// /proc/interrupts metrics

package lsvmi

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/emypar/linux-stats-victoriametrics-importer/procfs"
)

const (
	PROC_INTERRUPTS_METRICS_CONFIG_INTERVAL_DEFAULT            = "1s"
	PROC_INTERRUPTS_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT = 15

	// This generator id:
	PROC_INTERRUPTS_METRICS_ID = "proc_interrupts_metrics"
)

const (
	// METRIC{instance="INSTANCE",hostname="HOSTNAME",irq="IRQ",cpu="CPU"}:
	PROC_INTERRUPTS_DELTA_METRIC   = "proc_interrupts_delta"
	PROC_INTERRUPTS_IRQ_LABEL_NAME = "irq"
	PROC_INTERRUPTS_DEV_LABEL_NAME = "dev"
	PROC_INTERRUPTS_CPU_LABEL_NAME = "cpu"

	// METRIC{instance="INSTANCE",hostname="HOSTNAME",irq="IRQ",controller="CONTROLLER",hw_interrupt="HW_INTERRUPT",dev="DEV"}:
	PROC_INTERRUPTS_INFO_METRIC                  = "proc_interrupts_info"
	PROC_INTERRUPTS_INFO_IRQ_LABEL_NAME          = PROC_INTERRUPTS_IRQ_LABEL_NAME
	PROC_INTERRUPTS_INFO_CONTROLLER_LABEL_NAME   = "controller"
	PROC_INTERRUPTS_INFO_HW_INTERRUPT_LABEL_NAME = "hw_interrupt"
	PROC_INTERRUPTS_INFO_DEV_LABEL_NAME          = PROC_INTERRUPTS_DEV_LABEL_NAME

	// Interval since last generation, i.e. the interval underlying the deltas.
	// Normally this should be close to scan interval, but this is the actual
	// value, rather than the desired one:
	PROC_INTERRUPTS_INTERVAL_METRIC_NAME = "proc_interrupts_metrics_delta_sec"
)

var procInterruptsMetricsLog = NewCompLogger(PROC_INTERRUPTS_METRICS_ID)

type ProcInterruptsMetricsConfig struct {
	// How often to generate the metrics in time.ParseDuration() format:
	Interval string `yaml:"interval"`
	// Normally metrics are generated only if there is a change in value from
	// the previous scan. However every N cycles the full set is generated. Use
	// 0 to generate full metrics every cycle.
	FullMetricsFactor int `yaml:"full_metrics_factor"`
}

func DefaultProcInterruptsMetricsConfig() *ProcInterruptsMetricsConfig {
	return &ProcInterruptsMetricsConfig{
		Interval:          PROC_INTERRUPTS_METRICS_CONFIG_INTERVAL_DEFAULT,
		FullMetricsFactor: PROC_INTERRUPTS_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT,
	}
}

// Group together all data that is to be indexed by IRQ, this way only one
// lookup is required:
type ProcInterruptsMetricsIrqData struct {
	// Current cycle#:
	cycleNum int

	// Delta metric prefix (i.e. all but CPU#):
	// 		`METRIC{instance="INSTANCE",hostname="HOSTNAME",irq="IRQ" ...
	deltaMetricPrefix []byte

	// Info metric:
	infoMetric []byte

	// Delta metrics are generated with skip-zero-after-zero rule, i.e. if the
	// current and previous deltas are both zero, then the current metric is
	// skipped, save for full cycles. Keep track of zero deltas, indexed by
	// counter index (see procfs.Interrupts.Irq[].Counters)
	zeroDelta []bool
}

type ProcInterruptsMetrics struct {
	// id/task_id:
	id string
	// Scan interval:
	interval time.Duration
	// Dual storage for parsed stats used as previous, current:
	procInterrupts [2]*procfs.Interrupts
	// Timestamp when the stats were collected:
	procInterruptsTs [2]time.Time
	// Index for current stats, toggled after each use:
	currIndex int
	// Full metric factor:
	fullMetricsFactor int

	// Data indexed by IRQ:
	irqDataCache map[string]*ProcInterruptsMetricsIrqData

	// Delta metrics suffix cache (CPU#), indexed by counter#:
	//              ... cpu="CPU"} `
	deltaMetricsSuffixCache [][]byte

	// Interval metric:
	intervalMetric []byte

	// A buffer for the timestamp suffix:
	tsSuffixBuf *bytes.Buffer

	// The following are needed for testing only. Left to their default values,
	// the usual objects will be used.
	instance, hostname string
	timeNowFn          func() time.Time
	metricsQueue       MetricsQueue
	procfsRoot         string
}

