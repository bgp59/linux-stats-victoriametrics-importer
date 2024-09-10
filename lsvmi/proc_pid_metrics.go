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
	"os"
	"strconv"
	"time"

	"github.com/emypar/linux-stats-victoriametrics-importer/internal/utils"
	"github.com/emypar/linux-stats-victoriametrics-importer/procfs"
)

const (
	PROC_PID_METRICS_CONFIG_INTERVAL_DEFAULT                      = "1s"
	PROC_PID_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT           = 15
	PROC_PID_METRICS_CONFIG_THREAD_METRICS_DEFAULT                = true
	PROC_PID_METRICS_CONFIG_PID_LIST_CACHE_VALID_INTERVAL_DEFAULT = "900ms"
	PROC_PID_METRICS_CONFIG_NUM_PART_DEFAULT                      = -1
	PROC_PID_METRICS_USE_PID_STATUS_DEFAULT                       = true

	// This generator id:
	PROC_PID_METRICS_ID = "proc_pid_metrics"
)

// Metrics definitions:
const (
	// All metrics will have the following labels:
	PROC_PID_PID_LABEL_NAME = "pid"
	PROC_PID_TID_LABEL_NAME = "tid" // TID only

	// /proc/PID/stat:
	PROC_PID_STAT_STATE_METRIC     = "proc_pid_stat_state" // PID + TID
	PROC_PID_STAT_STATE_LABEL_NAME = "state"

	PROC_PID_STAT_COMM_METRIC          = "proc_pid_stat_comm" // PID + TID
	PROC_PID_STAT_COMM_LABEL_NAME      = "comm"
	PROC_PID_STAT_STARTTIME_LABEL_NAME = "starttime_msec"

	// If starttimeMsec cannot be parsed, use the following value:
	PROC_PID_STARTTIME_FALLBACK_VALUE = 0 // should default to epoch

	PROC_PID_STAT_INFO_METRIC        = "proc_pid_stat_info" // PID only
	PROC_PID_STAT_PPID_LABEL_NAME    = "ppid"
	PROC_PID_STAT_PGRP_LABEL_NAME    = "pgrp"
	PROC_PID_STAT_SESSION_LABEL_NAME = "session"
	PROC_PID_STAT_TTY_NR_LABEL_NAME  = "tty"
	PROC_PID_STAT_TPGID_LABEL_NAME   = "tpgid"
	PROC_PID_STAT_FLAGS_LABEL_NAME   = "flags"

	PROC_PID_STAT_PRIORITY_METRIC        = "proc_pid_stat_prio" // PID + TID
	PROC_PID_STAT_PRIORITY_LABEL_NAME    = "prio"
	PROC_PID_STAT_NICE_LABEL_NAME        = "nice"
	PROC_PID_STAT_RT_PRIORITY_LABEL_NAME = "rt_prio"
	PROC_PID_STAT_POLICY_LABEL_NAME      = "policy"

	PROC_PID_STAT_VSIZE_METRIC  = "proc_pid_stat_vsize_bytes"  // PID only
	PROC_PID_STAT_RSS_METRIC    = "proc_pid_stat_rss_bytes"    // PID only
	PROC_PID_STAT_RSSLIM_METRIC = "proc_pid_stat_rsslim_bytes" // PID only

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

	// /proc/PID/cmdline.
	PROC_PID_CMDLINE_METRIC              = "proc_pid_cmdline" // PID only, well behaved threads don't change their command line
	PROC_PID_CMDLINE_CMD_PATH_LABEL_NAME = "cmd_path"
	PROC_PID_CMDLINE_CMD_LABEL_NAME      = "cmd"
	PROC_PID_CMDLINE_ARGS_LABEL_NAME     = "args"

	// This generator's specific metrics, i.e. in addition to those described in
	// metrics_common.go:

	// They all have the following label:
	PROC_PID_PART_LABEL_NAME = "part" // partition

	// PID,TID counts:
	PROC_PID_TOTAL_COUNT_METRIC     = "proc_pid_total_count"     // total found in the dir scan
	PROC_PID_PARSE_OK_COUNT_METRIC  = "proc_pid_parse_ok_count"  // successfully parsed
	PROC_PID_PARSE_ERR_COUNT_METRIC = "proc_pid_parse_err_count" // parsing error
	PROC_PID_ACTIVE_COUNT_METRIC    = "proc_pid_active_count"    // found active
	PROC_PID_NEW_COUNT_METRIC       = "proc_pid_new_count"       // newly found
	PROC_PID_DEL_COUNT_METRIC       = "proc_pid_del_count"       // removed (out of scope or parsing error)

	// Interval since last generation, i.e. the interval underlying the deltas.
	// Normally this should be close to scan interval, but this is the actual
	// value, rather than the desired one:
	PROC_PID_INTERVAL_METRIC = "proc_pid_metrics_delta_sec"

	PROC_PID_SPECIFIC_METRICS_COUNT = 7
)

var procPidMetricsLog = NewCompLogger(PROC_PID_METRICS_ID)

// The list of PidStatus memory indexes used for PID+TID metrics; all the others
// are PID only. Note: it is implemented as a map for fast lookup:
var procPidStatusPidTidMetricMemoryIndex = map[int]bool{
	procfs.PID_STATUS_VM_STK: true,
}

type ProcPidMetricsConfig struct {
	// How often to generate the metrics in time.ParseDuration() format:
	Interval string `yaml:"interval"`
	// Normally metrics are generated only if there is a change in value from
	// the previous scan. However every N cycles the full set is generated. Use
	// 0 to generate full metrics every cycle.
	FullMetricsFactor int `yaml:"full_metrics_factor"`
	// How long the PID, TID cached list (shared among goroutines) is valid
	// before a new reading of /proc directory is required, in
	// time.ParseDuration() format:
	PidTidListCacheValidInterval string `yaml:"pid_list_cache_valid_interval"`
	// The number of partitions used to divide the process list; each partition
	// will generate a task and each task will run in a separate worker. A
	// negative value signifies the same value as the number of workers.
	NumPartitions int `yaml:"num_partitions"`
	// Whether to scan threads (/proc/PID/task/TID) and include thread metrics:
	ThreadMetrics bool `yaml:"thread_metrics"`
	// Whether to generate metrics based on /proc/PID/status or not.
	UsePidStatus bool `yaml:"use_pid_status"`
	// The list of the memory related fields in /proc/PID/status to use, as per
	// https://www.kernel.org/doc/Documentation/filesystems/proc.rst (see
	// "Contents of the status fields", "VmPeak" thru "HugetlbPages"). An
	// empty/nil list will cause all fields to be used.
	PidStatusMemoryFields []string `yaml:"pid_status_memory_fields"`
}

