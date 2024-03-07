// // go:build exclude

// Metrics based on /proc/procStat

package lsvmi

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/eparparita/linux-stats-victoriametrics-importer/internal/utils"
	"github.com/eparparita/linux-stats-victoriametrics-importer/procfs"
)

const (
	PROC_STAT_METRICS_CONFIG_INTERVAL_DEFAULT            = "200ms"
	PROC_STAT_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT = 25
	PROC_STAT_METRICS_CONFIG_SCALE_CPU_ALL_DEFAULT       = true

	// This generator id:
	PROC_STAT_METRICS_ID = "proc_stat_metrics"
)

const (
	// %CPU metrics:
	PROC_STAT_CPU_PCT_METRIC = "proc_stat_cpu_pct"

	PROC_STAT_CPU_PCT_TYPE_LABEL_NAME = "type"
	PROC_STAT_CPU_PCT_TYPE_USER       = "user"
	PROC_STAT_CPU_PCT_TYPE_NICE       = "nice"
	PROC_STAT_CPU_PCT_TYPE_SYSTEM     = "system"
	PROC_STAT_CPU_PCT_TYPE_IDLE       = "idle"
	PROC_STAT_CPU_PCT_TYPE_IOWAIT     = "iowait"
	PROC_STAT_CPU_PCT_TYPE_IRQ        = "irq"
	PROC_STAT_CPU_PCT_TYPE_SOFTIRQ    = "softirq"
	PROC_STAT_CPU_PCT_TYPE_STEAL      = "steal"
	PROC_STAT_CPU_PCT_TYPE_GUEST      = "guest"
	PROC_STAT_CPU_PCT_TYPE_GUEST_NICE = "guest_nice"

	PROC_STAT_CPU_LABEL_NAME      = "cpu"
	PROC_STAT_CPU_ALL_LABEL_VALUE = "all"

	// System uptime will be based on btime:
	PROC_STAT_BTIME_METRIC  = "proc_stat_btime_sec"
	PROC_STAT_UPTIME_METRIC = "proc_stat_uptime_sec"

	// Other metrics:
	PROC_STAT_PAGE_IN_COUNT_DELTA_METRIC  = "proc_stat_page_in_count_delta"
	PROC_STAT_PAGE_OUT_COUNT_DELTA_METRIC = "proc_stat_page_out_count_delta"
	PROC_STAT_SWAP_IN_COUNT_DELTA_METRIC  = "proc_stat_swap_in_count_delta"
	PROC_STAT_SWAP_OUT_COUNT_DELTA_METRIC = "proc_stat_swap_out_count_delta"
	PROC_STAT_CTXT_COUNT_DELTA_METRIC     = "proc_stat_swap_ctxt_count_delta"
	PROC_STAT_PROCESSES_COUNT_METRIC      = "proc_stat_processes_count"
	PROC_STAT_PROCS_RUNNING_COUNT_METRIC  = "proc_stat_procs_running_count"
	PROC_STAT_PROCS_BLOCKED_COUNT_METRIC  = "proc_stat_procs_blocked_count"

	// Interval since last generation, i.e. the interval underlying the deltas.
	// Normally this should be close to scan interval, but this the actual
	// value, rather than the desired one:
	PROC_STAT_INTERVAL_METRIC_NAME = "proc_stat_metrics_delta_sec"
)

// Map procfs.Stat PROC_STAT_CPU_ indexes into type label value:
var procStatCpuIndexTypeLabelValMap = map[int]string{
	procfs.STAT_CPU_USER_TICKS:       PROC_STAT_CPU_PCT_TYPE_USER,
	procfs.STAT_CPU_NICE_TICKS:       PROC_STAT_CPU_PCT_TYPE_NICE,
	procfs.STAT_CPU_SYSTEM_TICKS:     PROC_STAT_CPU_PCT_TYPE_SYSTEM,
	procfs.STAT_CPU_IDLE_TICKS:       PROC_STAT_CPU_PCT_TYPE_IDLE,
	procfs.STAT_CPU_IOWAIT_TICKS:     PROC_STAT_CPU_PCT_TYPE_IOWAIT,
	procfs.STAT_CPU_IRQ_TICKS:        PROC_STAT_CPU_PCT_TYPE_IRQ,
	procfs.STAT_CPU_SOFTIRQ_TICKS:    PROC_STAT_CPU_PCT_TYPE_SOFTIRQ,
	procfs.STAT_CPU_STEAL_TICKS:      PROC_STAT_CPU_PCT_TYPE_STEAL,
	procfs.STAT_CPU_GUEST_TICKS:      PROC_STAT_CPU_PCT_TYPE_GUEST,
	procfs.STAT_CPU_GUEST_NICE_TICKS: PROC_STAT_CPU_PCT_TYPE_GUEST_NICE,
}

