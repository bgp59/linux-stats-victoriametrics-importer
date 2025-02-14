// Metrics based on /proc/stat

package lsvmi

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/bgp59/linux-stats-victoriametrics-importer/internal/utils"
	"github.com/bgp59/linux-stats-victoriametrics-importer/procfs"
)

const (
	PROC_STAT_METRICS_CONFIG_INTERVAL_DEFAULT            = "200ms"
	PROC_STAT_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT = 25

	// This generator id:
	PROC_STAT_METRICS_ID = "proc_stat_metrics"
)

const (
	// %CPU metrics:
	PROC_STAT_CPU_PCT_METRIC = "proc_stat_cpu_pct"

	PROC_STAT_CPU_PCT_MODE_LABEL_NAME = "mode"
	PROC_STAT_CPU_PCT_MODE_USER       = "user"
	PROC_STAT_CPU_PCT_MODE_NICE       = "nice"
	PROC_STAT_CPU_PCT_MODE_SYSTEM     = "system"
	PROC_STAT_CPU_PCT_MODE_IDLE       = "idle"
	PROC_STAT_CPU_PCT_MODE_IOWAIT     = "iowait"
	PROC_STAT_CPU_PCT_MODE_IRQ        = "irq"
	PROC_STAT_CPU_PCT_MODE_SOFTIRQ    = "softirq"
	PROC_STAT_CPU_PCT_MODE_STEAL      = "steal"
	PROC_STAT_CPU_PCT_MODE_GUEST      = "guest"
	PROC_STAT_CPU_PCT_MODE_GUEST_NICE = "guest_nice"

	// CPU presence metric:
	PROC_STAT_CPU_UP_METRIC = "proc_stat_cpu_up"

	PROC_STAT_CPU_LABEL_NAME      = "cpu"
	PROC_STAT_CPU_ALL_LABEL_VALUE = "all"
	PROC_STAT_CPU_AVG_LABEL_VALUE = "avg" // % for ALL / number of CPUs

	// System uptime will be based on btime:
	PROC_STAT_BTIME_METRIC  = "proc_stat_btime_sec"
	PROC_STAT_UPTIME_METRIC = "proc_stat_uptime_sec"

	// Other metrics:
	PROC_STAT_PAGE_IN_DELTA_METRIC       = "proc_stat_page_in_delta"
	PROC_STAT_PAGE_OUT_DELTA_METRIC      = "proc_stat_page_out_delta"
	PROC_STAT_SWAP_IN_DELTA_METRIC       = "proc_stat_swap_in_delta"
	PROC_STAT_SWAP_OUT_DELTA_METRIC      = "proc_stat_swap_out_delta"
	PROC_STAT_CTXT_DELTA_METRIC          = "proc_stat_ctxt_delta"
	PROC_STAT_PROCESSES_DELTA_METRIC     = "proc_stat_processes_delta"
	PROC_STAT_PROCS_RUNNING_COUNT_METRIC = "proc_stat_procs_running_count"
	PROC_STAT_PROCS_BLOCKED_COUNT_METRIC = "proc_stat_procs_blocked_count"

	// Interval since last generation, i.e. the interval underlying the deltas.
	// Normally this should be close to scan interval, but this is the actual
	// value, rather than the desired one:
	PROC_STAT_INTERVAL_METRIC = "proc_stat_metrics_delta_sec"
)

// Map procfs.Stat PROC_STAT_CPU_ indexes into type label value:
var procStatCpuIndexTypeLabelValMap = map[int]string{
	procfs.STAT_CPU_USER_TICKS:       PROC_STAT_CPU_PCT_MODE_USER,
	procfs.STAT_CPU_NICE_TICKS:       PROC_STAT_CPU_PCT_MODE_NICE,
	procfs.STAT_CPU_SYSTEM_TICKS:     PROC_STAT_CPU_PCT_MODE_SYSTEM,
	procfs.STAT_CPU_IDLE_TICKS:       PROC_STAT_CPU_PCT_MODE_IDLE,
	procfs.STAT_CPU_IOWAIT_TICKS:     PROC_STAT_CPU_PCT_MODE_IOWAIT,
	procfs.STAT_CPU_IRQ_TICKS:        PROC_STAT_CPU_PCT_MODE_IRQ,
	procfs.STAT_CPU_SOFTIRQ_TICKS:    PROC_STAT_CPU_PCT_MODE_SOFTIRQ,
	procfs.STAT_CPU_STEAL_TICKS:      PROC_STAT_CPU_PCT_MODE_STEAL,
	procfs.STAT_CPU_GUEST_TICKS:      PROC_STAT_CPU_PCT_MODE_GUEST,
	procfs.STAT_CPU_GUEST_NICE_TICKS: PROC_STAT_CPU_PCT_MODE_GUEST_NICE,
}

