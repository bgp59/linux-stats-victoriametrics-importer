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
//s
//             +----------------+
//             | Next Task Heap |
//             +----------------+
//                     ^
//                     | task
//                     v
//             +----------------+
//             |   Dispatcher   |
//             +----------------+
//               ^            | task
//               | task       v
//         +----------+  +----------+
//         | Task Que |  | TODO Que |
//         +----------+  +----------+
//            ^  ^            |
//   new task |  |            |
//   ---------+  |  +---------+---- ... ----+
//              /   | task    | task        | task
//          +--+    v         v             v
//          |  +--------+ +--------+   +--------+
//          |  | Worker | | Worker |...| Worker |
//          |  +--------+ +--------+   +--------+
//          |       |         |             |
//          +-------+---------+----- ... ---+
//
//  Principles Of Operation
//  =======================
//
// The order of execution is set by the Next Task Heap, which is a min heap
// sorted by the task's deadline (i.e. the nearest deadline is at the top).
//
// The Dispatcher monitors the top of the Next Task Heap, waiting for the
// deadline. When the latter arrives, it pulls the task from heap and it adds it
// to the TODO Queue.
//
// The TODO Queue feeds the Worker Pool; the number of workers in the pool
// controls the level of concurrency of task execution and it allows for short
// tasks to be executed without having to wait for a long one to complete.
//
// A Worker will pull the next task from the TODO Queue, it will execute it and
// it will update the task's deadline for the next execution. The task is then
// added to the Task Queue.
//
// The Dispatcher pulls from the Task Queue a new/re added task and it inserts
// it into the heap. If the top of the latter changes (i.e. it added a task with
// a nearer deadline) then the Dispatcher reevaluates the wait for the next
// deadline.

import (
	"container/heap"
	"context"
	"sync"
	"time"
)

type TaskJob interface {
	Execute()
}

type Task struct {
	// Deadline:
	deadline time.Time
	// Interval:
	interval time.Duration
	// Job:
	job TaskJob
}

type NextTaskHeap struct {
	tasks []*Task
}

type Scheduler struct {
	// Next Task Heap:
	heap *NextTaskHeap
	// The task and TDOO queues:
	taskQ, todoQ chan *Task
	// The number of workers:
	numWorkers int
	// The state of the scheduler, whether it is running or not:
	state   int
	stateMu *sync.Mutex
	// The apparatus needed for clean shutdown:
	ctx      context.Context
	cancelFn context.CancelFunc
	wg       *sync.WaitGroup
}

var schedulerDummyDeadline = time.Now()

const (
	SCHEDULER_STATE_CREATED = iota
	SCHEDULER_STATE_RUNNING
	SCHEDULER_STATE_STOPPED
)

var schedulerStateMap = map[int]string{
	SCHEDULER_STATE_CREATED: "Created",
	SCHEDULER_STATE_RUNNING: "Running",
	SCHEDULER_STATE_STOPPED: "Stopped",
}

// Make time functions mockable for test purposes:
var schedulerTimeNowFn = time.Now

var schedulerLog = NewCompLogger("scheduler")

