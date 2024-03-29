// /proc/interrupts metrics

package lsvmi

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/eparparita/linux-stats-victoriametrics-importer/procfs"
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
	PROC_INTERRUPTS_CPU_LABEL_NAME = "cpu"

	// METRIC{instance="INSTANCE",hostname="HOSTNAME",irq="IRQ",controller="CONTROLLER",hw_interrupt="HW_INTERRUPT",dev="DEV"}:
	PROC_INTERRUPTS_INFO_METRIC                  = "proc_interrupts_info"
	PROC_INTERRUPTS_INFO_IRQ_LABEL_NAME          = PROC_INTERRUPTS_IRQ_LABEL_NAME
	PROC_INTERRUPTS_INFO_CONTROLLER_LABEL_NAME   = "controller"
	PROC_INTERRUPTS_INFO_HW_INTERRUPT_LABEL_NAME = "hw_interrupt"
	PROC_INTERRUPTS_INFO_DEV_LABEL_NAME          = "dev"

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
	crtIndex int
	// Current cycle#:
	cycleNum int
	// Full metric factor:
	fullMetricsFactor int

	// Delta metrics are generated with skip-zero-after-zero rule, i.e. if the
	// current and previous deltas are both zero, then the current metric is
	// skipped, save for full cycles. Keep track of zero deltas, indexed by irq
	// and counter index (see procfs.Interrupts.Irq[].Counters)
	zeroDeltaMap map[string][]bool

	// Delta metrics prefix cache (i.e. all but CPU#), indexed by IRQ:
	// 		`METRIC{instance="INSTANCE",hostname="HOSTNAME",irq="IRQ" ...
	deltaMetricsPrefixCache map[string][]byte

	// Delta metrics suffix cache (CPU#), indexed by counter#:
	//		... cpu="CPU"} `
	deltaMetricsSuffixCache [][]byte

	// Info metrics cache, indexed by IRQ:
	infoMetricsCache map[string][]byte

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
		id:                      PROC_INTERRUPTS_METRICS_ID,
		interval:                interval,
		zeroDeltaMap:            make(map[string][]bool),
		deltaMetricsPrefixCache: make(map[string][]byte),
		infoMetricsCache:        make(map[string][]byte),
		fullMetricsFactor:       procInterruptsMetricsCfg.FullMetricsFactor,
		tsSuffixBuf:             &bytes.Buffer{},
	}

	procInterruptsMetricsLog.Infof("id=%s", procInterruptsMetrics.id)
	procInterruptsMetricsLog.Infof("interval=%s", procInterruptsMetrics.interval)
	procInterruptsMetricsLog.Infof("full_metrics_factor=%d", procInterruptsMetrics.fullMetricsFactor)
	return procInterruptsMetrics, nil
}

func (pim *ProcInterruptsMetrics) updateDeltaMetricsPrefixCache(irq string) {
	instance, hostname := GlobalInstance, GlobalHostname
	if pim.instance != "" {
		instance = pim.instance
	}
	if pim.hostname != "" {
		hostname = pim.hostname
	}
	pim.deltaMetricsPrefixCache[irq] = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s",%s="%s"`,
		PROC_INTERRUPTS_INFO_METRIC,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
		PROC_INTERRUPTS_IRQ_LABEL_NAME, irq,
	))
}

func (pim *ProcInterruptsMetrics) updateDeltaMetricsSuffixCache() {
	interrupts := pim.procInterrupts[pim.crtIndex]
	if interrupts.CpuList == nil {
		// No CPU is missing, i.e. CPU# == counter index#
		numCpus := interrupts.NumCounters
		pim.deltaMetricsSuffixCache = make([][]byte, numCpus)
		for i := 0; i < numCpus; i++ {
			pim.deltaMetricsSuffixCache[i] = []byte(fmt.Sprintf(
				`,%s="%d"} `, // N.B. include space before value
				PROC_INTERRUPTS_CPU_LABEL_NAME, i,
			))
		}
	} else {
		pim.deltaMetricsSuffixCache = make([][]byte, len(interrupts.CpuList))
		for i, cpu := range interrupts.CpuList {
			pim.deltaMetricsSuffixCache[i] = []byte(fmt.Sprintf(
				`,%s="%d"} `, // N.B. include space before value
				PROC_INTERRUPTS_CPU_LABEL_NAME, cpu,
			))
		}
	}
}