// Map procfs.NumericFields indexes into delta metrics name:
var procStatIndexDeltaMetricNameMap = map[int]string{
	procfs.STAT_PAGE_IN:  PROC_STAT_PAGE_IN_COUNT_DELTA_METRIC,
	procfs.STAT_PAGE_OUT: PROC_STAT_PAGE_OUT_COUNT_DELTA_METRIC,
	procfs.STAT_SWAP_IN:  PROC_STAT_SWAP_IN_COUNT_DELTA_METRIC,
	procfs.STAT_SWAP_OUT: PROC_STAT_SWAP_OUT_COUNT_DELTA_METRIC,
	procfs.STAT_CTXT:     PROC_STAT_CTXT_COUNT_DELTA_METRIC,
}

// Map procfs.NumericFields indexes into metrics name:
var procStatIndexMetricNameMap = map[int]string{
	procfs.STAT_PROCESSES:     PROC_STAT_PROCESSES_COUNT_METRIC,
	procfs.STAT_PROCS_RUNNING: PROC_STAT_PROCS_RUNNING_COUNT_METRIC,
	procfs.STAT_PROCS_BLOCKED: PROC_STAT_PROCS_BLOCKED_COUNT_METRIC,
}

var procStatMetricsLog = NewCompLogger("proc_stat_metrics")

type ProcStatMetricsConfig struct {
	// How often to generate the metrics in time.ParseDuration() format:
	Interval string `yaml:"interval"`
	// Normally metrics are generated only if there is a change in value from
	// the previous scan. However every N cycles the full set is generated. Use
	// 0 to generate full metrics every cycle.
	FullMetricsFactor int `yaml:"full_metrics_factor"`
	// Whether to scale %CPU for all with the total number os CPUs or not; i.e.
	// should a 8 CPU totally idle host report 800% (scale=false) or 100% idle
	// (scale=true):
	ScaleCpuAll bool `yaml:"scale_cpu_all"`
}

func DefaultProcStatMetricsConfig() *ProcStatMetricsConfig {
	return &ProcStatMetricsConfig{
		Interval:          PROC_STAT_METRICS_CONFIG_INTERVAL_DEFAULT,
		FullMetricsFactor: PROC_STAT_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT,
		ScaleCpuAll:       PROC_STAT_METRICS_CONFIG_SCALE_CPU_ALL_DEFAULT,
	}
}

type ProcStatMetrics struct {
	// id/task_id:
	id string
	// Scan interval:
	interval time.Duration
	// Whether to scale the aggregate %CPU by the number of CPUs or not:
	scaleCpuAll bool
	// Dual storage for parsed stats used as previous, current:
	procStat [2]*procfs.Stat
	// Timestamp when the stats were collected:
	procStatTs [2]time.Time
	// Index for current stats, toggled after each use:
	crtIndex int
	// Current cycle#:
	cycleNum int
	// Full metric factor:
	fullMetricsFactor int

	// For %CPU no metrics will be generated for 0 after 0, except for full
	// cycles. Keep whether the previous value was 0 or not, indexed by CPU#,
	// STAT_CPU_...:
	zeroPcpuMap map[int][]bool

	// CPU metrics cache, indexed by CPU#, STAT_CPU_...:
	cpuMetricsCache map[int][][]byte

	// Bootime/uptime metrics cache:
	btimeMetricCache, uptimeMetricCache []byte
	// Cache the btime at the 1st reading to be used as a reference for uptime:
	btime time.Time

	// Other metrics caches, indexed by stat#:
	deltaMetricsCache, metricsCache map[int][]byte
	// Interval metric:
	intervalMetric []byte

	// A buffer for the timestamp suffix:
	tsSuffixBuf *bytes.Buffer

	// The following are needed for testing only. Left to their default values,
	// the usual objects will be used.
	instance, hostname string
	timeNowFn          func() time.Time
	timeSinceFn        func(t time.Time) time.Duration
	metricsQueue       MetricsQueue
	procfsRoot         string
	linuxClktckSec     float64
}

