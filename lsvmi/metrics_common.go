// Definitions common to all metrics and generators.

package lsvmi

import (
	"bytes"
	"fmt"
	"os"
	"strings"
)

const (
	// Whether internal metrics stats are snapped and cleared or not:
	STATS_SNAP_ONLY      = false
	STATS_SNAP_AND_CLEAR = true
)

// The following labels are common to all metrics:
const (
	INSTANCE_LABEL_NAME = "inst"
	HOSTNAME_LABEL_NAME = "node"
)

// A metrics generator satisfies the TaskActivity interface to be able to
// register with the scheduler.

// The generated metrics are written into *bytes.Buffer's which then queued into
// the metrics queue for transmission.

// The general flow of the TaskActivity implementation:
//  repeat until no more metrics
//  - buf <- MetricsQueue.GetBuf()
//  - fill buf it with metrics until it reaches MetricsQueue.GetTargetSize() or
//    there are no more metrics
//  - MetricsQueue.QueueBuf(buf)

// All metrics generators have the following configuration params:

type CommonMetricsGeneratorConfig struct {
	// How often to generate the metrics in time.ParseDuration() format:
	Interval string `yaml:"interval"`
	// Normally metrics are generated only if there is a change in value from
	// the previous scan. However every N cycles the full set is generated. Use
	// 0 to generate full metrics every cycle.
	FullMetricsFactor int `yaml:"full_metrics_factor"`
}

type MetricsQueue interface {
	GetBuf() *bytes.Buffer
	QueueBuf(b *bytes.Buffer)
	GetTargetSize() int
}

var commonMetricsLog = NewCompLogger("common_metrics")

func buildMetricsCommonLabels(instance, hostname string) []byte {
	return []byte(
		fmt.Sprintf(
			`%s="%s",%s="%s"`,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
		),
	)
}

func InitCommonMetrics(cfg any) error {
	var (
		globalCfg *GlobalConfig
		hostname  string
		err       error
	)

	switch cfg := cfg.(type) {
	case *LsvmiConfig:
		globalCfg = cfg.GlobalConfig
	case *GlobalConfig:
		globalCfg = cfg
	case nil:
		globalCfg = DefaultGlobalConfig()
	default:
		return fmt.Errorf("cfg: %T invalid type", cfg)
	}

	if *hostnameArg != "" {
		hostname = *hostnameArg
	} else {
		hostname, err = os.Hostname()
		if err != nil {
			return err
		}
		if globalCfg.UseShortHostname {
			i := strings.Index(hostname, ".")
			if i >= 0 {
				hostname = hostname[:i]
			}
		}
		if hostname == "" {
			return fmt.Errorf("empty hostname")
		}
	}

	GlobalMetricsCommonLabels = buildMetricsCommonLabels(globalCfg.Instance, hostname)
	commonMetricsLog.Infof("common labels='%s'", GlobalMetricsCommonLabels)

	return nil
}
