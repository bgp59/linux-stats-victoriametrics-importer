// Metrics based on /proc/softirqs

package lsvmi

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/bgp59/linux-stats-victoriametrics-importer/procfs"
)

const (
	PROC_SOFTIRQS_METRICS_CONFIG_INTERVAL_DEFAULT            = "1s"
	PROC_SOFTIRQS_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT = 15

	// This generator id:
	PROC_SOFTIRQS_METRICS_ID = "proc_softirqs_metrics"
)

const (
	// METRIC{instance="INSTANCE",hostname="HOSTNAME",irq="IRQ",cpu="CPU"}:
	PROC_SOFTIRQS_DELTA_METRIC   = "proc_softirqs_delta"
	PROC_SOFTIRQS_IRQ_LABEL_NAME = "irq"
	PROC_SOFTIRQS_CPU_LABEL_NAME = "cpu"

	// METRIC{instance="INSTANCE",hostname="HOSTNAME",irq="IRQ"}:
	PROC_SOFTIRQS_INFO_METRIC         = "proc_softirqs_info"
	PROC_SOFTIRQS_INFO_IRQ_LABEL_NAME = PROC_SOFTIRQS_IRQ_LABEL_NAME

	// Interval since last generation, i.e. the interval underlying the deltas.
	// Normally this should be close to scan interval, but this is the actual
	// value, rather than the desired one:
	PROC_SOFTIRQS_INTERVAL_METRIC = "proc_softirqs_metrics_delta_sec"
)

var procSoftirqsMetricsLog = NewCompLogger(PROC_SOFTIRQS_METRICS_ID)

type ProcSoftirqsMetricsConfig struct {
	// How often to generate the metrics in time.ParseDuration() format:
	Interval string `yaml:"interval"`
	// Normally metrics are generated only if there is a change in value from
	// the previous scan. However every N cycles the full set is generated. Use
	// 0 to generate full metrics every cycle.
	FullMetricsFactor int `yaml:"full_metrics_factor"`
}

func DefaultProcSoftirqsMetricsConfig() *ProcSoftirqsMetricsConfig {
	return &ProcSoftirqsMetricsConfig{
		Interval:          PROC_SOFTIRQS_METRICS_CONFIG_INTERVAL_DEFAULT,
		FullMetricsFactor: PROC_SOFTIRQS_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT,
	}
}

// Group together all data that is to be indexed by IRQ, this way only one
// lookup is required:
type ProcSoftirqsMetricsIrqData struct {
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
	// counter index (see procfs.Softirqs.SoftirqInfo[].Counters)
	zeroDelta []bool
}

