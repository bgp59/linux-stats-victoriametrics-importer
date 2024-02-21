// Scheduler metrics:

package lsvmi

import (
	"bytes"
	"fmt"
	"strconv"
)

const (
	TASK_SCHEDULED_COUNT_DELTA_METRIC      = "lsvmi_task_scheduled_delta"
	TASK_STATS_DELAYED_COUNT_DELTA_METRIC  = "lsvmi_task_delayed_delta"
	TASK_STATS_OVERRUN_COUNT_DELTA_METRIC  = "lsvmi_task_overrun_delta"
	TASK_STATS_EXECUTED_COUNT_DELTA_METRIC = "lsvmi_task_executed_delta"
	TASK_STATS_AVG_RUNTIME_SEC_METRIC      = "lsvmi_task_avg_runtime_sec"

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
}

var taskStatsUint64MetricsNameMap = map[int]string{
	TASK_STATS_SCHEDULED_COUNT: TASK_SCHEDULED_COUNT_DELTA_METRIC,
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
	indexMetricMap := make(taskStatsIndexMetricMap)
	for index, name := range taskStatsUint64MetricsNameMap {
		metric := fmt.Sprint(
			`%s{%s,%s="%s"} `, // N.B. include the whitespace separating the metric from value
			name, commonLabels, TASK_STATS_TASK_ID_LABEL_NAME, taskId,
		)
		indexMetricMap[index] = []byte(metric)
	}
	sim.uint64MetricsCache[taskId] = indexMetricMap

	indexMetricMap = make(taskStatsIndexMetricMap)
	for index, name := range taskStatsFloat64MetricsNameMap {
		metric := fmt.Sprint(
			`%s{%s,%s="%s"} `, // N.B. include the whitespace separating the metric from value
			name, commonLabels, TASK_STATS_TASK_ID_LABEL_NAME, taskId,
		)
		indexMetricMap[index] = []byte(metric)
	}
	sim.float64MetricsCache[taskId] = indexMetricMap
}

func (sim *SchedulerInternalMetrics) GenerateMetrics(buf *bytes.Buffer, fullCycle bool) {
	crtStatsIndx := sim.crtStatsIndx
	sim.stats[crtStatsIndx] = internalMetricsScheduler.SnapStats(sim.stats[crtStatsIndx], STATS_SNAP_AND_CLEAR)
	tsSuffix := internalMetricsTimestampSuffixFn(sim.tsSuffixBuf)

	crtStats, prevStats := sim.stats[crtStatsIndx], sim.stats[1-crtStatsIndx]
	if fullCycle {
		prevStats = nil
	}

	var prevTaskStats *TaskStats = nil
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
			}
		}

		for index, metric := range sim.float64MetricsCache[taskId] {
			crtVal := crtTaskStats.float64Stats[index]
			if prevTaskStats == nil || crtVal != prevTaskStats.float64Stats[index] {
				buf.Write(metric)
				buf.WriteString(strconv.FormatFloat(crtVal, 'f', 6, 64))
				buf.Write(tsSuffix)
			}
		}
	}

	// Flip the buffers:
	sim.crtStatsIndx = 1 - crtStatsIndx
}