// Map procfs.NumericFields indexes into delta metrics name:
var procStatIndexDeltaMetricNameMap = map[int]string{
	procfs.STAT_PAGE_IN:   PROC_STAT_PAGE_IN_DELTA_METRIC,
	procfs.STAT_PAGE_OUT:  PROC_STAT_PAGE_OUT_DELTA_METRIC,
	procfs.STAT_SWAP_IN:   PROC_STAT_SWAP_IN_DELTA_METRIC,
	procfs.STAT_SWAP_OUT:  PROC_STAT_SWAP_OUT_DELTA_METRIC,
	procfs.STAT_CTXT:      PROC_STAT_CTXT_DELTA_METRIC,
	procfs.STAT_PROCESSES: PROC_STAT_PROCESSES_DELTA_METRIC,
}

// Map procfs.NumericFields indexes into metrics name:
var procStatIndexMetricNameMap = map[int]string{
	procfs.STAT_PROCS_RUNNING: PROC_STAT_PROCS_RUNNING_COUNT_METRIC,
	procfs.STAT_PROCS_BLOCKED: PROC_STAT_PROCS_BLOCKED_COUNT_METRIC,
}

var procStatMetricsLog = NewCompLogger(PROC_STAT_METRICS_ID)

type ProcStatMetricsConfig struct {
	// How often to generate the metrics in time.ParseDuration() format:
	Interval string `yaml:"interval"`
	// Normally metrics are generated only if there is a change in value from
	// the previous scan. However every N cycles the full set is generated. Use
	// 0 to generate full metrics every cycle.
	FullMetricsFactor int `yaml:"full_metrics_factor"`
}

func DefaultProcStatMetricsConfig() *ProcStatMetricsConfig {
	return &ProcStatMetricsConfig{
		Interval:          PROC_STAT_METRICS_CONFIG_INTERVAL_DEFAULT,
		FullMetricsFactor: PROC_STAT_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT,
	}
}

// Group together info indexed by CPU# to minimize the number of lookups:
type ProcStatMetricsCpuInfo struct {
	// %CPU metrics cache, indexed by STAT_CPU_...:
	pCpuMetrics [][]byte
	// Up metric:
	upMetric []byte
	// Current cycle#:
	cycleNum int
	// For %CPU no metrics will be generated for 0 after 0, except for full
	// cycles. Keep whether the previous value was 0 or not, indexed by
	// STAT_CPU_...:
	zeroPcpu []bool
}

type ProcStatMetrics struct {
	// id/task_id:
	id string
	// Scan interval:
	interval time.Duration
	// Full metric factor(s):
	fullMetricsFactor int

	// Dual storage for parsed stats used as previous, current:
	procStat [2]*procfs.Stat
	// Timestamp when the stats were collected:
	procStatTs [2]time.Time
	// Index for current stats, toggled after each use:
	currIndex int

	// Per CPU# info:
	cpuInfo map[int]*ProcStatMetricsCpuInfo

	// Avg (`all' / number of CPUs) metrics, indexed by STAT_CPU_...:
	avgPCpuMetrics [][]byte
	avgCpuUpMetric []byte

	// Bootime/uptime metrics cache:
	btimeMetric, uptimeMetric []byte
	// Cache the btime at the 1st reading to be used as a reference for uptime:
	btime time.Time

	// Other metrics, indexed by stat#:
	otherMetrics   map[int][]byte
	otherZeroDelta []bool
	otherCycleNum  int

	// Interval metric:
	intervalMetric []byte

	// A buffer for the timestamp suffix:
	tsSuffixBuf *bytes.Buffer

	// Cache the total metrics count, this is revised every time the number of
	// observed CPUs increases:
	maxNumCpus        int
	totalMetricsCount int

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
		return nil, fmt.Errorf("NewProcStatMetrics: %T invalid config type", cfg)
	}

	interval, err := time.ParseDuration(procStatMetricsCfg.Interval)
	if err != nil {
		return nil, err
	}
	procStatMetrics := &ProcStatMetrics{
		id:                PROC_STAT_METRICS_ID,
		interval:          interval,
		cpuInfo:           make(map[int]*ProcStatMetricsCpuInfo),
		fullMetricsFactor: procStatMetricsCfg.FullMetricsFactor,
		otherZeroDelta:    make([]bool, procfs.STAT_NUMERIC_NUM_STATS),
		tsSuffixBuf:       &bytes.Buffer{},
	}
	procStatMetrics.updateMaxNumCpus(1)
	procStatMetrics.otherCycleNum = initialCycleNum.Get(procStatMetrics.fullMetricsFactor)

	procStatMetricsLog.Infof("id=%s", procStatMetrics.id)
	procStatMetricsLog.Infof("interval=%s", procStatMetrics.interval)
	procStatMetricsLog.Infof("full_metrics_factor=%d", procStatMetrics.fullMetricsFactor)
	return procStatMetrics, nil
}

