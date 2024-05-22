// statfs metrics, a-la df (disk free) command
package lsvmi

import (
	"bytes"
	"fmt"
	"time"

	"golang.org/x/sys/unix"

	"github.com/eparparita/linux-stats-victoriametrics-importer/procfs"
)

const (
	STATFS_METRICS_CONFIG_INTERVAL_DEFAULT            = "5s"
	STATFS_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT = 12
	STATFS_METRICS_CONFIG_MOUNTINFO_PID_DEFAULT       = 0 // i.e. self

	// This generator id:
	STATFS_METRICS_ID = "statfs_metrics"
)

const (
	STATFS_BSIZE_METRIC      = "statfs_bsize"
	STATFS_BLOCKS_METRIC     = "statfs_blocks"
	STATFS_BFREE_METRIC      = "statfs_bfree"
	STATFS_BAVAIL_METRIC     = "statfs_bavail"
	STATFS_FILES_METRIC      = "statfs_files"
	STATFS_FFREE_METRIC      = "statfs_ffree"
	STATFS_TOTAL_SIZE_METRIC = "statfs_total_size_kb"
	STATFS_AVAIL_SIZE_METRIC = "statfs_avail_size_kb"
	STATFS_USED_SIZE_METRIC  = "statfs_used_size_kb"
	STATFS_USED_PCT_METRIC   = "statfs_used_pct"

	STATFS_MOUNT_POINT_LABEL_NAME   = "mountPoint"
	STATFS_MOUNT_FS_LABEL_NAME      = "fs"
	STATFS_MOUNT_FS_TYPE_LABEL_NAME = "fs_type"
)

var statfsMetricsLog = NewCompLogger(STATFS_METRICS_ID)

// The default list of filesystem types to include; if nil then all are included:
var defaultIncludeFilesystemTypeList []string = nil

// The default list of filesystem types to exclude; if nil then none is
// excluded. If a filesystem type appears in both lists then exclude takes
// precedence. The usual suspects are remote file systems:
var defaultExcludeFilesystemTypeList = []string{
	"dav",
	"sftp",
	"smb",
	"cifs",
	"nfs",
	"davfs2",
	"sshfs",
}

type StatfsMetricsConfig struct {
	// How often to generate the metrics in time.ParseDuration() format:
	Interval string `yaml:"interval"`
	// Normally metrics are generated only if there is a change in value from
	// the previous scan. However every N cycles the full set is generated. Use
	// 0 to generate full metrics every cycle.
	FullMetricsFactor int `yaml:"full_metrics_factor"`
	// The PID to use for /proc/PID/mountinfo, use 0 for self:
	MountinfoPid int `yaml:"mountinfo_pid"`
	// The list list of filesystem types to include; if not defined/empty then
	// all are included:
	IncludeFilesystemTypes []string `yaml:"include_file_system_types"`
	// The list list of filesystem types to exclude; if not defined/empty then
	// none is excluded. If a filesystem type appears in both lists then exclude
	// takes precedence.
	ExcludeFilesystemTypes []string `yaml:"exclude_file_system_types"`
}

func DefaultStatfsMetricsConfig() *StatfsMetricsConfig {
	cfg := &StatfsMetricsConfig{
		Interval:          STATFS_METRICS_CONFIG_INTERVAL_DEFAULT,
		FullMetricsFactor: STATFS_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT,
		MountinfoPid:      STATFS_METRICS_CONFIG_MOUNTINFO_PID_DEFAULT,
	}

	if defaultIncludeFilesystemTypeList != nil {
		cfg.IncludeFilesystemTypes = make([]string, len(defaultIncludeFilesystemTypeList))
		copy(cfg.IncludeFilesystemTypes, defaultIncludeFilesystemTypeList)
	}

	if defaultExcludeFilesystemTypeList != nil {
		cfg.ExcludeFilesystemTypes = make([]string, len(defaultExcludeFilesystemTypeList))
		copy(cfg.ExcludeFilesystemTypes, defaultExcludeFilesystemTypeList)
	}

	return cfg
}

// Per mount source info:
type StatfsInfo struct {
	// Dual buffer for the sys call:
	statfsBuf [2]*unix.Statfs_t
	// Metrics cache:
	bsizeMetric     []byte
	blocksMetric    []byte
	bfreeMetric     []byte
	bavailMetric    []byte
	filesMetric     []byte
	ffreeMetric     []byte
	totalSizeMetric []byte
	availSizeMetric []byte
	usedSizeMetric  []byte
	usedPctMetric   []byte
	// Cycle#:
	cycleNum int
	// If statfs reports an error, mark this mount point as disabled:
	disabled bool
}

