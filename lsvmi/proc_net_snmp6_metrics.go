// /proc/net/snmp6 metrics

package lsvmi

const (
	PROC_NET_SNMP6_METRICS_CONFIG_INTERVAL_DEFAULT            = "1s"
	PROC_NET_SNMP6_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT = 15

	// This generator id:
	PROC_NET_SNMP6_METRICS_ID = "proc_net_snmp6_metrics"
)

type ProcNetSnmp6MetricsConfig struct {
	// How often to generate the metrics in time.ParseDuration() format:
	Interval string `yaml:"interval"`
	// Normally metrics are generated only if there is a change in value from
	// the previous scan. However every N cycles the full set is generated. Use
	// 0 to generate full metrics every cycle.
	FullMetricsFactor int `yaml:"full_metrics_factor"`
}

func DefaultProcNetSnmp6MetricsConfig() *ProcNetSnmp6MetricsConfig {
	return &ProcNetSnmp6MetricsConfig{
		Interval:          PROC_NET_SNMP6_METRICS_CONFIG_INTERVAL_DEFAULT,
		FullMetricsFactor: PROC_NET_SNMP6_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT,
	}
}
