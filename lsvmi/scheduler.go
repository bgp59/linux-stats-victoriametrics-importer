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
//             +------------------+
//             |  Next Task Heap  |
//             +------------------+
//                       ^
//                       | task
//                       v
//             +------------------+
//             |     Dispatcher   |
//             +------------------+
//               ^              | task
//               | task         v
//        +------------+ +------------+
//        | Task Queue | | TODO Queue |
//        +------------+ +------------+
//              ^               |
//     new task |               |
//                    +---------+---- ... ----+
//                    | task    | task        | task
//                    v         v             v
//              +--------+ +--------+   +--------+
//              | Worker | | Worker |...| Worker |
//              +--------+ +--------+   +--------+
//
//  Principles Of Operation
//  =======================
//
// The order of execution is set by the Next Task Heap, which is a min heap
// sorted by the task's deadline (i.e. the nearest deadline is at the top).
//
// The Dispatcher maintains a timer for the next deadline based on the top of
// the heap and it also monitors the Task Queue for new additions. Whichever
// comes first, it determines the next task to run. The latter's deadlines is
// updated and, if a new task, it is also added to heap; if not then the heap is
// updated in place. The task is marked as scheduled and it is placed into the
// TODO Queue.
//
// The TODO Queue feeds the Worker Pool; the number of workers in the pool
// controls the level of concurrency of task execution and it allows for short
// tasks to be executed without having to wait for a long one to complete.
//
// A Worker will pull the next task from the TODO Queue, it will execute it and
// it will then clear the scheduled flag.
//
// The flag prevents a task from being scheduled while pending execution;
// normally, with enough resources, this should never happen.

import (
	"container/heap"
	"sync"
	"time"
)

const (
	SCHEDULER_TASK_Q_LEN = 1
	SCHEDULER_TODO_Q_LEN = 64
)

const (
	SCHEDULER_STATE_CREATED = iota
	SCHEDULER_STATE_RUNNING
	SCHEDULER_STATE_STOPPED
)

type TaskAction interface {
	Execute()
}

type Task struct {
	// Deadline:
	deadline time.Time
	// Interval:
	interval time.Duration
	// Action:
	action TaskAction
	// Whether it is currently scheduled or not:
	scheduled bool
}

type Scheduler struct {
	// Next Task Heap:
	tasks []*Task
	// The task and TDOO queues:
	taskQ, todoQ chan *Task
	// The number of workers:
	numWorkers int
	// The state of the scheduler, whether it is running or not:
	state SchedulerState
	// General purpose lock for atomic operations: check task `scheduled` flag,
	// scheduler's `state`, etc. The lock is shared because the contention is
	// minimal, it doesn't make sense to use individual lock.
	mu *sync.Mutex
	// Goroutines exit sync:
	wg *sync.WaitGroup
}

type SchedulerState int

var schedulerStateMap = map[SchedulerState]string{
	SCHEDULER_STATE_CREATED: "Created",
	SCHEDULER_STATE_RUNNING: "Running",
	SCHEDULER_STATE_STOPPED: "Stopped",
}

func (s SchedulerState) String() string {
	return schedulerStateMap[s]
}

var schedulerLog = NewCompLogger("scheduler")

func NewTask(interval time.Duration, action TaskAction) *Task {
	return &Task{
		interval: interval,
		action:   action,
	}
}

func NewScheduler(numWorkers int) *Scheduler {
	return &Scheduler{
		tasks:      make([]*Task, 0),
		taskQ:      make(chan *Task, SCHEDULER_TASK_Q_LEN),
		todoQ:      make(chan *Task, SCHEDULER_TODO_Q_LEN),
		numWorkers: numWorkers,
		state:      SCHEDULER_STATE_CREATED,
		mu:         &sync.Mutex{},
		wg:         &sync.WaitGroup{},
	}
}

// The scheduler should be a heap, so define the expected interfaces:

// sort.Interface:
func (scheduler *Scheduler) Len() int {
	return len(scheduler.tasks)
}