func NewProcInterruptsMetrics(cfg any) (*ProcInterruptsMetrics, error) {
	var (
		err                      error
		procInterruptsMetricsCfg *ProcInterruptsMetricsConfig
	)

	switch cfg := cfg.(type) {
	case *LsvmiConfig:
		procInterruptsMetricsCfg = cfg.ProcInterruptsMetricsConfig
	case *ProcInterruptsMetricsConfig:
		procInterruptsMetricsCfg = cfg
	case nil:
		procInterruptsMetricsCfg = DefaultProcInterruptsMetricsConfig()
	default:
		return nil, fmt.Errorf("NewProcInterruptsMetrics: %T invalid config type", cfg)
	}

	interval, err := time.ParseDuration(procInterruptsMetricsCfg.Interval)
	if err != nil {
		return nil, err
	}
	procInterruptsMetrics := &ProcInterruptsMetrics{
		id:                PROC_INTERRUPTS_METRICS_ID,
		interval:          interval,
		irqDataCache:      make(map[string]*ProcInterruptsMetricsIrqData),
		fullMetricsFactor: procInterruptsMetricsCfg.FullMetricsFactor,
		tsSuffixBuf:       &bytes.Buffer{},
	}

	procInterruptsMetricsLog.Infof("id=%s", procInterruptsMetrics.id)
	procInterruptsMetricsLog.Infof("interval=%s", procInterruptsMetrics.interval)
	procInterruptsMetricsLog.Infof("full_metrics_factor=%d", procInterruptsMetrics.fullMetricsFactor)
	return procInterruptsMetrics, nil
}

// Update the IRQ data every time a new IRQ is discovered or there is a change
// to an existent IRQ:
func (pim *ProcInterruptsMetrics) updateIrqDataCache(irq string) *ProcInterruptsMetricsIrqData {
	instance, hostname := GlobalInstance, GlobalHostname
	if pim.instance != "" {
		instance = pim.instance
	}
	if pim.hostname != "" {
		hostname = pim.hostname
	}

	interrupts := pim.procInterrupts[pim.currIndex]
	irqInfo := interrupts.Info.IrqInfo[irq]

	irqData, ok := pim.irqDataCache[irq]
	if !ok {
		irqData = &ProcInterruptsMetricsIrqData{
			cycleNum:  initialCycleNum.Get(pim.fullMetricsFactor),
			zeroDelta: make([]bool, interrupts.NumCounters),
		}
		pim.irqDataCache[irq] = irqData
	}

	irqData.deltaMetricPrefix = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s",%s="%s",%s="%s"`,
		PROC_INTERRUPTS_DELTA_METRIC,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
		PROC_INTERRUPTS_IRQ_LABEL_NAME, irq,
		PROC_INTERRUPTS_DEV_LABEL_NAME, irqInfo.Devices,
	))

	irqData.infoMetric = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s",%s="%s",%s="%s",%s="%s",%s="%s"} `, // N.B. the space before the value is included!
		PROC_INTERRUPTS_INFO_METRIC,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
		PROC_INTERRUPTS_INFO_IRQ_LABEL_NAME, irq,
		PROC_INTERRUPTS_INFO_CONTROLLER_LABEL_NAME, irqInfo.Controller,
		PROC_INTERRUPTS_INFO_HW_INTERRUPT_LABEL_NAME, irqInfo.HWInterrupt,
		PROC_INTERRUPTS_INFO_DEV_LABEL_NAME, irqInfo.Devices,
	))

	return irqData
}

