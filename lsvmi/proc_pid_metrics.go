// Metrics bases on /proc/PID/... and/or /proc/PID/task/TID stat, status and cmdline files.

package lsvmi

// PID based metrics generation takes the same approach of producing metrics
// only for changes or non-zero deltas except every Nth cycle, when all metrics
// are generated (delta v. full metrics cycles, that is). Because these metrics
// are by far the most numerous, an additional reduction mechanism is used based
// on active processes/threads. A process/thread is deemed active if the sum of
// STIME + UTIME has an uptick from the previous sample; inactive
// processes/threads (based on that criterion) will be ignored for delta cycles.

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/emypar/linux-stats-victoriametrics-importer/internal/utils"
	"github.com/emypar/linux-stats-victoriametrics-importer/procfs"
)

const (
	PROC_PID_METRICS_CONFIG_INTERVAL_DEFAULT            = "1s"
	PROC_PID_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT = 15
	PROC_PID_METRICS_CONFIG_THREAD_METRICS_DEFAULT      = true
	PROC_PID_METRICS_CONFIG_NUM_PART_DEFAULT            = -1
	PROC_PID_METRICS_CONFIG_ACTIVE_ONLY_DELTA_DEFAULT   = true

	// This generator id:
	PROC_PID_METRICS_ID = "proc_pid_metrics"
)

// Metrics definitions:
const (
	// All metrics will have the following labels:
	PROC_PID_PID_LABEL_NAME = "pid"
	PROC_PID_TID_LABEL_NAME = "tid" // TID only

	// /proc/PID/stat:
	PROC_PID_STAT_STATE_METRIC         = "proc_pid_stat_state" // PID + TID
	PROC_PID_STAT_STATE_LABEL_NAME     = "state"
	PROC_PID_STAT_STARTTIME_LABEL_NAME = "starttime_msec"

	// If starttimeMsec cannot be parsed, use the following value:
	PROC_PID_STARTTIME_FALLBACK_VALUE = 0 // should default to epoch

	PROC_PID_STAT_INFO_METRIC            = "proc_pid_stat_info" // PID only
	PROC_PID_STAT_COMM_LABEL_NAME        = "comm"
	PROC_PID_STAT_PPID_LABEL_NAME        = "ppid"
	PROC_PID_STAT_PGRP_LABEL_NAME        = "pgrp"
	PROC_PID_STAT_SESSION_LABEL_NAME     = "session"
	PROC_PID_STAT_TTY_NR_LABEL_NAME      = "tty"
	PROC_PID_STAT_TPGID_LABEL_NAME       = "tpgid"
	PROC_PID_STAT_FLAGS_LABEL_NAME       = "flags"
	PROC_PID_STAT_PRIORITY_LABEL_NAME    = "prio"
	PROC_PID_STAT_NICE_LABEL_NAME        = "nice"
	PROC_PID_STAT_RT_PRIORITY_LABEL_NAME = "rt_prio"
	PROC_PID_STAT_POLICY_LABEL_NAME      = "policy"

	PROC_PID_STAT_VSIZE_METRIC  = "proc_pid_stat_vsize"  // PID only
	PROC_PID_STAT_RSS_METRIC    = "proc_pid_stat_rss"    // PID only
	PROC_PID_STAT_RSSLIM_METRIC = "proc_pid_stat_rsslim" // PID only

	PROC_PID_STAT_MINFLT_METRIC = "proc_pid_stat_minflt_delta" // PID + TID
	PROC_PID_STAT_MAJFLT_METRIC = "proc_pid_stat_majflt_delta" // PID + TID

	PROC_PID_STAT_UTIME_PCT_METRIC = "proc_pid_stat_utime_pcpu" // PID + TID
	PROC_PID_STAT_STIME_PCT_METRIC = "proc_pid_stat_stime_pcpu" // PID + TID
	PROC_PID_STAT_TIME_PCT_METRIC  = "proc_pid_stat_pcpu"       // PID + TID

	PROC_PID_STAT_CPU_NUM_METRIC = "proc_pid_stat_cpu_num" // PID + TID

	// /proc/PID/status:
	PROC_PID_STATUS_INFO_METRIC                  = "proc_pid_status_info" // PID only
	PROC_PID_STATUS_UID_LABEL_NAME               = "uid"
	PROC_PID_STATUS_GID_LABEL_NAME               = "gid"
	PROC_PID_STATUS_GROUPS_LABEL_NAME            = "groups"
	PROC_PID_STATUS_CPUS_ALLOWED_LIST_LABEL_NAME = "cpus_allowed"
	PROC_PID_STATUS_MEMS_ALLOWED_LIST_LABEL_NAME = "mems_allowed"

	PROC_PID_STATUS_VM_PEAK_METRIC      = "proc_pid_status_vm_peak"      // PID only
	PROC_PID_STATUS_VM_SIZE_METRIC      = "proc_pid_status_vm_size"      // PID only
	PROC_PID_STATUS_VM_LCK_METRIC       = "proc_pid_status_vm_lck"       // PID only
	PROC_PID_STATUS_VM_PIN_METRIC       = "proc_pid_status_vm_pin"       // PID only
	PROC_PID_STATUS_VM_HWM_METRIC       = "proc_pid_status_vm_hwm"       // PID only
	PROC_PID_STATUS_VM_RSS_METRIC       = "proc_pid_status_vm_rss"       // PID only
	PROC_PID_STATUS_RSS_ANON_METRIC     = "proc_pid_status_rss_anon"     // PID only
	PROC_PID_STATUS_RSS_FILE_METRIC     = "proc_pid_status_rss_file"     // PID only
	PROC_PID_STATUS_RSS_SHMEM_METRIC    = "proc_pid_status_rss_shmem"    // PID only
	PROC_PID_STATUS_VM_DATA_METRIC      = "proc_pid_status_vm_data"      // PID only
	PROC_PID_STATUS_VM_STK_METRIC       = "proc_pid_status_vm_stk"       // PID + TID
	PROC_PID_STATUS_VM_EXE_METRIC       = "proc_pid_status_vm_exe"       // PID only
	PROC_PID_STATUS_VM_LIB_METRIC       = "proc_pid_status_vm_lib"       // PID only
	PROC_PID_STATUS_VM_PTE_METRIC       = "proc_pid_status_vm_pte"       // PID only
	PROC_PID_STATUS_VM_PMD_METRIC       = "proc_pid_status_vm_pmd"       // PID only
	PROC_PID_STATUS_VM_SWAP_METRIC      = "proc_pid_status_vm_swap"      // PID only
	PROC_PID_STATUS_HUGETLBPAGES_METRIC = "proc_pid_status_hugetlbpages" // PID only
	PROC_PID_STATUS_VM_UNIT_LABEL_NAME  = "unit"

	PROC_PID_STATUS_VOLUNTARY_CTXT_SWITCHES_METRIC    = "proc_pid_status_vol_ctx_switch_delta"    // PID + TID
	PROC_PID_STATUS_NONVOLUNTARY_CTXT_SWITCHES_METRIC = "proc_pid_status_nonvol_ctx_switch_delta" // PID + TID

	// /proc/PID/cmdline. This metric is generated only for PID's, since it is assumed
	PROC_PID_CMDLINE_METRIC     = "proc_pid_cmdline" // PID only
	PROC_PID_CMDLINE_LABEL_NAME = "cmdline"

	// Interval since last generation, i.e. the interval underlying the deltas.
	// Normally this should be close to scan interval, but this is the actual
	// value, rather than the desired one:
	PROC_PID_INTERVAL_METRIC_NAME = "proc_pid_metrics_delta_sec"
)

