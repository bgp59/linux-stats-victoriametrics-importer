// statfs metrics, a-la df (disk free) command
package lsvmi

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"time"

	"golang.org/x/sys/unix"

	"github.com/emypar/linux-stats-victoriametrics-importer/procfs"
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
	STATFS_FREE_SIZE_METRIC  = "statfs_free_size_kb"
	STATFS_AVAIL_SIZE_METRIC = "statfs_avail_size_kb"
	STATFS_FREE_PCT_METRIC   = "statfs_free_pct"
	STATFS_AVAIL_PCT_METRIC  = "statfs_avail_pct"

	STATFS_PRESENCE_METRIC = "statfs_present"

	STATFS_DEVICE_NUM_METRICS = 11

	STATFS_MOUNT_POINT_LABEL_NAME   = "mount_point"
	STATFS_MOUNT_FS_LABEL_NAME      = "fs"
	STATFS_MOUNT_FS_TYPE_LABEL_NAME = "fs_type"

	STATFS_INTERVAL_METRIC = "statfs_metrics_delta_sec"
)

const (
	STATFS_FREE_PCT_METRIC_PREC  = 1
	STATFS_AVAIL_PCT_METRIC_PREC = 1
)

var statfsMetricsLog = NewCompLogger(STATFS_METRICS_ID)

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
	return cfg
}

// Per mount source info:
type StatfsInfo struct {
	// Relevant mount info:
	mountPoint  string
	mountSource string
	fsType      string
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
	freeSizeMetric  []byte
	availSizeMetric []byte
	freePctMetric   []byte
	availPctMetric  []byte
	enabledMetric   []byte
	// Cycle#:
	cycleNum int
	// If statfs reports an error, mark this mount point as disabled:
	disabled bool
	// Whether this was disabled before, needed to detect the transition to
	// disabled:
	wasDisabled bool
}

type StatfsMetrics struct {
	// id/task_id:
	id string
	// Scan interval:
	interval time.Duration

	// Stats indexed by mount source; there can be multiple mounts for the same
	// source, the first one encountered will be considered:
	statfsInfo map[string]*StatfsInfo
	// The list of out-of-scope present metrics, they should be published w/
	// 0 status:
	outOfScopePresentMetrics [][]byte
	// Timestamp when the stats were collected:
	statfsTs [2]time.Time
	// Whether there is a prev timestamp or not, essentially not first time
	// invocation:
	validPrevTs bool
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

	if len(statfsMetricsCfg.IncludeFilesystemTypes) > 0 {
		statfsMetrics.includeFilesystemTypes = make(map[string]bool)
		for _, fsType := range statfsMetricsCfg.IncludeFilesystemTypes {
			statfsMetrics.includeFilesystemTypes[fsType] = true
		}
	}

	if len(statfsMetricsCfg.ExcludeFilesystemTypes) > 0 {
		statfsMetrics.excludeFilesystemTypes = make(map[string]bool)
		for _, fsType := range statfsMetricsCfg.ExcludeFilesystemTypes {
			statfsMetrics.excludeFilesystemTypes[fsType] = true
		}
	}

	statfsMetricsLog.Infof("id=%s", statfsMetrics.id)
	statfsMetricsLog.Infof("interval=%s", statfsMetrics.interval)
	statfsMetricsLog.Infof("full_metrics_factor=%d", statfsMetrics.fullMetricsFactor)
	statfsMetricsLog.Infof("mountinfo_pid=%d", statfsMetrics.mountinfoPid)
	if statfsMetrics.includeFilesystemTypes != nil {
		fsTypeList := make([]string, len(statfsMetrics.includeFilesystemTypes))
		i := 0
		for fsType := range statfsMetrics.includeFilesystemTypes {
			fsTypeList[i] = fsType
			i++
		}
		sort.Strings(fsTypeList)
		statfsMetricsLog.Infof("include_file_system_types=%q", fsTypeList)
	} else {
		statfsMetricsLog.Info("include_file_system_types empty, all file system types will be included unless explicitly excluded")
	}
	if statfsMetrics.excludeFilesystemTypes != nil {
		fsTypeList := make([]string, len(statfsMetrics.excludeFilesystemTypes))
		i := 0
		for fsType := range statfsMetrics.excludeFilesystemTypes {
			fsTypeList[i] = fsType
			i++
		}
		sort.Strings(fsTypeList)
		statfsMetricsLog.Infof("exclude_file_system_types=%q", fsTypeList)
	} else {
		statfsMetricsLog.Info("exclude_file_system_types empty, no exclusions")
	}

	return statfsMetrics, nil
}