type StatfsMetrics struct {
	// id/task_id:
	id string
	// Scan interval:
	interval time.Duration

	// Stats indexed by mount source; there can be multiple mounts for the same
	// source, the first one encountered will be considered:
	statfsInfo map[string]*StatfsInfo
	// Timestamp when the stats were collected:
	statfsTs [2]time.Time
	// Index for current stats, toggled after each use:
	currIndex int

	// Mountinfo:
	mountinfoPid      int
	procMountinfo     *procfs.Mountinfo
	mountinfoCycleNum int

	// Filesystem types to include/exclude. A nil list means no restriction.
	// Exclude takes precedence over include.
	includeFilesystemTypes, excludeFilesystemTypes map[string]bool

	// Full metric factor:
	fullMetricsFactor int

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

func NewStatfsMetrics(cfg any) (*StatfsMetrics, error) {
	var (
		err              error
		statfsMetricsCfg *StatfsMetricsConfig
	)

	switch cfg := cfg.(type) {
	case *LsvmiConfig:
		statfsMetricsCfg = cfg.StatfsMetricsConfig
	case *StatfsMetricsConfig:
		statfsMetricsCfg = cfg
	case nil:
		statfsMetricsCfg = DefaultStatfsMetricsConfig()
	default:
		return nil, fmt.Errorf("NewStatfsMetrics: %T invalid config type", cfg)
	}

	interval, err := time.ParseDuration(statfsMetricsCfg.Interval)
	if err != nil {
		return nil, err
	}
	statfsMetrics := &StatfsMetrics{
		id:                STATFS_METRICS_ID,
		interval:          interval,
		statfsInfo:        make(map[string]*StatfsInfo),
		mountinfoPid:      statfsMetricsCfg.MountinfoPid,
		mountinfoCycleNum: initialCycleNum.Get(statfsMetricsCfg.FullMetricsFactor),
		fullMetricsFactor: statfsMetricsCfg.FullMetricsFactor,
		tsSuffixBuf:       &bytes.Buffer{},
	}

	if statfsMetricsCfg.IncludeFilesystemTypes != nil {
		statfsMetrics.includeFilesystemTypes = make(map[string]bool)
		for _, fsType := range statfsMetricsCfg.IncludeFilesystemTypes {
			statfsMetrics.includeFilesystemTypes[fsType] = true
		}
	}

	if statfsMetricsCfg.ExcludeFilesystemTypes != nil {
		statfsMetrics.excludeFilesystemTypes = make(map[string]bool)
		for _, fsType := range statfsMetricsCfg.ExcludeFilesystemTypes {
			statfsMetrics.excludeFilesystemTypes[fsType] = true
		}
	}

	statfsMetricsLog.Infof("id=%s", statfsMetrics.id)
	statfsMetricsLog.Infof("interval=%s", statfsMetrics.interval)
	statfsMetricsLog.Infof("full_metrics_factor=%d", statfsMetrics.fullMetricsFactor)
	statfsMetricsLog.Infof("mountinfoPid=%d", statfsMetrics.mountinfoPid)
	return statfsMetrics, nil
}

func (sfsm *StatfsMetrics) updateStatfsInfo() {
	instance, hostname := GlobalInstance, GlobalHostname
	if sfsm.instance != "" {
		instance = sfsm.instance
	}
	if sfsm.hostname != "" {
		hostname = sfsm.hostname
	}

	mountSources := make(map[string]bool)
	for _, parsedLine := range sfsm.procMountinfo.ParsedLines {
		fsType := string(parsedLine[procfs.MOUNTINFO_FS_TYPE])
		if sfsm.excludeFilesystemTypes != nil && sfsm.excludeFilesystemTypes[fsType] ||
			sfsm.includeFilesystemTypes != nil && !sfsm.includeFilesystemTypes[fsType] {
			continue
		}
		mountSource := string(parsedLine[procfs.MOUNTINFO_MOUNT_SOURCE])
		if mountSources[mountSource] {
			continue // already encountered
		}
		mountSources[mountSource] = true
		statfsInfo := sfsm.statfsInfo[mountSource]
		if statfsInfo == nil {
			statfsInfo = &StatfsInfo{
				cycleNum: initialCycleNum.Get(sfsm.fullMetricsFactor),
			}
			sfsm.statfsInfo[mountSource] = statfsInfo
		} else if statfsInfo.disabled {
			continue
		}
		labels := fmt.Sprintf(
			`%s="%s",%s="%s",%s="%s",%s="%s",%s="%s"`,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
			STATFS_MOUNT_POINT_LABEL_NAME, parsedLine[procfs.MOUNTINFO_MOUNT_POINT],
			STATFS_MOUNT_FS_LABEL_NAME, mountSource,
			STATFS_MOUNT_FS_TYPE_LABEL_NAME, fsType,
		)
		statfsInfo.bsizeMetric = []byte(fmt.Sprintf(
			`%s{%s} `, // N.B. space before value included
			STATFS_BSIZE_METRIC, labels,
		))
		statfsInfo.blocksMetric = []byte(fmt.Sprintf(
			`%s{%s} `, // N.B. space before value included
			STATFS_BLOCKS_METRIC, labels,
		))
		statfsInfo.bfreeMetric = []byte(fmt.Sprintf(
			`%s{%s} `, // N.B. space before value included
			STATFS_BFREE_METRIC, labels,
		))
		statfsInfo.bavailMetric = []byte(fmt.Sprintf(
			`%s{%s} `, // N.B. space before value included
			STATFS_BAVAIL_METRIC, labels,
		))
		statfsInfo.filesMetric = []byte(fmt.Sprintf(
			`%s{%s} `, // N.B. space before value included
			STATFS_FILES_METRIC, labels,
		))
		statfsInfo.ffreeMetric = []byte(fmt.Sprintf(
			`%s{%s} `, // N.B. space before value included
			STATFS_FFREE_METRIC, labels,
		))
		statfsInfo.totalSizeMetric = []byte(fmt.Sprintf(
			`%s{%s} `, // N.B. space before value included
			STATFS_TOTAL_SIZE_METRIC, labels,
		))
		statfsInfo.availSizeMetric = []byte(fmt.Sprintf(
			`%s{%s} `, // N.B. space before value included
			STATFS_AVAIL_SIZE_METRIC, labels,
		))
		statfsInfo.usedSizeMetric = []byte(fmt.Sprintf(
			`%s{%s} `, // N.B. space before value included
			STATFS_USED_SIZE_METRIC, labels,
		))
		statfsInfo.usedPctMetric = []byte(fmt.Sprintf(
			`%s{%s} `, // N.B. space before value included
			STATFS_USED_PCT_METRIC, labels,
		))
	}

	// Remove out of scope statfs info:
	for mountSource := range sfsm.statfsInfo {
		if !mountSources[mountSource] {
			delete(sfsm.statfsInfo, mountSource)
		}
	}
}
