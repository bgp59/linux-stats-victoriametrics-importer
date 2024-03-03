// Definitions common to all metrics and generators.

package lsvmi

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"sync"
)

const (
	// Whether internal metrics stats are snapped and cleared or not:
	STATS_SNAP_ONLY      = false
	STATS_SNAP_AND_CLEAR = true
)

// The following labels are common to all metrics:
const (
	INSTANCE_LABEL_NAME = "instance"
	HOSTNAME_LABEL_NAME = "hostname"
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

// Each metrics generator will maintain the following common stats:
const (
	// Indexes into the per generator []int stats:
	METRICS_GENERATOR_INVOCATION_COUNT = iota
	METRICS_GENERATOR_METRICS_COUNT
	METRICS_GENERATOR_BYTES_COUNT
	// Must be last:
	METRICS_GENERATOR_NUM_STATS
)

const (
	METRICS_GENERATOR_INVOCATION_COUNT_DELTA_METRIC = "lsvmi_metrics_gen_invocation_count_delta"
	METRICS_GENERATOR_METRICS_COUNT_DELTA_METRIC    = "lsvmi_metrics_gen_metrics_count_delta"
	METRICS_GENERATOR_BYTES_COUNT_DELTA_METRIC      = "lsvmi_metrics_gen_bytes_count_delta"

	METRICS_GENERATOR_ID_LABEL_NAME = "id"
)

type MetricsGeneratorStats map[string][]uint64

type MetricsGeneratorStatsContainer struct {
	// Stats proper:
	stats MetricsGeneratorStats
	// Lock:
	mu *sync.Mutex
}

var MetricsGeneratorStatsMetricsNameMap = map[int]string{
	METRICS_GENERATOR_INVOCATION_COUNT: METRICS_GENERATOR_INVOCATION_COUNT_DELTA_METRIC,
	METRICS_GENERATOR_METRICS_COUNT:    METRICS_GENERATOR_METRICS_COUNT_DELTA_METRIC,
	METRICS_GENERATOR_BYTES_COUNT:      METRICS_GENERATOR_BYTES_COUNT_DELTA_METRIC,
}

func NewMetricsGeneratorStatsContainer() *MetricsGeneratorStatsContainer {
	return &MetricsGeneratorStatsContainer{
		stats: make(MetricsGeneratorStats),
		mu:    &sync.Mutex{},
	}
}

func (mgs *MetricsGeneratorStatsContainer) Update(id string, metricsCount, byteCount uint64) {
	mgs.mu.Lock()
	defer mgs.mu.Unlock()

	gStats := mgs.stats[id]
	if gStats == nil {
		gStats = make([]uint64, METRICS_GENERATOR_NUM_STATS)
		mgs.stats[id] = gStats
	}
	gStats[METRICS_GENERATOR_INVOCATION_COUNT] += 1
	gStats[METRICS_GENERATOR_METRICS_COUNT] += metricsCount
	gStats[METRICS_GENERATOR_BYTES_COUNT] += byteCount
}

func (mgs *MetricsGeneratorStatsContainer) SnapStats(to MetricsGeneratorStats, clearStats bool) MetricsGeneratorStats {
	mgs.mu.Lock()
	defer mgs.mu.Unlock()
	if to == nil {
		to = make(MetricsGeneratorStats)
	}

	for taskId, gStats := range mgs.stats {
		toGStats := to[taskId]
		if toGStats == nil {
			toGStats = make([]uint64, METRICS_GENERATOR_NUM_STATS)
			to[taskId] = toGStats
		}
		copy(toGStats, gStats)
		if clearStats {
			clear(gStats)
		}
	}
	return to
}

// Initialize things common to all metrics; it should be invoke after the
// configuration was loaded and before task registration:
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

	GlobalInstance = globalCfg.Instance
	GlobalHostname = hostname
	GlobalProcfsRoot = globalCfg.ProcfsRoot
	GlobalMetricsGeneratorStatsContainer = NewMetricsGeneratorStatsContainer()

	return nil
}

// All metrics generators have to register with the scheduler as a task or
// tasks. Each generator will have a task builder function:
type TaskBuilderFunc func(config *LsvmiConfig) ([]*Task, error)

// The  metrics generators will register their specific builder into a list:
type TaskBuildersContainer struct {
	builders []TaskBuilderFunc
	mu       *sync.Mutex
}

func (tbc *TaskBuildersContainer) Register(tb TaskBuilderFunc) {
	tbc.mu.Lock()
	tbc.builders = append(tbc.builders, tb)
	tbc.mu.Unlock()
}

func (tbc *TaskBuildersContainer) List() []TaskBuilderFunc {
	return tbc.builders
}

var TaskBuilders = &TaskBuildersContainer{
	builders: make([]TaskBuilderFunc, 0),
	mu:       &sync.Mutex{},
}
