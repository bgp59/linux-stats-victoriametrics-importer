// /proc/diskstats and /proc/PID/mountinfo metrics

package lsvmi

import (
	"bytes"
	"fmt"
	"time"

	"github.com/eparparita/linux-stats-victoriametrics-importer/procfs"
)

const (
	PROC_DISKSTATS_METRICS_CONFIG_INTERVAL_DEFAULT            = "5s"
	PROC_DISKSTATS_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT = 12
	PROC_DISKSTATS_METRICS_CONFIG_MOUNTINFO_PID_DEFAULT       = 1

	// This generator id:
	PROC_DISKSTATS_METRICS_ID = "proc_diskstats_metrics"
)

const (
	// diskstats:
	PROC_DISKSTATS_NUM_READS_COMPLETED_DELTA_METRIC    = "proc_diskstats_num_reads_completed_delta"
	PROC_DISKSTATS_NUM_READS_MERGED_DELTA_METRIC       = "proc_diskstats_num_reads_merged_delta"
	PROC_DISKSTATS_NUM_READ_SECTORS_DELTA_METRIC       = "proc_diskstats_num_read_sectors_delta"
	PROC_DISKSTATS_READ_PCT_METRIC                     = "proc_diskstats_read_pct"
	PROC_DISKSTATS_NUM_WRITES_COMPLETED_DELTA_METRIC   = "proc_diskstats_num_writes_completed_delta"
	PROC_DISKSTATS_NUM_WRITES_MERGED_DELTA_METRIC      = "proc_diskstats_num_writes_merged_delta"
	PROC_DISKSTATS_NUM_WRITE_SECTORS_DELTA_METRIC      = "proc_diskstats_num_write_sectors_delta"
	PROC_DISKSTATS_WRITE_PCT_METRIC                    = "proc_diskstats_write_pct"
	PROC_DISKSTATS_NUM_IO_IN_PROGRESS_DELTA_METRIC     = "proc_diskstats_num_io_in_progress_delta"
	PROC_DISKSTATS_IO_PCT_METRIC                       = "proc_diskstats_io_pct"
	PROC_DISKSTATS_IO_WEIGTHED_PCT_METRIC              = "proc_diskstats_io_weigthed_pct"
	PROC_DISKSTATS_NUM_DISCARDS_COMPLETED_DELTA_METRIC = "proc_diskstats_num_discards_completed_delta"
	PROC_DISKSTATS_NUM_DISCARDS_MERGED_DELTA_METRIC    = "proc_diskstats_num_discards_merged_delta"
	PROC_DISKSTATS_NUM_DISCARD_SECTORS_DELTA_METRIC    = "proc_diskstats_num_discard_sectors_delta"
	PROC_DISKSTATS_DISCARD_PCT_METRIC                  = "proc_diskstats_discard_pct"
	PROC_DISKSTATS_NUM_FLUSH_REQUESTS_DELTA_METRIC     = "proc_diskstats_num_flush_requests_delta"
	PROC_DISKSTATS_FLUSH_PCT_METRIC                    = "proc_diskstats_flush_pct"

	PROC_DISKSTATS_MAJ_MIN_LABEL_NAME = "maj_min"
	PROC_DISKSTATS_NAME_LABEL_NAME    = "name"

	// mountinfo:
	PROC_MOUNTINFO_METRIC                  = "proc_mountinfo"
	PROC_MOUNTINFO_PID_LABEL_NAME          = "pid"
	PROC_MOUNTINFO_MAJ_MIN_LABEL_NAME      = "maj_min"
	PROC_MOUNTINFO_ROOT_LABEL_NAME         = "root"
	PROC_MOUNTINFO_MOUNT_POINT_LABEL_NAME  = "mount_point"
	PROC_MOUNTINFO_FS_TYPE_LABEL_NAME      = "fs_type"
	PROC_MOUNTINFO_MOUNT_SOURCE_LABEL_NAME = "source"
)

// Certain values are used to generate rates:
type ProcDiskstatsPct struct {
	factor float64 // dVal/dTime * factor
	prec   int     // FormatFloat prec arg
}