// Maintain N x cycle counter groups; each PID/TID will belong to PID/TID % N
// group. It would be impractical to keep the counters per PID/TID since their
// total count is significantly greater than the full metric factor, so the full
// cycle would coincide for many of them anyway. Have N = power of 2 to make % N
// is very efficient.
const (
	PROC_PID_METRICS_CYCLE_NUM_COUNTERS = (1 << 4) // i.e. ~ default full metrics factor
	PROC_PID_METRICS_CYCLE_NUM_MASK     = PROC_PID_METRICS_CYCLE_NUM_COUNTERS - 1
)

var procPidMetricsLog = NewCompLogger(PROC_PID_METRICS_ID)

var procPidStatusMemoryMetricsPidTidIndexes = []int{
	procfs.PID_STATUS_VM_STK,
}

type ProcPidMetricsConfig struct {
	// How often to generate the metrics in time.ParseDuration() format:
	Interval string `yaml:"interval"`
	// Normally metrics are generated only if there is a change in value from
	// the previous scan. However every N cycles the full set is generated. Use
	// 0 to generate full metrics every cycle.
	FullMetricsFactor int `yaml:"full_metrics_factor"`
	// Whether to scan threads (/proc/PID/task/TID) and include thread metrics:
	ThreadMetrics bool `yaml:"thread_metrics"`
	// The number of partitions used to divide the process list; each partition
	// will generate a task and each task will run in a separate worker. A
	// negative value signifies the same value as the number of workers.
	NumPartitions int `yaml:"num_partitions"`
	// Whether to skip metrics for inactive processes/threads or not, during
	// delta cycles. Active is defined by an uptick in UTIME + STIME.
	ActiveOnlyDelta bool `yaml:"active_only_delta"`
}

func DefaultProcPidMetricsConfig() *ProcPidMetricsConfig {
	return &ProcPidMetricsConfig{
		Interval:          PROC_PID_METRICS_CONFIG_INTERVAL_DEFAULT,
		FullMetricsFactor: PROC_PID_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT,
		ThreadMetrics:     PROC_PID_METRICS_CONFIG_THREAD_METRICS_DEFAULT,
		NumPartitions:     PROC_PID_METRICS_CONFIG_NUM_PART_DEFAULT,
		ActiveOnlyDelta:   PROC_PID_METRICS_CONFIG_ACTIVE_ONLY_DELTA_DEFAULT,
	}
}

