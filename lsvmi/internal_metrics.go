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
	INTERNAL_METRICS_CONFIG_INTERVAL_DEFAULT = "5s"

	// Heartbeat metric:
	LSVMI_UPTIME_METRIC_NAME = "lsvmi_uptime_sec"
	LSVMI_VERSION_LABEL_NAME = "version"

	// Interval since last generation, i.e. the interval underlying the deltas.
	// Normally this should be close to scan interval, but this the actual
	// value, rather than the desired one:
	LSVMI_INTERNAL_METRICS_INTERVAL_METRIC_NAME = "lsvmi_internal_metrics_delta_sec"

	// This generator id:
	INTERNAL_METRICS_ID = "internal_metrics"
)

var internalMetricsLog = NewCompLogger(INTERNAL_METRICS_ID)

type InternalMetricsConfig struct {
	// How often to generate the metrics in time.ParseDuration() format:
	Interval string `yaml:"interval"`
}

func DefaultInternalMetricsConfig() *InternalMetricsConfig {
	return &InternalMetricsConfig{
		Interval: INTERNAL_METRICS_CONFIG_INTERVAL_DEFAULT,
	}
}

type InternalMetrics struct {
	// id/task_id:
	id string
	// How often to generate the metrics:
	interval time.Duration
	// The timestamp of the previous scan:
	prevTs time.Time

	// Cache heartbeat metric:
	uptimeMetric []byte

	// Cache interval metric:
	intervalMetric []byte

	// Scheduler specific metrics:
	schedulerMetrics *SchedulerInternalMetrics

	// Compressor pool specific metrics:
	compressorPoolMetrics *CompressorPoolInternalMetrics

	// HTTP Endpoint Pool specific metrics:
	httpEndpointPoolMetrics *HttpEndpointPoolInternalMetrics

	// Go specific metrics:
	goMetrics *GoInternalMetrics

	// OS metrics related to this process:
	osMetrics *OsInternalMetrics

	// Common metrics generators stats:
	metricsGenStats MetricsGeneratorStats
	// Cache the metrics:
	metricsGenStatsMetricsCache map[string][][]byte

	// When this metrics was created, used as the base for uptime:
	startTs time.Time

	// A buffer for the timestamp suffix:
	tsSuffixBuf *bytes.Buffer

	// The following are needed for testing only. Left to their default values,
	// the usual objects will be used.
	instance, hostname string
	timeNowFn          func() time.Time
	metricsQueue       MetricsQueue
	scheduler          *Scheduler
	compressorPool     *CompressorPool
	httpEndpointPool   *HttpEndpointPool
	mgsStatsContainer  *MetricsGeneratorStatsContainer
	procfsRoot         string
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
	now := time.Now()
	internalMetrics := &InternalMetrics{
		id:                          INTERNAL_METRICS_ID,
		interval:                    interval,
		prevTs:                      now,
		metricsGenStats:             make(MetricsGeneratorStats),
		metricsGenStatsMetricsCache: make(map[string][][]byte),
		startTs:                     time.Now(),
		tsSuffixBuf:                 &bytes.Buffer{},
	}
	internalMetrics.schedulerMetrics = NewSchedulerInternalMetrics(internalMetrics)
	internalMetrics.compressorPoolMetrics = NewCompressorPoolInternalMetrics(internalMetrics)
	internalMetrics.httpEndpointPoolMetrics = NewHttpEndpointPoolInternalMetrics(internalMetrics)
	internalMetrics.goMetrics = NewGoInternalMetrics(internalMetrics)
	internalMetrics.osMetrics = NewOsInternalMetrics(internalMetrics)
	internalMetricsLog.Infof("id=%s", internalMetrics.id)
	internalMetricsLog.Infof("interval=%s", internalMetrics.interval)
	return internalMetrics, nil
}

func (internalMetrics *InternalMetrics) getTsSuffix() []byte {
	timeNowFn := time.Now
	if internalMetrics.timeNowFn != nil {
		timeNowFn = internalMetrics.timeNowFn
	}
	return []byte(fmt.Sprintf(" %d\n", timeNowFn().UnixMilli()))
}

