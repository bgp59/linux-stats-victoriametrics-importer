// qdisc stats metrics

package lsvmi

import (
	"bytes"
	"fmt"
	"time"

	"github.com/eparparita/linux-stats-victoriametrics-importer/qdisc"
)

const (
	QDISC_METRICS_CONFIG_INTERVAL_DEFAULT            = "1s"
	QDISC_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT = 15

	// This generator id:
	QDISC_METRICS_ID = "qdisc_metrics"
)

const (
	// Stats based metrics:
	QDISC_RATE_METRICS             = "qdisc_rate_kbps"
	QDISC_PACKETS_DELTA_METRIC     = "qdisc_packets_delta"
	QDISC_DROPS_DELTA_METRIC       = "qdisc_drops_delta"
	QDISC_REQUEUES_DELTA_METRIC    = "qdisc_requeues_delta"
	QDISC_OVERLIMITS_DELTA_METRIC  = "qdisc_overlimits_delta"
	QDISC_QLEN_METRIC              = "qdisc_qlen"
	QDISC_BACKLOG_METRIC           = "qdisc_backlog"
	QDISC_GCLOWS_DELTA_METRIC      = "qdisc_gclows_delta"
	QDISC_THROTTLED_DELTA_METRIC   = "qdisc_throttled_delta"
	QDISC_FLOWSPLIMIT_DELTA_METRIC = "qdisc_flowsplimit_delta"

	// Qdisc presence metric:
	QDISC_PRESENCE_METRIC = "qdisc_present"

	QDISC_KIND_LABEL_NAME   = "kind"
	QDISC_HANDLE_LABEL_NAME = "handle"
	QDISC_PARENT_LABEL_NAME = "parent"
	QDISC_IF_LABEL_NAME     = "if" // interface

	// Interval since last generation, i.e. the interval underlying the deltas.
	// Normally this should be close to scan interval, but this is the actual
	// value, rather than the desired one:
	QDISC_INTERVAL_METRIC_NAME = "qdisc_metrics_delta_sec"
)

// Map uint32/64 indexes into metrics, if they are not present below then the
// metric will bw skipped:
var qdiscUint32IndexToDeltaMetricNameMap = map[int]string{
	qdisc.QDISC_PACKETS:    QDISC_PACKETS_DELTA_METRIC,
	qdisc.QDISC_DROPS:      QDISC_DROPS_DELTA_METRIC,
	qdisc.QDISC_REQUEUES:   QDISC_REQUEUES_DELTA_METRIC,
	qdisc.QDISC_OVERLIMITS: QDISC_OVERLIMITS_DELTA_METRIC,
}

var qdiscUint32IndexToMetricNameMap = map[int]string{
	qdisc.QDISC_QLEN:    QDISC_QLEN_METRIC,
	qdisc.QDISC_BACKLOG: QDISC_BACKLOG_METRIC,
}

var qdiscUint64IndexToDeltaMetricNameMap = map[int]string{
	qdisc.QDISC_BYTES:       QDISC_RATE_METRICS,
	qdisc.QDISC_GCFLOWS:     QDISC_GCLOWS_DELTA_METRIC,
	qdisc.QDISC_THROTTLED:   QDISC_THROTTLED_DELTA_METRIC,
	qdisc.QDISC_FLOWSPLIMIT: QDISC_FLOWSPLIMIT_DELTA_METRIC,
}

// Certain values are used to generate rates:
type QdiscRate struct {
	factor float64 // dVal/dTime * factor
	prec   int     // FormatFloat prec arg
}

var qdiscUint32IndexRate = [qdisc.QDISK_UINT32_NUM_STATS]*QdiscRate{}

var qdiscUint64IndexRate = [qdisc.QDISK_UINT64_NUM_STATS]*QdiscRate{
	qdisc.QDISC_BYTES: {8. / 1000., 1},
}

var qdiscMetricsLog = NewCompLogger(QDISC_METRICS_ID)

type QdiscMetricsConfig struct {
	// How often to generate the metrics in time.ParseDuration() format:
	Interval string `yaml:"interval"`
	// Normally metrics are generated only if there is a change in value from
	// the previous scan. However every N cycles the full set is generated. Use
	// 0 to generate full metrics every cycle.
	FullMetricsFactor int `yaml:"full_metrics_factor"`
}

func DefaultQdiscMetricsConfig() *QdiscMetricsConfig {
	return &QdiscMetricsConfig{
		Interval:          QDISC_METRICS_CONFIG_INTERVAL_DEFAULT,
		FullMetricsFactor: QDISC_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT,
	}
}

// Each qdisc will have some info cached, such as metrics, cycle#, etc:
type QdiscMetricsInfo struct {
	// Metrics, indexed by QDISK_... index:
	uint32DeltaMetrics map[int][]byte
	uint32Metrics      map[int][]byte
	uint64DeltaMetrics map[int][]byte
	presenceMetric     []byte

	// Delta metrics are skipped for zero-after-zero, keep track of previous
	// condition:
	uint32ZeroDelta []bool
	uint64ZeroDelta []bool

	// Cycle#:
	cycleNum int
}