func (psm *ProcStatMetrics) updateMaxNumCpus(numCpus int) {
	psm.maxNumCpus = numCpus
	psm.totalMetricsCount = (
	// (...+2 for all and avg)*(...+1 for up metric)
	(psm.maxNumCpus+1)*(procfs.STAT_CPU_NUM_STATS+1) +
		// Non-CPU stats:
		len(procStatIndexDeltaMetricNameMap) +
		len(procStatIndexMetricNameMap) +
		// +2 for btime and uptime; +1 for interval
		2 + 1)
}

func (psm *ProcStatMetrics) updateCpuInfo(cpu int) {
	instance, hostname := GlobalInstance, GlobalHostname
	if psm.instance != "" {
		instance = psm.instance
	}
	if psm.hostname != "" {
		hostname = psm.hostname
	}

	pCpuMetrics := make([][]byte, procfs.STAT_CPU_NUM_STATS)
	var cpuLabelVal string
	var avgPCpuMetrics [][]byte
	if cpu == procfs.STAT_CPU_ALL {
		cpuLabelVal = PROC_STAT_CPU_ALL_LABEL_VALUE
		avgPCpuMetrics = make([][]byte, procfs.STAT_CPU_NUM_STATS)
	} else {
		cpuLabelVal = strconv.Itoa(cpu)
	}
	for index, typeLabelVal := range procStatCpuIndexTypeLabelValMap {
		pCpuMetrics[index] = []byte(fmt.Sprintf(
			`%s{%s="%s",%s="%s",%s="%s",%s="%s"} `, // N.B. include space before val
			PROC_STAT_CPU_PCT_METRIC,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
			PROC_STAT_CPU_PCT_MODE_LABEL_NAME, typeLabelVal,
			PROC_STAT_CPU_LABEL_NAME, cpuLabelVal,
		))
		if avgPCpuMetrics != nil {
			avgPCpuMetrics[index] = []byte(fmt.Sprintf(
				`%s{%s="%s",%s="%s",%s="%s",%s="%s"} `, // N.B. include space before val
				PROC_STAT_CPU_PCT_METRIC,
				INSTANCE_LABEL_NAME, instance,
				HOSTNAME_LABEL_NAME, hostname,
				PROC_STAT_CPU_PCT_MODE_LABEL_NAME, typeLabelVal,
				PROC_STAT_CPU_LABEL_NAME, PROC_STAT_CPU_AVG_LABEL_VALUE,
			))
		}
	}
	psm.cpuInfo[cpu] = &ProcStatMetricsCpuInfo{
		pCpuMetrics: pCpuMetrics,
		upMetric: []byte(fmt.Sprintf(
			`%s{%s="%s",%s="%s",%s="%s"} `, // N.B. include space before val
			PROC_STAT_CPU_UP_METRIC,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
			PROC_STAT_CPU_LABEL_NAME, cpuLabelVal,
		)),
		cycleNum: initialCycleNum.Get(psm.fullMetricsFactor),
		zeroPcpu: make([]bool, procfs.STAT_CPU_NUM_STATS),
	}
	if avgPCpuMetrics != nil {
		psm.avgPCpuMetrics = avgPCpuMetrics
		psm.avgCpuUpMetric = []byte(fmt.Sprintf(
			`%s{%s="%s",%s="%s",%s="%s"} `, // N.B. include space before val
			PROC_STAT_CPU_UP_METRIC,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
			PROC_STAT_CPU_LABEL_NAME, PROC_STAT_CPU_AVG_LABEL_VALUE,
		))
	}
}