func NewProcStatMetrics(cfg any) (*ProcStatMetrics, error) {
	var (
		err                error
		procStatMetricsCfg *ProcStatMetricsConfig
	)

	switch cfg := cfg.(type) {
	case *LsvmiConfig:
		procStatMetricsCfg = cfg.ProcStatMetricsConfig
	case *ProcStatMetricsConfig:
		procStatMetricsCfg = cfg
	case nil:
		procStatMetricsCfg = DefaultProcStatMetricsConfig()
	default:
		return nil, fmt.Errorf("NewInternalMetrics: %T invalid config type", cfg)
	}

	interval, err := time.ParseDuration(procStatMetricsCfg.Interval)
	if err != nil {
		return nil, err
	}
	proStatMetrics := &ProcStatMetrics{
		id:                PROC_STAT_METRICS_ID,
		interval:          interval,
		scaleCpuAll:       procStatMetricsCfg.ScaleCpuAll,
		zeroPcpuMap:       make(map[int][]bool),
		cpuMetricsCache:   make(map[int][][]byte),
		fullMetricsFactor: procStatMetricsCfg.FullMetricsFactor,
		tsSuffixBuf:       &bytes.Buffer{},
	}
	return proStatMetrics, nil
}

func (psm *ProcStatMetrics) updateCpuMetricsCache(cpu int) {
	instance, hostname := GlobalInstance, GlobalHostname
	if psm.instance != "" {
		instance = psm.instance
	}
	if psm.hostname != "" {
		hostname = psm.hostname
	}

	cpuMetrics := make([][]byte, procfs.STAT_CPU_NUM_STATS)
	var cpuLabelVal string
	if cpu == procfs.STAT_CPU_ALL {
		cpuLabelVal = PROC_STAT_CPU_ALL_LABEL_VALUE
	} else {
		cpuLabelVal = strconv.Itoa(cpu)
	}
	for index, typeLabelVal := range procStatCpuIndexTypeLabelValMap {
		cpuMetrics[index] = []byte(fmt.Sprintf(
			`%s{%s="%s",%s="%s",%s="%s",%s="%s"} `, // N.B. include space before val
			PROC_STAT_CPU_PCT_METRIC,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
			PROC_STAT_CPU_PCT_TYPE_LABEL_NAME, typeLabelVal,
			PROC_STAT_CPU_LABEL_NAME, cpuLabelVal,
		))
	}
	psm.cpuMetricsCache[cpu] = cpuMetrics
}

func (psm *ProcStatMetrics) updateBtimeUptimeMetricsCache() {
	instance, hostname := GlobalInstance, GlobalHostname
	if psm.instance != "" {
		instance = psm.instance
	}
	if psm.hostname != "" {
		hostname = psm.hostname
	}
	btime := int64(psm.procStat[psm.crtIndex].NumericFields[procfs.STAT_BTIME])
	psm.btime = time.Unix(btime, 0)
	psm.btimeMetricCache = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s"} %d`, // N.B. include the value!
		PROC_STAT_BTIME_METRIC,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
		btime,
	))
	psm.uptimeMetricCache = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s"} `, // N.B. include space before val
		PROC_STAT_UPTIME_METRIC,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
	))
}