// PID/TID specific cached info:
type ProcPidTidMetricsInfo struct {
	// Parsers, used to maintain the previous state:
	pidStat   *procfs.PidStat
	pidStatus *procfs.PidStatus

	// When the previous stats above were collected:
	prevTs time.Time

	// Cache PID/TID labels: `PID="PID"[,TID="TID"]}';
	pidTidLabels string

	// Starttime label value converted to milliseconds:
	starttimeMsec string

	// Zero deltas:
	pidStatFltZeroDelta   []bool
	pidStatusCtxZeroDelta []bool

	// Scan#, used to detect out-of-scope PID/TID's:
	scanNum int
}

// Some metrics require the pairing of an index (in a parser returned slice) and
// a metric format or prefix:
type ProcPidMetricsIndexFmt struct {
	index int
	fmt   string
}

// The main pid metrics data structure; there are NumPartitions (above)
// instances.

// Musical Chairs Approach For Deltas:
//
// Due to the large number of PID/TID, an alternative to the dual buffer approach
// is used for the previous/current approach:
//  - the previous state is cached in a parser, on a per PID/TID basis
//  - there is an unbound parser used to get the current state for a given PID/TID
//  - once the metrics are generated, the latter is swapped with the cached one,
//    i.e. it becomes the previous, while the freed previous will be used as the
//    current parser for the next PID/TID in the list.

type ProcPidMetrics struct {
	// id/task_id:
	id string
	// Scan interval:
	interval time.Duration
	// Full metric factor(s):
	fullMetricsFactor int

	// The pid_list cache, shared among ProcPidMetrics instances:
	pidListCache *procfs.PidListCache
	// The partition for the above:
	nPart int
	// Destination storage for the above:
	pidTidList []procfs.PidTid

	// The cycle# counters:
	cycleNum [PROC_PID_METRICS_CYCLE_NUM_COUNTERS]int

	// Individual metrics cache, indexed by PID/TID:
	metricsInfo map[procfs.PidTid]*ProcPidTidMetricsInfo

	// Unbound parsers, see Musical Chairs Approach For Deltas above:
	pidStat   *procfs.PidStat
	pidStatus *procfs.PidStatus

	// The command line is not cached, it is parsed for every full metrics cycle
	// when the metrics is generated. A single parser is used for all PID/TID:
	pidCmdline *procfs.PidCmdline

	// Scan#, used to detect out-of-scope PID/TID's. This counter is incremented
	// for every scan and it is used to update the scan# for the cached PID/TID
	// info. At the end of the metrics generation, all the cache entries left
	// with an outdated scan# will be deleted.
	scanNum int

	// Cache metrics in a generic format that is applicable to all PID/TID and
	// other labels. This can be either as fragments that get combined with
	// PID/TID specific values, or format strings (args for fmt.Sprintf).

	// PidStat based metric formats:
	pidStatStateMetricFmt  string
	pidStatInfoMetricFmt   string
	pidStatCpuNumMetricFmt string
	pidStatMemoryMetricFmt []*ProcPidMetricsIndexFmt
	pidStatPcpuMetricFmt   []*ProcPidMetricsIndexFmt
	pidStatFltMetricFmt    []*ProcPidMetricsIndexFmt

	// PidStatus based metric formats:
	pidStatusInfoMetricFmt          string
	pidStatusPidOnlyMemoryMetricFmt []*ProcPidMetricsIndexFmt
	pidStatusPidTidMemoryMetricFmt  []*ProcPidMetricsIndexFmt
	pidStatusCtxMetricFmt           []*ProcPidMetricsIndexFmt

	// PidCmdline metric format:
	pidCmdlineMetricFmt string

	// A buffer for the timestamp:
	tsBuf *bytes.Buffer

	// The following will be initialized with the usual values and they may be
	// overwritten for testing purposes.
	instance, hostname string
	timeNowFn          func() time.Time
	metricsQueue       MetricsQueue
	procfsRoot         string
	linuxClktckSec     float64
	osBtimeMilli       int64
}

func (pm *ProcPidMetrics) buildMetricFmt(metricName string, valFmt string, labelNames ...string) string {
	metricFmt := fmt.Sprintf(
		`%s{%s="%s",%s="%s"`,
		metricName,
		INSTANCE_LABEL_NAME, pm.instance, HOSTNAME_LABEL_NAME, pm.hostname,
	)
	for _, label := range labelNames {
		metricFmt += fmt.Sprintf(`,%s="%%s"`, label)
	}
	metricFmt += fmt.Sprintf("} %s %%s\n", valFmt)
	return metricFmt
}