// Update an info metric cache entry and return the previous content, if
// changed. It is possible, yet unlikely, that update was triggered by a change
// in the line underlying the IRQ but after parsing, the relevant parts were the
// same.
func (pim *ProcInterruptsMetrics) updateInfoMetricsCache(
	irq string,
	irqInfo *procfs.InterruptsIrqInfo,
) []byte {
	instance, hostname := GlobalInstance, GlobalHostname
	if pim.instance != "" {
		instance = pim.instance
	}
	if pim.hostname != "" {
		hostname = pim.hostname
	}

	prevInfoMetric := pim.infoMetricsCache[irq]
	updatedInfoMetrics := []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s",%s="%s",%s="%s,%s="%s",%s="%s"} `, // N.B. the space before the value is included!
		PROC_INTERRUPTS_DELTA_METRIC,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
		PROC_INTERRUPTS_INFO_IRQ_LABEL_NAME, irq,
		PROC_INTERRUPTS_INFO_CONTROLLER_LABEL_NAME, irqInfo.Controller,
		PROC_INTERRUPTS_INFO_HW_INTERRUPT_LABEL_NAME, irqInfo.HWInterrupt,
		PROC_INTERRUPTS_INFO_DEV_LABEL_NAME, irqInfo.Devices,
	))
	pim.infoMetricsCache[irq] = updatedInfoMetrics

	if bytes.Equal(prevInfoMetric, updatedInfoMetrics) {
		// No material change:
		return nil
	}
	return prevInfoMetric
}

func (pim *ProcInterruptsMetrics) updateMetricsCache() {
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

func (pim *ProcInterruptsMetrics) generateMetrics(buf *bytes.Buffer) int {
	metricsCount := 0
	crtProcInterrupts, prevProcInterrupts := pim.procInterrupts[pim.crtIndex], pim.procInterrupts[1-pim.crtIndex]

	// All metrics are deltas, so must have previous stats:
	if prevProcInterrupts != nil {
		crtInfo := crtProcInterrupts.Info

		crtTs, prevTs := pim.procInterruptsTs[pim.crtIndex], pim.procInterruptsTs[1-pim.crtIndex]
		pim.tsSuffixBuf.Reset()
		fmt.Fprintf(
			pim.tsSuffixBuf, " %d\n", crtTs.UnixMilli(),
		)
		promTs := pim.tsSuffixBuf.Bytes()

		deltaSec := crtTs.Sub(prevTs).Seconds()

		// Counter deltas:
		fullMetrics := pim.cycleNum == 0
		deltaFullMetrics := fullMetrics || crtInfo.CpuListChanged

		// If there was a CPU list change, then build CPU# -> counter index# for
		// the previous list. Given a current counter index#, first map into
		// CPU# using the current CpuList then map it into prev counter index#
		// using the map.
		var prevCpuNumToCounterIndexMap map[int]int

		if crtInfo.CpuListChanged {
			// Delta metrics suffix holds the CPU#, so it needs updating:
			pim.updateDeltaMetricsSuffixCache()

			// Build the prev CPU# -> counter index# map:
			prevCpuNumToCounterIndexMap = make(map[int]int)
			if prevProcInterrupts.CpuList == nil {
				for i := 0; i < prevProcInterrupts.NumCounters; i++ {
					prevCpuNumToCounterIndexMap[i] = i
				}
			} else {
				for i, cpu := range prevProcInterrupts.CpuList {
					prevCpuNumToCounterIndexMap[i] = cpu
				}
			}
		}

		for irq, crtCounters := range crtProcInterrupts.Counters {
			prevCounters := prevProcInterrupts.Counters[irq]
			if prevCounters == nil {
				// This is a new IRQ, no deltas for it:
				continue
			}

			deltaMetricPrefix := pim.deltaMetricsPrefixCache[irq]
			if deltaMetricPrefix == nil {
				pim.updateDeltaMetricsPrefixCache(irq)
				deltaMetricPrefix = pim.deltaMetricsPrefixCache[irq]
			}

			if crtInfo.CpuListChanged {
				// The zeroDeltaMap should be reinitialized for this IRQ:
				pim.zeroDeltaMap[irq] = make([]bool, crtProcInterrupts.NumCounters)
			}
			irqZeroDelta := pim.zeroDeltaMap[irq]

			for crtI, crtCounter := range crtCounters {
				prevI, ok := crtI, true
				if prevCpuNumToCounterIndexMap != nil {
					prevI, ok = prevCpuNumToCounterIndexMap[crtProcInterrupts.CpuList[crtI]]
					if !ok {
						// This CPU didn't exist before, so no delta for it:
						continue
					}
				}
				delta := crtCounter - prevCounters[prevI]

				if deltaFullMetrics || delta > 0 || !irqZeroDelta[crtI] {
					buf.Write(deltaMetricPrefix)
					buf.Write(pim.deltaMetricsSuffixCache[crtI])
					buf.WriteString(strconv.FormatUint(delta, 10))
					buf.Write(promTs)
					metricsCount++
				}
				irqZeroDelta[crtI] = delta == 0
			}
		}

		// Info:
		if fullMetrics || crtInfo.IrqChanged {
			for irq, irqInfo := range crtInfo.IrqInfo {
				if irqInfo.Changed {
					prevMetric := pim.updateInfoMetricsCache(irq, irqInfo)
					if prevMetric != nil {
						buf.Write(prevMetric)
						buf.WriteByte('0')
						buf.Write(promTs)
						metricsCount++
					}
				}
				if fullMetrics || irqInfo.Changed {
					buf.Write(pim.infoMetricsCache[irq])
					buf.WriteByte('1')
					buf.Write(promTs)
					metricsCount++
				}
			}
		}

		// Interval:
		if pim.intervalMetric == nil {
			pim.updateMetricsCache()
		}
		buf.Write(pim.intervalMetric)
		buf.WriteString(strconv.FormatFloat(deltaSec, 'f', 6, 64))
		buf.Write(promTs)
		metricsCount++
	}

	// Toggle the buffers, update the collection time and the cycle#:
	pim.crtIndex = 1 - pim.crtIndex
	if pim.cycleNum++; pim.cycleNum >= pim.fullMetricsFactor {
		pim.cycleNum = 0
	}

	return metricsCount
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

	crtProcInterrupts := pim.procInterrupts[pim.crtIndex]
	if crtProcInterrupts == nil {
		prevProcInterrupts := pim.procInterrupts[1-pim.crtIndex]
		if prevProcInterrupts != nil {
			crtProcInterrupts = prevProcInterrupts.Clone(false)
		} else {
			procfsRoot := GlobalProcfsRoot
			if pim.procfsRoot != "" {
				procfsRoot = pim.procfsRoot
			}
			crtProcInterrupts = procfs.NewInterrupts(procfsRoot)
		}
		pim.procInterrupts[pim.crtIndex] = crtProcInterrupts
	}
	err := crtProcInterrupts.Parse()
	if err != nil {
		procInterruptsMetricsLog.Warnf("%v: proc interrupts metrics will be disabled", err)
		return false
	}
	pim.procInterruptsTs[pim.crtIndex] = timeNowFn()

	buf := metricsQueue.GetBuf()
	metricsCount := pim.generateMetrics(buf)
	byteCount := buf.Len()
	metricsQueue.QueueBuf(buf)

	GlobalMetricsGeneratorStatsContainer.Update(
		pim.id, uint64(metricsCount), uint64(byteCount),
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