func (psm *ProcStatMetrics) updateMetricsCache() {
	instance, hostname := GlobalInstance, GlobalHostname
	if psm.instance != "" {
		instance = psm.instance
	}
	if psm.hostname != "" {
		hostname = psm.hostname
	}

	psm.deltaMetricsCache = make(map[int][]byte)
	for index, name := range procStatIndexDeltaMetricNameMap {
		psm.deltaMetricsCache[index] = []byte(fmt.Sprintf(
			`%s{%s="%s",%s="%s"} `, // N.B. include space before val
			name,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
		))
	}

	psm.metricsCache = make(map[int][]byte)
	for index, name := range procStatIndexMetricNameMap {
		psm.metricsCache[index] = []byte(fmt.Sprintf(
			`%s{%s="%s",%s="%s"} `, // N.B. include space before val
			name,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
		))
	}

	psm.intervalMetric = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s"} `, // N.B. include space before val
		PROC_STAT_INTERVAL_METRIC_NAME,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
	))
}

func (psm *ProcStatMetrics) generateMetrics(buf *bytes.Buffer) int {
	crtProcStat, prevProcStat := psm.procStat[psm.crtIndex], psm.procStat[1-psm.crtIndex]
	crtTs, prevTs := psm.procStatTs[psm.crtIndex], psm.procStatTs[1-psm.crtIndex]

	psm.tsSuffixBuf.Reset()
	fmt.Fprintf(
		psm.tsSuffixBuf, " %d\n", crtTs.UnixMilli(),
	)
	promTs := psm.tsSuffixBuf.Bytes()

	metricsCount := 0
	fullMetrics := psm.cycleNum == 0

	crtNumericFields := crtProcStat.NumericFields

	var prevNumericFields []uint64 = nil

	if prevProcStat != nil {
		// %CPU require prev stats:
		linuxClktckSec := utils.LinuxClktckSec
		if psm.linuxClktckSec > 0 {
			linuxClktckSec = psm.linuxClktckSec
		}
		deltaSec := crtTs.Sub(prevTs).Seconds()
		pCpuFactor := linuxClktckSec * 100. / deltaSec // %CPU = delta(ticks) * pCpuFactor

		for cpu, crtCpuStats := range crtProcStat.Cpu {
			prevCpuStats := prevProcStat.Cpu[cpu]
			zeroPcpu := psm.zeroPcpuMap[cpu]
			if zeroPcpu == nil {
				zeroPcpu = make([]bool, procfs.STAT_CPU_NUM_STATS)
				psm.zeroPcpuMap[cpu] = zeroPcpu
			}
			if prevCpuStats != nil {
				numCpus := len(crtProcStat.Cpu) - 1
				cpuMetrics := psm.cpuMetricsCache[cpu]
				if cpuMetrics == nil {
					psm.updateCpuMetricsCache(cpu)
					cpuMetrics = psm.cpuMetricsCache[cpu]
				}
				for index, metric := range cpuMetrics {
					dCpuTicks := crtCpuStats[index] - prevCpuStats[index]
					if dCpuTicks != 0 || fullMetrics || !zeroPcpu[index] {
						val := float64(dCpuTicks)
						if cpu == procfs.STAT_CPU_ALL && numCpus > 0 && psm.scaleCpuAll {
							val /= float64(numCpus)
						}
						buf.Write(metric)
						buf.WriteString(strconv.FormatFloat(val*pCpuFactor, 'f', 1, 64))
						buf.Write(promTs)
						metricsCount++
					}
					zeroPcpu[index] = dCpuTicks == 0
				}
			}
		}
		// CPU's may be unplugged dynamically. Check an clear zero %CPU flags as needed.
		if len(psm.zeroPcpuMap) > len(crtProcStat.Cpu) {
			for cpu := range psm.zeroPcpuMap {
				if _, ok := crtProcStat.Cpu[cpu]; !ok {
					delete(psm.zeroPcpuMap, cpu)
				}
			}
		}

		// Delta metrics require prev stats.
		prevNumericFields = prevProcStat.NumericFields
		deltaMetricsCache := psm.deltaMetricsCache
		if deltaMetricsCache == nil {
			psm.updateMetricsCache()
			deltaMetricsCache = psm.deltaMetricsCache
		}
		for index, metric := range deltaMetricsCache {
			val := crtNumericFields[index] - prevNumericFields[index]
			buf.Write(metric)
			buf.WriteString(strconv.FormatUint(val, 10))
			buf.Write(promTs)
			metricsCount++
		}

		// Interval requires prev stats.
		buf.Write(psm.intervalMetric)
		buf.WriteString(strconv.FormatFloat(deltaSec, 'f', 6, 64))
		buf.Write(promTs)
		metricsCount++
	}

	// Boot/up-time metrics:
	if psm.btimeMetricCache == nil {
		psm.updateBtimeUptimeMetricsCache()
	}
	if fullMetrics {
		buf.Write(psm.btimeMetricCache)
		buf.Write(promTs)
		metricsCount++
	}
	buf.Write(psm.uptimeMetricCache)
	timeSinceFn := time.Since
	if psm.timeSinceFn != nil {
		timeSinceFn = psm.timeSinceFn
	}
	buf.WriteString(strconv.FormatFloat(timeSinceFn(psm.btime).Seconds(), 'f', 3, 64))
	buf.Write(promTs)
	metricsCount++

	// Other metrics:
	metricsCache := psm.metricsCache
	if metricsCache == nil {
		psm.updateMetricsCache()
		metricsCache = psm.metricsCache
	}
	for index, metric := range metricsCache {
		val := crtNumericFields[index]
		if fullMetrics || prevNumericFields == nil || val != prevNumericFields[index] {
			buf.Write(metric)
			buf.WriteString(strconv.FormatUint(val, 10))
			buf.Write(promTs)
			metricsCount++
		}
	}

	// Toggle the buffers, update the collection time and the cycle#:
	psm.crtIndex = 1 - psm.crtIndex
	if psm.cycleNum++; psm.cycleNum >= psm.fullMetricsFactor {
		psm.cycleNum = 0
	}

	return metricsCount
}

