// Scheduler metrics:

package lsvmi

import (
	"bytes"
	"fmt"
	"strconv"
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
	// Internal metrics, for common values:
	internalMetrics *InternalMetrics
	// Dual buffer holding current, previous delta stats:
	stats [2]SchedulerStats
	// Which one is current:
	crtStatsIndx int
	// Cache the full metrics for each taskId and stats index:
	uint64MetricsCache  map[string]taskStatsIndexMetricMap
	float64MetricsCache map[string]taskStatsIndexMetricMap
	// A buffer for the timestamp suffix:
	tsSuffixBuf *bytes.Buffer
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

func NewSchedulerInternalMetrics(internalMetrics *InternalMetrics) *SchedulerInternalMetrics {
	return &SchedulerInternalMetrics{
		internalMetrics:     internalMetrics,
		uint64MetricsCache:  make(map[string]taskStatsIndexMetricMap),
		float64MetricsCache: make(map[string]taskStatsIndexMetricMap),
		tsSuffixBuf:         &bytes.Buffer{},
	}
}

func (sim *SchedulerInternalMetrics) updateMetricsCache(taskId string) {
	instance, hostname := GlobalInstance, GlobalHostname
	if sim.internalMetrics.instance != "" {
		instance = sim.internalMetrics.instance
	}
	if sim.internalMetrics.hostname != "" {
		hostname = sim.internalMetrics.hostname
	}

	indexMetricMap := make(taskStatsIndexMetricMap)
	for index, name := range taskStatsUint64MetricsNameMap {
		metric := fmt.Sprintf(
			`%s{%s="%s",%s="%s",%s="%s"} `, // N.B. include the whitespace separating the metric from value
			name,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
			TASK_STATS_TASK_ID_LABEL_NAME, taskId,
		)
		indexMetricMap[index] = []byte(metric)
	}
	sim.uint64MetricsCache[taskId] = indexMetricMap

	indexMetricMap = make(taskStatsIndexMetricMap)
	for index, name := range taskStatsFloat64MetricsNameMap {
		metric := fmt.Sprintf(
			`%s{%s="%s",%s="%s",%s="%s"} `, // N.B. include the whitespace separating the metric from value
			name,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
			TASK_STATS_TASK_ID_LABEL_NAME, taskId,
		)
		indexMetricMap[index] = []byte(metric)
	}
	sim.float64MetricsCache[taskId] = indexMetricMap
}

func (sim *SchedulerInternalMetrics) generateMetrics(
	buf *bytes.Buffer, fullCycle bool, tsSuffix []byte,
) int {
	crtStatsIndx := sim.crtStatsIndx
	crtStats, prevStats := sim.stats[crtStatsIndx], sim.stats[1-crtStatsIndx]
	if fullCycle {
		prevStats = nil
	}

	if tsSuffix == nil {
		// This should happen only during unit testing:
		tsSuffix = sim.internalMetrics.getTsSuffix()
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
			crtVal := crtTaskStats.Uint64Stats[index]
			if prevTaskStats == nil || crtVal != prevTaskStats.Uint64Stats[index] {
				buf.Write(metric)
				buf.WriteString(strconv.FormatUint(crtVal, 10))
				buf.Write(tsSuffix)
				metricsCount++
			}
		}

		for index, metric := range sim.float64MetricsCache[taskId] {
			crtVal := crtTaskStats.Float64Stats[index]
			if prevTaskStats == nil || crtVal != prevTaskStats.Float64Stats[index] {
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