func DefaultProcPidMetricsConfig() *ProcPidMetricsConfig {
	return &ProcPidMetricsConfig{
		Interval:                     PROC_PID_METRICS_CONFIG_INTERVAL_DEFAULT,
		FullMetricsFactor:            PROC_PID_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT,
		ThreadMetrics:                PROC_PID_METRICS_CONFIG_THREAD_METRICS_DEFAULT,
		PidTidListCacheValidInterval: PROC_PID_METRICS_CONFIG_PID_LIST_CACHE_VALID_INTERVAL_DEFAULT,
		NumPartitions:                PROC_PID_METRICS_CONFIG_NUM_PART_DEFAULT,
		UsePidStatus:                 PROC_PID_METRICS_USE_PID_STATUS_DEFAULT,
	}
}

// PID, TID specific cached info:
type ProcPidTidMetricsInfo struct {
	// LRU pointers:
	next, prev *ProcPidTidMetricsInfo

	// Parsers, used to maintain the previous state:
	pidStat   procfs.PidStatParser
	pidStatus procfs.PidStatusParser

	// The time stamp when stats above were collected:
	prevTs time.Time

	// PID, TID cached info:
	pidTid       procfs.PidTid
	pidTidPath   string
	pidTidLabels string

	// Starttime label value converted to milliseconds:
	starttimeMsec string

	// Zero deltas:
	pidStatFltZeroDelta   []bool
	pidStatusCtxZeroDelta []bool

	// Cycle#, used for full metrics cycles:
	cycleNum int

	// Scan#, used to detect outdated PID, TID's:
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
// Due to the large number of PID, TID, an alternative to the dual buffer approach
// is used for the previous/current approach:
//  - the previous state is cached in a parser on a per PID, TID basis
//  - there is an unbound parser used to get the current state for a given PID, TID
//  - once the metrics are generated, the latter is swapped with the cached one,
//    i.e. it becomes the previous, while the freed previous will be used as the
//    current parser for the next PID, TID in the list.

type ProcPidMetrics struct {
	// id/task_id:
	id string
	// Scan interval:
	interval time.Duration
	// Full metric factor(s):
	fullMetricsFactor int
	// Whether to use /proc/PID/status metrics or not:
	usePidStatus bool
	// The list of PidStatus memory indexes used for metrics; if empty then they
	// are all used. Note: it is implemented as a map for fast lookup (is-in
	// function).
	pidStatusMemKeepIndex map[int]bool

	// The PidTid list cache, shared among ProcPidMetrics instances:
	pidTidListCache procfs.PidTidListCacheIF
	// The partition for the above:
	partNo int
	// Destination storage for the above:
	pidTidList []procfs.PidTid

	// Individual metrics cache, indexed by PID, TID:
	pidTidMetricsInfo map[procfs.PidTid]*ProcPidTidMetricsInfo
	// Also organized as a LRU list; the list will start with the entries not
	// found in the most recent scan, identified by scan# mismatch; those
	// entries will be deleted to keep the cache up-to-date, with no need to
	// scan the entire cache.
	pidTidMetricsInfoHead, pidTidMetricsInfoTail *ProcPidTidMetricsInfo

	// Unbound parsers, see Musical Chairs Approach For Deltas above:
	pidStat   procfs.PidStatParser
	pidStatus procfs.PidStatusParser

	// The command line is not cached, it is parsed for every full metrics cycle
	// when the metrics is generated. A single parser is used for all PID, TID:
	pidCmdline procfs.PidCmdlineParser

	// Scan#, used to detect outdated PID, TID's. This counter is incremented
	// for every scan and it is used to update the scan# for the cached PID, TID
	// info. At the end of the metrics generation, all the cache entries left
	// with an outdated scan# will be deleted.
	scanNum int

	// PID metrics (name+labels) are too numerous to be cached individually,
	// rather cache the format (the arg for fmt.Fprintf, that is) which has
	// format descriptors for label values such as PID, TID. The formats have to
	// be built at runtime to include the actual instance and hostname.
	intialized bool

	// PidStat based metric formats:
	pidStatStateMetricFmt    string
	pidStatCommMetricFmt     string
	pidStatInfoMetricFmt     string
	pidStatPriorityMetricFmt string
	pidStatCpuNumMetricFmt   string
	pidStatRssMetricFmt      string
	pidStatMemoryMetricFmt   []*ProcPidMetricsIndexFmt
	pidStatPcpuMetricFmt     []*ProcPidMetricsIndexFmt
	pidStatFltMetricFmt      []*ProcPidMetricsIndexFmt

	// PidStatus based metric formats:
	pidStatusInfoMetricFmt          string
	pidStatusPidOnlyMemoryMetricFmt []*ProcPidMetricsIndexFmt
	pidStatusPidTidMemoryMetricFmt  []*ProcPidMetricsIndexFmt
	pidStatusCtxMetricFmt           []*ProcPidMetricsIndexFmt

	// PidCmdline metric format:
	pidCmdlineMetricFmt string
	// Fallback for kernel threads and zombie processes where cmdline is empty:
	pidCmdlineCommMetricFmt string // use stat COMM field

	// Total metric counts per PID, determined once at the format update:
	perPidTidMetricCount  int
	perPidOnlyMetricCount int

	// Generator specific metrics formats:
	pidTotalCountMetricFmt    string
	pidParseOkCountMetricFmt  string
	pidParseErrCountMetricFmt string
	pidActiveCountMetricFmt   string
	pidNewCountMetricFmt      string
	pidDelCountMetricFmt      string
	intervalMetricFmt         string

	// Timestamp for the previous generator specific metrics:
	prevTs time.Time

	// A buffer for the timestamp:
	tsBuf *bytes.Buffer

	// Page size, needed to convert some of the memory stats:
	pageSize uint64

	// The following are needed for testing only. Left to their default values,
	// the usual objects will be used.
	instance, hostname  string
	timeNowFn           func() time.Time
	metricsQueue        MetricsQueue
	procfsRoot          string
	linuxClktckSec      float64
	boottimeMsec        int64
	newPidStatParser    procfs.NewPidStatParser
	newPidStatusParser  procfs.NewPidStatusParser
	newPidCmdlineParser procfs.NewPidCmdlineParser
}

func NewProcProcPidMetrics(cfg any, partNo int, pidTidListCache procfs.PidTidListCacheIF) (*ProcPidMetrics, error) {
	var (
		err                  error
		procPidMetricsConfig *ProcPidMetricsConfig
	)

	switch cfg := cfg.(type) {
	case *LsvmiConfig:
		procPidMetricsConfig = cfg.ProcPidMetricsConfig
	case *ProcPidMetricsConfig:
		procPidMetricsConfig = cfg
	case nil:
		procPidMetricsConfig = DefaultProcPidMetricsConfig()
	default:
		return nil, fmt.Errorf("NewProcProcPidMetrics: %T invalid config type", cfg)
	}

	interval, err := time.ParseDuration(procPidMetricsConfig.Interval)
	if err != nil {
		return nil, err
	}

	procPidMetrics := &ProcPidMetrics{
		id:                  fmt.Sprintf("%s#%d", PROC_PID_METRICS_ID, partNo),
		interval:            interval,
		fullMetricsFactor:   procPidMetricsConfig.FullMetricsFactor,
		usePidStatus:        procPidMetricsConfig.UsePidStatus,
		pidTidListCache:     pidTidListCache,
		partNo:              partNo,
		pidTidMetricsInfo:   make(map[procfs.PidTid]*ProcPidTidMetricsInfo),
		tsBuf:               &bytes.Buffer{},
		pageSize:            uint64(os.Getpagesize()),
		instance:            GlobalInstance,
		hostname:            GlobalHostname,
		timeNowFn:           time.Now,
		metricsQueue:        GlobalMetricsQueue,
		procfsRoot:          GlobalProcfsRoot,
		linuxClktckSec:      utils.LinuxClktckSec,
		boottimeMsec:        utils.OSBtime.UnixMilli(),
		newPidStatParser:    procfs.NewPidStat,
		newPidStatusParser:  procfs.NewPidStatus,
		newPidCmdlineParser: procfs.NewPidCmdline,
	}

	procPidMetricsLog.Infof("id=%s", procPidMetrics.id)
	procPidMetricsLog.Infof("interval=%s", procPidMetrics.interval)
	procPidMetricsLog.Infof("full_metrics_factor=%d", procPidMetrics.fullMetricsFactor)
	procPidMetricsLog.Infof("use_pid_status=%v", procPidMetrics.usePidStatus)

	if procPidMetrics.usePidStatus {
		if len(procPidMetricsConfig.PidStatusMemoryFields) > 0 {
			procPidMetrics.pidStatusMemKeepIndex = make(map[int]bool)
			for _, name := range procPidMetricsConfig.PidStatusMemoryFields {
				index := procfs.PidStatusNameToIndex(name)
				if index < 0 {
					return nil, fmt.Errorf("%q: invalid pid status memory metric selector", name)
				}
				procPidMetrics.pidStatusMemKeepIndex[index] = true
			}
		}
		procPidMetricsLog.Infof("pid_status_memory_fields=%v", procPidMetricsConfig.PidStatusMemoryFields)
	}

	return procPidMetrics, nil
}

func (pm *ProcPidMetrics) buildGeneratorSpecificMetricFmt(metricName string, valFmt string, labelNames ...string) string {
	metricFmt := fmt.Sprintf(
		`%s{%s="%s",%s="%s",%s="%d"`,
		metricName,
		INSTANCE_LABEL_NAME, pm.instance, HOSTNAME_LABEL_NAME, pm.hostname, PROC_PID_PART_LABEL_NAME, pm.partNo,
	)
	for _, label := range labelNames {
		metricFmt += fmt.Sprintf(`,%s="%%s"`, label)
	}
	metricFmt += fmt.Sprintf("} %s %%s\n", valFmt)
	return metricFmt
}

func (pm *ProcPidMetrics) buildMetricFmt(metricName string, valFmt string, labelNames ...string) string {
	metricFmt := fmt.Sprintf(
		`%s{%s="%s",%s="%s",%%s`,
		metricName,
		INSTANCE_LABEL_NAME, pm.instance, HOSTNAME_LABEL_NAME, pm.hostname,
	)
	for _, label := range labelNames {
		metricFmt += fmt.Sprintf(`,%s="%%s"`, label)
	}
	metricFmt += fmt.Sprintf("} %s %%s\n", valFmt)
	return metricFmt
}

func (pm *ProcPidMetrics) initMetricsCache() {
	pm.pidStatStateMetricFmt = pm.buildMetricFmt(
		PROC_PID_STAT_STATE_METRIC,
		"%c",
		PROC_PID_STAT_STATE_LABEL_NAME,
	)
	pm.perPidTidMetricCount++

	pm.pidStatCommMetricFmt = pm.buildMetricFmt(
		PROC_PID_STAT_COMM_METRIC,
		"%c",
		PROC_PID_STAT_STARTTIME_LABEL_NAME,
		PROC_PID_STAT_COMM_LABEL_NAME,
	)
	pm.perPidTidMetricCount++

	pm.pidStatInfoMetricFmt = pm.buildMetricFmt(
		PROC_PID_STAT_INFO_METRIC,
		"%c",
		PROC_PID_STAT_PPID_LABEL_NAME,
		PROC_PID_STAT_PGRP_LABEL_NAME,
		PROC_PID_STAT_SESSION_LABEL_NAME,
		PROC_PID_STAT_TTY_NR_LABEL_NAME,
		PROC_PID_STAT_TPGID_LABEL_NAME,
		PROC_PID_STAT_FLAGS_LABEL_NAME,
	)
	pm.perPidOnlyMetricCount++

	pm.pidStatPriorityMetricFmt = pm.buildMetricFmt(
		PROC_PID_STAT_PRIORITY_METRIC,
		"%c",
		PROC_PID_STAT_PRIORITY_LABEL_NAME,
		PROC_PID_STAT_NICE_LABEL_NAME,
		PROC_PID_STAT_RT_PRIORITY_LABEL_NAME,
		PROC_PID_STAT_POLICY_LABEL_NAME,
	)
	pm.perPidTidMetricCount++

	pm.pidStatRssMetricFmt = pm.buildMetricFmt(PROC_PID_STAT_RSS_METRIC, "%d")
	pm.pidStatMemoryMetricFmt = []*ProcPidMetricsIndexFmt{
		{
			procfs.PID_STAT_VSIZE,
			pm.buildMetricFmt(PROC_PID_STAT_VSIZE_METRIC, "%s"),
		},
		{
			procfs.PID_STAT_RSSLIM,
			pm.buildMetricFmt(PROC_PID_STAT_RSSLIM_METRIC, "%s"),
		},
	}
	pm.perPidOnlyMetricCount += len(pm.pidStatMemoryMetricFmt)

	pm.pidStatCpuNumMetricFmt = pm.buildMetricFmt(
		PROC_PID_STAT_CPU_NUM_METRIC, "%s",
	)
	pm.perPidTidMetricCount++

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
	pm.perPidTidMetricCount += len(pm.pidStatFltMetricFmt)

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
	pm.perPidTidMetricCount += len(pm.pidStatPcpuMetricFmt)

	if pm.usePidStatus {
		pm.pidStatusInfoMetricFmt = pm.buildMetricFmt(
			PROC_PID_STATUS_INFO_METRIC,
			"%c",
			PROC_PID_STATUS_UID_LABEL_NAME,
			PROC_PID_STATUS_GID_LABEL_NAME,
			PROC_PID_STATUS_GROUPS_LABEL_NAME,
			PROC_PID_STATUS_CPUS_ALLOWED_LIST_LABEL_NAME,
			PROC_PID_STATUS_MEMS_ALLOWED_LIST_LABEL_NAME,
		)
		pm.perPidOnlyMetricCount++

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

		pm.pidStatusPidOnlyMemoryMetricFmt = make(
			[]*ProcPidMetricsIndexFmt,
			0,
			len(pidStatusMemoryFmt)-len(procPidStatusPidTidMetricMemoryIndex),
		)
		pm.pidStatusPidTidMemoryMetricFmt = make(
			[]*ProcPidMetricsIndexFmt,
			0,
			len(procPidStatusPidTidMetricMemoryIndex),
		)
		for _, indexFmt := range pidStatusMemoryFmt {
			if len(pm.pidStatusMemKeepIndex) > 0 && !pm.pidStatusMemKeepIndex[indexFmt.index] {
				continue
			}
			if procPidStatusPidTidMetricMemoryIndex[indexFmt.index] {
				pm.pidStatusPidTidMemoryMetricFmt = append(pm.pidStatusPidTidMemoryMetricFmt, indexFmt)
			} else {
				pm.pidStatusPidOnlyMemoryMetricFmt = append(pm.pidStatusPidOnlyMemoryMetricFmt, indexFmt)
			}
		}

		pm.perPidTidMetricCount += len(pm.pidStatusPidTidMemoryMetricFmt)
		pm.perPidOnlyMetricCount += len(pm.pidStatusPidOnlyMemoryMetricFmt)

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
		pm.perPidTidMetricCount += len(pm.pidStatusCtxMetricFmt)
	}

	pm.pidCmdlineMetricFmt = pm.buildMetricFmt(
		PROC_PID_CMDLINE_METRIC, "%c",
		PROC_PID_CMDLINE_CMD_PATH_LABEL_NAME, PROC_PID_CMDLINE_ARGS_LABEL_NAME, PROC_PID_CMDLINE_CMD_LABEL_NAME,
	)
	pm.pidCmdlineCommMetricFmt = pm.buildMetricFmt(
		PROC_PID_CMDLINE_METRIC, "%c",
		PROC_PID_CMDLINE_CMD_LABEL_NAME,
	)
	pm.perPidOnlyMetricCount += 2

	pm.pidTotalCountMetricFmt = pm.buildGeneratorSpecificMetricFmt(PROC_PID_TOTAL_COUNT_METRIC, "%d")
	pm.pidParseOkCountMetricFmt = pm.buildGeneratorSpecificMetricFmt(PROC_PID_PARSE_OK_COUNT_METRIC, "%d")
	pm.pidParseErrCountMetricFmt = pm.buildGeneratorSpecificMetricFmt(PROC_PID_PARSE_ERR_COUNT_METRIC, "%d")
	pm.pidActiveCountMetricFmt = pm.buildGeneratorSpecificMetricFmt(PROC_PID_ACTIVE_COUNT_METRIC, "%d")
	pm.pidNewCountMetricFmt = pm.buildGeneratorSpecificMetricFmt(PROC_PID_NEW_COUNT_METRIC, "%d")
	pm.pidDelCountMetricFmt = pm.buildGeneratorSpecificMetricFmt(PROC_PID_DEL_COUNT_METRIC, "%d")
	pm.intervalMetricFmt = pm.buildGeneratorSpecificMetricFmt(PROC_PID_INTERVAL_METRIC, "%.6f")
}

func (pm *ProcPidMetrics) initialize() {
	pm.initMetricsCache()
	// Note the dummy PID, TID next; they will be overwritten in parser args:
	pm.pidStat = pm.newPidStatParser()
	if pm.usePidStatus {
		pm.pidStatus = pm.newPidStatusParser()
	}
	pm.pidCmdline = pm.newPidCmdlineParser()
	pm.intialized = true
}

func (pm *ProcPidMetrics) initPidTidMetricsInfo(pidTid procfs.PidTid, pidTidPath string) *ProcPidTidMetricsInfo {
	pidTidLabels := fmt.Sprintf(`%s="%d"`, PROC_PID_PID_LABEL_NAME, pidTid.Pid)
	if pidTid.Tid != procfs.PID_ONLY_TID {
		pidTidLabels += fmt.Sprintf(`,%s="%d"`, PROC_PID_TID_LABEL_NAME, pidTid.Tid)
	}

	pidStatBSF, _ := pm.pidStat.GetData()
	starttimeTck, err := strconv.ParseFloat(string(pidStatBSF[procfs.PID_STAT_STARTTIME]), 64)
	if err != nil {
		procPidMetricsLog.Warnf(
			`PID: %d, TID: %d, starttime: %v`,
			pidTid.Pid, pidTid.Tid, err,
		)
		starttimeTck = PROC_PID_STARTTIME_FALLBACK_VALUE
	}

	pidTidMetricsInfo := &ProcPidTidMetricsInfo{
		pidStat:               pm.newPidStatParser(),
		pidTid:                pidTid,
		pidTidPath:            pidTidPath,
		pidTidLabels:          pidTidLabels,
		starttimeMsec:         strconv.FormatInt(pm.boottimeMsec+int64(starttimeTck*pm.linuxClktckSec*1000.), 10),
		pidStatFltZeroDelta:   make([]bool, 2),
		pidStatusCtxZeroDelta: make([]bool, 2),
		cycleNum:              initialCycleNum.Get(pm.fullMetricsFactor),
	}
	if pm.usePidStatus {
		pidTidMetricsInfo.pidStatus = pm.newPidStatusParser()
	}

	return pidTidMetricsInfo
}

func (pm *ProcPidMetrics) generateMetrics(
	pidTidMetricsInfo *ProcPidTidMetricsInfo,
	hasPrev bool,
	isPid bool,
	fullMetrics bool,
	currTs time.Time,
	buf *bytes.Buffer,
) int {
	var (
		currPidStat, prevPidStat       procfs.PidStatParser
		currPidStatBSF, prevPidStatBSF [][]byte
		currPidStatNF, prevPidStatNF   []uint64

		currPidStatus, prevPidStatus       procfs.PidStatusParser
		currPidStatusBSF, prevPidStatusBSF [][]byte
		currPidStatusBSFU                  [][]byte
		currPidStatusNF, prevPidStatusNF   []uint64

		changed bool
	)

	currPidStat = pm.pidStat
	currPidStatBSF, currPidStatNF = currPidStat.GetData()
	if hasPrev {
		prevPidStat = pidTidMetricsInfo.pidStat
		prevPidStatBSF, prevPidStatNF = prevPidStat.GetData()
	}

	if pm.usePidStatus {
		currPidStatus = pm.pidStatus
		currPidStatusBSF, currPidStatusBSFU, currPidStatusNF = currPidStatus.GetData()
		if hasPrev {
			prevPidStatus = pidTidMetricsInfo.pidStatus
			prevPidStatusBSF, _, prevPidStatusNF = prevPidStatus.GetData()
		}
	}

	pm.tsBuf.Reset()
	fmt.Fprintf(pm.tsBuf, "%d", currTs.UnixMilli())
	ts := pm.tsBuf.Bytes()

	actualMetricsCount := 0
	fullMetricsNoPrev := fullMetrics || !hasPrev

	// PID + TID metrics:
	changed = hasPrev && !bytes.Equal(
		prevPidStatBSF[procfs.PID_STAT_STATE],
		currPidStatBSF[procfs.PID_STAT_STATE])
	if changed {
		// Clear previous state:
		fmt.Fprintf(
			buf,
			pm.pidStatStateMetricFmt,
			pidTidMetricsInfo.pidTidLabels,
			prevPidStatBSF[procfs.PID_STAT_STATE],
			'0',
			ts,
		)
		actualMetricsCount++
	}
	if fullMetricsNoPrev || changed {
		fmt.Fprintf(
			buf,
			pm.pidStatStateMetricFmt,
			pidTidMetricsInfo.pidTidLabels,
			currPidStatBSF[procfs.PID_STAT_STATE],
			'1',
			ts,
		)
		actualMetricsCount++
	}

	statCommChanged := hasPrev && !bytes.Equal(
		prevPidStatBSF[procfs.PID_STAT_COMM],
		currPidStatBSF[procfs.PID_STAT_COMM])
	if statCommChanged {
		// Clear previous state:
		fmt.Fprintf(
			buf,
			pm.pidStatCommMetricFmt,
			pidTidMetricsInfo.pidTidLabels,
			pidTidMetricsInfo.starttimeMsec,
			prevPidStatBSF[procfs.PID_STAT_COMM],
			'0',
			ts,
		)
		actualMetricsCount++
	}
	if fullMetricsNoPrev || statCommChanged {
		fmt.Fprintf(
			buf,
			pm.pidStatCommMetricFmt,
			pidTidMetricsInfo.pidTidLabels,
			pidTidMetricsInfo.starttimeMsec,
			currPidStatBSF[procfs.PID_STAT_COMM],
			'1',
			ts,
		)
		actualMetricsCount++
	}

	changed = false
	if hasPrev {
		// Check for change:
		for _, index := range []int{
			procfs.PID_STAT_PRIORITY,
			procfs.PID_STAT_NICE,
			procfs.PID_STAT_RT_PRIORITY,
			procfs.PID_STAT_POLICY,
		} {
			if changed = !bytes.Equal(
				prevPidStatBSF[index],
				currPidStatBSF[index]); changed {
				break
			}
		}
	}
	if changed {
		// Clear previous state:
		fmt.Fprintf(
			buf,
			pm.pidStatPriorityMetricFmt,
			pidTidMetricsInfo.pidTidLabels,
			prevPidStatBSF[procfs.PID_STAT_PRIORITY],
			prevPidStatBSF[procfs.PID_STAT_NICE],
			prevPidStatBSF[procfs.PID_STAT_RT_PRIORITY],
			prevPidStatBSF[procfs.PID_STAT_POLICY],
			'0',
			ts,
		)
		actualMetricsCount++
	}
	if fullMetricsNoPrev || changed {
		fmt.Fprintf(
			buf,
			pm.pidStatPriorityMetricFmt,
			pidTidMetricsInfo.pidTidLabels,
			currPidStatBSF[procfs.PID_STAT_PRIORITY],
			currPidStatBSF[procfs.PID_STAT_NICE],
			currPidStatBSF[procfs.PID_STAT_RT_PRIORITY],
			currPidStatBSF[procfs.PID_STAT_POLICY],
			'1',
			ts,
		)
		actualMetricsCount++
	}

	fmt.Fprintf(
		buf,
		pm.pidStatCpuNumMetricFmt,
		pidTidMetricsInfo.pidTidLabels,
		currPidStatBSF[procfs.PID_STAT_PROCESSOR],
		ts,
	)
	actualMetricsCount++

	if pm.usePidStatus {
		for _, indexFmt := range pm.pidStatusPidTidMemoryMetricFmt {
			if fullMetricsNoPrev || !bytes.Equal(
				prevPidStatusBSF[indexFmt.index],
				currPidStatusBSF[indexFmt.index]) {
				fmt.Fprintf(
					buf,
					indexFmt.fmt,
					pidTidMetricsInfo.pidTidLabels,
					currPidStatusBSFU[indexFmt.index],
					currPidStatusBSF[indexFmt.index],
					ts,
				)
				actualMetricsCount++
			}
		}
	}

	// Delta metrics require previous:
	if hasPrev {
		for i, indexFmt := range pm.pidStatFltMetricFmt {
			delta := currPidStatNF[indexFmt.index] - prevPidStatNF[indexFmt.index]
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

		totalPcpuMetricFmt := ""
		totalCpuDelta := uint64(0)
		deltaSec := currTs.Sub(pidTidMetricsInfo.prevTs).Seconds()
		pcpuFactor := pm.linuxClktckSec / deltaSec * 100.
		for _, indexFmt := range pm.pidStatPcpuMetricFmt {
			if indexFmt.index < 0 { // i.e. total (user + system)
				totalPcpuMetricFmt = indexFmt.fmt
				continue
			}
			delta := currPidStatNF[indexFmt.index] - prevPidStatNF[indexFmt.index]
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

		if pm.usePidStatus {
			for i, indexFmt := range pm.pidStatusCtxMetricFmt {
				delta := currPidStatusNF[indexFmt.index] - prevPidStatusNF[indexFmt.index]
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
	}

	if !isPid {
		return actualMetricsCount
	}

	// PID only metrics:
	changed = false
	if hasPrev {
		// Check for change:
		for _, index := range []int{
			procfs.PID_STAT_PPID,
			procfs.PID_STAT_PGRP,
			procfs.PID_STAT_SESSION,
			procfs.PID_STAT_TTY_NR,
			procfs.PID_STAT_TPGID,
			procfs.PID_STAT_FLAGS,
		} {
			if changed = !bytes.Equal(
				prevPidStatBSF[index],
				currPidStatBSF[index]); changed {
				break
			}
		}
	}
	if changed {
		// Clear previous state:
		fmt.Fprintf(
			buf,
			pm.pidStatInfoMetricFmt,
			pidTidMetricsInfo.pidTidLabels,
			prevPidStatBSF[procfs.PID_STAT_PPID],
			prevPidStatBSF[procfs.PID_STAT_PGRP],
			prevPidStatBSF[procfs.PID_STAT_SESSION],
			prevPidStatBSF[procfs.PID_STAT_TTY_NR],
			prevPidStatBSF[procfs.PID_STAT_TPGID],
			prevPidStatBSF[procfs.PID_STAT_FLAGS],
			'0',
			ts,
		)
		actualMetricsCount++
	}
	if fullMetricsNoPrev || changed {
		fmt.Fprintf(
			buf,
			pm.pidStatInfoMetricFmt,
			pidTidMetricsInfo.pidTidLabels,
			currPidStatBSF[procfs.PID_STAT_PPID],
			currPidStatBSF[procfs.PID_STAT_PGRP],
			currPidStatBSF[procfs.PID_STAT_SESSION],
			currPidStatBSF[procfs.PID_STAT_TTY_NR],
			currPidStatBSF[procfs.PID_STAT_TPGID],
			currPidStatBSF[procfs.PID_STAT_FLAGS],
			'1',
			ts,
		)
		actualMetricsCount++
	}

	if rss := currPidStatNF[procfs.PID_STAT_RSS]; fullMetricsNoPrev || rss != prevPidStatNF[procfs.PID_STAT_RSS] {
		fmt.Fprintf(
			buf,
			pm.pidStatRssMetricFmt,
			pidTidMetricsInfo.pidTidLabels,
			rss*pm.pageSize,
			ts,
		)
		actualMetricsCount++
	}

	for _, indexFmt := range pm.pidStatMemoryMetricFmt {
		if fullMetricsNoPrev || !bytes.Equal(
			prevPidStatBSF[indexFmt.index],
			currPidStatBSF[indexFmt.index]) {
			fmt.Fprintf(
				buf,
				indexFmt.fmt,
				pidTidMetricsInfo.pidTidLabels,
				currPidStatBSF[indexFmt.index],
				ts,
			)
			actualMetricsCount++
		}
	}

	if pm.usePidStatus {
		// Check for change:
		changed = false
		if hasPrev {
			for _, index := range []int{
				procfs.PID_STATUS_UID,
				procfs.PID_STATUS_GID,
				procfs.PID_STATUS_GROUPS,
				procfs.PID_STATUS_CPUS_ALLOWED_LIST,
				procfs.PID_STATUS_MEMS_ALLOWED_LIST,
			} {
				if changed = !bytes.Equal(
					prevPidStatusBSF[index],
					currPidStatusBSF[index]); changed {
					break
				}
			}
		}
		if changed {
			// Clear prev metric:
			fmt.Fprintf(
				buf,
				pm.pidStatusInfoMetricFmt,
				pidTidMetricsInfo.pidTidLabels,
				prevPidStatusBSF[procfs.PID_STATUS_UID],
				prevPidStatusBSF[procfs.PID_STATUS_GID],
				prevPidStatusBSF[procfs.PID_STATUS_GROUPS],
				prevPidStatusBSF[procfs.PID_STATUS_CPUS_ALLOWED_LIST],
				prevPidStatusBSF[procfs.PID_STATUS_MEMS_ALLOWED_LIST],
				'0',
				ts,
			)
			actualMetricsCount++
		}
		if fullMetricsNoPrev || changed {
			fmt.Fprintf(
				buf,
				pm.pidStatusInfoMetricFmt,
				pidTidMetricsInfo.pidTidLabels,
				currPidStatusBSF[procfs.PID_STATUS_UID],
				currPidStatusBSF[procfs.PID_STATUS_GID],
				currPidStatusBSF[procfs.PID_STATUS_GROUPS],
				currPidStatusBSF[procfs.PID_STATUS_CPUS_ALLOWED_LIST],
				currPidStatusBSF[procfs.PID_STATUS_MEMS_ALLOWED_LIST],
				'1',
				ts,
			)
			actualMetricsCount++
		}

		for _, indexFmt := range pm.pidStatusPidOnlyMemoryMetricFmt {
			if fullMetricsNoPrev || !bytes.Equal(
				prevPidStatusBSF[indexFmt.index],
				currPidStatusBSF[indexFmt.index]) {
				fmt.Fprintf(
					buf,
					indexFmt.fmt,
					pidTidMetricsInfo.pidTidLabels,
					currPidStatusBSFU[indexFmt.index],
					currPidStatusBSF[indexFmt.index],
					ts,
				)
				actualMetricsCount++
			}
		}
	}

	if fullMetricsNoPrev {
		cmdPath, args, cmd := pm.pidCmdline.GetData()
		if len(cmdPath) != 0 {
			fmt.Fprintf(
				buf,
				pm.pidCmdlineMetricFmt,
				pidTidMetricsInfo.pidTidLabels,
				cmdPath, args, cmd,
				'1',
				ts,
			)
		} else {
			if statCommChanged {
				fmt.Fprintf(
					buf,
					pm.pidCmdlineCommMetricFmt,
					pidTidMetricsInfo.pidTidLabels,
					prevPidStatBSF[procfs.PID_STAT_COMM],
					'0',
					ts,
				)
				actualMetricsCount++
			}
			// Fallback over stat COMM field:
			fmt.Fprintf(
				buf,
				pm.pidCmdlineCommMetricFmt,
				pidTidMetricsInfo.pidTidLabels,
				currPidStatBSF[procfs.PID_STAT_COMM],
				'1',
				ts,
			)
		}
		actualMetricsCount++
	}

	return actualMetricsCount
}

// Satisfy the TaskActivity interface:
func (pm *ProcPidMetrics) Execute() bool {
	// If this is the 1st call, initialize various structures:
	hasPrev := pm.intialized
	if !hasPrev {
		pm.initialize()
	}

	// Get the current list of PID, TID to be handled by this generator:
	pidTidList, err := pm.pidTidListCache.GetPidTidList(pm.partNo, pm.pidTidList)
	if err != nil {
		procPidMetricsLog.Errorf("GetPidTidList(part=%d): %v", pm.partNo, err)
		return false
	}
	// The list stoarge will be reused next time:
	pm.pidTidList = pidTidList

	// Advance the scan number; never use 0 since PID, TID cache entries are
	// initialized w/ 0 and they may appear to be up-to-date when in fact they
	// aren't:
	scanNum := pm.scanNum + 1
	if scanNum == 0 {
		scanNum = 1
	}

	actualMetricsCount := 0
	bufTargetSize := pm.metricsQueue.GetTargetSize()
	pidTidCount, pidOnlyCount, activePidTidCount, addPidCount, delPidCount := 0, 0, 0, 0, 0
	byteCount := 0
	var buf *bytes.Buffer

	var (
		currPidStatBSF, prevPidStatBSF [][]byte
		currPidStatNF, prevPidStatNF   []uint64
	)

	for _, pidTid := range pidTidList {
		pidTidPath := ""

		isPid := pidTid.Tid == procfs.PID_ONLY_TID

		// Parse PID/stat first; this will be needed to determine whether this
		// is the same process/thread from the previous scan and whether it is
		// active or not:
		pidTidMetricsInfo, hasPrev := pm.pidTidMetricsInfo[pidTid]
		if !hasPrev {
			pidTidPath = procfs.BuildPidTidPath(pm.procfsRoot, pidTid.Pid, pidTid.Tid)
		} else {
			pidTidPath = pidTidMetricsInfo.pidTidPath
			// Remove from LRU:
			if pidTidMetricsInfo.next != nil {
				pidTidMetricsInfo.next.prev = pidTidMetricsInfo.prev
			} else {
				pm.pidTidMetricsInfoTail = pidTidMetricsInfo.prev
			}
			if pidTidMetricsInfo.prev != nil {
				pidTidMetricsInfo.prev.next = pidTidMetricsInfo.next
			} else {
				pm.pidTidMetricsInfoHead = pidTidMetricsInfo.next
			}
			pidTidMetricsInfo.next = nil
			pidTidMetricsInfo.prev = nil
		}
		err = pm.pidStat.Parse(pidTidPath)
		if err != nil {
			procPidMetricsLog.Error(err)
			if hasPrev {
				delete(pm.pidTidMetricsInfo, pidTid)
				delPidCount++
			}
			continue
		}

		fullMetrics := false
		if hasPrev {
			currPidStatBSF, currPidStatNF = pm.pidStat.GetData()
			prevPidStatBSF, prevPidStatNF = pidTidMetricsInfo.pidStat.GetData()

			// Check if the PID[,TID] was in fact reused since last scan, based
			// on starttime. This is only a theoretical possiblity, but still
			// robust conding and all.

			if !bytes.Equal(currPidStatBSF[procfs.PID_STAT_STARTTIME], prevPidStatBSF[procfs.PID_STAT_STARTTIME]) {
				hasPrev = false
				// Technically a delete:
				delPidCount++
			} else {
				fullMetrics = pidTidMetricsInfo.cycleNum == 0
			}
		}

		// Active?
		active := false
		if !hasPrev {
			// By definition 1st time PID, TID is deemed active:
			active = true
			pidTidMetricsInfo = pm.initPidTidMetricsInfo(pidTid, pidTidPath)
		} else if currPidStatNF[procfs.PID_STAT_UTIME] != prevPidStatNF[procfs.PID_STAT_UTIME] ||
			currPidStatNF[procfs.PID_STAT_STIME] != prevPidStatNF[procfs.PID_STAT_STIME] {
			active = true
		} else if !fullMetrics {
			// Inactive, non full metrics cycle. Mark it as scanned but otherwise do nothing:
			pidTidMetricsInfo.cycleNum++
			if pidTidMetricsInfo.cycleNum >= pm.fullMetricsFactor {
				pidTidMetricsInfo.cycleNum = 0
			}
			pidTidMetricsInfo.scanNum = scanNum
			pidTidMetricsInfo.prevTs = pm.timeNowFn()
			// (Re)add to the tail of LRU:
			if pm.pidTidMetricsInfoTail != nil {
				pm.pidTidMetricsInfoTail.next = pidTidMetricsInfo
				pidTidMetricsInfo.prev = pm.pidTidMetricsInfoTail
			} else {
				pm.pidTidMetricsInfoHead = pidTidMetricsInfo
			}
			pm.pidTidMetricsInfoTail = pidTidMetricsInfo
			// Update PID/TID counts:
			pidTidCount++
			if isPid {
				pidOnlyCount++
			}
			continue
		}

		if pm.usePidStatus {
			err = pm.pidStatus.Parse(pidTidPath)
			if err != nil {
				procPidMetricsLog.Error(err)
				if hasPrev {
					delete(pm.pidTidMetricsInfo, pidTid)
					delPidCount++
				}
				continue
			}
		}
		if isPid && (fullMetrics || !hasPrev) {
			err = pm.pidCmdline.Parse(pidTidPath)
			if err != nil {
				procPidMetricsLog.Error(err)
				if hasPrev {
					delete(pm.pidTidMetricsInfo, pidTid)
					delPidCount++
				}
				continue
			}
		}

		currTs := pm.timeNowFn()
		if buf == nil {
			buf = pm.metricsQueue.GetBuf()
		}
		actualMetricsCount += pm.generateMetrics(pidTidMetricsInfo, hasPrev, isPid, fullMetrics, currTs, buf)
		if buf.Len() > bufTargetSize {
			byteCount += buf.Len()
			pm.metricsQueue.QueueBuf(buf)
			buf = nil
		}
		// Swap the per PID, TID parsers w/ the metrics generator ones:
		pidTidMetricsInfo.pidStat, pm.pidStat = pm.pidStat, pidTidMetricsInfo.pidStat
		if pm.usePidStatus {
			pidTidMetricsInfo.pidStatus, pm.pidStatus = pm.pidStatus, pidTidMetricsInfo.pidStatus
		}
		// Mark it as scanned:
		pidTidMetricsInfo.prevTs = currTs
		pidTidMetricsInfo.cycleNum++
		if pidTidMetricsInfo.cycleNum >= pm.fullMetricsFactor {
			pidTidMetricsInfo.cycleNum = 0
		}
		pidTidMetricsInfo.scanNum = scanNum
		// Store it in the cache as needed:
		if !hasPrev {
			pm.pidTidMetricsInfo[pidTid] = pidTidMetricsInfo
			addPidCount++
		}
		// (Re)add to the tail of LRU:
		if pm.pidTidMetricsInfoTail != nil {
			pm.pidTidMetricsInfoTail.next = pidTidMetricsInfo
			pidTidMetricsInfo.prev = pm.pidTidMetricsInfoTail
		} else {
			pm.pidTidMetricsInfoHead = pidTidMetricsInfo
		}
		pm.pidTidMetricsInfoTail = pidTidMetricsInfo
		// Update PID/TID counts:
		pidTidCount++
		if isPid {
			pidOnlyCount++
		}
		if active {
			activePidTidCount++
		}
	}

	// Remove outdated PID, TID's from cache:
	for pidTidMetricsInfo := pm.pidTidMetricsInfoHead; pidTidMetricsInfo != nil; {
		if pidTidMetricsInfo.scanNum == scanNum {
			break
		}
		delete(pm.pidTidMetricsInfo, pidTidMetricsInfo.pidTid)
		delPidCount++
		pidTidMetricsInfo = pidTidMetricsInfo.next
		pm.pidTidMetricsInfoHead = pidTidMetricsInfo
		pidTidMetricsInfo.prev = nil
	}

	// This generator's specific metrics:
	currTs := pm.timeNowFn()
	pm.tsBuf.Reset()
	fmt.Fprintf(pm.tsBuf, "%d", currTs.UnixMilli())
	ts := pm.tsBuf.Bytes()
	if buf == nil {
		buf = pm.metricsQueue.GetBuf()
	}
	pidTidTotalCount := len(pm.pidTidList)
	fmt.Fprintf(buf, pm.pidTotalCountMetricFmt, pidTidTotalCount, ts)
	fmt.Fprintf(buf, pm.pidParseOkCountMetricFmt, pidTidCount, ts)
	fmt.Fprintf(buf, pm.pidParseErrCountMetricFmt, pidTidTotalCount-pidTidCount, ts)
	fmt.Fprintf(buf, pm.pidActiveCountMetricFmt, activePidTidCount, ts)
	fmt.Fprintf(buf, pm.pidNewCountMetricFmt, addPidCount, ts)
	fmt.Fprintf(buf, pm.pidDelCountMetricFmt, delPidCount, ts)
	actualMetricsCount += PROC_PID_SPECIFIC_METRICS_COUNT - 1
	if hasPrev {
		fmt.Fprintf(buf, pm.intervalMetricFmt, currTs.Sub(pm.prevTs).Seconds(), ts)
		actualMetricsCount++
	}
	byteCount += buf.Len()
	pm.metricsQueue.QueueBuf(buf)
	pm.prevTs = currTs

	// Generator stats:
	totalMetricsCount := pm.perPidTidMetricCount*pidTidCount + pm.perPidOnlyMetricCount*pidOnlyCount + PROC_PID_SPECIFIC_METRICS_COUNT
	GlobalMetricsGeneratorStatsContainer.Update(
		pm.id, uint64(actualMetricsCount), uint64(totalMetricsCount), uint64(byteCount),
	)

	// Update scan#:
	pm.scanNum = scanNum

	return true
}

// Define and register the task builder:
func ProcPidMetricsTaskBuilder(cfg *LsvmiConfig) ([]*Task, error) {
	procPidMetricsConfig := cfg.ProcPidMetricsConfig

	interval, err := time.ParseDuration(procPidMetricsConfig.Interval)
	if err != nil {
		return nil, fmt.Errorf("interval: %v", err)
	}

	if interval <= 0 {
		procPidMetricsLog.Info("proc PID metrics disabled")
		return nil, nil
	}

	numPart := procPidMetricsConfig.NumPartitions
	if numPart <= 0 {
		numPart = GlobalScheduler.numWorkers
	}
	validFor, err := time.ParseDuration(procPidMetricsConfig.PidTidListCacheValidInterval)
	if err != nil {
		return nil, fmt.Errorf("pid_list_cache_valid_interval: %v", err)
	}
	flags := procfs.PID_LIST_CACHE_PID_ENABLED
	if procPidMetricsConfig.ThreadMetrics {
		flags |= procfs.PID_LIST_CACHE_TID_ENABLED
	}
	procPidMetricsLog.Infof(
		"num_partitions=%d (config), %d (using)",
		procPidMetricsConfig.NumPartitions,
		numPart,
	)
	procPidMetricsLog.Infof("pid_list_cache_valid_interval=%s", validFor)
	pidTidListCache := procfs.NewPidTidListCache(GlobalProcfsRoot, numPart, validFor, flags)

	tasks := make([]*Task, numPart)
	for partNo := 0; partNo < numPart; partNo++ {
		pm, err := NewProcProcPidMetrics(procPidMetricsConfig, partNo, pidTidListCache)
		if err != nil {
			return nil, err
		}
		tasks[partNo] = NewTask(pm.id, pm.interval, pm)
	}
	return tasks, nil
}

func init() {
	TaskBuilders.Register(ProcPidMetricsTaskBuilder)
}