type ProcSoftirqsMetrics struct {
	// id/task_id:
	id string
	// Scan interval:
	interval time.Duration
	// Dual storage for parsed stats used as previous, current:
	procSoftirqs [2]*procfs.Softirqs
	// Timestamp when the stats were collected:
	procSoftirqsTs [2]time.Time
	// Index for current stats, toggled after each use:
	currIndex int
	// Full metric factor:
	fullMetricsFactor int

	// Data indexed by IRQ:
	irqDataCache map[string]*ProcSoftirqsMetricsIrqData

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

func NewProcSoftirqsMetrics(cfg any) (*ProcSoftirqsMetrics, error) {
	var (
		err                    error
		procSoftirqsMetricsCfg *ProcSoftirqsMetricsConfig
	)

	switch cfg := cfg.(type) {
	case *LsvmiConfig:
		procSoftirqsMetricsCfg = cfg.ProcSoftirqsMetricsConfig
	case *ProcSoftirqsMetricsConfig:
		procSoftirqsMetricsCfg = cfg
	case nil:
		procSoftirqsMetricsCfg = DefaultProcSoftirqsMetricsConfig()
	default:
		return nil, fmt.Errorf("NewProcSoftirqsMetrics: %T invalid config type", cfg)
	}

	interval, err := time.ParseDuration(procSoftirqsMetricsCfg.Interval)
	if err != nil {
		return nil, err
	}
	procSoftirqsMetrics := &ProcSoftirqsMetrics{
		id:                PROC_SOFTIRQS_METRICS_ID,
		interval:          interval,
		irqDataCache:      make(map[string]*ProcSoftirqsMetricsIrqData),
		fullMetricsFactor: procSoftirqsMetricsCfg.FullMetricsFactor,
		tsSuffixBuf:       &bytes.Buffer{},
	}

	procSoftirqsMetricsLog.Infof("id=%s", procSoftirqsMetrics.id)
	procSoftirqsMetricsLog.Infof("interval=%s", procSoftirqsMetrics.interval)
	procSoftirqsMetricsLog.Infof("full_metrics_factor=%d", procSoftirqsMetrics.fullMetricsFactor)
	return procSoftirqsMetrics, nil
}

// Update the IRQ data every time a new IRQ is discovered:
func (psirqm *ProcSoftirqsMetrics) updateIrqDataCache(irq string) *ProcSoftirqsMetricsIrqData {
	instance, hostname := GlobalInstance, GlobalHostname
	if psirqm.instance != "" {
		instance = psirqm.instance
	}
	if psirqm.hostname != "" {
		hostname = psirqm.hostname
	}

	softirqs := psirqm.procSoftirqs[psirqm.currIndex]
	irqData, ok := psirqm.irqDataCache[irq]
	if !ok {
		irqData = &ProcSoftirqsMetricsIrqData{
			cycleNum:  initialCycleNum.Get(psirqm.fullMetricsFactor),
			zeroDelta: make([]bool, softirqs.NumCounters),
		}
		psirqm.irqDataCache[irq] = irqData
	}

	irqData.deltaMetricPrefix = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s",%s="%s"`,
		PROC_SOFTIRQS_DELTA_METRIC,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
		PROC_SOFTIRQS_IRQ_LABEL_NAME, irq,
	))

	irqData.infoMetric = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s",%s="%s"} `, // N.B. the space before the value is included!
		PROC_SOFTIRQS_INFO_METRIC,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
		PROC_SOFTIRQS_INFO_IRQ_LABEL_NAME, irq,
	))

	return irqData
}

// Update suffix cache every time there is a change to the CPU list; return the
// mapping from current to previous counter index such that they target the same
// CPU#:
func (psirqm *ProcSoftirqsMetrics) updateCpuList() map[int]int {
	curr_softirqs, prev_softirqs := psirqm.procSoftirqs[psirqm.currIndex], psirqm.procSoftirqs[1-psirqm.currIndex]

	// Suffix cache:
	if curr_softirqs.CpuList == nil {
		// No CPU is missing, i.e. CPU# == counter index#
		numCpus := curr_softirqs.NumCounters
		psirqm.deltaMetricsSuffixCache = make([][]byte, numCpus)
		for i := 0; i < numCpus; i++ {
			psirqm.deltaMetricsSuffixCache[i] = []byte(fmt.Sprintf(
				`,%s="%d"} `, // N.B. include space before value
				PROC_SOFTIRQS_CPU_LABEL_NAME, i,
			))
		}
	} else {
		psirqm.deltaMetricsSuffixCache = make([][]byte, len(curr_softirqs.CpuList))
		for i, cpu := range curr_softirqs.CpuList {
			psirqm.deltaMetricsSuffixCache[i] = []byte(fmt.Sprintf(
				`,%s="%d"} `, // N.B. include space before value
				PROC_SOFTIRQS_CPU_LABEL_NAME, cpu,
			))
		}
	}

	// Mapping:
	currToPrevCounterIndexMap := make(map[int]int)
	prevCpuNumToCounterIndexMap := make(map[int]int)
	if prev_softirqs.CpuList == nil {
		for i := 0; i < prev_softirqs.NumCounters; i++ {
			prevCpuNumToCounterIndexMap[i] = i
		}
	} else {
		for i, cpuNum := range prev_softirqs.CpuList {
			prevCpuNumToCounterIndexMap[cpuNum] = i
		}
	}
	if curr_softirqs.CpuList == nil {
		for i := 0; i < curr_softirqs.NumCounters; i++ {
			if prevI, ok := prevCpuNumToCounterIndexMap[i]; ok {
				currToPrevCounterIndexMap[i] = prevI
			}
		}
	} else {
		for currI, cpuNum := range curr_softirqs.CpuList {
			if prevI, ok := prevCpuNumToCounterIndexMap[cpuNum]; ok {
				currToPrevCounterIndexMap[currI] = prevI
			}
		}
	}
	return currToPrevCounterIndexMap
}