func (sfsm *StatfsMetrics) keepFsType(fsType string) bool {
	return ((sfsm.excludeFilesystemTypes == nil || !sfsm.excludeFilesystemTypes[fsType]) &&
		(sfsm.includeFilesystemTypes == nil || sfsm.includeFilesystemTypes[fsType]))
}

func (sfsm *StatfsMetrics) updateStatfsInfo() {
	instance, hostname := GlobalInstance, GlobalHostname
	if sfsm.instance != "" {
		instance = sfsm.instance
	}
	if sfsm.hostname != "" {
		hostname = sfsm.hostname
	}

	sfsm.outOfScopePresentMetrics = nil

	mountSources := make(map[string]bool)
	for _, parsedLine := range sfsm.procMountinfo.ParsedLines {
		fsType := string(parsedLine[procfs.MOUNTINFO_FS_TYPE])
		if !sfsm.keepFsType(fsType) {
			continue
		}

		mountSource := string(parsedLine[procfs.MOUNTINFO_MOUNT_SOURCE])
		if mountSources[mountSource] {
			continue // already encountered
		}
		mountSources[mountSource] = true

		mountPoint := string(parsedLine[procfs.MOUNTINFO_MOUNT_POINT])
		statfsInfo := sfsm.statfsInfo[mountSource]
		if statfsInfo == nil {
			statfsInfo = &StatfsInfo{
				cycleNum:    initialCycleNum.Get(sfsm.fullMetricsFactor),
				mountPoint:  mountPoint,
				mountSource: mountSource,
				fsType:      fsType,
			}
			sfsm.statfsInfo[mountSource] = statfsInfo
		} else if statfsInfo.disabled {
			continue
		} else if statfsInfo.mountPoint != mountPoint ||
			statfsInfo.mountSource != mountSource ||
			statfsInfo.fsType != fsType {
			sfsm.outOfScopePresentMetrics = append(sfsm.outOfScopePresentMetrics, statfsInfo.enabledMetric)
			statfsInfo.mountPoint = mountPoint
			statfsInfo.mountSource = mountSource
			statfsInfo.fsType = fsType
		}
		labels := fmt.Sprintf(
			`%s="%s",%s="%s",%s="%s",%s="%s",%s="%s"`,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
			STATFS_MOUNT_POINT_LABEL_NAME, statfsInfo.mountPoint,
			STATFS_MOUNT_FS_LABEL_NAME, statfsInfo.mountSource,
			STATFS_MOUNT_FS_TYPE_LABEL_NAME, statfsInfo.fsType,
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
		statfsInfo.freeSizeMetric = []byte(fmt.Sprintf(
			`%s{%s} `, // N.B. space before value included
			STATFS_FREE_SIZE_METRIC, labels,
		))
		statfsInfo.availSizeMetric = []byte(fmt.Sprintf(
			`%s{%s} `, // N.B. space before value included
			STATFS_AVAIL_SIZE_METRIC, labels,
		))
		statfsInfo.freePctMetric = []byte(fmt.Sprintf(
			`%s{%s} `, // N.B. space before value included
			STATFS_FREE_PCT_METRIC, labels,
		))
		statfsInfo.availPctMetric = []byte(fmt.Sprintf(
			`%s{%s} `, // N.B. space before value included
			STATFS_AVAIL_PCT_METRIC, labels,
		))
		statfsInfo.enabledMetric = []byte(fmt.Sprintf(
			`%s{%s} `, // N.B. space before value included
			STATFS_PRESENCE_METRIC, labels,
		))
	}

	// Handle out of scope mount sources:
	for mountSource, statfsInfo := range sfsm.statfsInfo {
		if !mountSources[mountSource] {
			if !statfsInfo.wasDisabled {
				sfsm.outOfScopePresentMetrics = append(sfsm.outOfScopePresentMetrics, statfsInfo.enabledMetric)
			}
			delete(sfsm.statfsInfo, mountSource)
		}
	}
}

func (sfsm *StatfsMetrics) updateIntervalMetricsCache() {
	instance, hostname := GlobalInstance, GlobalHostname
	if sfsm.instance != "" {
		instance = sfsm.instance
	}
	if sfsm.hostname != "" {
		hostname = sfsm.hostname
	}

	sfsm.intervalMetric = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s"} `, // N.B. include space before val
		STATFS_INTERVAL_METRIC,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
	))
}