// Update suffix cache every time there is a change to the CPU list; return the
// mapping from current to previous counter index such that they target the same
// CPU#:
func (pim *ProcInterruptsMetrics) updateCpuList() map[int]int {
	curr_interrupts, prev_interrupts := pim.procInterrupts[pim.currIndex], pim.procInterrupts[1-pim.currIndex]

	// Suffix cache:
	if curr_interrupts.CpuList == nil {
		// No CPU is missing, i.e. CPU# == counter index#
		numCpus := curr_interrupts.NumCounters
		pim.deltaMetricsSuffixCache = make([][]byte, numCpus)
		for i := 0; i < numCpus; i++ {
			pim.deltaMetricsSuffixCache[i] = []byte(fmt.Sprintf(
				`,%s="%d"} `, // N.B. include space before value
				PROC_INTERRUPTS_CPU_LABEL_NAME, i,
			))
		}
	} else {
		pim.deltaMetricsSuffixCache = make([][]byte, len(curr_interrupts.CpuList))
		for i, cpu := range curr_interrupts.CpuList {
			pim.deltaMetricsSuffixCache[i] = []byte(fmt.Sprintf(
				`,%s="%d"} `, // N.B. include space before value
				PROC_INTERRUPTS_CPU_LABEL_NAME, cpu,
			))
		}
	}

	// Mapping:
	currToPrevCounterIndexMap := make(map[int]int)
	prevCpuNumToCounterIndexMap := make(map[int]int)
	if prev_interrupts.CpuList == nil {
		for i := 0; i < prev_interrupts.NumCounters; i++ {
			prevCpuNumToCounterIndexMap[i] = i
		}
	} else {
		for i, cpuNum := range prev_interrupts.CpuList {
			prevCpuNumToCounterIndexMap[cpuNum] = i
		}
	}
	if curr_interrupts.CpuList == nil {
		for i := 0; i < curr_interrupts.NumCounters; i++ {
			if prevI, ok := prevCpuNumToCounterIndexMap[i]; ok {
				currToPrevCounterIndexMap[i] = prevI
			}
		}
	} else {
		for currI, cpuNum := range curr_interrupts.CpuList {
			if prevI, ok := prevCpuNumToCounterIndexMap[cpuNum]; ok {
				currToPrevCounterIndexMap[currI] = prevI
			}
		}
	}
	return currToPrevCounterIndexMap
}

func (pim *ProcInterruptsMetrics) updateIntervalMetricsCache() {
	instance, hostname := GlobalInstance, GlobalHostname
	if pim.instance != "" {
		instance = pim.instance
	}
	if pim.hostname != "" {
		hostname = pim.hostname
	}
	pim.intervalMetric = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s"} `, // N.B. include space before val
		PROC_INTERRUPTS_INTERVAL_METRIC_NAME,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
	))
}

func (pim *ProcInterruptsMetrics) generateMetrics(buf *bytes.Buffer) (int, int) {
	actualMetricsCount := 0
	currProcInterrupts, prevProcInterrupts := pim.procInterrupts[pim.currIndex], pim.procInterrupts[1-pim.currIndex]

	// All metrics are deltas, so must have previous stats:
	if prevProcInterrupts != nil {
		currInfo := currProcInterrupts.Info

		currTs, prevTs := pim.procInterruptsTs[pim.currIndex], pim.procInterruptsTs[1-pim.currIndex]
		pim.tsSuffixBuf.Reset()
		fmt.Fprintf(
			pim.tsSuffixBuf, " %d\n", currTs.UnixMilli(),
		)
		promTs := pim.tsSuffixBuf.Bytes()
		deltaSec := currTs.Sub(prevTs).Seconds()

		// If there was a CPU list change, then build curr to previous counter
		// index# map such that they refer to the same CPU#.
		var currToPrevCounterIndexMap map[int]int = nil

		if currInfo.CpuListChanged {
			currToPrevCounterIndexMap = pim.updateCpuList()
		} else if pim.deltaMetricsSuffixCache == nil {
			// 1st time, no mapping required:
			pim.updateCpuList()
		}

		for irq, currCounters := range currProcInterrupts.Counters {
			prevCounters := prevProcInterrupts.Counters[irq]
			if prevCounters == nil {
				// This is a new IRQ, no deltas for it:
				continue
			}

			currIrqInfo := currInfo.IrqInfo[irq]

			irqData := pim.irqDataCache[irq]
			fullMetrics := irqData == nil || // 1st time IRQ
				currIrqInfo.Changed || // something changed
				irqData.cycleNum == 0 // regular full cycle
			var prevInfoMetric []byte = nil
			if irqData == nil {
				// 1st time IRQ:
				irqData = pim.updateIrqDataCache(irq)
			} else if currIrqInfo.Changed {
				// Info changed, may have to 0 the previous info metric:
				prevInfoMetric = irqData.infoMetric
				irqData = pim.updateIrqDataCache(irq)
			}

			if currInfo.CpuListChanged {
				// Previous zero delta is no longer valid:
				irqData.zeroDelta = make([]bool, currProcInterrupts.NumCounters)
			}

			deltaMetricPrefix := irqData.deltaMetricPrefix
			irqZeroDelta := irqData.zeroDelta

			// Delta metrics:
			for currI, currCounter := range currCounters {
				prevI, ok := currI, true
				if currToPrevCounterIndexMap != nil {
					prevI, ok = currToPrevCounterIndexMap[currI]
					if !ok {
						// This CPU didn't exist before, so no delta for it:
						continue
					}
				}
				delta := currCounter - prevCounters[prevI]
				if fullMetrics || delta > 0 || !irqZeroDelta[currI] {
					buf.Write(deltaMetricPrefix)
					buf.Write(pim.deltaMetricsSuffixCache[currI])
					buf.WriteString(strconv.FormatUint(delta, 10))
					buf.Write(promTs)
					actualMetricsCount++
				}
				irqZeroDelta[currI] = delta == 0
			}

			// Info metric:
			if fullMetrics {
				currInfoMetric := irqData.infoMetric
				if prevInfoMetric != nil && !bytes.Equal(prevInfoMetric, currInfoMetric) {
					// Must 0 previous info metric:
					buf.Write(prevInfoMetric)
					buf.WriteByte('0')
					buf.Write(promTs)
					actualMetricsCount++
				}
				buf.Write(currInfoMetric)
				buf.WriteByte('1')
				buf.Write(promTs)
				actualMetricsCount++
			}

			// Update cycle#:
			if irqData.cycleNum++; irqData.cycleNum >= pim.fullMetricsFactor {
				irqData.cycleNum = 0
			}
		}

		// Clear info for removed IRQ's, if any:
		if len(pim.irqDataCache) != len(currInfo.IrqInfo) {
			for irq, prevIrqData := range pim.irqDataCache {
				if _, ok := currProcInterrupts.Counters[irq]; !ok {
					buf.Write(prevIrqData.infoMetric)
					buf.WriteByte('0')
					buf.Write(promTs)
					actualMetricsCount++
					delete(pim.irqDataCache, irq)
				}
			}
		}

		// Interval metric:
		if pim.intervalMetric == nil {
			pim.updateIntervalMetricsCache()
		}
		buf.Write(pim.intervalMetric)
		buf.WriteString(strconv.FormatFloat(deltaSec, 'f', 6, 64))
		buf.Write(promTs)
		actualMetricsCount++
	}

	// The total number of metrics:
	//		delta metrics#: number of IRQs * number of counter
	//		info metrics#:  number of IRQs
	//		interval metric#: 1
	totalMetricsCount := len(currProcInterrupts.Counters)*(currProcInterrupts.NumCounters+1) + 1

	// Toggle the buffers:
	pim.currIndex = 1 - pim.currIndex

	return actualMetricsCount, totalMetricsCount
}