// Satisfy the TaskActivity interface:
func (internalMetrics *InternalMetrics) Execute() bool {
	timeNowFn := time.Now
	if internalMetrics.timeNowFn != nil {
		timeNowFn = internalMetrics.timeNowFn
	}

	metricsQueue := GlobalMetricsQueue
	if internalMetrics.metricsQueue != nil {
		metricsQueue = internalMetrics.metricsQueue
	}

	// Collect stats from various sources:
	// Scheduler:
	scheduler, schedulerMetrics := GlobalScheduler, internalMetrics.schedulerMetrics
	if internalMetrics.scheduler != nil {
		scheduler = internalMetrics.scheduler
	}
	schedulerMetrics.stats[schedulerMetrics.crtIndex] = scheduler.SnapStats(
		schedulerMetrics.stats[schedulerMetrics.crtIndex],
	)

	// Compressor pool:
	compressorPool, compressorPoolMetrics := GlobalCompressorPool, internalMetrics.compressorPoolMetrics
	if internalMetrics.compressorPool != nil {
		compressorPool = internalMetrics.compressorPool
	}
	if compressorPool != nil {
		compressorPoolMetrics.stats[compressorPoolMetrics.crtIndex] = compressorPool.SnapStats(
			compressorPoolMetrics.stats[compressorPoolMetrics.crtIndex],
		)
	} else {
		compressorPoolMetrics = nil
	}

	// HTTP Endpoint Pool metrics:
	httpEndpointPool, httpEndpointPoolMetrics := GlobalHttpEndpointPool, internalMetrics.httpEndpointPoolMetrics
	if internalMetrics.httpEndpointPool != nil {
		httpEndpointPool = internalMetrics.httpEndpointPool
	}
	if httpEndpointPool != nil {
		httpEndpointPoolMetrics.stats[httpEndpointPoolMetrics.crtIndex] = httpEndpointPool.SnapStats(
			httpEndpointPoolMetrics.stats[httpEndpointPoolMetrics.crtIndex],
		)
	} else {
		httpEndpointPoolMetrics = nil
	}

	// Go metrics:
	goMetrics := internalMetrics.goMetrics
	goMetrics.SnapStats()

	// OS metrics:
	osMetrics := internalMetrics.osMetrics
	if osMetrics != nil {
		osMetrics.SnapStats()
	}

	// Common metrics generators metrics:
	mgsStatsContainer := GlobalMetricsGeneratorStatsContainer
	if internalMetrics.mgsStatsContainer != nil {
		mgsStatsContainer = internalMetrics.mgsStatsContainer
	}
	mgsStatsContainer.SnapStats(internalMetrics.metricsGenStats, STATS_SNAP_AND_CLEAR)

	// Timestamp when all stats were collected:
	ts := timeNowFn()
	internalMetrics.tsSuffixBuf.Reset()
	fmt.Fprintf(internalMetrics.tsSuffixBuf, " %d\n", ts.UnixMilli())
	tsSuffix := internalMetrics.tsSuffixBuf.Bytes()

	// Generate metrics from the collected stats:
	metricsCount := 0
	buf := metricsQueue.GetBuf()

	uptimeMetric := internalMetrics.uptimeMetric
	if uptimeMetric == nil {
		uptimeMetric = internalMetrics.updateUptimeMetric()
	}
	buf.Write(uptimeMetric)
	buf.WriteString(strconv.FormatFloat(ts.Sub(internalMetrics.startTs).Seconds(), 'f', 6, 64))
	buf.Write(tsSuffix)
	metricsCount++

	intervalMetric := internalMetrics.intervalMetric
	if intervalMetric == nil {
		intervalMetric = internalMetrics.updateIntervalMetric()
	}
	buf.Write(intervalMetric)
	intervalDelta := ts.Sub(internalMetrics.prevTs).Seconds()
	buf.WriteString(strconv.FormatFloat(intervalDelta, 'f', 6, 64))
	buf.Write(tsSuffix)
	metricsCount++
	internalMetrics.prevTs = ts

	metricsCount += schedulerMetrics.generateMetrics(buf, tsSuffix)
	if compressorPoolMetrics != nil {
		metricsCount += compressorPoolMetrics.generateMetrics(buf, tsSuffix)
	}
	if httpEndpointPoolMetrics != nil {
		metricsCount += httpEndpointPoolMetrics.generateMetrics(buf, tsSuffix)
	}
	metricsCount += goMetrics.generateMetrics(buf, tsSuffix)
	if osMetrics != nil {
		osMetrics.generateMetrics(buf, tsSuffix)
	}

	for id, metricsGenStats := range internalMetrics.metricsGenStats {
		metrics := internalMetrics.metricsGenStatsMetricsCache[id]
		if metrics == nil {
			metrics = internalMetrics.updateMetricsCache(id)
		}
		for indx, val := range metricsGenStats {
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
		metrics := internalMetrics.metricsGenStatsMetricsCache[internalMetrics.id]
		if metrics == nil {
			metrics = internalMetrics.updateMetricsCache(internalMetrics.id)
		}

		buf.Write(metrics[METRICS_GENERATOR_INVOCATION_COUNT])
		buf.WriteByte('1')
		buf.Write(tsSuffix)

		metricsCount += METRICS_GENERATOR_NUM_STATS
		buf.Write(metrics[METRICS_GENERATOR_ACTUAL_METRICS_COUNT])
		buf.WriteString(strconv.Itoa(metricsCount))
		buf.Write(tsSuffix)

		buf.Write(metrics[METRICS_GENERATOR_TOTAL_METRICS_COUNT])
		buf.WriteString(strconv.Itoa(metricsCount)) // No delta for internal metrics
		buf.Write(tsSuffix)

		buf.Write(metrics[METRICS_GENERATOR_BYTES_COUNT])
		// Let l denote the number of bytes in buf without k bytes needed to
		// encode l+k. Then k is the smallest number such that:
		//  l + k < 10**k
		// This is equivalent to k being the largest number such that
		//  l + (k - 1) >= 10**(k-1)
		// which is equivalent w/ k being the value that stops the loop
		//  10*k <= l+k
		l := (buf.Len() + len(tsSuffix) + 1)
		pow10, n := 10, l+1
		for pow10 <= n {
			n++
			pow10 *= 10
		}
		buf.WriteString(strconv.Itoa(n))
		buf.Write(tsSuffix)
		buf.WriteByte('\n')
	}
	metricsQueue.QueueBuf(buf)

	return true
}

func (internalMetrics *InternalMetrics) updateUptimeMetric() []byte {
	instance, hostname := GlobalInstance, GlobalHostname
	if internalMetrics.instance != "" {
		instance = internalMetrics.instance
	}
	if internalMetrics.hostname != "" {
		hostname = internalMetrics.hostname
	}
	internalMetrics.uptimeMetric = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s",%s="%s"} `, // N.B. whitespace before value!
		LSVMI_UPTIME_METRIC_NAME,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
		LSVMI_VERSION_LABEL_NAME, LsvmiVersion,
	))
	return internalMetrics.uptimeMetric
}