func NewTask(interval time.Duration, job TaskJob) *Task {
	return &Task{
		interval: interval,
		job:      job,
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

// Add a task to the heap. Return true if heap top was changed.
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

// Return (deadline, valid) pair:
func (h *NextTaskHeap) PeekNextDeadline() (time.Time, bool) {
	if len(h.tasks) > 0 {
		return h.tasks[0].deadline, true
	}
	return schedulerDummyDeadline, false
}

func (h *NextTaskHeap) PopNextTask() *Task {
	if len(h.tasks) > 0 {
		return heap.Pop(h).(*Task)
	}
	return nil
}

func NewScheduler(queLen, numWorkers int) *Scheduler {
	ctx, cancelFn := context.WithCancel(context.Background())
	return &Scheduler{
		heap:       NewNextTaskHeap(),
		taskQ:      make(chan *Task, queLen),
		todoQ:      make(chan *Task, queLen),
		numWorkers: numWorkers,
		state:      SCHEDULER_STATE_CREATED,
		stateMu:    &sync.Mutex{},
		ctx:        ctx,
		cancelFn:   cancelFn,
		wg:         &sync.WaitGroup{},
	}
}

func (scheduler *Scheduler) dispatcherLoop() {
	schedulerLog.Info("start dispatcher loop")

	defer scheduler.wg.Done()

	timer := time.NewTimer(1 * time.Hour)
	if !timer.Stop() {
		<-timer.C
	}
	activeTimer := false

	defer func() {
		schedulerLog.Info("stop dispatcher loop")
		if activeTimer && !timer.Stop() {
			<-timer.C
		}
		schedulerLog.Info("dispatcher stopped")
	}()

	for {
		if !activeTimer {
			deadline, valid := scheduler.heap.PeekNextDeadline()
			if valid {
				timer.Reset(time.Until(deadline))
				activeTimer = true
			}
		}

		select {
		case <-scheduler.ctx.Done():
			return
		case task := <-scheduler.taskQ:
			heapChanged := scheduler.heap.AddTask(task)
			if heapChanged && activeTimer {
				if !timer.Stop() {
					<-timer.C
				}
				activeTimer = false
			}
		case <-timer.C:
			activeTimer = false
			scheduler.todoQ <- scheduler.heap.PopNextTask()
		}
	}
}

func (scheduler *Scheduler) workerLoop(workerId int) {
	schedulerLog.Infof("start worker# %d", workerId)

	defer func() {
		schedulerLog.Infof("stop worker# %d", workerId)
		scheduler.wg.Done()
	}()

	var (
		task   *Task
		isOpen bool
	)
	for {
		select {
		case <-scheduler.ctx.Done():
			return
		case task, isOpen = <-scheduler.todoQ:
			if !isOpen {
				return
			}
		}
		if task.job != nil {
			task.job.Execute()
		}
		select {
		case <-scheduler.ctx.Done():
			return
		case scheduler.taskQ <- task:
		}
	}
}

func (scheduler *Scheduler) Start() {
	scheduler.stateMu.Lock()
	defer scheduler.stateMu.Unlock()

	if scheduler.state != SCHEDULER_STATE_CREATED {
		schedulerLog.Warnf(
			"scheduler can only be started from state %d '%s', not from %d '%s'",
			SCHEDULER_STATE_CREATED, schedulerStateMap[SCHEDULER_STATE_CREATED],
			scheduler.state, schedulerStateMap[scheduler.state],
		)
		return
	}

	schedulerLog.Info("start scheduler")

	scheduler.wg.Add(1)
	go scheduler.dispatcherLoop()

	for workerId := 0; workerId < scheduler.numWorkers; workerId++ {
		scheduler.wg.Add(1)
		go scheduler.workerLoop(workerId)
	}

	scheduler.state = SCHEDULER_STATE_RUNNING
	schedulerLog.Info("scheduler started")
}

func (scheduler *Scheduler) Shutdown() {
	scheduler.stateMu.Lock()
	defer scheduler.stateMu.Unlock()

	if scheduler.state == SCHEDULER_STATE_STOPPED {
		schedulerLog.Warnf(
			"scheduler already in state %d '%s'",
			SCHEDULER_STATE_STOPPED, schedulerStateMap[SCHEDULER_STATE_STOPPED],
		)
		return
	}

	schedulerLog.Info("stop scheduler")

	scheduler.cancelFn()
	scheduler.wg.Wait()

	scheduler.state = SCHEDULER_STATE_STOPPED
	schedulerLog.Info("scheduler stopped")
}

func (scheduler *Scheduler) AddTask(task *Task) {
	select {
	case <-scheduler.ctx.Done():
	case scheduler.taskQ <- task:
	}
}