func (sfsm *StatfsMetrics) generateMetrics(buf *bytes.Buffer) (int, int) {
	const KBYTE = 1024

	currIndex := sfsm.currIndex
	prevIndex := 1 - currIndex

	currTs := sfsm.statfsTs[currIndex]
	sfsm.tsSuffixBuf.Reset()
	fmt.Fprintf(
		sfsm.tsSuffixBuf, " %d\n", currTs.UnixMilli(),
	)
	promTs := sfsm.tsSuffixBuf.Bytes()

	actualMetricsCount := 0
	totalMetricsCount := len(sfsm.statfsInfo)*STATFS_DEVICE_NUM_METRICS + 1
	for _, statfsInfo := range sfsm.statfsInfo {
		currStatfsBuf, prevStatfsBuf := statfsInfo.statfsBuf[currIndex], statfsInfo.statfsBuf[prevIndex]
		allMetrics := prevStatfsBuf == nil || statfsInfo.cycleNum == 0
		if !statfsInfo.disabled {
			bsize := uint64(currStatfsBuf.Bsize)
			// If bsize changes then force a full cycle:
			if !allMetrics && bsize != uint64(prevStatfsBuf.Bsize) {
				allMetrics = true
			}
			if allMetrics {
				buf.Write(statfsInfo.bsizeMetric)
				buf.WriteString(strconv.FormatUint(bsize, 10))
				buf.Write(promTs)
				actualMetricsCount += 1
			}

			updateFreePct, updateAvailPct := false, false
			if allMetrics || currStatfsBuf.Blocks != prevStatfsBuf.Blocks {
				buf.Write(statfsInfo.blocksMetric)
				buf.WriteString(strconv.FormatUint(currStatfsBuf.Blocks, 10))
				buf.Write(promTs)

				buf.Write(statfsInfo.totalSizeMetric)
				buf.WriteString(strconv.FormatUint(currStatfsBuf.Blocks*bsize/KBYTE, 10))
				buf.Write(promTs)

				updateAvailPct = true
				updateFreePct = true
				actualMetricsCount += 2
			}

			if allMetrics || currStatfsBuf.Bfree != prevStatfsBuf.Bfree {
				buf.Write(statfsInfo.bfreeMetric)
				buf.WriteString(strconv.FormatUint(currStatfsBuf.Bfree, 10))
				buf.Write(promTs)

				buf.Write(statfsInfo.freeSizeMetric)
				buf.WriteString(strconv.FormatUint(currStatfsBuf.Bfree*bsize/KBYTE, 10))
				buf.Write(promTs)

				updateFreePct = true
				actualMetricsCount += 2
			}

			if allMetrics || currStatfsBuf.Bavail != prevStatfsBuf.Bavail {
				buf.Write(statfsInfo.bavailMetric)
				buf.WriteString(strconv.FormatUint(currStatfsBuf.Bavail, 10))
				buf.Write(promTs)

				buf.Write(statfsInfo.availSizeMetric)
				buf.WriteString(strconv.FormatUint(currStatfsBuf.Bavail*bsize/KBYTE, 10))
				buf.Write(promTs)

				updateAvailPct = true
				actualMetricsCount += 2
			}

			if updateFreePct {
				buf.Write(statfsInfo.freePctMetric)
				buf.WriteString(strconv.FormatFloat(
					float64(currStatfsBuf.Bfree)/float64(currStatfsBuf.Blocks)*100,
					'f', STATFS_FREE_PCT_METRIC_PREC, 64,
				))
				buf.Write(promTs)
				actualMetricsCount += 1
			}

			if updateAvailPct {
				buf.Write(statfsInfo.availPctMetric)
				buf.WriteString(strconv.FormatFloat(
					float64(currStatfsBuf.Bavail)/float64(currStatfsBuf.Blocks)*100,
					'f', STATFS_AVAIL_PCT_METRIC_PREC, 64,
				))
				buf.Write(promTs)
				actualMetricsCount += 1
			}

			if allMetrics || currStatfsBuf.Files != prevStatfsBuf.Files {
				buf.Write(statfsInfo.filesMetric)
				buf.WriteString(strconv.FormatUint(currStatfsBuf.Files, 10))
				buf.Write(promTs)
				actualMetricsCount += 1
			}

			if allMetrics || currStatfsBuf.Ffree != prevStatfsBuf.Ffree {
				buf.Write(statfsInfo.ffreeMetric)
				buf.WriteString(strconv.FormatUint(currStatfsBuf.Ffree, 10))
				buf.Write(promTs)
				actualMetricsCount += 1
			}

			if allMetrics {
				buf.Write(statfsInfo.enabledMetric)
				buf.WriteByte('1')
				buf.Write(promTs)
				actualMetricsCount += 1
			}
		} else if allMetrics || !statfsInfo.wasDisabled {
			buf.Write(statfsInfo.enabledMetric)
			buf.WriteByte('0')
			buf.Write(promTs)
			actualMetricsCount += 1
			statfsInfo.wasDisabled = true
		}

		if statfsInfo.cycleNum += 1; statfsInfo.cycleNum >= sfsm.fullMetricsFactor {
			statfsInfo.cycleNum = 0
		}
	}

	if sfsm.outOfScopePresentMetrics != nil {
		for _, metric := range sfsm.outOfScopePresentMetrics {
			buf.Write(metric)
			buf.WriteByte('0')
			buf.Write(promTs)
			actualMetricsCount += 1
		}
		sfsm.outOfScopePresentMetrics = nil
	}

	if sfsm.validPrevTs {
		deltaSec := currTs.Sub(sfsm.statfsTs[prevIndex]).Seconds()
		if sfsm.intervalMetric == nil {
			sfsm.updateIntervalMetricsCache()
		}
		buf.Write(sfsm.intervalMetric)
		buf.WriteString(strconv.FormatFloat(deltaSec, 'f', 6, 64))
		buf.Write(promTs)
		actualMetricsCount++
	}

	sfsm.currIndex = 1 - sfsm.currIndex

	return actualMetricsCount, totalMetricsCount
}

