// Tests for scheduler internal metrics

package lsvmi

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/eparparita/linux-stats-victoriametrics-importer/internal/testutils"
)

type SchedulerInternalMetricsTestWantTaskStatsIndex struct {
	uint64Index, float64Index []int
}

type SchedulerInternalMetricsTestCase struct {
	instance            string
	hostname            string
	promTs              int64
	fullCycle           bool
	crtStats, prevStats SchedulerStats
	wantIndex           map[string]*SchedulerInternalMetricsTestWantTaskStatsIndex
	wantMetricsCount    int
	reportExtra         bool
}

var schedulerInternalMetricsTestUint64Fmts = map[int]string{
	TASK_STATS_SCHEDULED_COUNT: fmt.Sprintf(
		`%s{%s="%%s",%s="%%s",%s="%%s"} %%d %%d`,
		TASK_STATS_SCHEDULED_COUNT_DELTA_METRIC, INSTANCE_LABEL_NAME, HOSTNAME_LABEL_NAME, TASK_STATS_TASK_ID_LABEL_NAME,
	),
	TASK_STATS_DELAYED_COUNT: fmt.Sprintf(
		`%s{%s="%%s",%s="%%s",%s="%%s"} %%d %%d`,
		TASK_STATS_DELAYED_COUNT_DELTA_METRIC, INSTANCE_LABEL_NAME, HOSTNAME_LABEL_NAME, TASK_STATS_TASK_ID_LABEL_NAME,
	),
	TASK_STATS_OVERRUN_COUNT: fmt.Sprintf(
		`%s{%s="%%s",%s="%%s",%s="%%s"} %%d %%d`,
		TASK_STATS_OVERRUN_COUNT_DELTA_METRIC, INSTANCE_LABEL_NAME, HOSTNAME_LABEL_NAME, TASK_STATS_TASK_ID_LABEL_NAME,
	),
	TASK_STATS_EXECUTED_COUNT: fmt.Sprintf(
		`%s{%s="%%s",%s="%%s",%s="%%s"} %%d %%d`,
		TASK_STATS_EXECUTED_COUNT_DELTA_METRIC, INSTANCE_LABEL_NAME, HOSTNAME_LABEL_NAME, TASK_STATS_TASK_ID_LABEL_NAME,
	),
}

var schedulerInternalMetricsTestFloat64Fmts = map[int]string{
	TASK_STATS_AVG_RUNTIME_SEC: fmt.Sprintf(
		`%s{%s="%%s",%s="%%s",%s="%%s"} %%.6f %%d`,
		TASK_STATS_AVG_RUNTIME_SEC_METRIC, INSTANCE_LABEL_NAME, HOSTNAME_LABEL_NAME, TASK_STATS_TASK_ID_LABEL_NAME,
	),
}

func newTestSchedulerInternalMetrics(
	instance, hostname string,
	timeNowFn func() time.Time,
	scheduler *Scheduler,
) *SchedulerInternalMetrics {
	sim := NewSchedulerInternalMetrics()
	if instance != "" && hostname != "" {
		sim.metricsCommonLabels = buildMetricsCommonLabels(instance, hostname)
	}
	sim.timeNowFn = timeNowFn
	sim.scheduler = scheduler
	return sim
}

func duplicateTestSchedulerStats(stats SchedulerStats) SchedulerStats {
	dupStats := make(SchedulerStats)
	for taskId, taskStats := range stats {
		dupTaskStats := NewTaskStats()
		copy(dupTaskStats.uint64Stats, taskStats.uint64Stats)
		copy(dupTaskStats.float64Stats, taskStats.float64Stats)
		dupStats[taskId] = dupTaskStats
	}
	return dupStats
}

func testSchedulerInternalMetrics(tc *SchedulerInternalMetricsTestCase, t *testing.T) {
	tlc := testutils.NewTestingLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	testScheduler, err := NewScheduler(nil)
	if err != nil {
		tlc.Fatal(err)
	}
	timeNowRetVal := time.UnixMilli(tc.promTs)
	schedulerInternalMetrics := newTestSchedulerInternalMetrics(
		tc.instance, tc.hostname,
		func() time.Time { return timeNowRetVal },
		testScheduler,
	)
	schedulerInternalMetrics.stats[1-schedulerInternalMetrics.crtStatsIndx] = tc.prevStats

	testScheduler.stats = duplicateTestSchedulerStats(tc.crtStats)
	testMetricsQueue := testutils.NewTestMetricsQueue(0)

	buf := testMetricsQueue.GetBuf()
	gotMetricsCount := schedulerInternalMetrics.GenerateMetrics(buf, tc.fullCycle)
	testMetricsQueue.QueueBuf(buf)

	errBuf := &bytes.Buffer{}

	if tc.wantMetricsCount != gotMetricsCount {
		fmt.Fprintf(
			errBuf,
			"\nmetrics count: want: %d, got: %d",
			tc.wantMetricsCount, gotMetricsCount,
		)
	}

	wantMetrics := make([]string, 0)

	for taskId, wantsTaskStatsIndex := range tc.wantIndex {
		if wantsTaskStatsIndex.uint64Index != nil {
			taskUint64Stats := tc.crtStats[taskId].uint64Stats
			for _, indx := range wantsTaskStatsIndex.uint64Index {
				metric := fmt.Sprintf(
					schedulerInternalMetricsTestUint64Fmts[indx],
					tc.instance, tc.hostname, taskId, taskUint64Stats[indx], tc.promTs,
				)
				wantMetrics = append(wantMetrics, metric)
			}
		}
		if wantsTaskStatsIndex.float64Index != nil {
			taskFloat64Stats := tc.crtStats[taskId].float64Stats
			for _, indx := range wantsTaskStatsIndex.float64Index {
				metric := fmt.Sprintf(
					schedulerInternalMetricsTestFloat64Fmts[indx],
					tc.instance, tc.hostname, taskId, taskFloat64Stats[indx], tc.promTs,
				)
				wantMetrics = append(wantMetrics, metric)
			}
		}
	}

	testMetricsQueue.GenerateReport(wantMetrics, tc.reportExtra, errBuf)

	if errBuf.Len() > 0 {
		tlc.Fatal(errBuf)
	}
}