func (internalMetrics *InternalMetrics) updateIntervalMetric() []byte {
	instance, hostname := GlobalInstance, GlobalHostname
	if internalMetrics.instance != "" {
		instance = internalMetrics.instance
	}
	if internalMetrics.hostname != "" {
		hostname = internalMetrics.hostname
	}
	internalMetrics.intervalMetric = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s"} `, // N.B. whitespace before value!
		LSVMI_INTERNAL_METRICS_INTERVAL_METRIC_NAME,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
	))
	return internalMetrics.intervalMetric
}

func (internalMetrics *InternalMetrics) updateMetricsCache(id string) [][]byte {
	instance, hostname := GlobalInstance, GlobalHostname
	if internalMetrics.instance != "" {
		instance = internalMetrics.instance
	}
	if internalMetrics.hostname != "" {
		hostname = internalMetrics.hostname
	}
	metricsGenStatsMetrics := make([][]byte, METRICS_GENERATOR_NUM_STATS)
	internalMetrics.metricsGenStatsMetricsCache[id] = metricsGenStatsMetrics
	for index, name := range MetricsGeneratorStatsMetricsNameMap {
		metricsGenStatsMetrics[index] = []byte(fmt.Sprintf(
			`%s{%s="%s",%s="%s",%s="%s"} `, // N.B. whitespace before value!
			name,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
			METRICS_GENERATOR_ID_LABEL_NAME, id,
		))
	}
	return metricsGenStatsMetrics
}

// Define and register the task builder:
func InternalMetricsTaskBuilder(cfg *LsvmiConfig) ([]*Task, error) {
	internalMetrics, err := NewInternalMetrics(cfg)
	if err != nil {
		return nil, err
	}
	if internalMetrics.interval <= 0 {
		internalMetricsLog.Infof(
			"interval=%s, metrics disabled", internalMetrics.interval,
		)
		return nil, nil
	}
	tasks := []*Task{
		NewTask(internalMetrics.id, internalMetrics.interval, internalMetrics),
	}
	return tasks, nil
}

func init() {
	TaskBuilders.Register(InternalMetricsTaskBuilder)
}
