// Metrics bases on /proc/PID/... files

package lsvmi

const (
	PROC_PID_METRICS_CONFIG_INTERVAL_DEFAULT            = "1s"
	PROC_PID_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT = 15
	PROC_PID_METRICS_CONFIG_PROCESS_METRICS_DEFAULT     = true
	PROC_PID_METRICS_CONFIG_THREAD_METRICS_DEFAULT      = true
	PROC_PID_METRICS_CONFIG_NUM_PART_DEFAULT            = -1
	PROC_PID_METRICS_CONFIG_ACTIVE_ONLY_DELTA_DEFAULT   = true

	// This generator id:
	PROC_PID_METRICS_ID = "proc_pid_metrics"
)

var procPidMetricsLog = NewCompLogger(PROC_PID_METRICS_ID)

type ProcPidMetricsConfig struct {
	// How often to generate the metrics in time.ParseDuration() format:
	Interval string `yaml:"interval"`
	// Normally metrics are generated only if there is a change in value from
	// the previous scan. However every N cycles the full set is generated. Use
	// 0 to generate full metrics every cycle.
	FullMetricsFactor int `yaml:"full_metrics_factor"`
	// Whether to scan processes (/proc/PID):
	ProcessMetrics bool `yaml:"process_metrics"`
	// Whether to scan threads (/proc/PID/task/TID):
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
		ProcessMetrics:    PROC_PID_METRICS_CONFIG_PROCESS_METRICS_DEFAULT,
		ThreadMetrics:     PROC_PID_METRICS_CONFIG_THREAD_METRICS_DEFAULT,
		NumPartitions:     PROC_PID_METRICS_CONFIG_NUM_PART_DEFAULT,
		ActiveOnlyDelta:   PROC_PID_METRICS_CONFIG_ACTIVE_ONLY_DELTA_DEFAULT,
	}
}