func (scheduler *Scheduler) Less(i, j int) bool {
	return scheduler.tasks[i].deadline.Before(scheduler.tasks[j].deadline)
}

func (scheduler *Scheduler) Swap(i, j int) {
	scheduler.tasks[i], scheduler.tasks[j] = scheduler.tasks[j], scheduler.tasks[i]
}

// heap.Interface:
func (scheduler *Scheduler) Push(x any) {
	if task, ok := x.(*Task); ok {
		scheduler.tasks = append(scheduler.tasks, task)
	}
}

func (scheduler *Scheduler) Pop() any {
	newLen := len(scheduler.tasks) - 1
	task := scheduler.tasks[newLen]
	scheduler.tasks = scheduler.tasks[:newLen]
	return task
}

// Add a new task:
func (scheduler *Scheduler) AddTask(task *Task) {
	scheduler.taskQ <- task
}

func (scheduler *Scheduler) dispatcherLoop() {
	schedulerLog.Info("start dispatcher loop")

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
		schedulerLog.Info("close TODO Queue")
		close(scheduler.todoQ)
		schedulerLog.Info("dispatcher stopped")
		scheduler.wg.Done()
	}()

	var (
		task   *Task
		isOpen bool
	)

	for {
		if !activeTimer && len(scheduler.tasks) > 0 {
			timer.Reset(time.Until(scheduler.tasks[0].deadline))
			activeTimer = true
		}

		select {
		case task, isOpen = <-scheduler.taskQ:
			if !isOpen {
				return
			}
			// Add the task to the heap, with the deadline set to the nearest
			// future multiple of interval:
			task.deadline = time.Now().Truncate(task.interval).Add(task.interval)
			heap.Push(scheduler, task)
			// Any other pending timer is no longer applicable:
			if activeTimer {
				if !timer.Stop() {
					<-timer.C
				}
				activeTimer = false
			}
		case <-timer.C:
			activeTimer = false
			task = scheduler.tasks[0]
			// Update its deadline to be set to the nearest future multiple of
			// interval:
			task.deadline = time.Now().Truncate(task.interval).Add(task.interval)
			heap.Fix(scheduler, 0)
		}

		if task != nil {
			scheduler.mu.Lock()
			queueIt := !task.scheduled
			if queueIt {
				task.scheduled = true
			}
			scheduler.mu.Unlock()
			if queueIt {
				scheduler.todoQ <- task
			}
		}
	}
}

func (scheduler *Scheduler) workerLoop(workerId int) {
	schedulerLog.Infof("start worker# %d", workerId)

	defer func() {
		schedulerLog.Infof("worker# %d stopped", workerId)
		scheduler.wg.Done()
	}()

	for {
		task, isOpen := <-scheduler.todoQ
		if !isOpen {
			return
		}
		if task.action != nil {
			task.action.Execute()
		}
		scheduler.mu.Lock()
		task.scheduled = false
		scheduler.mu.Unlock()
	}
}

func (scheduler *Scheduler) Start() {
	scheduler.mu.Lock()
	defer scheduler.mu.Unlock()

	if scheduler.state != SCHEDULER_STATE_CREATED {
		schedulerLog.Warnf(
			"scheduler can only be started from state %d '%s', not from %d '%s'",
			SCHEDULER_STATE_CREATED, SchedulerState(SCHEDULER_STATE_CREATED),
			scheduler.state, scheduler.state,
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
	scheduler.mu.Lock()
	defer scheduler.mu.Unlock()

	if scheduler.state == SCHEDULER_STATE_STOPPED {
		schedulerLog.Warnf(
			"scheduler already in state %d '%s'",
			scheduler.state, scheduler.state,
		)
		return
	}

	schedulerLog.Info("stop scheduler")
	close(scheduler.taskQ)
	scheduler.wg.Wait()

	scheduler.state = SCHEDULER_STATE_STOPPED
	schedulerLog.Info("scheduler stopped")
}
