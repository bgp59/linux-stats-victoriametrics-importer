// internal metrics

package lsvmi

// Generate internal metrics:

const (
	INTERNAL_METRICS_CONFIG_INTERVAL_DEFAULT            = "5s"
	INTERNAL_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT = 12
)

type InternalMetricsConfig struct {
	// How often to generate the metrics in time.ParseDuration() format:
	Interval string `yaml:"interval"`
	// Normally metrics are generated only if there is a change in value from
	// the previous scan. However every N cycles the full set is generated. Use
	// 0 to fully generate with every cycle.
	FullMetricsFactor int `yaml:"full_metrics_factor"`
}

func DefaultInternalMetricsConfig() *InternalMetricsConfig {
	return &InternalMetricsConfig{
		Interval:          INTERNAL_METRICS_CONFIG_INTERVAL_DEFAULT,
		FullMetricsFactor: INTERNAL_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT,
	}
}