func TestSchedulerInternalMetrics(t *testing.T) {
	instance := "lsvmi_test"
	hostname := "lsvmi-test"
	promTs := int64(123456789000)

	for _, tc := range []*SchedulerInternalMetricsTestCase{
		{
			instance: instance,
			hostname: hostname,
			promTs:   promTs,
			crtStats: SchedulerStats{
				"taskA": &TaskStats{
					uint64Stats:  []uint64{1, 2, 3, 4},
					float64Stats: []float64{0.1, 0.1 / 4},
				},
			},
			wantMetricsCount: 5,
			wantIndex: map[string]*SchedulerInternalMetricsTestWantTaskStatsIndex{
				"taskA": {
					uint64Index:  []int{0, 1, 2, 3},
					float64Index: []int{1},
				},
			},
			reportExtra: true,
		},
		{
			instance:  instance,
			hostname:  hostname,
			promTs:    promTs,
			fullCycle: true,
			crtStats: SchedulerStats{
				"taskA": &TaskStats{
					uint64Stats:  []uint64{1, 2, 3, 4},
					float64Stats: []float64{0.1, 0.1 / 4},
				},
			},
			prevStats: SchedulerStats{
				"taskA": &TaskStats{
					uint64Stats:  []uint64{1, 2, 3, 4},
					float64Stats: []float64{0.1, 0.1 / 4},
				},
			},
			wantMetricsCount: 5,
			wantIndex: map[string]*SchedulerInternalMetricsTestWantTaskStatsIndex{
				"taskA": {
					uint64Index:  []int{0, 1, 2, 3},
					float64Index: []int{1},
				},
			},
			reportExtra: true,
		},
		{
			instance:  instance,
			hostname:  hostname,
			promTs:    promTs,
			fullCycle: false,
			crtStats: SchedulerStats{
				"taskA": &TaskStats{
					uint64Stats:  []uint64{1, 2, 3, 4},
					float64Stats: []float64{0.1, 0.1 / 4},
				},
			},
			prevStats: SchedulerStats{
				"taskA": &TaskStats{
					uint64Stats:  []uint64{1, 2, 3, 4},
					float64Stats: []float64{0.1, 0.1 / 4},
				},
			},
			wantMetricsCount: 0,
			reportExtra:      true,
		},
		{
			instance:  instance,
			hostname:  hostname,
			promTs:    promTs,
			fullCycle: false,
			crtStats: SchedulerStats{
				"taskA": &TaskStats{
					uint64Stats:  []uint64{1, 2, 3, 4},
					float64Stats: []float64{0.1, 0.1 / 4},
				},
			},
			prevStats: SchedulerStats{
				"taskA": &TaskStats{
					uint64Stats:  []uint64{11, 2, 3, 4},
					float64Stats: []float64{0.1, 0.1 / 4},
				},
			},
			wantMetricsCount: 1,
			wantIndex: map[string]*SchedulerInternalMetricsTestWantTaskStatsIndex{
				"taskA": {
					uint64Index: []int{0},
				},
			},
			reportExtra: true,
		},
		{
			instance:  instance,
			hostname:  hostname,
			promTs:    promTs,
			fullCycle: false,
			crtStats: SchedulerStats{
				"taskA": &TaskStats{
					uint64Stats:  []uint64{1, 12, 3, 4},
					float64Stats: []float64{0.1, 0.1 / 4},
				},
			},
			prevStats: SchedulerStats{
				"taskA": &TaskStats{
					uint64Stats:  []uint64{1, 2, 3, 4},
					float64Stats: []float64{0.1, 0.1 / 4},
				},
			},
			wantMetricsCount: 1,
			wantIndex: map[string]*SchedulerInternalMetricsTestWantTaskStatsIndex{
				"taskA": {
					uint64Index: []int{1},
				},
			},
			reportExtra: true,
		},
		{
			instance:  instance,
			hostname:  hostname,
			promTs:    promTs,
			fullCycle: false,
			crtStats: SchedulerStats{
				"taskA": &TaskStats{
					uint64Stats:  []uint64{1, 2, 3, 4},
					float64Stats: []float64{0.1, 0.1 / 4},
				},
				"taskB": &TaskStats{
					uint64Stats:  []uint64{1, 2, 3, 14},
					float64Stats: []float64{1, 1. / 14},
				},
			},
			prevStats: SchedulerStats{
				"taskB": &TaskStats{
					uint64Stats:  []uint64{1, 2, 3, 4},
					float64Stats: []float64{0.1, 0.1 / 4},
				},
			},
			wantMetricsCount: 7,
			wantIndex: map[string]*SchedulerInternalMetricsTestWantTaskStatsIndex{
				"taskA": {
					uint64Index:  []int{0, 1, 2, 3},
					float64Index: []int{1},
				},
				"taskB": {
					uint64Index:  []int{3},
					float64Index: []int{1},
				},
			},
			reportExtra: true,
		},
	} {
		t.Run(
			"",
			func(t *testing.T) { testSchedulerInternalMetrics(tc, t) },
		)
	}
}