func (psm *ProcStatMetrics) updateOtherMetrics() {
	instance, hostname := GlobalInstance, GlobalHostname
	if psm.instance != "" {
		instance = psm.instance
	}
	if psm.hostname != "" {
		hostname = psm.hostname
	}

	btime := int64(psm.procStat[psm.currIndex].NumericFields[procfs.STAT_BTIME])
	psm.btime = time.Unix(btime, 0)
	psm.btimeMetric = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s"} %d`, // N.B. include the value!
		PROC_STAT_BTIME_METRIC,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
		btime,
	))
	psm.uptimeMetric = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s"} `, // N.B. include space before val
		PROC_STAT_UPTIME_METRIC,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
	))

	psm.otherMetrics = make(map[int][]byte)
	for index, name := range procStatIndexDeltaMetricNameMap {
		psm.otherMetrics[index] = []byte(fmt.Sprintf(
			`%s{%s="%s",%s="%s"} `, // N.B. include space before val
			name,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
		))
	}
	for index, name := range procStatIndexMetricNameMap {
		psm.otherMetrics[index] = []byte(fmt.Sprintf(
			`%s{%s="%s",%s="%s"} `, // N.B. include space before val
			name,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
		))
	}

	psm.intervalMetric = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s"} `, // N.B. include space before val
		PROC_STAT_INTERVAL_METRIC,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
	))
}

