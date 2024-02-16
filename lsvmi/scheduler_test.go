// Tests for scheduler.go

package lsvmi

import (
	"testing"
	"time"
)

type SchedulerTestTimeNowFunc func() time.Time

type NextTaskHeapTestCase struct {
	timeNowFn       SchedulerTestTimeNowFunc
	intervals       []string
	wantHeapChanged []bool
	wantOrder       []int
}

func testNextTaskHeap(tc *NextTaskHeapTestCase, t *testing.T) {
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
	for i, intervalSpec := range tc.intervals {
		interval, err := time.ParseDuration(intervalSpec)
		if err != nil {
			t.Fatal(err)
		}
		tasks[i] = NewTask(interval)
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

func TestNextTaskHeap(t *testing.T) {
	timeNowRetVal := time.Now().Truncate(
		2 * 5 * 33 * time.Second,
	)
	timeNowFn := func() time.Time { return timeNowRetVal }

	for _, tc := range []*NextTaskHeapTestCase{
		{
			intervals:       []string{"1s"},
			wantHeapChanged: []bool{true},
			wantOrder:       []int{0},
		},
		{
			timeNowFn:       timeNowFn,
			intervals:       []string{"2s", "5s"},
			wantHeapChanged: []bool{true, false},
			wantOrder:       []int{0, 1},
		},
		{
			timeNowFn:       timeNowFn,
			intervals:       []string{"5s", "2s"},
			wantHeapChanged: []bool{true, true},
			wantOrder:       []int{1, 0},
		},
		{
			timeNowFn:       timeNowFn,
			intervals:       []string{"2s", "5s", "33s"},
			wantHeapChanged: []bool{true, false, false},
			wantOrder:       []int{0, 1, 2},
		},
		{
			timeNowFn:       timeNowFn,
			intervals:       []string{"5s", "2s", "33s"},
			wantHeapChanged: []bool{true, true, false},
			wantOrder:       []int{1, 0, 2},
		},
		{
			timeNowFn:       timeNowFn,
			intervals:       []string{"5s", "33s", "2s"},
			wantHeapChanged: []bool{true, true, true},
			wantOrder:       []int{2, 0, 1},
		},
	} {
		t.Run(
			"",
			func(t *testing.T) { testNextTaskHeap(tc, t) },
		)
	}
}