// Satisfy the TaskActivity interface:
func (pim *ProcInterruptsMetrics) Execute() bool {
	timeNowFn := time.Now
	if pim.timeNowFn != nil {
		timeNowFn = pim.timeNowFn
	}

	metricsQueue := GlobalMetricsQueue
	if pim.metricsQueue != nil {
		metricsQueue = pim.metricsQueue
	}

	currProcInterrupts := pim.procInterrupts[pim.currIndex]
	if currProcInterrupts == nil {
		prevProcInterrupts := pim.procInterrupts[1-pim.currIndex]
		if prevProcInterrupts != nil {
			currProcInterrupts = prevProcInterrupts.Clone(false)
		} else {
			procfsRoot := GlobalProcfsRoot
			if pim.procfsRoot != "" {
				procfsRoot = pim.procfsRoot
			}
			currProcInterrupts = procfs.NewInterrupts(procfsRoot)
		}
		pim.procInterrupts[pim.currIndex] = currProcInterrupts
	}
	err := currProcInterrupts.Parse()
	if err != nil {
		procInterruptsMetricsLog.Warnf("%v: proc interrupts metrics will be disabled", err)
		return false
	}
	pim.procInterruptsTs[pim.currIndex] = timeNowFn()

	buf := metricsQueue.GetBuf()
	actualMetricsCount, totalMetricsCount := pim.generateMetrics(buf)
	byteCount := buf.Len()
	metricsQueue.QueueBuf(buf)

	GlobalMetricsGeneratorStatsContainer.Update(
		pim.id, uint64(actualMetricsCount), uint64(totalMetricsCount), uint64(byteCount),
	)

	return true
}

// Define and register the task builder:
func ProcInterruptsMetricsTaskBuilder(cfg *LsvmiConfig) ([]*Task, error) {
	pim, err := NewProcInterruptsMetrics(cfg)
	if err != nil {
		return nil, err
	}
	if pim.interval <= 0 {
		procInterruptsMetricsLog.Infof(
			"interval=%s, metrics disabled", pim.interval,
		)
		return nil, nil
	}
	tasks := []*Task{
		NewTask(pim.id, pim.interval, pim),
	}
	return tasks, nil
}

func init() {
	TaskBuilders.Register(ProcInterruptsMetricsTaskBuilder)
}