func (psm *ProcStatMetrics) generateMetrics(buf *bytes.Buffer) (int, int) {
	currProcStat, prevProcStat := psm.procStat[psm.currIndex], psm.procStat[1-psm.currIndex]
	actualMetricsCount := 0
	numCpus := currProcStat.NumCpus
	if numCpus > psm.maxNumCpus {
		psm.updateMaxNumCpus(numCpus)
	}

	// Since most stats are deltas, wait until a prev stats:
	if prevProcStat != nil {
		currTs, prevTs := psm.procStatTs[psm.currIndex], psm.procStatTs[1-psm.currIndex]
		psm.tsSuffixBuf.Reset()
		fmt.Fprintf(
			psm.tsSuffixBuf, " %d\n", currTs.UnixMilli(),
		)
		promTs := psm.tsSuffixBuf.Bytes()

		// %CPU:
		linuxClktckSec := utils.LinuxClktckSec
		if psm.linuxClktckSec > 0 {
			linuxClktckSec = psm.linuxClktckSec
		}
		deltaSec := currTs.Sub(prevTs).Seconds()
		pCpuFactor := linuxClktckSec * 100. / deltaSec // %CPU = delta(ticks) * pCpuFactor
		var avgPCpuMetrics [][]byte
		for cpu, currCpuStats := range currProcStat.Cpu {
			prevCpuStats := prevProcStat.Cpu[cpu]
			if prevCpuStats == nil {
				continue
			}
			cpuInfo := psm.cpuInfo[cpu]
			fullMetrics := cpuInfo == nil || cpuInfo.cycleNum == 0
			if cpuInfo == nil {
				psm.updateCpuInfo(cpu)
				cpuInfo = psm.cpuInfo[cpu]
				fullMetrics = true
			}
			zeroPcpu := cpuInfo.zeroPcpu
			if cpu == procfs.STAT_CPU_ALL && numCpus > 0 {
				// This was updated by the call for `all` above:
				avgPCpuMetrics = psm.avgPCpuMetrics
			} else {
				avgPCpuMetrics = nil
			}
			for index, metric := range cpuInfo.pCpuMetrics {
				dCpuTicks := currCpuStats[index] - prevCpuStats[index]
				if dCpuTicks != 0 || fullMetrics || !zeroPcpu[index] {
					pct := float64(dCpuTicks) * pCpuFactor
					buf.Write(metric)
					buf.WriteString(strconv.FormatFloat(pct, 'f', 1, 64))
					buf.Write(promTs)
					actualMetricsCount++
					if avgPCpuMetrics != nil {
						buf.Write(avgPCpuMetrics[index])
						buf.WriteString(strconv.FormatFloat(pct/float64(numCpus), 'f', 1, 64))
						buf.Write(promTs)
						actualMetricsCount++
					}
					zeroPcpu[index] = dCpuTicks == 0
				}
			}
			if fullMetrics {
				buf.Write(cpuInfo.upMetric)
				buf.WriteByte('1')
				buf.Write(promTs)
				actualMetricsCount++
				if avgPCpuMetrics != nil {
					buf.Write(psm.avgCpuUpMetric)
					buf.WriteByte('1')
					buf.Write(promTs)
					actualMetricsCount++
				}
			}
			if cpuInfo.cycleNum++; cpuInfo.cycleNum >= psm.fullMetricsFactor {
				cpuInfo.cycleNum = 0
			}
		}
		// CPU's may be unplugged dynamically; remove out-of-scope CPUs:
		if len(psm.cpuInfo) > len(currProcStat.Cpu) {
			for cpu, cpuInfo := range psm.cpuInfo {
				if _, ok := currProcStat.Cpu[cpu]; !ok {
					// This CPU is out of scope:
					buf.Write(cpuInfo.upMetric)
					buf.WriteByte('0')
					buf.Write(promTs)
					actualMetricsCount++
					delete(psm.cpuInfo, cpu)
					if cpu == procfs.STAT_CPU_ALL {
						buf.Write(psm.avgCpuUpMetric)
						buf.WriteByte('0')
						buf.Write(promTs)
						actualMetricsCount++
					}
				}
			}
		}

		// Other metrics:
		otherMetrics := psm.otherMetrics
		otherFullMetrics := otherMetrics == nil || psm.otherCycleNum == 0
		if otherMetrics == nil {
			psm.updateOtherMetrics()
			otherMetrics = psm.otherMetrics
		}
		currNumericFields, prevNumericFields := currProcStat.NumericFields, prevProcStat.NumericFields

		// Other metrics - deltas:
		otherZeroDelta := psm.otherZeroDelta
		for index := range procStatIndexDeltaMetricNameMap {
			delta := currNumericFields[index] - prevNumericFields[index]
			if otherFullMetrics || delta != 0 || !otherZeroDelta[index] {
				buf.Write(otherMetrics[index])
				buf.WriteString(strconv.FormatUint(delta, 10))
				buf.Write(promTs)
				actualMetricsCount++
			}
			otherZeroDelta[index] = delta == 0
		}

		// Other metrics - non-deltas:
		for index := range procStatIndexMetricNameMap {
			val := currNumericFields[index]
			if otherFullMetrics || val != prevNumericFields[index] {
				buf.Write(otherMetrics[index])
				buf.WriteString(strconv.FormatUint(val, 10))
				buf.Write(promTs)
				actualMetricsCount++
			}
		}

		// Boot/up-time metrics:
		if otherFullMetrics {
			buf.Write(psm.btimeMetric)
			buf.Write(promTs)
			timeSinceFn := time.Since
			if psm.timeSinceFn != nil {
				timeSinceFn = psm.timeSinceFn
			}
			buf.Write(psm.uptimeMetric)
			buf.WriteString(strconv.FormatFloat(timeSinceFn(psm.btime).Seconds(), 'f', 3, 64))
			buf.Write(promTs)
			actualMetricsCount += 2
		}

		// Interval:
		buf.Write(psm.intervalMetric)
		buf.WriteString(strconv.FormatFloat(deltaSec, 'f', 6, 64))
		buf.Write(promTs)
		actualMetricsCount++

		if psm.otherCycleNum++; psm.otherCycleNum >= psm.fullMetricsFactor {
			psm.otherCycleNum = 0
		}
	}

	// Toggle the buffers:
	psm.currIndex = 1 - psm.currIndex

	return actualMetricsCount, psm.totalMetricsCount
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

	currProcStat := psm.procStat[psm.currIndex]
	if currProcStat == nil {
		prevProcStat := psm.procStat[1-psm.currIndex]
		if prevProcStat != nil {
			currProcStat = prevProcStat.Clone(false)
		} else {
			procfsRoot := GlobalProcfsRoot
			if psm.procfsRoot != "" {
				procfsRoot = psm.procfsRoot
			}
			currProcStat = procfs.NewStat(procfsRoot)
		}
		psm.procStat[psm.currIndex] = currProcStat
	}
	err := currProcStat.Parse()
	if err != nil {
		procStatMetricsLog.Warnf("%v: proc stat metrics will be disabled", err)
		return false
	}
	psm.procStatTs[psm.currIndex] = timeNowFn()

	buf := metricsQueue.GetBuf()
	actualMetricsCount, totalMetricsCount := psm.generateMetrics(buf)
	byteCount := buf.Len()
	metricsQueue.QueueBuf(buf)

	GlobalMetricsGeneratorStatsContainer.Update(
		psm.id, uint64(actualMetricsCount), uint64(totalMetricsCount), uint64(byteCount),
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
	}
	tasks := []*Task{
		NewTask(psm.id, psm.interval, psm),
	}
	return tasks, nil
}

func init() {
	TaskBuilders.Register(ProcStatMetricsTaskBuilder)
}