func (pm *ProcPidMetrics) updateMetricsCache() {
	pm.pidStatStateMetricFmt = pm.buildMetricFmt(
		PROC_PID_STAT_STATE_METRIC,
		"%c",
		PROC_PID_STAT_STARTTIME_LABEL_NAME,
		PROC_PID_STAT_STATE_LABEL_NAME,
	)

	pm.pidStatInfoMetricFmt = pm.buildMetricFmt(
		PROC_PID_STAT_STATE_METRIC,
		"%c",
		PROC_PID_STAT_COMM_LABEL_NAME,
		PROC_PID_STAT_PPID_LABEL_NAME,
		PROC_PID_STAT_PGRP_LABEL_NAME,
		PROC_PID_STAT_SESSION_LABEL_NAME,
		PROC_PID_STAT_TTY_NR_LABEL_NAME,
		PROC_PID_STAT_TPGID_LABEL_NAME,
		PROC_PID_STAT_FLAGS_LABEL_NAME,
		PROC_PID_STAT_PRIORITY_LABEL_NAME,
		PROC_PID_STAT_NICE_LABEL_NAME,
		PROC_PID_STAT_RT_PRIORITY_LABEL_NAME,
		PROC_PID_STAT_POLICY_LABEL_NAME,
	)

	pm.pidStatMemoryMetricFmt = []*ProcPidMetricsIndexFmt{
		{
			procfs.PID_STAT_VSIZE,
			pm.buildMetricFmt(PROC_PID_STAT_VSIZE_METRIC, "%s"),
		},
		{
			procfs.PID_STAT_RSS,
			pm.buildMetricFmt(PROC_PID_STAT_RSS_METRIC, "%s"),
		},
		{
			procfs.PID_STAT_RSSLIM,
			pm.buildMetricFmt(PROC_PID_STAT_RSSLIM_METRIC, "%s"),
		},
	}

	pm.pidStatCpuNumMetricFmt = pm.buildMetricFmt(
		PROC_PID_STAT_CPU_NUM_METRIC, "%s",
	)

	pm.pidStatFltMetricFmt = []*ProcPidMetricsIndexFmt{
		{
			procfs.PID_STAT_MINFLT,
			pm.buildMetricFmt(PROC_PID_STAT_MINFLT_METRIC, "%d"),
		},
		{
			procfs.PID_STAT_MAJFLT,
			pm.buildMetricFmt(PROC_PID_STAT_MAJFLT_METRIC, "%d"),
		},
	}

	pm.pidStatPcpuMetricFmt = []*ProcPidMetricsIndexFmt{
		{
			-1, // i.e. synthetic:
			pm.buildMetricFmt(PROC_PID_STAT_TIME_PCT_METRIC, "%.1f"),
		},
		{
			procfs.PID_STAT_STIME,
			pm.buildMetricFmt(PROC_PID_STAT_STIME_PCT_METRIC, "%.1f"),
		},
		{
			procfs.PID_STAT_UTIME,
			pm.buildMetricFmt(PROC_PID_STAT_UTIME_PCT_METRIC, "%.1f"),
		},
	}

	pm.pidStatusInfoMetricFmt = pm.buildMetricFmt(
		PROC_PID_STATUS_INFO_METRIC,
		"%c",
		PROC_PID_STATUS_UID_LABEL_NAME,
		PROC_PID_STATUS_GID_LABEL_NAME,
		PROC_PID_STATUS_GROUPS_LABEL_NAME,
		PROC_PID_STATUS_CPUS_ALLOWED_LIST_LABEL_NAME,
		PROC_PID_STATUS_MEMS_ALLOWED_LIST_LABEL_NAME,
	)

	pidStatusMemoryFmt := []*ProcPidMetricsIndexFmt{
		{
			procfs.PID_STATUS_VM_PEAK,
			pm.buildMetricFmt(PROC_PID_STATUS_VM_PEAK_METRIC, "%s", PROC_PID_STATUS_VM_UNIT_LABEL_NAME),
		},
		{
			procfs.PID_STATUS_VM_SIZE,
			pm.buildMetricFmt(PROC_PID_STATUS_VM_SIZE_METRIC, "%s", PROC_PID_STATUS_VM_UNIT_LABEL_NAME),
		},
		{
			procfs.PID_STATUS_VM_LCK,
			pm.buildMetricFmt(PROC_PID_STATUS_VM_LCK_METRIC, "%s", PROC_PID_STATUS_VM_UNIT_LABEL_NAME),
		},
		{
			procfs.PID_STATUS_VM_PIN,
			pm.buildMetricFmt(PROC_PID_STATUS_VM_PIN_METRIC, "%s", PROC_PID_STATUS_VM_UNIT_LABEL_NAME),
		},
		{
			procfs.PID_STATUS_VM_HWM,
			pm.buildMetricFmt(PROC_PID_STATUS_VM_HWM_METRIC, "%s", PROC_PID_STATUS_VM_UNIT_LABEL_NAME),
		},
		{
			procfs.PID_STATUS_VM_RSS,
			pm.buildMetricFmt(PROC_PID_STATUS_VM_RSS_METRIC, "%s", PROC_PID_STATUS_VM_UNIT_LABEL_NAME),
		},
		{
			procfs.PID_STATUS_RSS_ANON,
			pm.buildMetricFmt(PROC_PID_STATUS_RSS_ANON_METRIC, "%s", PROC_PID_STATUS_VM_UNIT_LABEL_NAME),
		},
		{
			procfs.PID_STATUS_RSS_FILE,
			pm.buildMetricFmt(PROC_PID_STATUS_RSS_FILE_METRIC, "%s", PROC_PID_STATUS_VM_UNIT_LABEL_NAME),
		},
		{
			procfs.PID_STATUS_RSS_SHMEM,
			pm.buildMetricFmt(PROC_PID_STATUS_RSS_SHMEM_METRIC, "%s", PROC_PID_STATUS_VM_UNIT_LABEL_NAME),
		},
		{
			procfs.PID_STATUS_VM_DATA,
			pm.buildMetricFmt(PROC_PID_STATUS_VM_DATA_METRIC, "%s", PROC_PID_STATUS_VM_UNIT_LABEL_NAME),
		},
		{
			procfs.PID_STATUS_VM_STK,
			pm.buildMetricFmt(PROC_PID_STATUS_VM_STK_METRIC, "%s", PROC_PID_STATUS_VM_UNIT_LABEL_NAME),
		},
		{
			procfs.PID_STATUS_VM_EXE,
			pm.buildMetricFmt(PROC_PID_STATUS_VM_EXE_METRIC, "%s", PROC_PID_STATUS_VM_UNIT_LABEL_NAME),
		},
		{
			procfs.PID_STATUS_VM_LIB,
			pm.buildMetricFmt(PROC_PID_STATUS_VM_LIB_METRIC, "%s", PROC_PID_STATUS_VM_UNIT_LABEL_NAME),
		},
		{
			procfs.PID_STATUS_VM_PTE,
			pm.buildMetricFmt(PROC_PID_STATUS_VM_PTE_METRIC, "%s", PROC_PID_STATUS_VM_UNIT_LABEL_NAME),
		},
		{
			procfs.PID_STATUS_VM_PMD,
			pm.buildMetricFmt(PROC_PID_STATUS_VM_PMD_METRIC, "%s", PROC_PID_STATUS_VM_UNIT_LABEL_NAME),
		},
		{
			procfs.PID_STATUS_VM_SWAP,
			pm.buildMetricFmt(PROC_PID_STATUS_VM_SWAP_METRIC, "%s", PROC_PID_STATUS_VM_UNIT_LABEL_NAME),
		},
		{
			procfs.PID_STATUS_HUGETLBPAGES,
			pm.buildMetricFmt(PROC_PID_STATUS_HUGETLBPAGES_METRIC, "%s", PROC_PID_STATUS_VM_UNIT_LABEL_NAME),
		},
	}

	pidStatusPidTidMemoryMetricIndex := make(map[int]bool)
	for _, index := range procPidStatusMemoryMetricsPidTidIndexes {
		pidStatusPidTidMemoryMetricIndex[index] = true
	}

	pm.pidStatusPidOnlyMemoryMetricFmt = make(
		[]*ProcPidMetricsIndexFmt,
		0,
		len(pidStatusMemoryFmt)-len(procPidStatusMemoryMetricsPidTidIndexes),
	)
	pm.pidStatusPidTidMemoryMetricFmt = make(
		[]*ProcPidMetricsIndexFmt,
		0,
		len(procPidStatusMemoryMetricsPidTidIndexes),
	)
	for _, indexFmt := range pidStatusMemoryFmt {
		if pidStatusPidTidMemoryMetricIndex[indexFmt.index] {
			pm.pidStatusPidTidMemoryMetricFmt = append(pm.pidStatusPidTidMemoryMetricFmt, indexFmt)
		} else {
			pm.pidStatusPidOnlyMemoryMetricFmt = append(pm.pidStatusPidOnlyMemoryMetricFmt, indexFmt)
		}
	}

	pm.pidStatusCtxMetricFmt = []*ProcPidMetricsIndexFmt{
		{
			procfs.PID_STATUS_VOLUNTARY_CTXT_SWITCHES,
			pm.buildMetricFmt(PROC_PID_STATUS_VOLUNTARY_CTXT_SWITCHES_METRIC, "%d"),
		},
		{
			procfs.PID_STATUS_NONVOLUNTARY_CTXT_SWITCHES,
			pm.buildMetricFmt(PROC_PID_STATUS_NONVOLUNTARY_CTXT_SWITCHES_METRIC, "%d"),
		},
	}

	pm.pidCmdlineMetricFmt = pm.buildMetricFmt(PROC_PID_CMDLINE_METRIC, "%c", PROC_PID_CMDLINE_LABEL_NAME)
}

