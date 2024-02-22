// Scheduler metrics:

package lsvmi

import (
	"bytes"
	"fmt"
	"strconv"
	"time"
)

const (
	TASK_STATS_SCHEDULED_COUNT_DELTA_METRIC = "lsvmi_task_scheduled_delta"
	TASK_STATS_DELAYED_COUNT_DELTA_METRIC   = "lsvmi_task_delayed_delta"
	TASK_STATS_OVERRUN_COUNT_DELTA_METRIC   = "lsvmi_task_overrun_delta"
	TASK_STATS_EXECUTED_COUNT_DELTA_METRIC  = "lsvmi_task_executed_delta"
	TASK_STATS_AVG_RUNTIME_SEC_METRIC       = "lsvmi_task_avg_runtime_sec"

	TASK_STATS_TASK_ID_LABEL_NAME = "task_id"
)

type taskStatsIndexMetricMap map[int][]byte

type SchedulerInternalMetrics struct {
	// Dual buffer holding current, previous delta stats:
	stats [2]SchedulerStats
	// Which one is current:
	crtStatsIndx int
	// Cache the full metrics for each taskId and index:
	uint64MetricsCache  map[string]taskStatsIndexMetricMap
	float64MetricsCache map[string]taskStatsIndexMetricMap
	// A buffer for the timestamp suffix:
	tsSuffixBuf *bytes.Buffer
	// The following are needed for testing, if nil they default to the usual
	// common values:
	metricsCommonLabels []byte
	timeNowFn           func() time.Time
	scheduler           *Scheduler
}

var taskStatsUint64MetricsNameMap = map[int]string{
	TASK_STATS_SCHEDULED_COUNT: TASK_STATS_SCHEDULED_COUNT_DELTA_METRIC,
	TASK_STATS_DELAYED_COUNT:   TASK_STATS_DELAYED_COUNT_DELTA_METRIC,
	TASK_STATS_OVERRUN_COUNT:   TASK_STATS_OVERRUN_COUNT_DELTA_METRIC,
	TASK_STATS_EXECUTED_COUNT:  TASK_STATS_EXECUTED_COUNT_DELTA_METRIC,
}

var taskStatsFloat64MetricsNameMap = map[int]string{
	TASK_STATS_AVG_RUNTIME_SEC: TASK_STATS_AVG_RUNTIME_SEC_METRIC,
}

func NewSchedulerInternalMetrics() *SchedulerInternalMetrics {
	return &SchedulerInternalMetrics{
		uint64MetricsCache:  make(map[string]taskStatsIndexMetricMap),
		float64MetricsCache: make(map[string]taskStatsIndexMetricMap),
		tsSuffixBuf:         &bytes.Buffer{},
	}
}

func (sim *SchedulerInternalMetrics) updateMetricsCache(taskId string) {
	metricsCommonLabels := GlobalMetricsCommonLabels
	if sim.metricsCommonLabels != nil {
		metricsCommonLabels = sim.metricsCommonLabels
	}
	indexMetricMap := make(taskStatsIndexMetricMap)
	for index, name := range taskStatsUint64MetricsNameMap {
		metric := fmt.Sprintf(
			`%s{%s,%s="%s"} `, // N.B. include the whitespace separating the metric from value
			name, metricsCommonLabels, TASK_STATS_TASK_ID_LABEL_NAME, taskId,
		)
		indexMetricMap[index] = []byte(metric)
	}
	sim.uint64MetricsCache[taskId] = indexMetricMap

	indexMetricMap = make(taskStatsIndexMetricMap)
	for index, name := range taskStatsFloat64MetricsNameMap {
		metric := fmt.Sprintf(
			`%s{%s,%s="%s"} `, // N.B. include the whitespace separating the metric from value
			name, metricsCommonLabels, TASK_STATS_TASK_ID_LABEL_NAME, taskId,
		)
		indexMetricMap[index] = []byte(metric)
	}
	sim.float64MetricsCache[taskId] = indexMetricMap
}

func (sim *SchedulerInternalMetrics) GenerateMetrics(buf *bytes.Buffer, fullCycle bool) int {
	scheduler, timeNowFn := GlobalScheduler, time.Now
	if sim.scheduler != nil {
		scheduler = sim.scheduler
	}
	if sim.timeNowFn != nil {
		timeNowFn = sim.timeNowFn
	}

	crtStatsIndx := sim.crtStatsIndx
	sim.stats[crtStatsIndx] = scheduler.SnapStats(sim.stats[crtStatsIndx], STATS_SNAP_AND_CLEAR)
	ts := timeNowFn()
	sim.tsSuffixBuf.Reset()
	fmt.Fprintf(sim.tsSuffixBuf, " %d\n", ts.UnixMilli())
	tsSuffix := sim.tsSuffixBuf.Bytes()

	crtStats, prevStats := sim.stats[crtStatsIndx], sim.stats[1-crtStatsIndx]
	if fullCycle {
		prevStats = nil
	}

	var prevTaskStats *TaskStats = nil
	metricsCount := 0
	for taskId, crtTaskStats := range crtStats {
		if prevStats != nil {
			prevTaskStats = prevStats[taskId]
		}

		uint64IndexMetricMap := sim.uint64MetricsCache[taskId]
		if uint64IndexMetricMap == nil {
			sim.updateMetricsCache(taskId)
			uint64IndexMetricMap = sim.uint64MetricsCache[taskId]
		}
		for index, metric := range uint64IndexMetricMap {
			crtVal := crtTaskStats.uint64Stats[index]
			if prevTaskStats == nil || crtVal != prevTaskStats.uint64Stats[index] {
				buf.Write(metric)
				buf.WriteString(strconv.FormatUint(crtVal, 10))
				buf.Write(tsSuffix)
				metricsCount++
			}
		}

		for index, metric := range sim.float64MetricsCache[taskId] {
			crtVal := crtTaskStats.float64Stats[index]
			if prevTaskStats == nil || crtVal != prevTaskStats.float64Stats[index] {
				buf.Write(metric)
				buf.WriteString(strconv.FormatFloat(crtVal, 'f', 6, 64))
				buf.Write(tsSuffix)
				metricsCount++
			}
		}
	}

	// Flip the buffers:
	sim.crtStatsIndx = 1 - crtStatsIndx

	return metricsCount
}
