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
	"time"

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
	PROC_PID_TID_LABEL_NAME = "tid" // PID metrics will associate an empty value

	// /proc/PID/stat:
	PROC_PID_STAT_STATE_METRIC     = "proc_pid_stat_state" // PID + TID
	PROC_PID_STAT_STATE_LABEL_NAME = "state"
	PROC_PID_STARTTIME_LABEL_NAME  = "starttime_msec"

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

	PROC_PID_STAT_PROCESSOR_NUM_METRIC = "proc_pid_stat_cpu_num" // PID + TID

	// /proc/PID/status:
	PROC_PID_STATUS_INFO_METRIC                  = "proc_pid_status_info" // PID only
	PROC_PID_STATUS_UID_LABEL_NAME               = "uid"
	PROC_PID_STATUS_GID_LABEL_NAME               = "gig"
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

// PID/TID specific (cached) info:
type ProcPidTidMetricsInfo struct {
	// Parsers, used to maintain the previous state:
	pidStat   *procfs.PidStat
	pidStatus *procfs.PidStatus

	// Cache PID/TID labels part (pid="PID",tid="TID"):
	pidTidLabels []byte

	// Cache starttime to millisec conversion:
	starttime_msec int64

	// Scan#, used to detect out-of-scope PID/TID's:
	scanNum int
}

// The main pid metrics data structure; there are NumPartitions (above)
// instances:
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

	// The cycle# counters:
	cycleNum [PROC_PID_METRICS_CYCLE_NUM_COUNTERS]int

	// Individual metrics cache, indexed by PID/TID:
	metricsInfo map[procfs.PidTid]*ProcPidTidMetricsInfo

	// Scan#, used to detect out-of-scope PID/TID's. This counter is incremented
	// for every scan and it is used to update the scan# for the cached PID/TID
	// info. At the end of the metrics generation, all the cache entries left
	// with an outdated scan# will be deleted.
	scanNum int

	// The following are needed for testing only. Left to their default values,
	// the usual objects will be used.
	instance, hostname string
	timeNowFn          func() time.Time
	metricsQueue       MetricsQueue
	procfsRoot         string
	linuxClktckSec     float64
}
