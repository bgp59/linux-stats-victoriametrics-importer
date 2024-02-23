// Tests for scheduler internal metrics

package lsvmi

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/eparparita/linux-stats-victoriametrics-importer/internal/testutils"
)

type SchedIMTestWantTaskStatsIndex struct {
	uint64Index, float64Index []int
}

type SchedIMTestCase struct {
	instance            string
	hostname            string
	promTs              int64
	fullCycle           bool
	crtStats, prevStats SchedulerStats
	wantIndex           map[string]*SchedIMTestWantTaskStatsIndex
	wantMetricsCount    int
	reportExtra         bool
}

// The following are redefined here to detect unwanted changes in the code:
var (
	schedIMTestInstLbl   = "inst"
	schedIMTestHostLbl   = "node"
	schedIMTestTaskIdLbl = "task_id"

	schedIMTestUint64MetricName = map[int]string{
		TASK_STATS_SCHEDULED_COUNT: "lsvmi_task_scheduled_delta",
		TASK_STATS_DELAYED_COUNT:   "lsvmi_task_delayed_delta",
		TASK_STATS_OVERRUN_COUNT:   "lsvmi_task_overrun_delta",
		TASK_STATS_EXECUTED_COUNT:  "lsvmi_task_executed_delta",
	}

	schedIMTestFloat4MetricName = map[int]string{
		TASK_STATS_AVG_RUNTIME_SEC: "lsvmi_task_avg_runtime_sec",
	}
)

func newTestSchedulerInternalMetrics(tc *SchedIMTestCase) (*InternalMetrics, error) {
	internalMetrics, err := NewInternalMetrics(nil)
	if err != nil {
		return nil, err
	}
	scheduler, err := NewScheduler(nil)
	if err != nil {
		return nil, err
	}

	internalMetrics.instance = tc.instance
	internalMetrics.hostname = tc.hostname
	timeNowRetVal := time.UnixMilli(tc.promTs)
	internalMetrics.timeNowFn = func() time.Time { return timeNowRetVal }
	internalMetrics.scheduler = scheduler

	return internalMetrics, nil
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

func testSchedulerInternalMetrics(tc *SchedIMTestCase, t *testing.T) {
	tlc := testutils.NewTestingLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	internalMetrics, err := newTestSchedulerInternalMetrics(tc)
	if err != nil {
		tlc.Fatal(err)
	}
	schedulerInternalMetrics := internalMetrics.schedulerMetrics
	schedulerInternalMetrics.stats[1-schedulerInternalMetrics.crtStatsIndx] = tc.prevStats
	internalMetrics.scheduler.stats = duplicateTestSchedulerStats(tc.crtStats)
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
					`%s{%s="%s",%s="%s",%s="%s"} %d %d`,
					schedIMTestUint64MetricName[indx],
					schedIMTestInstLbl, tc.instance,
					schedIMTestHostLbl, tc.hostname,
					schedIMTestTaskIdLbl, taskId,
					taskUint64Stats[indx], tc.promTs,
				)
				wantMetrics = append(wantMetrics, metric)
			}
		}
		if wantsTaskStatsIndex.float64Index != nil {
			taskFloat64Stats := tc.crtStats[taskId].float64Stats
			for _, indx := range wantsTaskStatsIndex.float64Index {
				metric := fmt.Sprintf(
					`%s{%s="%s",%s="%s",%s="%s"} %.6f %d`,
					schedIMTestFloat4MetricName[indx],
					schedIMTestInstLbl, tc.instance,
					schedIMTestHostLbl, tc.hostname,
					schedIMTestTaskIdLbl, taskId,
					taskFloat64Stats[indx], tc.promTs,
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

	for _, tc := range []*SchedIMTestCase{
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
			wantIndex: map[string]*SchedIMTestWantTaskStatsIndex{
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
			wantIndex: map[string]*SchedIMTestWantTaskStatsIndex{
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
			wantIndex: map[string]*SchedIMTestWantTaskStatsIndex{
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
			wantIndex: map[string]*SchedIMTestWantTaskStatsIndex{
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
			wantIndex: map[string]*SchedIMTestWantTaskStatsIndex{
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
