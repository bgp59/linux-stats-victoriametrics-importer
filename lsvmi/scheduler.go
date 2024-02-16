// Scheduler for metrics generation.

package lsvmi

//  Task Definition
//  ===============
//
// For the purpose of scheduling, each metrics generator is a periodic task
// (task for short).
//
// The task attributes relevant for scheduling:
//  - the interval by which it is to be repeated
//  - the next deadline
//
//  Scheduler Architecture
//  ======================
//
//                   +----------------+        task
//         +---------| Next Task Heap | <-------------------+
//         | task    +----------------+                     |
//         V                  notification             +----------+
//   +------------+    +-------------------------------| Add Task |
//   | Dispatcher | <--+                               +----------+
//   +------------+                     +-------------+    ^  ^
//         | task   +------------+ task | Worker Pool |    |  |
//         +------->|    TODO    |--------> Worker --------+  |
//                  |    Queue   |      |   Worker    |       |
//                  +------------+      +-------------+    new task
//
//  Principles Of Operation
//  =======================
//
// The order of execution is set by the Next Task Heap, which is a min heap
// sorted by the task's deadline; the nearest deadline is as the top.
//
// The dispatcher monitors the top of the Next Task Heap, waiting for the
// deadline. When the latter arrives, it pulls the task from heap and it adds it
// to the TODO Queue.
//
// The TODO Queue feeds the Worker Pool; the number of workers in the pool
// controls the level of concurrency of task execution and it allows for short
// tasks to be executed without having to wait for a long one to complete.
//
// A Worker will pull the next task from the TODO Queue, it will execute it and
// it will update the task's deadline for the next execution. The task is then
// added to the Next Task Heap.
//
// Add Task is responsible for (re-)adding tasks to the Next Task Heap. After
// each addition the former checks if it changed the top of the latter (i.e. it
// added a task with a nearer deadline). If that is the case then it notifies
// the dispatcher to reevaluate the wait for the next deadline.

import (
	"container/heap"
	"time"
)

type Task struct {
	// Deadline:
	deadline time.Time
	// Interval:
	interval time.Duration
}

type NextTaskHeap struct {
	tasks []*Task
}

var SchedulerDummyDeadline = time.Now()

// Make time functions mockable for test purposes:
var schedulerTimeNowFn = time.Now

func NewTask(interval time.Duration) *Task {
	return &Task{
		interval: interval,
	}
}

func NewNextTaskHeap() *NextTaskHeap {
	return &NextTaskHeap{
		tasks: make([]*Task, 0),
	}
}

// sort.Interface:
func (h *NextTaskHeap) Len() int {
	return len(h.tasks)
}

func (h *NextTaskHeap) Less(i, j int) bool {
	return h.tasks[i].deadline.Before(h.tasks[j].deadline)
}

func (h *NextTaskHeap) Swap(i, j int) {
	h.tasks[i], h.tasks[j] = h.tasks[j], h.tasks[i]
}

// heap.Interface:
func (h *NextTaskHeap) Push(x any) {
	if task, ok := x.(*Task); ok {
		h.tasks = append(h.tasks, task)
	}
}

func (h *NextTaskHeap) Pop() any {
	newLen := len(h.tasks) - 1
	task := h.tasks[newLen]
	h.tasks = h.tasks[:newLen]
	return task
}

// Add a new/existent task to the heap. Return true if heap top was changed.
func (h *NextTaskHeap) AddTask(task *Task) bool {
	// The deadline is the nearest future multiple of task interval:
	task.deadline = schedulerTimeNowFn().Truncate(task.interval).Add(task.interval)

	// The top of heap changes if either the heap was empty or the new deadline
	// is more recent:
	hasChanged := len(h.tasks) == 0 ||
		task.deadline.Before(h.tasks[0].deadline)
	heap.Push(h, task)
	return hasChanged
}

// Return deadline, valid pair:
func (h *NextTaskHeap) PeekNextDeadline() (time.Time, bool) {
	if len(h.tasks) > 0 {
		return h.tasks[0].deadline, true
	}
	return SchedulerDummyDeadline, false
}

func (h *NextTaskHeap) PopNextTask() *Task {
	if len(h.tasks) > 0 {
		return heap.Pop(h).(*Task)
	}
	return nil
}
