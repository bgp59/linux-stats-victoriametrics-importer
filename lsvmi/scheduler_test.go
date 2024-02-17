// Tests for scheduler.go

package lsvmi

import (
	"strconv"
	"testing"
	"time"

	"github.com/eparparita/linux-stats-victoriametrics-importer/internal/testutils"
)

type SchedulerExecuteTestCase struct {
	intervals   []time.Duration
	runDuration time.Duration
}

type TestSchedulerTaskAction struct {
	taskId     string
	task       *Task
	timestamps []time.Time
}

func (action *TestSchedulerTaskAction) Execute() {
	action.timestamps = append(action.timestamps, time.Now())
	schedulerLog.Infof(
		"Execute task %s: interval: %s, next deadline: %s",
		action.taskId, action.task.interval, action.task.deadline.Format(time.RFC3339Nano),
	)
}

func testSchedulerExecute(tc *SchedulerExecuteTestCase, t *testing.T) {
	tlc := testutils.NewTestingLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	scheduler := NewScheduler(4)
	scheduler.Start()

	for i, interval := range tc.intervals {
		action := &TestSchedulerTaskAction{
			taskId: strconv.Itoa(i),
		}
		task := NewTask(interval, action)
		action.task = task
		scheduler.AddTask(task)
	}

	time.Sleep(tc.runDuration)
	scheduler.Shutdown()

}

func TestSchedulerExecute(t *testing.T) {
	timeUnit := 100 * time.Millisecond

	for _, tc := range []*SchedulerExecuteTestCase{
		{
			intervals: []time.Duration{
				1 * timeUnit,
			},
			runDuration: 10 * timeUnit,
		},
		{
			intervals: []time.Duration{
				7 * timeUnit, 3 * timeUnit, 5 * timeUnit, 1 * timeUnit,
			},
			runDuration: 43 * timeUnit,
		},
	} {
		t.Run(
			"",
			func(t *testing.T) { testSchedulerExecute(tc, t) },
		)
	}
}
