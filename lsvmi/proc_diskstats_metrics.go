// /proc/diskstats and /proc/mountinfo metrics

package lsvmi

const (
	PROC_DISKSTATS_METRICS_CONFIG_INTERVAL_DEFAULT            = "5s"
	PROC_DISKSTATS_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT = 12
	PROC_DISKSTATS_METRICS_CONFIG_INCLUDE_MOUNTINFO_DEFAULT   = false

	// This generator id:
	PROC_DISKSTATS_METRICS_ID = "proc_diskstats_metrics"
)

type ProcDiskstatsMetricsConfig struct {
	// How often to generate the metrics in time.ParseDuration() format:
	Interval string `yaml:"interval"`
	// Normally metrics are generated only if there is a change in value from
	// the previous scan. However every N cycles the full set is generated. Use
	// 0 to generate full metrics every cycle.
	FullMetricsFactor int `yaml:"full_metrics_factor"`
	// Whether to include detailed mountinfo or not; the mount point is a label
	// of diskstats regardless:
	IncludeMountinfo bool `yaml:"include_mountinfo"`
}

func DefaultProcDiskstatsMetricsConfig() *ProcDiskstatsMetricsConfig {
	return &ProcDiskstatsMetricsConfig{
		Interval:          PROC_DISKSTATS_METRICS_CONFIG_INTERVAL_DEFAULT,
		FullMetricsFactor: PROC_DISKSTATS_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT,
		IncludeMountinfo:  PROC_DISKSTATS_METRICS_CONFIG_INCLUDE_MOUNTINFO_DEFAULT,
	}
}