func (psirqm *ProcSoftirqsMetrics) updateIntervalMetricsCache() {
	instance, hostname := GlobalInstance, GlobalHostname
	if psirqm.instance != "" {
		instance = psirqm.instance
	}
	if psirqm.hostname != "" {
		hostname = psirqm.hostname
	}
	psirqm.intervalMetric = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s"} `, // N.B. include space before val
		PROC_SOFTIRQS_INTERVAL_METRIC,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
	))
}

func (psirqm *ProcSoftirqsMetrics) generateMetrics(buf *bytes.Buffer) (int, int) {
	actualMetricsCount := 0
	currProcSoftirqs, prevProcSoftirqs := psirqm.procSoftirqs[psirqm.currIndex], psirqm.procSoftirqs[1-psirqm.currIndex]

	// All metrics are deltas, so must have previous stats:
	if prevProcSoftirqs != nil {
		currCounters := currProcSoftirqs.Counters
		prevCounters := prevProcSoftirqs.Counters

		currTs, prevTs := psirqm.procSoftirqsTs[psirqm.currIndex], psirqm.procSoftirqsTs[1-psirqm.currIndex]
		psirqm.tsSuffixBuf.Reset()
		fmt.Fprintf(
			psirqm.tsSuffixBuf, " %d\n", currTs.UnixMilli(),
		)
		promTs := psirqm.tsSuffixBuf.Bytes()

		deltaSec := currTs.Sub(prevTs).Seconds()

		// If there was a CPU list change, then build curr to previous counter
		// index# map such that they refer to the same CPU#.
		var currToPrevCounterIndexMap map[int]int = nil

		if currProcSoftirqs.CpuListChanged {
			currToPrevCounterIndexMap = psirqm.updateCpuList()
		} else if psirqm.deltaMetricsSuffixCache == nil {
			// 1st time, no need of mapping:
			psirqm.updateCpuList()
		}

		for irq, currIrqCounters := range currCounters {
			prevIrqCounters := prevCounters[irq]
			if prevIrqCounters == nil {
				// This is a new IRQ, no deltas for it:
				continue
			}

			irqData := psirqm.irqDataCache[irq]
			fullMetrics := irqData == nil || // 1st time IRQ
				irqData.cycleNum == 0 // regular full cycle
			if irqData == nil {
				// 1st time IRQ:
				irqData = psirqm.updateIrqDataCache(irq)
			}

			if currProcSoftirqs.CpuListChanged {
				// Previous zero delta is no longer valid:
				irqData.zeroDelta = make([]bool, currProcSoftirqs.NumCounters)
			}

			deltaMetricPrefix := irqData.deltaMetricPrefix
			irqZeroDelta := irqData.zeroDelta

			// Delta metrics:
			for currI, currCounter := range currIrqCounters {
				prevI, ok := currI, true
				if currToPrevCounterIndexMap != nil {
					prevI, ok = currToPrevCounterIndexMap[currI]
					if !ok {
						// This CPU didn't exist before, so no delta for it:
						continue
					}
				}
				delta := currCounter - prevIrqCounters[prevI]
				if fullMetrics || delta > 0 || !irqZeroDelta[currI] {
					buf.Write(deltaMetricPrefix)
					buf.Write(psirqm.deltaMetricsSuffixCache[currI])
					buf.WriteString(strconv.FormatUint(delta, 10))
					buf.Write(promTs)
					actualMetricsCount++
				}
				irqZeroDelta[currI] = delta == 0
			}

			// Info metric:
			if fullMetrics {
				currInfoMetric := irqData.infoMetric
				buf.Write(currInfoMetric)
				buf.WriteByte('1')
				buf.Write(promTs)
				actualMetricsCount++
			}

			// Update cycle#:
			if irqData.cycleNum++; irqData.cycleNum >= psirqm.fullMetricsFactor {
				irqData.cycleNum = 0
			}
		}

		// Clear info for removed IRQ's, if any:
		if len(psirqm.irqDataCache) != len(currCounters) {
			for irq, prevIrqData := range psirqm.irqDataCache {
				if _, ok := currCounters[irq]; !ok {
					buf.Write(prevIrqData.infoMetric)
					buf.WriteByte('0')
					buf.Write(promTs)
					actualMetricsCount++
					delete(psirqm.irqDataCache, irq)
				}
			}
		}

		// Interval metric:
		if psirqm.intervalMetric == nil {
			psirqm.updateIntervalMetricsCache()
		}
		buf.Write(psirqm.intervalMetric)
		buf.WriteString(strconv.FormatFloat(deltaSec, 'f', 6, 64))
		buf.Write(promTs)
		actualMetricsCount++
	}

	// The total number of metrics:
	//		delta metrics#: number of IRQs * number of counter
	//		info metrics#:  number of IRQs
	//		interval metric#: 1
	totalMetricsCount := len(currProcSoftirqs.Counters)*(currProcSoftirqs.NumCounters+1) + 1

	// Toggle the buffers:
	psirqm.currIndex = 1 - psirqm.currIndex

	return actualMetricsCount, totalMetricsCount
}

