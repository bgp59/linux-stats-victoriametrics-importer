// Scheduler metrics:

package lsvmi

import (
	"bytes"
	"fmt"
	"strconv"
)

const (
	TASK_STATS_SCHEDULED_COUNT_DELTA_METRIC = "lsvmi_task_scheduled_count_delta"
	TASK_STATS_DELAYED_COUNT_DELTA_METRIC   = "lsvmi_task_delayed_count_delta"
	TASK_STATS_OVERRUN_COUNT_DELTA_METRIC   = "lsvmi_task_overrun_count_delta"
	TASK_STATS_EXECUTED_COUNT_DELTA_METRIC  = "lsvmi_task_executed_count_delta"
	TASK_STATS_INTERVAL_AVG_RUNTIME_METRIC  = "lsvmi_task_interval_avg_runtime_sec"

	TASK_STATS_TASK_ID_LABEL_NAME = "task_id"
)

type taskStatsIndexMetricMap map[int][]byte

type SchedulerInternalMetrics struct {
	// Internal metrics, for common values:
	internalMetrics *InternalMetrics
	// Dual storage for snapping the stats, used as current, previous, toggled
	// after every metrics generation:
	stats [2]SchedulerStats
	// The current index:
	crtIndex int
	// Cache the full metrics for each taskId and stats index:
	uint64DeltaMetricsCache map[string]taskStatsIndexMetricMap
	// Cache the avg runtime metrics for each taskId:
	avgRuntimeMetricsCache map[string][]byte
}

// The following stats will be used to generate deltas:
var taskStatsUint64DeltaMetricsNameMap = map[int]string{
	TASK_STATS_SCHEDULED_COUNT: TASK_STATS_SCHEDULED_COUNT_DELTA_METRIC,
	TASK_STATS_DELAYED_COUNT:   TASK_STATS_DELAYED_COUNT_DELTA_METRIC,
	TASK_STATS_OVERRUN_COUNT:   TASK_STATS_OVERRUN_COUNT_DELTA_METRIC,
	TASK_STATS_EXECUTED_COUNT:  TASK_STATS_EXECUTED_COUNT_DELTA_METRIC,
}

func NewSchedulerInternalMetrics(internalMetrics *InternalMetrics) *SchedulerInternalMetrics {
	return &SchedulerInternalMetrics{
		internalMetrics:         internalMetrics,
		uint64DeltaMetricsCache: make(map[string]taskStatsIndexMetricMap),
		avgRuntimeMetricsCache:  make(map[string][]byte),
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
	for index, name := range taskStatsUint64DeltaMetricsNameMap {
		metric := fmt.Sprintf(
			`%s{%s="%s",%s="%s",%s="%s"} `, // N.B. include the whitespace separating the metric from value
			name,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
			TASK_STATS_TASK_ID_LABEL_NAME, taskId,
		)
		indexMetricMap[index] = []byte(metric)
	}
	sim.uint64DeltaMetricsCache[taskId] = indexMetricMap

	sim.avgRuntimeMetricsCache[taskId] = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s",%s="%s"} `, // N.B. include the whitespace separating the metric from value
		TASK_STATS_INTERVAL_AVG_RUNTIME_METRIC,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
		TASK_STATS_TASK_ID_LABEL_NAME, taskId,
	))
}

func (sim *SchedulerInternalMetrics) generateMetrics(
	buf *bytes.Buffer, tsSuffix []byte,
) int {
	if tsSuffix == nil {
		// This should happen only during unit testing:
		tsSuffix = sim.internalMetrics.getTsSuffix()
	}

	metricsCount := 0
	crtStats, prevStats := sim.stats[sim.crtIndex], sim.stats[1-sim.crtIndex]
	var prevTaskStats *TaskStats
	for taskId, crtTaskStats := range crtStats {
		execCountDelta, hasExecCountDelta := uint64(0), false
		if prevStats != nil {
			prevTaskStats = prevStats[taskId]
		} else {
			prevTaskStats = nil
		}
		uint64IndexMetricMap := sim.uint64DeltaMetricsCache[taskId]
		if uint64IndexMetricMap == nil {
			// N.B. This will also update sim.avgRuntimeMetricsCache.
			sim.updateMetricsCache(taskId)
			uint64IndexMetricMap = sim.uint64DeltaMetricsCache[taskId]
		}
		for index, metric := range uint64IndexMetricMap {
			val := crtTaskStats.Uint64Stats[index]
			if prevTaskStats != nil {
				val -= prevTaskStats.Uint64Stats[index]
			}
			buf.Write(metric)
			buf.WriteString(strconv.FormatUint(val, 10))
			buf.Write(tsSuffix)
			if index == TASK_STATS_EXECUTED_COUNT {
				execCountDelta, hasExecCountDelta = val, true
			}
			metricsCount++
		}

		{
			runtimeAvg := 0.
			// Safeguard against dropping exec count delta from metrics set:
			if !hasExecCountDelta {
				execCountDelta = crtTaskStats.Uint64Stats[TASK_STATS_EXECUTED_COUNT]
				if prevTaskStats != nil {
					execCountDelta -= prevTaskStats.Uint64Stats[TASK_STATS_EXECUTED_COUNT]
				}
			}
			if execCountDelta > 0 {
				runtimeDelta := crtTaskStats.RuntimeTotal
				if prevTaskStats != nil {
					runtimeDelta -= prevTaskStats.RuntimeTotal
				}
				runtimeAvg = runtimeDelta.Seconds() / float64(execCountDelta)
			}
			metric := sim.avgRuntimeMetricsCache[taskId]
			buf.Write(metric)
			buf.WriteString(strconv.FormatFloat(runtimeAvg, 'f', 6, 64))
			buf.Write(tsSuffix)
			metricsCount++
		}
	}

	// Flip the stats storage:
	sim.crtIndex = 1 - sim.crtIndex

	return metricsCount
}