// Satisfy the TaskActivity interface:
func (psm *ProcStatMetrics) Execute() bool {
	timeNowFn := time.Now
	if psm.timeNowFn != nil {
		timeNowFn = psm.timeNowFn
	}

	metricsQueue := GlobalMetricsQueue
	if psm.metricsQueue != nil {
		metricsQueue = psm.metricsQueue
	}

	crtProcStat := psm.procStat[psm.crtIndex]
	if crtProcStat == nil {
		prevProcStat := psm.procStat[1-psm.crtIndex]
		if prevProcStat != nil {
			crtProcStat = prevProcStat.Clone(false)
		} else {
			procfsRoot := GlobalProcfsRoot
			if psm.procfsRoot != "" {
				procfsRoot = psm.procfsRoot
			}
			crtProcStat = procfs.NewStat(procfsRoot)
		}
		psm.procStat[psm.crtIndex] = crtProcStat
	}
	err := crtProcStat.Parse()
	if err != nil {
		procStatMetricsLog.Warnf("%v: proc stat metrics will be disabled", err)
		return false
	}
	psm.procStatTs[psm.crtIndex] = timeNowFn()

	buf := metricsQueue.GetBuf()
	metricsCount := psm.generateMetrics(buf)
	byteCount := buf.Len()
	metricsQueue.QueueBuf(buf)

	GlobalMetricsGeneratorStatsContainer.Update(
		psm.id, uint64(metricsCount), uint64(byteCount),
	)

	return true
}

// Define and register the task builder:
func ProcStatMetricsTaskBuilder(cfg *LsvmiConfig) ([]*Task, error) {
	psm, err := NewProcStatMetrics(cfg)
	if err != nil {
		return nil, err
	}
	if psm.interval <= 0 {
		procStatMetricsLog.Infof(
			"interval=%s, metrics disabled", psm.interval,
		)
		return nil, nil
	} else {
		procStatMetricsLog.Infof(
			"interval=%s, fullMetricsFactor=%d, scaleCpuAll=%v",
			psm.interval, psm.fullMetricsFactor, psm.scaleCpuAll,
		)
	}
	tasks := []*Task{
		NewTask(psm.id, psm.interval, psm),
	}
	return tasks, nil
}

func init() {
	TaskBuilders.Register(ProcStatMetricsTaskBuilder)
}