type QdiscMetrics struct {
	// id/task_id:
	id string
	// Scan interval:
	interval time.Duration

	// Full metric factor:
	fullMetricsFactor int

	// Dual storage for parsed stats used as previous, current:
	qdiscStats [2]*qdisc.QdiscStats
	// Timestamp when the stats were collected:
	qdiscDevTs [2]time.Time
	// Index for current stats, toggled after each use:
	currIndex int

	// Qdisc info, indexed by qdisc.QdiscInfoKey:
	qdiscMetricsInfoMap map[qdisc.QdiscInfoKey]*QdiscMetricsInfo

	// Interval metric:
	intervalMetric []byte

	// A buffer for the timestamp suffix:
	tsSuffixBuf *bytes.Buffer

	// The total number of metrics, evaluated every time there is a change:
	totalMetricsCount int

	// The following are needed for testing only. Left to their default values,
	// the usual objects will be used.
	instance, hostname string
	timeNowFn          func() time.Time
	metricsQueue       MetricsQueue
}

func NewQdiscMetrics(cfg any) (*QdiscMetrics, error) {
	var (
		err             error
		qdiscMetricsCfg *QdiscMetricsConfig
	)

	switch cfg := cfg.(type) {
	case *LsvmiConfig:
		qdiscMetricsCfg = cfg.QdiscMetricsConfig
	case *QdiscMetricsConfig:
		qdiscMetricsCfg = cfg
	case nil:
		qdiscMetricsCfg = DefaultQdiscMetricsConfig()
	default:
		return nil, fmt.Errorf("NewQdiscMetrics: %T invalid config type", cfg)
	}

	interval, err := time.ParseDuration(qdiscMetricsCfg.Interval)
	if err != nil {
		return nil, err
	}
	qdiscMetrics := &QdiscMetrics{
		id:                  PROC_NET_DEV_METRICS_ID,
		interval:            interval,
		fullMetricsFactor:   qdiscMetricsCfg.FullMetricsFactor,
		qdiscMetricsInfoMap: make(map[qdisc.QdiscInfoKey]*QdiscMetricsInfo),
		tsSuffixBuf:         &bytes.Buffer{},
	}

	qdiscMetricsLog.Infof("id=%s", qdiscMetrics.id)
	qdiscMetricsLog.Infof("interval=%s", qdiscMetrics.interval)
	qdiscMetricsLog.Infof("full_metrics_factor=%d", qdiscMetrics.fullMetricsFactor)
	return qdiscMetrics, nil
}

func (qm *QdiscMetrics) updateQdiscMetricsInfo(qiKey qdisc.QdiscInfoKey, qi qdisc.QdiscInfo) {
	instance, hostname := GlobalInstance, GlobalHostname
	if qm.instance != "" {
		instance = qm.instance
	}
	if qm.hostname != "" {
		hostname = qm.hostname
	}

	qmi := &QdiscMetricsInfo{
		uint32DeltaMetrics: make(map[int][]byte),
		uint32Metrics:      make(map[int][]byte),
		uint64DeltaMetrics: map[int][]byte{},
		uint32ZeroDelta:    make([]bool, qdisc.QDISK_UINT32_NUM_STATS),
		uint64ZeroDelta:    make([]bool, qdisc.QDISK_UINT64_NUM_STATS),
		cycleNum:           initialCycleNum.Get(qm.fullMetricsFactor),
	}

	handle := qi.Uint32[qdisc.QDISC_HANDLE]
	parent := qi.Uint32[qdisc.QDISC_PARENT]

	commonLabels := fmt.Sprintf(
		`%s="%s",%s="%s",%s="%s",%s="%04x:%04x",%s="%04x:%04x",%s="%s"`,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
		QDISC_KIND_LABEL_NAME, qi.Kind,
		QDISC_HANDLE_LABEL_NAME, (handle >> 16), (handle & 0xfff),
		QDISC_PARENT_LABEL_NAME, (parent >> 16), (parent & 0xfff),
		QDISC_IF_LABEL_NAME, qi.IfName,
	)

	for i, name := range qdiscUint32IndexToDeltaMetricNameMap {
		qmi.uint32DeltaMetrics[i] = []byte(fmt.Sprintf(
			`%s{%s} `, // N.B. include space before value
			name, commonLabels,
		))
	}
	for i, name := range qdiscUint32IndexToMetricNameMap {
		qmi.uint32Metrics[i] = []byte(fmt.Sprintf(
			`%s{%s} `, // N.B. include space before value
			name, commonLabels,
		))
	}
	for i, name := range qdiscUint32IndexToDeltaMetricNameMap {
		qmi.uint32DeltaMetrics[i] = []byte(fmt.Sprintf(
			`%s{%s} `, // N.B. include space before value
			name, commonLabels,
		))
	}
	for i, name := range qdiscUint64IndexToDeltaMetricNameMap {
		qmi.uint64DeltaMetrics[i] = []byte(fmt.Sprintf(
			`%s{%s} `, // N.B. include space before value
			name, commonLabels,
		))
	}

	qmi.presenceMetric = []byte(fmt.Sprintf(
		`%s{%s} `, // N.B. include space before value
		QDISC_PRESENCE_METRIC, commonLabels,
	))
}
