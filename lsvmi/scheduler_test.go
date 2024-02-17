// Tests for scheduler.go

package lsvmi

import (
	"strconv"
	"testing"
	"time"

	"github.com/eparparita/linux-stats-victoriametrics-importer/internal/testutils"
)

type SchedulerTestTimeNowFunc func() time.Time

type SchedulerNextTaskHeapTestCase struct {
	timeNowFn       SchedulerTestTimeNowFunc
	intervals       []time.Duration
	wantHeapChanged []bool
	wantOrder       []int
}

type SchedulerExecuteTestCase struct {
	intervals   []time.Duration
	runDuration time.Duration
}

type TestSchedulerTaskJob struct {
	task       *Task
	id         string
	deadlines  []time.Time
	timestamps []time.Time
}

func (job *TestSchedulerTaskJob) Execute() {
	job.timestamps = append(job.timestamps, time.Now())
	job.deadlines = append(job.deadlines, job.task.deadline)
	schedulerLog.Infof(
		"Execute task(deadline=%s, interval=%s, id=%s)",
		job.task.deadline.Format(time.RFC3339Nano), job.task.interval, job.id,
	)
}

func testSchedulerNextTaskHeap(tc *SchedulerNextTaskHeapTestCase, t *testing.T) {
	if tc.timeNowFn != nil {
		savedSchedulerTimeNowFn := schedulerTimeNowFn
		schedulerTimeNowFn = tc.timeNowFn
		defer func() { schedulerTimeNowFn = savedSchedulerTimeNowFn }()
	}

	if tc.wantHeapChanged != nil && len(tc.intervals) != len(tc.wantHeapChanged) {
		t.Fatalf(
			"len(tc.intervals) %d != %d len(tc.wantHeapChanged)",
			len(tc.intervals), len(tc.wantHeapChanged),
		)
	}
	if tc.wantOrder != nil && len(tc.intervals) != len(tc.wantOrder) {
		t.Fatalf(
			"len(tc.intervals) %d != %d len(tc.wantOrder)",
			len(tc.intervals), len(tc.wantOrder),
		)
	}

	tasks := make([]*Task, len(tc.intervals))
	taskMap := make(map[*Task]int)
	for i, interval := range tc.intervals {
		tasks[i] = NewTask(interval, nil)
		taskMap[tasks[i]] = i
	}

	heap := NewNextTaskHeap()
	gotHeapChanged := make([]bool, len(tc.intervals))
	for i, task := range tasks {
		gotHeapChanged[i] = heap.AddTask(task)
	}

	if tc.wantHeapChanged != nil {
		match := true
		for i, wantChanged := range tc.wantHeapChanged {
			if wantChanged != gotHeapChanged[i] {
				match = false
				break
			}
			if !match {
				t.Fatalf("Heap changed mismatch\nwant: %v\n got: %v", tc.wantHeapChanged, gotHeapChanged)
			}
		}
	}

	gotOrder := make([]int, len(tc.wantOrder))
	for i := 0; i < len(gotOrder); i++ {
		task := heap.PopNextTask()
		if task == nil {
			t.Fatalf("Unexpected empty heap at pop# %d", i+1)
		}
		if index, ok := taskMap[task]; ok {
			gotOrder[i] = index
		} else {
			t.Fatalf("Unexpected task(interval=%s) at pop# %d", task.interval, i+1)
		}
	}
	if task := heap.PopNextTask(); task != nil {
		t.Fatalf("Unexpected task(interval=%s), the heap should be empty", task.interval)
	}

	if tc.wantOrder != nil {
		match := true
		for i, index := range tc.wantOrder {
			if index != gotOrder[i] {
				match = false
				break
			}
		}
		if !match {
			t.Fatalf("Heap order mismatch\nwant: %v\n got: %v", tc.wantOrder, gotOrder)
		}
	}
}

func testSchedulerExecute(tc *SchedulerExecuteTestCase, t *testing.T) {
	tlc := testutils.NewTestingLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	scheduler := NewScheduler(256, 4)
	defer scheduler.Shutdown()

	scheduler.Start()

	for i, interval := range tc.intervals {
		job := &TestSchedulerTaskJob{
			id: strconv.Itoa(i),
		}
		task := NewTask(interval, job)
		job.task = task
		scheduler.AddTask(task)
	}

	time.Sleep(tc.runDuration)
}

func TestSchedulerNextTaskHeap(t *testing.T) {
	timeNowRetVal := time.Now().Truncate(
		2 * 5 * 33 * time.Second,
	)
	timeNowFn := func() time.Time { return timeNowRetVal }

	sec := 1 * time.Second

	for _, tc := range []*SchedulerNextTaskHeapTestCase{
		{
			intervals:       []time.Duration{1 * sec},
			wantHeapChanged: []bool{true},
			wantOrder:       []int{0},
		},
		{
			timeNowFn:       timeNowFn,
			intervals:       []time.Duration{2 * sec, 5 * sec},
			wantHeapChanged: []bool{true, false},
			wantOrder:       []int{0, 1},
		},
		{
			timeNowFn:       timeNowFn,
			intervals:       []time.Duration{5 * sec, 2 * sec},
			wantHeapChanged: []bool{true, true},
			wantOrder:       []int{1, 0},
		},
		{
			timeNowFn:       timeNowFn,
			intervals:       []time.Duration{2 * sec, 5 * sec, 33 * sec},
			wantHeapChanged: []bool{true, false, false},
			wantOrder:       []int{0, 1, 2},
		},
		{
			timeNowFn:       timeNowFn,
			intervals:       []time.Duration{5 * sec, 2 * sec, 33 * sec},
			wantHeapChanged: []bool{true, true, false},
			wantOrder:       []int{1, 0, 2},
		},
		{
			timeNowFn:       timeNowFn,
			intervals:       []time.Duration{5 * sec, 33 * sec, 2 * sec},
			wantHeapChanged: []bool{true, true, true},
			wantOrder:       []int{2, 0, 1},
		},
	} {
		t.Run(
			"",
			func(t *testing.T) { testSchedulerNextTaskHeap(tc, t) },
		)
	}
}

func TestSchedulerExecute(t *testing.T) {
	timeUnit := 100 * time.Millisecond

	for _, tc := range []*SchedulerExecuteTestCase{
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