var ProcDiskstatsIndexPct = [procfs.DISKSTATS_VALUE_FIELDS_NUM]*ProcDiskstatsPct{
	procfs.DISKSTATS_READ_MILLISEC:        {100. / 1000., 2},
	procfs.DISKSTATS_WRITE_MILLISEC:       {100. / 1000., 2},
	procfs.DISKSTATS_IO_MILLISEC:          {100. / 1000., 2},
	procfs.DISKSTATS_IO_WEIGTHED_MILLISEC: {100. / 1000., 2},
	procfs.DISKSTATS_DISCARD_MILLISEC:     {100. / 1000., 2},
	procfs.DISKSTATS_FLUSH_MILLISEC:       {100. / 1000., 2},
}

var procDiskstatsMetricsLog = NewCompLogger(PROC_DISKSTATS_METRICS_ID)

// Diskstats index to metrics name map; indexes not in the map will be ignored:
var procDiskstatsIndexToMetricNameMap = map[int]string{
	procfs.DISKSTATS_NUM_READS_COMPLETED:    PROC_DISKSTATS_NUM_READS_COMPLETED_DELTA_METRIC,
	procfs.DISKSTATS_NUM_READS_MERGED:       PROC_DISKSTATS_NUM_READS_MERGED_DELTA_METRIC,
	procfs.DISKSTATS_NUM_READ_SECTORS:       PROC_DISKSTATS_NUM_READ_SECTORS_DELTA_METRIC,
	procfs.DISKSTATS_READ_MILLISEC:          PROC_DISKSTATS_READ_PCT_METRIC,
	procfs.DISKSTATS_NUM_WRITES_COMPLETED:   PROC_DISKSTATS_NUM_WRITES_COMPLETED_DELTA_METRIC,
	procfs.DISKSTATS_NUM_WRITES_MERGED:      PROC_DISKSTATS_NUM_WRITES_MERGED_DELTA_METRIC,
	procfs.DISKSTATS_NUM_WRITE_SECTORS:      PROC_DISKSTATS_NUM_WRITE_SECTORS_DELTA_METRIC,
	procfs.DISKSTATS_WRITE_MILLISEC:         PROC_DISKSTATS_WRITE_PCT_METRIC,
	procfs.DISKSTATS_NUM_IO_IN_PROGRESS:     PROC_DISKSTATS_NUM_IO_IN_PROGRESS_DELTA_METRIC,
	procfs.DISKSTATS_IO_MILLISEC:            PROC_DISKSTATS_IO_PCT_METRIC,
	procfs.DISKSTATS_IO_WEIGTHED_MILLISEC:   PROC_DISKSTATS_IO_WEIGTHED_PCT_METRIC,
	procfs.DISKSTATS_NUM_DISCARDS_COMPLETED: PROC_DISKSTATS_NUM_DISCARDS_COMPLETED_DELTA_METRIC,
	procfs.DISKSTATS_NUM_DISCARDS_MERGED:    PROC_DISKSTATS_NUM_DISCARDS_MERGED_DELTA_METRIC,
	procfs.DISKSTATS_NUM_DISCARD_SECTORS:    PROC_DISKSTATS_NUM_DISCARD_SECTORS_DELTA_METRIC,
	procfs.DISKSTATS_DISCARD_MILLISEC:       PROC_DISKSTATS_DISCARD_PCT_METRIC,
	procfs.DISKSTATS_NUM_FLUSH_REQUESTS:     PROC_DISKSTATS_NUM_FLUSH_REQUESTS_DELTA_METRIC,
	procfs.DISKSTATS_FLUSH_MILLISEC:         PROC_DISKSTATS_FLUSH_PCT_METRIC,
}

// List of Mountinfo indexes used for labels; to ensure predictable label order,
// they are grouped a list of pairs:
type MountinfoIndexLabelPair struct {
	index int
	label string
}