func (pm *ProcPidMetrics) initPidTidMetricsInfo(pidTid procfs.PidTid) *ProcPidTidMetricsInfo {
	pidTidLabels := fmt.Sprintf(`%s="%d"`, PROC_PID_PID_LABEL_NAME, pidTid.Pid)
	if pidTid.Tid != procfs.PID_STAT_PID_ONLY_TID {
		pidTidLabels += fmt.Sprintf(`,%s="%d"`, PROC_PID_TID_LABEL_NAME, pidTid.Tid)
	}

	starttimeMsec, err := strconv.ParseFloat(string(pm.pidStat.ByteSliceFields[procfs.PID_STAT_STARTTIME]), 64)
	if err != nil {
		procPidMetricsLog.Warnf(
			`PID: %d, TID: %d, starttime: %v`,
			pidTid.Pid, pidTid.Tid, err,
		)
		starttimeMsec = PROC_PID_STARTTIME_FALLBACK_VALUE
	}

	pidTidMetricsInfo := &ProcPidTidMetricsInfo{
		pidStat:               procfs.NewPidStat(pm.procfsRoot, pidTid.Pid, pidTid.Tid),
		pidStatus:             procfs.NewPidStatus(pm.procfsRoot, pidTid.Pid, pidTid.Tid),
		pidTidLabels:          pidTidLabels,
		starttimeMsec:         strconv.FormatInt(pm.osBtimeMilli+int64(starttimeMsec*pm.linuxClktckSec*1000.), 10),
		pidStatFltZeroDelta:   make([]bool, 2),
		pidStatusCtxZeroDelta: make([]bool, 2),
	}

	return pidTidMetricsInfo
}

