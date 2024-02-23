// internal metrics

package lsvmi

import (
	"bytes"
	"fmt"
	"strconv"
	"time"
)

// Generate internal metrics:
const (
	INTERNAL_METRICS_CONFIG_INTERVAL_DEFAULT            = "5s"
	INTERNAL_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT = 12

	INTERNAL_METRICS_ID = "internal_metrics"
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

type InternalMetrics struct {
	// id/task_id:
	id string
	// How often to generate the metrics:
	interval time.Duration
	// Every Nth cycle full all metrics are generated, regardless of having
	// changed since the previous cycle of not:
	fullMetricsFactor int
	// Current cycle#:
	cycleNum int
	// Scheduler specific metrics:
	schedulerMetrics *SchedulerInternalMetrics

	// Common metrics generators stats:
	mgStats MetricsGeneratorStats
	// Cache the metrics:
	mgStatsMetricsCache map[string][][]byte
	// A buffer for the timestamp suffix:
	tsSuffixBuf *bytes.Buffer

	// The following are needed for testing, normally they are set to nil/empty,
	// their default values, that is:
	instance, hostname string
	timeNowFn          func() time.Time
	metricsQueue       MetricsQueue
	scheduler          *Scheduler
	mgsStatsContainer  *MetricsGeneratorStatsContainer
}

func NewInternalMetrics(cfg any) (*InternalMetrics, error) {
	var (
		err                error
		internalMetricsCfg *InternalMetricsConfig
	)

	switch cfg := cfg.(type) {
	case *LsvmiConfig:
		internalMetricsCfg = cfg.InternalMetricsConfig
	case *InternalMetricsConfig:
		internalMetricsCfg = cfg
	case nil:
		internalMetricsCfg = DefaultInternalMetricsConfig()
	default:
		return nil, fmt.Errorf("NewInternalMetrics: %T invalid config type", cfg)
	}

	interval, err := time.ParseDuration(internalMetricsCfg.Interval)
	if err != nil {
		return nil, err
	}
	internalMetrics := &InternalMetrics{
		id:                  INTERNAL_METRICS_ID,
		interval:            interval,
		fullMetricsFactor:   internalMetricsCfg.FullMetricsFactor,
		mgStats:             make(MetricsGeneratorStats),
		mgStatsMetricsCache: make(map[string][][]byte),
		tsSuffixBuf:         &bytes.Buffer{},
	}
	internalMetrics.schedulerMetrics = NewSchedulerInternalMetrics(internalMetrics)
	internalMetrics.updateMGStatsMetricsCache(internalMetrics.id)
	return internalMetrics, nil
}

// Satisfy the TaskActivity interface:
func (internalMetrics *InternalMetrics) Execute() {
	timeNowFn := time.Now
	if internalMetrics.timeNowFn != nil {
		timeNowFn = internalMetrics.timeNowFn
	}

	metricsQueue := GlobalMetricsQueue
	if internalMetrics.metricsQueue != nil {
		metricsQueue = internalMetrics.metricsQueue
	}

	fullCycle := internalMetrics.cycleNum == internalMetrics.fullMetricsFactor-1

	metricsCount := 0
	buf := metricsQueue.GetBuf()

	// Scheduler metrics:
	metricsCount += internalMetrics.schedulerMetrics.GenerateMetrics(buf, fullCycle)

	// Common metrics generators metrics:
	mgsStatsContainer := GlobalMetricsGeneratorStatsContainer
	if internalMetrics.mgsStatsContainer != nil {
		mgsStatsContainer = internalMetrics.mgsStatsContainer
	}
	mgsStatsContainer.SnapStats(internalMetrics.mgStats, STATS_SNAP_AND_CLEAR)
	ts := timeNowFn()
	internalMetrics.tsSuffixBuf.Reset()
	fmt.Fprintf(internalMetrics.tsSuffixBuf, " %d\n", ts.UnixMilli())
	tsSuffix := internalMetrics.tsSuffixBuf.Bytes()
	for id, mgStats := range internalMetrics.mgStats {
		metrics := internalMetrics.mgStatsMetricsCache[id]
		if metrics == nil {
			metrics = internalMetrics.updateMGStatsMetricsCache(id)
		}
		for indx, val := range mgStats {
			buf.Write(metrics[indx])
			buf.WriteString(strconv.FormatUint(val, 10))
			buf.Write(tsSuffix)
			metricsCount++
		}
	}

	// Update by hand own generator metrics; this is required since the number
	// of metrics and bytes were unknown up to this point. This has to be the
	// last step before queueing the buffer.
	{
		metrics := internalMetrics.mgStatsMetricsCache[internalMetrics.id]

		buf.Write(metrics[METRICS_GENERATOR_INVOCATION_COUNT])
		buf.WriteByte('1')
		buf.Write(tsSuffix)

		buf.Write(metrics[METRICS_GENERATOR_METRICS_COUNT])
		buf.WriteString(strconv.Itoa(metricsCount + METRICS_GENERATOR_NUM_STATS))
		buf.Write(tsSuffix)

		buf.Write(metrics[METRICS_GENERATOR_BYTES_COUNT])
		// Assuming that buf size is < 100k, estimate 5 digits for bytes count value:
		buf.WriteString(strconv.Itoa(buf.Len() + 5 + len(tsSuffix)))
		buf.Write(tsSuffix)
	}
	metricsQueue.QueueBuf(buf)

	// Update the cycle number:
	if internalMetrics.cycleNum++; internalMetrics.cycleNum >= internalMetrics.fullMetricsFactor {
		internalMetrics.cycleNum = 0
	}
}

func (internalMetrics *InternalMetrics) updateMGStatsMetricsCache(id string) [][]byte {
	instance, hostname := GlobalInstance, GlobalHostname
	if internalMetrics.instance != "" {
		instance = internalMetrics.instance
	}
	if internalMetrics.hostname != "" {
		hostname = internalMetrics.hostname
	}
	mgStatsMetrics := make([][]byte, METRICS_GENERATOR_NUM_STATS)
	internalMetrics.mgStatsMetricsCache[id] = mgStatsMetrics
	for index, name := range MetricsGeneratorStatsMetricsNameMap {
		mgStatsMetrics[index] = []byte(fmt.Sprintf(
			`%s{%s="%s",%s="%s",%s="%s"} `, // N.B. whitespace before value
			name,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
			METRICS_GENERATOR_ID_LABEL_NAME, id,
		))
	}
	return mgStatsMetrics
}

// Define and register the task builder:
func InternalMetricsTaskBuilder(cfg *LsvmiConfig) ([]*Task, error) {
	internalMetrics, err := NewInternalMetrics(cfg)
	if err != nil {
		return nil, err
	}
	tasks := []*Task{
		NewTask(internalMetrics.id, internalMetrics.interval, internalMetrics),
	}
	return tasks, nil
}

func init() {
	TaskBuilders.Register(InternalMetricsTaskBuilder)
}