var procMountinfoIndexToMetricLabelList = []*MountinfoIndexLabelPair{
	{procfs.MOUNTINFO_MAJOR_MINOR, PROC_MOUNTINFO_MAJ_MIN_LABEL_NAME},
	{procfs.MOUNTINFO_ROOT, PROC_MOUNTINFO_ROOT_LABEL_NAME},
	{procfs.MOUNTINFO_MOUNT_POINT, PROC_MOUNTINFO_MOUNT_POINT_LABEL_NAME},
	{procfs.MOUNTINFO_FS_TYPE, PROC_MOUNTINFO_FS_TYPE_LABEL_NAME},
	{procfs.MOUNTINFO_MOUNT_SOURCE, PROC_MOUNTINFO_MOUNT_SOURCE_LABEL_NAME},
}

type ProcDiskstatsMetricsConfig struct {
	// How often to generate the metrics in time.ParseDuration() format:
	Interval string `yaml:"interval"`
	// Normally metrics are generated only if there is a change in value from
	// the previous scan. However every N cycles the full set is generated. Use
	// 0 to generate full metrics every cycle.
	FullMetricsFactor int `yaml:"full_metrics_factor"`
	// The PID to use for /proc/PID/mountinfo, use 0 for self:
	MountinfoPid int `yaml:"mountinfo_pid"`
}

func DefaultProcDiskstatsMetricsConfig() *ProcDiskstatsMetricsConfig {
	return &ProcDiskstatsMetricsConfig{
		Interval:          PROC_DISKSTATS_METRICS_CONFIG_INTERVAL_DEFAULT,
		FullMetricsFactor: PROC_DISKSTATS_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT,
		MountinfoPid:      PROC_DISKSTATS_METRICS_CONFIG_MOUNTINFO_PID_DEFAULT,
	}
}