func (pm *ProcPidMetrics) generateMetrics(
	pidTid procfs.PidTid,
	fullMetrics bool,
	currTs time.Time,
	buf *bytes.Buffer,
) int {
	actualMetricsCount := 0

	pidTidMetricsInfo, hasPrev := pm.metricsInfo[pidTid]
	if !hasPrev {
		pidTidMetricsInfo = pm.initPidTidMetricsInfo(pidTid)
	}
	currPidStat, prevPidStat := pm.pidStat, pidTidMetricsInfo.pidStat
	currPidStatus, prevPidStatus := pm.pidStatus, pidTidMetricsInfo.pidStatus

	fmt.Fprintf(pm.tsBuf, "%d", currTs.UnixMilli())
	ts := pm.tsBuf.Bytes()

	// PID + TID metrics:
	if changed := hasPrev && !bytes.Equal(
		prevPidStat.ByteSliceFields[procfs.PID_STAT_STATE],
		currPidStat.ByteSliceFields[procfs.PID_STAT_STATE]); changed || fullMetrics {
		if changed {
			// Clear previous state:
			fmt.Fprintf(
				buf,
				pm.pidStatStateMetricFmt,
				pidTidMetricsInfo.starttimeMsec,
				prevPidStat.ByteSliceFields[procfs.PID_STAT_STATE],
				pidTidMetricsInfo.pidTidLabels,
				'0',
				ts,
			)
			actualMetricsCount++
		}
		fmt.Fprintf(
			buf,
			pm.pidStatStateMetricFmt,
			pidTidMetricsInfo.starttimeMsec,
			currPidStat.ByteSliceFields[procfs.PID_STAT_STATE],
			pidTidMetricsInfo.pidTidLabels,
			'1',
			ts,
		)
		actualMetricsCount++
	}

	fmt.Fprintf(
		buf,
		pm.pidStatCpuNumMetricFmt,
		pidTidMetricsInfo.pidTidLabels,
		currPidStat.ByteSliceFields[procfs.PID_STAT_PROCESSOR],
		ts,
	)
	actualMetricsCount++

	for _, indexFmt := range pm.pidStatusPidTidMemoryMetricFmt {
		if fullMetrics || !hasPrev || !bytes.Equal(
			prevPidStatus.ByteSliceFields[indexFmt.index],
			currPidStatus.ByteSliceFields[indexFmt.index]) {
			fmt.Fprintf(
				buf,
				indexFmt.fmt,
				currPidStatus.ByteSliceFieldUnit[indexFmt.index],
				currPidStatus.ByteSliceFields[indexFmt.index],
				ts,
			)
			actualMetricsCount++
		}
	}

	// Delta metrics require previous:
	if hasPrev {
		for i, indexFmt := range pm.pidStatFltMetricFmt {
			delta := currPidStat.NumericFields[indexFmt.index] - prevPidStat.NumericFields[indexFmt.index]
			if delta != 0 || fullMetrics || !pidTidMetricsInfo.pidStatFltZeroDelta[i] {
				fmt.Fprintf(
					buf,
					indexFmt.fmt,
					pidTidMetricsInfo.pidTidLabels,
					delta,
					ts,
				)
				actualMetricsCount++
			}
			pidTidMetricsInfo.pidStatFltZeroDelta[i] = delta == 0
		}

		linuxClktckSec := utils.LinuxClktckSec
		if pm.linuxClktckSec > 0 {
			linuxClktckSec = pm.linuxClktckSec
		}
		totalPcpuMetricFmt := ""
		totalCpuDelta := uint64(0)
		deltaSec := currTs.Sub(pidTidMetricsInfo.prevTs).Seconds()
		pcpuFactor := linuxClktckSec / deltaSec * 100.
		for _, indexFmt := range pm.pidStatPcpuMetricFmt {
			if indexFmt.index < 0 {
				totalPcpuMetricFmt = indexFmt.fmt
				break
			}
			delta := currPidStat.NumericFields[indexFmt.index] - prevPidStat.NumericFields[indexFmt.index]
			totalCpuDelta += delta
			fmt.Fprintf(
				buf,
				indexFmt.fmt,
				pidTidMetricsInfo.pidTidLabels,
				float64(delta)*pcpuFactor,
				ts,
			)
			actualMetricsCount++
		}
		fmt.Fprintf(
			buf,
			totalPcpuMetricFmt,
			pidTidMetricsInfo.pidTidLabels,
			float64(totalCpuDelta)*pcpuFactor,
			ts,
		)
		actualMetricsCount++

		for i, indexFmt := range pm.pidStatusCtxMetricFmt {
			delta := currPidStatus.NumericFields[indexFmt.index] - prevPidStatus.NumericFields[indexFmt.index]
			if delta != 0 || fullMetrics || !pidTidMetricsInfo.pidStatusCtxZeroDelta[i] {
				fmt.Fprintf(
					buf,
					indexFmt.fmt,
					pidTidMetricsInfo.pidTidLabels,
					delta,
					ts,
				)
				actualMetricsCount++
			}
			pidTidMetricsInfo.pidStatusCtxZeroDelta[i] = delta == 0
		}
	}

	if pidTid.Tid != procfs.PID_STAT_PID_ONLY_TID {
		return actualMetricsCount
	}

	// PID only metrics:
	if fullMetrics || !hasPrev {
		if hasPrev {
			// Check for change:
			changed := false
			for _, index := range []int{
				procfs.PID_STAT_COMM,
				procfs.PID_STAT_PPID,
				procfs.PID_STAT_PGRP,
				procfs.PID_STAT_SESSION,
				procfs.PID_STAT_TTY_NR,
				procfs.PID_STAT_TPGID,
				procfs.PID_STAT_FLAGS,
				procfs.PID_STAT_PRIORITY,
				procfs.PID_STAT_NICE,
				procfs.PID_STAT_RT_PRIORITY,
				procfs.PID_STAT_POLICY,
			} {
				if changed = !bytes.Equal(
					prevPidStat.ByteSliceFields[index],
					currPidStat.ByteSliceFields[index]); changed {
					break
				}
			}
			if changed {
				// Clear previous state:
				fmt.Fprintf(
					buf,
					pm.pidStatInfoMetricFmt,
					prevPidStat.ByteSliceFields[procfs.PID_STAT_COMM],
					prevPidStat.ByteSliceFields[procfs.PID_STAT_PPID],
					prevPidStat.ByteSliceFields[procfs.PID_STAT_PGRP],
					prevPidStat.ByteSliceFields[procfs.PID_STAT_SESSION],
					prevPidStat.ByteSliceFields[procfs.PID_STAT_TTY_NR],
					prevPidStat.ByteSliceFields[procfs.PID_STAT_TPGID],
					prevPidStat.ByteSliceFields[procfs.PID_STAT_FLAGS],
					prevPidStat.ByteSliceFields[procfs.PID_STAT_PRIORITY],
					prevPidStat.ByteSliceFields[procfs.PID_STAT_NICE],
					prevPidStat.ByteSliceFields[procfs.PID_STAT_RT_PRIORITY],
					prevPidStat.ByteSliceFields[procfs.PID_STAT_POLICY],
					'0',
					ts,
				)
				actualMetricsCount++
			}
			fmt.Fprintf(
				buf,
				pm.pidStatInfoMetricFmt,
				currPidStat.ByteSliceFields[procfs.PID_STAT_COMM],
				currPidStat.ByteSliceFields[procfs.PID_STAT_PPID],
				currPidStat.ByteSliceFields[procfs.PID_STAT_PGRP],
				currPidStat.ByteSliceFields[procfs.PID_STAT_SESSION],
				currPidStat.ByteSliceFields[procfs.PID_STAT_TTY_NR],
				currPidStat.ByteSliceFields[procfs.PID_STAT_TPGID],
				currPidStat.ByteSliceFields[procfs.PID_STAT_FLAGS],
				currPidStat.ByteSliceFields[procfs.PID_STAT_PRIORITY],
				currPidStat.ByteSliceFields[procfs.PID_STAT_NICE],
				currPidStat.ByteSliceFields[procfs.PID_STAT_RT_PRIORITY],
				currPidStat.ByteSliceFields[procfs.PID_STAT_POLICY],
				'1',
				ts,
			)
			actualMetricsCount++
		}
	}

	for _, indexFmt := range pm.pidStatMemoryMetricFmt {
		if fullMetrics || !hasPrev || !bytes.Equal(
			prevPidStat.ByteSliceFields[indexFmt.index],
			currPidStat.ByteSliceFields[indexFmt.index]) {
			fmt.Fprintf(
				buf,
				indexFmt.fmt,
				currPidStat.ByteSliceFields[indexFmt.index],
				ts,
			)
			actualMetricsCount++
		}
	}

	if hasPrev {
		// Check for change:
		changed := false
		for _, index := range []int{
			procfs.PID_STATUS_UID,
			procfs.PID_STATUS_GID,
			procfs.PID_STATUS_GROUPS,
			procfs.PID_STATUS_CPUS_ALLOWED_LIST,
			procfs.PID_STATUS_MEMS_ALLOWED_LIST,
		} {
			if changed = !bytes.Equal(
				prevPidStatus.ByteSliceFields[index],
				currPidStatus.ByteSliceFields[index]); changed {
				break
			}
		}
		if changed {
			// Clear prev metric:
			fmt.Fprintf(
				buf,
				pm.pidStatusInfoMetricFmt,
				prevPidStatus.ByteSliceFields[procfs.PID_STATUS_UID],
				prevPidStatus.ByteSliceFields[procfs.PID_STATUS_GID],
				prevPidStatus.ByteSliceFields[procfs.PID_STATUS_GROUPS],
				prevPidStatus.ByteSliceFields[procfs.PID_STATUS_CPUS_ALLOWED_LIST],
				prevPidStatus.ByteSliceFields[procfs.PID_STATUS_MEMS_ALLOWED_LIST],
				'0',
				ts,
			)
			actualMetricsCount++
		}
	}
	fmt.Fprintf(
		buf,
		pm.pidStatusInfoMetricFmt,
		currPidStatus.ByteSliceFields[procfs.PID_STATUS_UID],
		currPidStatus.ByteSliceFields[procfs.PID_STATUS_GID],
		currPidStatus.ByteSliceFields[procfs.PID_STATUS_GROUPS],
		currPidStatus.ByteSliceFields[procfs.PID_STATUS_CPUS_ALLOWED_LIST],
		currPidStatus.ByteSliceFields[procfs.PID_STATUS_MEMS_ALLOWED_LIST],
		'1',
		ts,
	)
	actualMetricsCount++

	for _, indexFmt := range pm.pidStatusPidOnlyMemoryMetricFmt {
		if fullMetrics || !hasPrev || !bytes.Equal(
			prevPidStatus.ByteSliceFields[indexFmt.index],
			currPidStatus.ByteSliceFields[indexFmt.index]) {
			fmt.Fprintf(
				buf,
				indexFmt.fmt,
				currPidStatus.ByteSliceFieldUnit[indexFmt.index],
				currPidStatus.ByteSliceFields[indexFmt.index],
				ts,
			)
			actualMetricsCount++
		}
	}

	if fullMetrics {
		fmt.Fprintf(
			buf,
			pm.pidCmdlineMetricFmt,
			pm.pidCmdline.Cmdline.Bytes(),
			'1',
			ts,
		)
	}

	return actualMetricsCount
}