// Satisfy the TaskActivity interface:
func (sfsm *StatfsMetrics) Execute() bool {
	timeNowFn := time.Now
	if sfsm.timeNowFn != nil {
		timeNowFn = sfsm.timeNowFn
	}

	metricsQueue := GlobalMetricsQueue
	if sfsm.metricsQueue != nil {
		metricsQueue = sfsm.metricsQueue
	}

	firstTime := sfsm.procMountinfo == nil
	if firstTime {
		procfsRoot := GlobalProcfsRoot
		if sfsm.procfsRoot != "" {
			procfsRoot = sfsm.procfsRoot
		}
		sfsm.procMountinfo = procfs.NewMountinfo(procfsRoot, sfsm.mountinfoPid)
	}

	if firstTime || sfsm.mountinfoCycleNum == 0 {
		err := sfsm.procMountinfo.Parse()
		if err != nil {
			statfsMetricsLog.Warnf("%v: statfs (disk free) metrics will be disabled", err)
			return false
		}
	}
	if sfsm.mountinfoCycleNum += 1; sfsm.mountinfoCycleNum >= sfsm.fullMetricsFactor {
		sfsm.mountinfoCycleNum = 0
	}

	if firstTime || sfsm.procMountinfo.Changed {
		sfsm.updateStatfsInfo()
	}

	currIndex := sfsm.currIndex

	for _, statfsInfo := range sfsm.statfsInfo {
		if statfsInfo.disabled {
			continue
		}
		statfsBuf := statfsInfo.statfsBuf[currIndex]
		if statfsBuf == nil {
			statfsBuf = &unix.Statfs_t{}
			statfsInfo.statfsBuf[currIndex] = statfsBuf
		}
		err := unix.Statfs(statfsInfo.mountPoint, statfsBuf)
		if err != nil {
			statfsMetricsLog.Warnf(
				"statfs(%q): %v, statfs for %q, %q will be disabled",
				statfsInfo.mountPoint, err,
				statfsInfo.mountSource, statfsInfo.fsType,
			)
			statfsInfo.disabled = true
		}
	}

	sfsm.statfsTs[currIndex] = timeNowFn()
	sfsm.validPrevTs = !firstTime

	buf := metricsQueue.GetBuf()
	actualMetricsCount, totalMetricsCount := sfsm.generateMetrics(buf)
	byteCount := buf.Len()
	metricsQueue.QueueBuf(buf)

	GlobalMetricsGeneratorStatsContainer.Update(
		sfsm.id, uint64(actualMetricsCount), uint64(totalMetricsCount), uint64(byteCount),
	)

	return true
}

// Define and register the task builder:
func StatfsMetricsTaskBuilder(cfg *LsvmiConfig) ([]*Task, error) {
	sfsm, err := NewStatfsMetrics(cfg)
	if err != nil {
		return nil, err
	}
	if sfsm.interval <= 0 {
		procInterruptsMetricsLog.Infof(
			"interval=%s, metrics disabled", sfsm.interval,
		)
		return nil, nil
	}
	tasks := []*Task{
		NewTask(sfsm.id, sfsm.interval, sfsm),
	}
	return tasks, nil
}

func init() {
	TaskBuilders.Register(StatfsMetricsTaskBuilder)
}
