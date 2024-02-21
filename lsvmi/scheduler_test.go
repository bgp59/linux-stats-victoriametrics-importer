// Tests for scheduler.go

package lsvmi

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/eparparita/linux-stats-victoriametrics-importer/internal/testutils"
)

type SchedulerExecuteTestCase struct {
	numWorkers int
	// The unit for intervals, execTimes and runTime:
	timeUnitSec float64
	// Set one task for each interval:
	intervals []float64
	// The execution times for each:
	execTimes [][]float64
	// How long to run the scheduler for:
	runTime float64
	// The scheduled intervals will be checked against the desired one and they
	// should be in the range of:
	//  (1 - scheduleIntervalPct/100)*interval .. (1 + scheduleIntervalPct/100)*interval
	// Use -1 to disable.
	scheduleIntervalPct float64
	// The maximum allowed number of irregular scheduling intervals, as
	// determined by the above:
	wantIrregularIntervalMaxCount []int
}

type TestSchedulerTaskAction struct {
	task         *Task
	execTimes    []time.Duration
	execTimeIndx int
	timestamps   []time.Time
}

func (action *TestSchedulerTaskAction) Execute() {
	action.timestamps = append(action.timestamps, time.Now())
	schedulerLog.Infof(
		"Execute task %s: interval=%s, deadline=%s",
		action.task.id, action.task.interval, action.task.deadline.Format(time.RFC3339Nano),
	)
	n := len(action.execTimes)
	if n > 0 {
		time.Sleep(action.execTimes[action.execTimeIndx])
		action.execTimeIndx++
		if action.execTimeIndx >= n {
			action.execTimeIndx = 0
		}
	}
}

func testSchedulerDurationFromSec(sec float64) time.Duration {
	return time.Duration(
		sec * float64(time.Second),
	)
}

func testSchedulerBuildTaskList(tc *SchedulerExecuteTestCase) []*Task {
	tasks := make([]*Task, len(tc.intervals))
	for i, interval := range tc.intervals {
		action := &TestSchedulerTaskAction{}
		task := NewTask(strconv.Itoa(i), testSchedulerDurationFromSec(interval*tc.timeUnitSec), action)
		action.task = task
		if tc.execTimes != nil {
			execTimes := tc.execTimes[i]
			if execTimes != nil {
				action.execTimes = make([]time.Duration, len(execTimes))
				for k, execTime := range execTimes {
					action.execTimes[k] = testSchedulerDurationFromSec(execTime * tc.timeUnitSec)
				}
			}
		}
		tasks[i] = task
	}
	return tasks
}

func testSchedulerExecute(tc *SchedulerExecuteTestCase, t *testing.T) {
	tlc := testutils.NewTestingLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	numWorkers := tc.numWorkers
	if numWorkers <= 0 {
		numWorkers = 1
	}
	scheduler, err := NewScheduler(&SchedulerConfig{NumWorkers: tc.numWorkers})
	if err != nil {
		tlc.Fatal(err)
	}
	scheduler.Start()
	tasks := testSchedulerBuildTaskList(tc)
	for _, task := range tasks {
		scheduler.AddNewTask(task)
	}
	time.Sleep(testSchedulerDurationFromSec(tc.runTime * tc.timeUnitSec))
	scheduler.Shutdown()

	// Verify that each task was scheduled roughly at the expected intervals and
	// that it wasn't skipped:

	errBuf := &bytes.Buffer{}

	stats := scheduler.SnapStats(nil, STATS_SNAP_ONLY)

	type IrregularInterval struct {
		k        int
		interval float64
	}
	for i, task := range tasks {
		taskStats := stats[task.id]
		if taskStats == nil {
			fmt.Fprintf(errBuf, "\n task %s: missing stats", task.id)
			continue
		}
		pct := tc.scheduleIntervalPct / 100.
		intervalSec := task.interval.Seconds()
		minIntervalSec := (1 - pct) * intervalSec
		maxIntervalSec := (1 + pct) * intervalSec

		timestamps := task.action.(*TestSchedulerTaskAction).timestamps
		// timestamp#0 -> #1 may be irregular, but everything #(k-1) -> #k, k >=
		// 2, should be checked:
		irregularIntervals := make([]*IrregularInterval, 0)
		for k := 2; k < len(timestamps); k++ {
			gotIntervalSec := timestamps[k].Sub(timestamps[k-1]).Seconds()
			if gotIntervalSec < minIntervalSec || maxIntervalSec < gotIntervalSec {
				irregularIntervals = append(
					irregularIntervals,
					&IrregularInterval{k, gotIntervalSec},
				)
			}
		}
		wantIrregularIntervalMaxCount := 0
		if tc.wantIrregularIntervalMaxCount != nil {
			wantIrregularIntervalMaxCount = tc.wantIrregularIntervalMaxCount[i]
		}
		if len(irregularIntervals) > wantIrregularIntervalMaxCount {
			for _, irregularInterval := range irregularIntervals {
				fmt.Fprintf(
					errBuf,
					"\ntask %s execute# %d: want: %.06f..%.06f, got: %.06f sec from previous execution",
					task.id, irregularInterval.k,
					minIntervalSec, maxIntervalSec, irregularInterval.interval,
				)
			}
		}
		if taskStats.uint64Stats[TASK_STATS_OVERRUN_COUNT] > uint64(wantIrregularIntervalMaxCount) {
			fmt.Fprintf(
				errBuf,
				"\ntask %s TASK_STATS_OVERRUN_COUNT: want max: %d, got: %d",
				task.id, wantIrregularIntervalMaxCount,
				taskStats.uint64Stats[TASK_STATS_OVERRUN_COUNT],
			)
		}

	}

	if errBuf.Len() > 0 {
		tlc.Fatal(errBuf)
	}

}

func TestSchedulerExecute(t *testing.T) {
	scheduleIntervalPct := 10.

	for _, tc := range []*SchedulerExecuteTestCase{
		{
			numWorkers:  1,
			timeUnitSec: .1,
			intervals: []float64{
				1,
			},
			runTime:             40,
			scheduleIntervalPct: scheduleIntervalPct,
		},
		{
			numWorkers:  1,
			timeUnitSec: .1,
			intervals: []float64{
				4, 7, 3, 5, 1,
			},
			runTime:             43,
			scheduleIntervalPct: scheduleIntervalPct,
		},
		{
			numWorkers:  5,
			timeUnitSec: .1,
			intervals: []float64{
				4, 7, 3, 5, 1,
			},
			runTime:             43,
			scheduleIntervalPct: scheduleIntervalPct,
		},
		{
			numWorkers:  5,
			timeUnitSec: .1,
			intervals: []float64{
				4,
				7,
				3,
				5,
				1,
			},
			execTimes: [][]float64{
				{3},
				{6},
				{2},
				{4},
				nil,
			},
			runTime:             43,
			scheduleIntervalPct: scheduleIntervalPct,
		},
		{
			numWorkers:  5,
			timeUnitSec: .1,
			intervals: []float64{
				4,
				7,
				3,
				5,
				1,
			},
			execTimes: [][]float64{
				{3, 5, 3, 3, 3, 3},
				{6},
				{2},
				{4},
				nil,
			},
			runTime:             43,
			scheduleIntervalPct: scheduleIntervalPct,
			wantIrregularIntervalMaxCount: []int{
				2,
				0,
				0,
				0,
				0,
			},
		},
	} {
		t.Run(
			"",
			func(t *testing.T) { testSchedulerExecute(tc, t) },
		)
	}
}