type ProcDiskstatsMetrics struct {
	// id/task_id:
	id string
	// Scan interval:
	interval time.Duration
	// Diskstats:
	// Dual storage for parsed stats used as previous, current:
	procDiskstats [2]*procfs.Diskstats
	// Timestamp when the stats were collected:
	procDiskstatsTs [2]time.Time
	// Index for current stats, toggled after each use:
	currIndex int
	// Current cycle#, indexed by maj:min:
	diskstatsCycleNum map[string]int
	// Mountinfo:
	mountinfoPid      int
	procMountinfo     *procfs.Mountinfo
	mountinfoCycleNum int
	// Ensure predictable label
	// If mountinfo parser encounters an error, do not disable the entire
	// metrics generator, only the info part; keep track of such condition
	// separately:
	mountifoDisabled bool
	// Full metric factor:
	fullMetricsFactor int

	// Delta metrics are generated with skip-zero-after-zero rule, i.e. if the
	// current and previous deltas are both zero, then the current metric is
	// skipped, save for full cycles. Keep track of zero deltas, indexed by
	// maj:min and diskstats index:
	diskstatsZeroDeltaMap map[string][]bool

	// Metrics cache, maj:min and diskstats index:
	diskstatsMetricsCache map[string][][]byte

	// Mountinfo metrics cache, rebuilt every time mountinfo changes:
	mountinfoMetricsCache [][]byte

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

func NewProcDiskstatsMetrics(cfg any) (*ProcDiskstatsMetrics, error) {
	var (
		err                     error
		procDiskstatsMetricsCfg *ProcDiskstatsMetricsConfig
	)

	switch cfg := cfg.(type) {
	case *LsvmiConfig:
		procDiskstatsMetricsCfg = cfg.ProcDiskstatsMetricsConfig
	case *ProcDiskstatsMetricsConfig:
		procDiskstatsMetricsCfg = cfg
	case nil:
		procDiskstatsMetricsCfg = DefaultProcDiskstatsMetricsConfig()
	default:
		return nil, fmt.Errorf("NewProcDiskstatsMetrics: %T invalid config type", cfg)
	}

	interval, err := time.ParseDuration(procDiskstatsMetricsCfg.Interval)
	if err != nil {
		return nil, err
	}
	procDiskstatsMetrics := &ProcDiskstatsMetrics{
		id:                    PROC_DISKSTATS_METRICS_ID,
		interval:              interval,
		diskstatsCycleNum:     make(map[string]int),
		mountinfoPid:          procDiskstatsMetricsCfg.MountinfoPid,
		mountinfoCycleNum:     initialCycleNum.Get(procDiskstatsMetricsCfg.FullMetricsFactor),
		diskstatsZeroDeltaMap: make(map[string][]bool),
		diskstatsMetricsCache: make(map[string][][]byte),
		fullMetricsFactor:     procDiskstatsMetricsCfg.FullMetricsFactor,
		tsSuffixBuf:           &bytes.Buffer{},
	}

	procDiskstatsMetricsLog.Infof("id=%s", procDiskstatsMetrics.id)
	procDiskstatsMetricsLog.Infof("interval=%s", procDiskstatsMetrics.interval)
	procDiskstatsMetricsLog.Infof("full_metrics_factor=%d", procDiskstatsMetrics.fullMetricsFactor)
	procDiskstatsMetricsLog.Infof("mountinfoPid=%d", procDiskstatsMetrics.mountinfoPid)
	return procDiskstatsMetrics, nil
}

func (pdsm *ProcDiskstatsMetrics) updateDiskstatsMetricsCache(majMin, diskName string) {
	instance, hostname := GlobalInstance, GlobalHostname
	if pdsm.instance != "" {
		instance = pdsm.instance
	}
	if pdsm.hostname != "" {
		hostname = pdsm.hostname
	}

	metrics := make([][]byte, procfs.DISKSTATS_VALUE_FIELDS_NUM)
	for index, name := range procDiskstatsIndexToMetricNameMap {
		metrics[index] = []byte(fmt.Sprintf(
			`%s{%s="%s",%s="%s",%s="%s",%s="%s"} `, // N.B. space before value included
			name,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
			PROC_DISKSTATS_MAJ_MIN_LABEL_NAME, majMin,
			PROC_DISKSTATS_NAME_LABEL_NAME, diskName,
		))
	}
	pdsm.diskstatsMetricsCache[majMin] = metrics
}

// When updating mountinfo return an iterable object with the metrics that went
// out of scope; they should be pushed w/ the associated value set to 0.
func (pdsm *ProcDiskstatsMetrics) updateMountinfoMetricsCache() map[string]bool {
	instance, hostname := GlobalInstance, GlobalHostname
	if pdsm.instance != "" {
		instance = pdsm.instance
	}
	if pdsm.hostname != "" {
		hostname = pdsm.hostname
	}

	outOfScopeMetrics := make(map[string]bool)
	for _, metric := range pdsm.mountinfoMetricsCache {
		outOfScopeMetrics[string(metric)] = true
	}

	parsedLines := pdsm.procMountinfo.ParsedLines
	pdsm.mountinfoMetricsCache = make([][]byte, 0)
	buf := &bytes.Buffer{}
	fmt.Fprintf(
		buf,
		`%s{%s="%s",%s="%s",%s="%d"`,
		PROC_MOUNTINFO_METRIC,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
		PROC_MOUNTINFO_PID_LABEL_NAME, pdsm.mountinfoPid,
	)
	prefixLen := buf.Len()
	diskstats := pdsm.procDiskstats[pdsm.currIndex]
	for _, parsedLine := range parsedLines {
		// Keep only info that has a maj:min matching diskstats:
		if diskstats.DevInfoMap[string((*parsedLine)[procfs.MOUNTINFO_MAJOR_MINOR])] == nil {
			continue
		}
		buf.Truncate(prefixLen)
		for _, indexLabel := range procMountinfoIndexToMetricLabelList {
			fmt.Fprintf(
				buf,
				`,%s="%s"`,
				indexLabel.label, (*parsedLine)[indexLabel.index],
			)
		}
		buf.WriteString(`} `) // N.B. space before value included
		pdsm.mountinfoMetricsCache = append(pdsm.mountinfoMetricsCache, bytes.Clone(buf.Bytes()))
		delete(outOfScopeMetrics, buf.String())
	}
	return outOfScopeMetrics
}