// Satisfy the TaskActivity interface:
func (pm *ProcPidMetrics) Execute() bool {
	var (
		fullMetrics             bool
		buf                     *bytes.Buffer
		totalActualMetricsCount int
		activePidTidCount       int
		totalPidTidCount        int
		bufTargetSize           = pm.metricsQueue.GetTargetSize()
	)

	pidTidList, err := pm.pidListCache.GetPidTidList(pm.nPart, pm.pidTidList)
	if err != nil {
		procPidMetricsLog.Errorf("GetPidTidList(part=%d): %v", pm.nPart, err)
		return false
	}
	pm.pidTidList = pidTidList // to be reused next time

	for _, pidTid := range pidTidList {
		if pidTid.Tid == procfs.PID_STAT_PID_ONLY_TID {
			fullMetrics = pm.cycleNum[pidTid.Pid&PROC_PID_METRICS_CYCLE_NUM_MASK] == 0
		} else {
			fullMetrics = pm.cycleNum[pidTid.Tid&PROC_PID_METRICS_CYCLE_NUM_MASK] == 0
		}

		pidTidMetricsInfo, hasPrev := pm.metricsInfo[pidTid]
		if !hasPrev {
			pidTidMetricsInfo = pm.initPidTidMetricsInfo(pidTid)
		}
		if pm.pidStat == nil {
			pm.pidStat = procfs.NewPidStat(pm.procfsRoot, 0, 0) // they will be overwritten
		}
		err = pm.pidStat.Parse(pidTidMetricsInfo.pidStat)
		if err != nil {
			procPidMetricsLog.Error(err)
			continue
		}
		if !hasPrev ||
			pm.pidStat.NumericFields[procfs.PID_STAT_UTIME] != pidTidMetricsInfo.pidStat.NumericFields[procfs.PID_STAT_UTIME] ||
			pm.pidStat.NumericFields[procfs.PID_STAT_STIME] != pidTidMetricsInfo.pidStat.NumericFields[procfs.PID_STAT_STIME] {
			activePidTidCount++
		} else if !fullMetrics {
			continue
		}
		if pm.pidStatus == nil {
			pm.pidStatus = procfs.NewPidStatus(pm.procfsRoot, 0, 0) // they will be overwritten
		}
		err = pm.pidStatus.Parse(pidTidMetricsInfo.pidStatus)
		if err != nil {
			procPidMetricsLog.Error(err)
			continue
		}
		if fullMetrics {
			if pm.pidCmdline == nil {
				pm.pidCmdline = procfs.NewPidCmdline(pm.procfsRoot, 0, 0) // they will be overwritten
			}
			err = pm.pidCmdline.Parse(pidTid.Pid, pidTid.Tid)
			if err != nil {
				procPidMetricsLog.Error(err)
				continue
			}
		}
		currTs := pm.timeNowFn()
		if buf == nil {
			buf = pm.metricsQueue.GetBuf()
		}
		actualMetricsCount := pm.generateMetrics(pidTid, fullMetrics, currTs, buf)
		totalActualMetricsCount += actualMetricsCount
		if buf.Len() > bufTargetSize {
			pm.metricsQueue.QueueBuf(buf)
			buf = nil
		}
	}

	if buf != nil {
		pm.metricsQueue.QueueBuf(buf)
	}
	return true
}