// Satisfy the TaskActivity interface:
func (psirqm *ProcSoftirqsMetrics) Execute() bool {
	timeNowFn := time.Now
	if psirqm.timeNowFn != nil {
		timeNowFn = psirqm.timeNowFn
	}

	metricsQueue := GlobalMetricsQueue
	if psirqm.metricsQueue != nil {
		metricsQueue = psirqm.metricsQueue
	}

	currProcSoftirqs := psirqm.procSoftirqs[psirqm.currIndex]
	if currProcSoftirqs == nil {
		prevProcSoftirqs := psirqm.procSoftirqs[1-psirqm.currIndex]
		if prevProcSoftirqs != nil {
			currProcSoftirqs = prevProcSoftirqs.Clone(false)
		} else {
			procfsRoot := GlobalProcfsRoot
			if psirqm.procfsRoot != "" {
				procfsRoot = psirqm.procfsRoot
			}
			currProcSoftirqs = procfs.NewSoftirqs(procfsRoot)
		}
		psirqm.procSoftirqs[psirqm.currIndex] = currProcSoftirqs
	}
	err := currProcSoftirqs.Parse()
	if err != nil {
		procSoftirqsMetricsLog.Warnf("%v: proc softirqs metrics will be disabled", err)
		return false
	}
	psirqm.procSoftirqsTs[psirqm.currIndex] = timeNowFn()

	buf := metricsQueue.GetBuf()
	actualMetricsCount, totalMetricsCount := psirqm.generateMetrics(buf)
	byteCount := buf.Len()
	metricsQueue.QueueBuf(buf)

	GlobalMetricsGeneratorStatsContainer.Update(
		psirqm.id, uint64(actualMetricsCount), uint64(totalMetricsCount), uint64(byteCount),
	)

	return true
}

// Define and register the task builder:
func ProcSoftirqsMetricsTaskBuilder(cfg *LsvmiConfig) ([]*Task, error) {
	psirqm, err := NewProcSoftirqsMetrics(cfg)
	if err != nil {
		return nil, err
	}
	if psirqm.interval <= 0 {
		procSoftirqsMetricsLog.Infof(
			"interval=%s, metrics disabled", psirqm.interval,
		)
		return nil, nil
	}
	tasks := []*Task{
		NewTask(psirqm.id, psirqm.interval, psirqm),
	}
	return tasks, nil
}

func init() {
	TaskBuilders.Register(ProcSoftirqsMetricsTaskBuilder)
}
