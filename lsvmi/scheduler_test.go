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
	numWorkers  int
	intervals   []time.Duration
	runDuration time.Duration
	// The scheduled intervals will be checked against the desired one and they
	// should be in the range of:
	//  (1 - scheduleIntervalPct/100)*interval .. (1 + scheduleIntervalPct/100)*interval
	// Set to 0 to disable:
	scheduleIntervalPct float64
}

type TestSchedulerTaskAction struct {
	task       *Task
	timestamps []time.Time
}

func (action *TestSchedulerTaskAction) Execute() {
	action.timestamps = append(action.timestamps, time.Now())
	schedulerLog.Infof(
		"Execute task %s: interval: %s, next deadline: %s",
		action.task.id, action.task.interval, action.task.deadline.Format(time.RFC3339Nano),
	)
	//time.Sleep(2 * action.task.interval) // force over-scheduling
}

func testSchedulerExecute(tc *SchedulerExecuteTestCase, t *testing.T) {
	tlc := testutils.NewTestingLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	numWorkers := tc.numWorkers
	if numWorkers <= 0 {
		numWorkers = 1
	}
	scheduler := NewScheduler(numWorkers)
	scheduler.Start()

	tasks := make([]*Task, len(tc.intervals))
	for i, interval := range tc.intervals {
		action := &TestSchedulerTaskAction{}
		task := NewTask(strconv.Itoa(i), interval, action)
		action.task = task
		scheduler.AddTask(task)
		tasks[i] = task
	}

	time.Sleep(tc.runDuration)
	scheduler.Shutdown()

	// Verify that each task was scheduled roughly at the expected intervals and
	// that it wasn't skipped:

	errBuf := &bytes.Buffer{}

	stats := scheduler.SnapStats(nil)
	for _, task := range tasks {
		taskStats := stats[task.id]
		if taskStats == nil {
			fmt.Fprintf(errBuf, "\n task %s: missing stats", task.id)
			continue
		} else if taskStats.uint64Stats[TASK_STATS_OVER_SCHEDULED_COUNT] > 0 {
			fmt.Fprintf(
				errBuf, "\n task %s: over/scheduled %d/%d time(s)",
				task.id,
				taskStats.uint64Stats[TASK_STATS_OVER_SCHEDULED_COUNT],
				taskStats.uint64Stats[TASK_STATS_SCHEDULED_COUNT],
			)
			continue
		}
		if tc.scheduleIntervalPct <= 0 || tc.scheduleIntervalPct > 100 {
			continue
		}
		pct := tc.scheduleIntervalPct / 100.
		intervalSec := task.interval.Seconds()
		minIntervalSec := (1 - pct) * intervalSec
		maxIntervalSec := (1 + pct) * intervalSec

		timestamps := task.action.(*TestSchedulerTaskAction).timestamps
		// timestamp#0 -> #1 may be irregular, but everything #(k-1) -> #k, k >=
		// 2, should be checked:
		for k := 2; k < len(timestamps); k++ {
			gotIntervalSec := timestamps[k].Sub(timestamps[k-1]).Seconds()
			if gotIntervalSec < minIntervalSec || maxIntervalSec < gotIntervalSec {
				fmt.Fprintf(
					errBuf,
					"\ntask %s interval# %d: want: %.06f..%.06f, got: %.06f sec",
					task.id, k, minIntervalSec, maxIntervalSec, gotIntervalSec,
				)
			}
		}
	}
	if errBuf.Len() > 0 {
		tlc.Fatal(errBuf)
	}

}

func TestSchedulerExecute(t *testing.T) {
	timeUnit := 100 * time.Millisecond
	scheduleIntervalPct := 10.

	for _, tc := range []*SchedulerExecuteTestCase{
		{
			intervals: []time.Duration{
				1 * timeUnit,
			},
			runDuration:         10 * timeUnit,
			scheduleIntervalPct: scheduleIntervalPct,
		},
		{
			intervals: []time.Duration{
				4 * timeUnit, 7 * timeUnit, 3 * timeUnit, 5 * timeUnit, 1 * timeUnit,
			},
			runDuration:         43 * timeUnit,
			scheduleIntervalPct: scheduleIntervalPct,
		},
		{
			numWorkers: 4,
			intervals: []time.Duration{
				4 * timeUnit, 7 * timeUnit, 3 * timeUnit, 5 * timeUnit, 1 * timeUnit,
			},
			runDuration:         43 * timeUnit,
			scheduleIntervalPct: scheduleIntervalPct,
		},
	} {
		t.Run(
			"",
			func(t *testing.T) { testSchedulerExecute(tc, t) },
		)
	}
}
