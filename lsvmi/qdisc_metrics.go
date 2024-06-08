// qdisc stats metrics

package lsvmi

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/emypar/linux-stats-victoriametrics-importer/qdisc"
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

var qdiscUint64IndexToMetricNameMap = map[int]string{}

var qdiscMetricsCount = (len(qdiscUint32IndexToDeltaMetricNameMap) +
	len(qdiscUint32IndexToMetricNameMap) +
	len(qdiscUint64IndexToDeltaMetricNameMap) +
	len(qdiscUint64IndexToMetricNameMap))

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

var qdiscMajMinFmt = fmt.Sprintf(`%%0%dx:%%0%dx`, (qdisc.QDISC_MAJ_NUM_BITS+3)/4, (qdisc.QDISC_MIN_NUM_BITS+3)/4)

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
	uint64Metrics      map[int][]byte
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
	qdiscStatsTs [2]time.Time
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
		id:                  QDISC_METRICS_ID,
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

func qdiscMajMinLabelVal(val uint32) string {
	const qdisc_min_mask = (uint32(1) << qdisc.QDISC_MIN_NUM_BITS) - 1
	return fmt.Sprintf(qdiscMajMinFmt, (val >> qdisc.QDISC_MIN_NUM_BITS), (val & qdisc_min_mask))
}

func (qm *QdiscMetrics) updateQdiscMetricsInfo(qiKey qdisc.QdiscInfoKey, qi *qdisc.QdiscInfo) {
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

	commonLabels := fmt.Sprintf(
		`%s="%s",%s="%s",%s="%s",%s="%s",%s="%s",%s="%s"`,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
		QDISC_KIND_LABEL_NAME, qi.Kind,
		QDISC_HANDLE_LABEL_NAME, qdiscMajMinLabelVal(qi.Uint32[qdisc.QDISC_HANDLE]),
		QDISC_PARENT_LABEL_NAME, qdiscMajMinLabelVal(qi.Uint32[qdisc.QDISC_PARENT]),
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
	for i, name := range qdiscUint64IndexToDeltaMetricNameMap {
		qmi.uint64DeltaMetrics[i] = []byte(fmt.Sprintf(
			`%s{%s} `, // N.B. include space before value
			name, commonLabels,
		))
	}
	for i, name := range qdiscUint64IndexToMetricNameMap {
		qmi.uint64Metrics[i] = []byte(fmt.Sprintf(
			`%s{%s} `, // N.B. include space before value
			name, commonLabels,
		))
	}

	qmi.presenceMetric = []byte(fmt.Sprintf(
		`%s{%s} `, // N.B. include space before value
		QDISC_PRESENCE_METRIC, commonLabels,
	))

	qm.qdiscMetricsInfoMap[qiKey] = qmi
}

func (qm *QdiscMetrics) updateIntervalMetric() {
	instance, hostname := GlobalInstance, GlobalHostname
	if qm.instance != "" {
		instance = qm.instance
	}
	if qm.hostname != "" {
		hostname = qm.hostname
	}

	qm.intervalMetric = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s"} `, // N.B. include space before value
		QDISC_INTERVAL_METRIC_NAME,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
	))
}

