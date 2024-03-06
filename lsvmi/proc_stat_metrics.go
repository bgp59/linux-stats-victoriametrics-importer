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
	PROC_STAT_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT = 15
)

const (
	// CPU metrics:
	PROC_STAT_CPU_USER_METRIC       = "proc_stat_cpu_user_pct"
	PROC_STAT_CPU_NICE_METRIC       = "proc_stat_cpu_nice_pct"
	PROC_STAT_CPU_SYSTEM_METRIC     = "proc_stat_cpu_system_pct"
	PROC_STAT_CPU_IDLE_METRIC       = "proc_stat_cpu_idle_pct"
	PROC_STAT_CPU_IOWAIT_METRIC     = "proc_stat_cpu_iowait_pct"
	PROC_STAT_CPU_IRQ_METRIC        = "proc_stat_cpu_irq_pct"
	PROC_STAT_CPU_SOFTIRQ_METRIC    = "proc_stat_cpu_softirq_pct"
	PROC_STAT_CPU_STEAL_METRIC      = "proc_stat_cpu_steal_pct"
	PROC_STAT_CPU_GUEST_METRIC      = "proc_stat_cpu_guest_pct"
	PROC_STAT_CPU_GUEST_NICE_METRIC = "proc_stat_cpu_guest_nice_pct"

	PROC_STAT_CPU_LABEL_NAME = "cpu"
)

// Map procfs.Stat PROC_STAT_CPU_ index into metrics name:
var procStatCpuIndexMetricNameMap = map[int]string{
	procfs.STAT_CPU_USER_TICKS:       PROC_STAT_CPU_USER_METRIC,
	procfs.STAT_CPU_NICE_TICKS:       PROC_STAT_CPU_NICE_METRIC,
	procfs.STAT_CPU_SYSTEM_TICKS:     PROC_STAT_CPU_SYSTEM_METRIC,
	procfs.STAT_CPU_IDLE_TICKS:       PROC_STAT_CPU_IDLE_METRIC,
	procfs.STAT_CPU_IOWAIT_TICKS:     PROC_STAT_CPU_IOWAIT_METRIC,
	procfs.STAT_CPU_IRQ_TICKS:        PROC_STAT_CPU_IRQ_METRIC,
	procfs.STAT_CPU_SOFTIRQ_TICKS:    PROC_STAT_CPU_SOFTIRQ_METRIC,
	procfs.STAT_CPU_STEAL_TICKS:      PROC_STAT_CPU_STEAL_METRIC,
	procfs.STAT_CPU_GUEST_TICKS:      PROC_STAT_CPU_GUEST_METRIC,
	procfs.STAT_CPU_GUEST_NICE_TICKS: PROC_STAT_CPU_GUEST_NICE_METRIC,
}

// Special mapping to CPU# -> label value, the default is to use it as-is:
var procStatCpuToLabelVal = map[int]string{
	procfs.STAT_CPU_ALL: "all",
}

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

type ProcStatMetrics struct {
	// Scan interval:
	interval time.Duration
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

	// A buffer for the timestamp suffix:
	tsSuffixBuf *bytes.Buffer

	// The following are needed for testing only. Left to their default values,
	// the usual objects will be used.
	instance, hostname string
	timeNowFn          func() time.Time
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
		interval:          interval,
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
	cpuLabelVal := procStatCpuToLabelVal[cpu]
	if cpuLabelVal == "" {
		cpuLabelVal = strconv.Itoa(cpu)
	}
	for index, name := range procStatCpuIndexMetricNameMap {
		cpuMetrics[index] = []byte(fmt.Sprintf(
			`%s{%s="%s",%s="%s",%s="%s"} `, // N.B. include space before val
			name,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
			PROC_STAT_CPU_LABEL_NAME, cpuLabelVal,
		))
	}
	psm.cpuMetricsCache[cpu] = cpuMetrics
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

	// %CPU require prev stats:
	if prevProcStat != nil {
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
				cpuMetrics := psm.cpuMetricsCache[cpu]
				if cpuMetrics == nil {
					psm.updateCpuMetricsCache(cpu)
					cpuMetrics = psm.cpuMetricsCache[cpu]
				}
				for index, metric := range cpuMetrics {
					dCpuTicks := crtCpuStats[index] - prevCpuStats[index]
					if dCpuTicks != 0 || fullMetrics || !zeroPcpu[index] {
						buf.Write(metric)
						if dCpuTicks == 0 {
							buf.WriteByte('0')
						} else {
							buf.WriteString(strconv.FormatFloat(float64(dCpuTicks)*pCpuFactor, 'f', 1, 64))
						}
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
	}

	// Toggle the buffers, update the collection time and the cycle#:
	psm.crtIndex = 1 - psm.crtIndex
	if psm.cycleNum++; psm.cycleNum >= psm.fullMetricsFactor {
		psm.cycleNum = 0
	}

	return metricsCount
}