func (qm *QdiscMetrics) generateMetrics(buf *bytes.Buffer) (int, int) {
	currQdiscStats, prevQdiscStats := qm.qdiscStats[qm.currIndex], qm.qdiscStats[1-qm.currIndex]
	currTs, prevTs := qm.qdiscStatsTs[qm.currIndex], qm.qdiscStatsTs[1-qm.currIndex]
	qm.currIndex = 1 - qm.currIndex

	// Since most metrics are delta, a previous state is required:
	if prevQdiscStats == nil {
		return 0, 0
	}

	actualMetricsCount := 0
	qm.tsSuffixBuf.Reset()
	fmt.Fprintf(
		qm.tsSuffixBuf, " %d\n", currTs.UnixMilli(),
	)
	promTs := qm.tsSuffixBuf.Bytes()
	deltaSec := currTs.Sub(prevTs).Seconds()
	evalTotalMetricsCount := qm.totalMetricsCount == 0

	for qiKey, currQi := range currQdiscStats.Info {
		prevQi := prevQdiscStats.Info[qiKey]
		if prevQi == nil {
			continue
		}

		qdiscMetricsInfo := qm.qdiscMetricsInfoMap[qiKey]
		// Sanity check that no info has changed for this since the previous
		// parse; if it did then the metrics will have to be regenerated:
		if qdiscMetricsInfo != nil && (currQi.IfName != prevQi.IfName ||
			currQi.Kind != prevQi.Kind ||
			currQi.Uint32[qdisc.QDISC_PARENT] != prevQi.Uint32[qdisc.QDISC_PARENT]) {
			// Clear the prev presence metric:
			buf.Write(qdiscMetricsInfo.presenceMetric)
			buf.WriteByte('0')
			buf.Write(promTs)
			actualMetricsCount++
			// Force regeneration:
			qdiscMetricsInfo = nil
		}
		fullMetrics := qdiscMetricsInfo == nil || qdiscMetricsInfo.cycleNum == 0
		if qdiscMetricsInfo == nil {
			qm.updateQdiscMetricsInfo(qiKey, currQi)
			qdiscMetricsInfo = qm.qdiscMetricsInfoMap[qiKey]
		}

		for index, metric := range qdiscMetricsInfo.uint32DeltaMetrics {
			val := currQi.Uint32[index] - prevQi.Uint32[index]
			if fullMetrics || val > 0 || !qdiscMetricsInfo.uint32ZeroDelta[index] {
				buf.Write(metric)
				if rate := qdiscUint32IndexRate[index]; rate != nil {
					buf.WriteString(strconv.FormatFloat(
						float64(val)/deltaSec*rate.factor, 'f', rate.prec, 64,
					))
				} else {
					buf.WriteString(strconv.FormatUint(uint64(val), 10))
				}
				buf.Write(promTs)
				actualMetricsCount++
			}
			qdiscMetricsInfo.uint32ZeroDelta[index] = val == 0
		}
		for index, metric := range qdiscMetricsInfo.uint32Metrics {
			val := currQi.Uint32[index]
			if fullMetrics || val != prevQi.Uint32[index] {
				buf.Write(metric)
				buf.WriteString(strconv.FormatUint(uint64(val), 10))
				buf.Write(promTs)
				actualMetricsCount++
			}
		}

		for index, metric := range qdiscMetricsInfo.uint64DeltaMetrics {
			val := currQi.Uint64[index] - prevQi.Uint64[index]
			if fullMetrics || val > 0 || !qdiscMetricsInfo.uint64ZeroDelta[index] {
				buf.Write(metric)
				if rate := qdiscUint64IndexRate[index]; rate != nil {
					buf.WriteString(strconv.FormatFloat(
						float64(val)/deltaSec*rate.factor, 'f', rate.prec, 64,
					))
				} else {
					buf.WriteString(strconv.FormatUint(val, 10))
				}
				buf.Write(promTs)
				actualMetricsCount++
			}
			qdiscMetricsInfo.uint64ZeroDelta[index] = val == 0
		}
		for index, metric := range qdiscMetricsInfo.uint64Metrics {
			val := currQi.Uint64[index]
			if fullMetrics || val != prevQi.Uint64[index] {
				buf.Write(metric)
				buf.WriteString(strconv.FormatUint(val, 10))
				buf.Write(promTs)
				actualMetricsCount++
			}
		}

		if fullMetrics {
			buf.Write(qdiscMetricsInfo.presenceMetric)
			buf.WriteByte('1')
			buf.Write(promTs)
			actualMetricsCount++
		}

		if qdiscMetricsInfo.cycleNum++; qdiscMetricsInfo.cycleNum >= qm.fullMetricsFactor {
			qdiscMetricsInfo.cycleNum = 0
		}

	}

	// Clear out-of-scope qdiscs as needed:
	if len(qm.qdiscMetricsInfoMap) != len(currQdiscStats.Info) {
		for qiKey, qdiscMetricsInfo := range qm.qdiscMetricsInfoMap {
			if currQdiscStats.Info[qiKey] == nil {
				// Clear the prev presence metric:
				buf.Write(qdiscMetricsInfo.presenceMetric)
				buf.WriteByte('0')
				buf.Write(promTs)
				actualMetricsCount++

				delete(qm.qdiscMetricsInfoMap, qiKey)
			}
		}
	}

	if qm.intervalMetric == nil {
		qm.updateIntervalMetric()
	}
	buf.Write(qm.intervalMetric)
	buf.WriteString(strconv.FormatFloat(deltaSec, 'f', 6, 64))
	buf.Write(promTs)
	actualMetricsCount++

	if evalTotalMetricsCount {
		// The total number of metrics:
		//		qdisc metrics#: (number of qdisc) * (number of counters + 1 (presence))
		//		interval metric#: 1
		qm.totalMetricsCount = len(currQdiscStats.Info)*(qdiscMetricsCount+1) + 1
	}

	return actualMetricsCount, qm.totalMetricsCount
}

// Satisfy the TaskActivity interface:
func (qm *QdiscMetrics) Execute() bool {
	timeNowFn := time.Now
	if qm.timeNowFn != nil {
		timeNowFn = qm.timeNowFn
	}

	metricsQueue := GlobalMetricsQueue
	if qm.metricsQueue != nil {
		metricsQueue = qm.metricsQueue
	}

	currQdiscStats := qm.qdiscStats[qm.currIndex]
	if currQdiscStats == nil {
		prevQdiscStats := qm.qdiscStats[1-qm.currIndex]
		if prevQdiscStats != nil {
			currQdiscStats = prevQdiscStats.Clone()
		} else {
			currQdiscStats = qdisc.NewQdiscStats()
		}
		qm.qdiscStats[qm.currIndex] = currQdiscStats
	}
	err := currQdiscStats.Parse()
	if err != nil {
		qdiscMetricsLog.Warnf("%v: qdisc metrics will be disabled", err)
		return false
	}
	qm.qdiscStatsTs[qm.currIndex] = timeNowFn()

	buf := metricsQueue.GetBuf()
	actualMetricsCount, totalMetricsCount := qm.generateMetrics(buf)
	if totalMetricsCount > 0 {
		byteCount := buf.Len()
		metricsQueue.QueueBuf(buf)
		GlobalMetricsGeneratorStatsContainer.Update(
			qm.id, uint64(actualMetricsCount), uint64(totalMetricsCount), uint64(byteCount),
		)
	} else {
		metricsQueue.ReturnBuf(buf)
	}

	return true
}

// Define and register the task builder:
func QdiscMetricsTaskBuilder(cfg *LsvmiConfig) ([]*Task, error) {
	qm, err := NewQdiscMetrics(cfg)
	if err != nil {
		return nil, err
	}
	if qm.interval <= 0 {
		qdiscMetricsLog.Infof(
			"interval=%s, metrics disabled", qm.interval,
		)
		return nil, nil
	}
	tasks := []*Task{
		NewTask(qm.id, qm.interval, qm),
	}
	return tasks, nil
}

func init() {
	if qdisc.QdiscAvailable {
		TaskBuilders.Register(QdiscMetricsTaskBuilder)
	}
}
