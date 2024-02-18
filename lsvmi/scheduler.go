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
//                   +----------+--- ... ----+
//                   | task     | task       | task
//                   v          v            v
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
// the heap and it also monitors the Task Queue for new additions, whichever
// comes first. Based on those, it selects the next task to run. The latter's
// deadlines is updated and, if a new task, it is also added to heap; if not
// then the heap is updated in place. Each task has a flag indicating whether it
// is currently being scheduled or not. This flag prevents a task from being
// scheduled while pending execution; normally, with enough resources, this
// should never happen. If not scheduled then the task is marked as scheduled
// and it is placed into the TODO Queue.
//
// The TODO Queue feeds the Worker Pool; the number of workers in the pool
// controls the level of concurrency of task execution and it allows for short
// tasks to be executed without having to wait for a long one to complete.
//
// A Worker will pull the next task from the TODO Queue, it will execute it and
// it will then clear the scheduled flag.

import (
	"container/heap"
	"fmt"
	"sync"
	"time"

	"github.com/eparparita/linux-stats-victoriametrics-importer/internal/utils"
)

const (
	SCHEDULER_CONFIG_NUM_WORKERS_DEFAULT = -1
	SCHEDULER_CONFIG_NUM_WORKERS_MAX     = 4
)

const (
	SCHEDULER_TASK_Q_LEN = 1
	SCHEDULER_TODO_Q_LEN = 64
)

const (
	// Indexes into Scheduler.stats.[id].uint64Stats

	// How many times the task was scheduled, indexed by Task.id:
	TASK_STATS_SCHEDULED_COUNT = iota

	// How many time the task was not queued for execution because it was
	// pending a previously scheduled one, indexed by Task.id:
	TASK_STATS_OVER_SCHEDULED_COUNT

	// Must be last:
	TASK_STATS_UINT64_LEN
)

const (
	// Indexes into Scheduler.stats.[id].float64Stats

	// Total run time, in seconds:
	TASK_STATS_RUNTIME_SEC = iota

	// Must be last:
	TASK_STATS_FLOAT64_LEN
)

type TaskAction interface {
	Execute()
}

type Task struct {
	// Id, used for stats:
	id string
	// Deadline:
	deadline time.Time
	// Interval:
	interval time.Duration
	// Action:
	action TaskAction
	// Whether it is currently scheduled or not:
	scheduled bool
}

type TaskStats struct {
	uint64Stats  []uint64
	float64Stats []float64
}

type SchedulerStats map[string]*TaskStats

type Scheduler struct {
	// Next Task Heap:
	tasks []*Task
	// The task and TDOO queues:
	taskQ, todoQ chan *Task
	// The number of workers:
	numWorkers int
	// The state of the scheduler, whether it is running or not:
	state SchedulerState
	// Stats:
	stats SchedulerStats
	// General purpose lock for atomic operations: check task `scheduled` flag,
	// scheduler's `state`, etc. The lock is shared because the contention is
	// minimal, it doesn't make sense to use individual lock.
	mu *sync.Mutex
	// Goroutines exit sync:
	wg *sync.WaitGroup
}

type SchedulerConfig struct {
	// The number of workers. If set to -1 it will match the number of
	// available cores but not more than SCHEDULER_CONFIG_NUM_WORKERS_MAX:
	NumWorkers int `yaml:"num_workers"`
}

type SchedulerState int

var (
	SchedulerStateCreated SchedulerState = 0
	SchedulerStateRunning SchedulerState = 1
	SchedulerStateStopped SchedulerState = 2
)

var schedulerStateMap = map[SchedulerState]string{
	SchedulerStateCreated: "Created",
	SchedulerStateRunning: "Running",
	SchedulerStateStopped: "Stopped",
}

func (state SchedulerState) String() string {
	return schedulerStateMap[state]
}

var schedulerLog = NewCompLogger("scheduler")

func NewTask(id string, interval time.Duration, action TaskAction) *Task {
	return &Task{
		id:       id,
		interval: interval,
		action:   action,
	}
}

func NewTaskStats() *TaskStats {
	return &TaskStats{
		uint64Stats:  make([]uint64, TASK_STATS_UINT64_LEN),
		float64Stats: make([]float64, TASK_STATS_FLOAT64_LEN),
	}
}

func NewScheduler(cfg any) (*Scheduler, error) {
	var (
		schedulerCfg *SchedulerConfig
	)

	switch cfg := cfg.(type) {
	case *LsvmiConfig:
		schedulerCfg = cfg.SchedulerConfig
	case *SchedulerConfig:
		schedulerCfg = cfg
	case nil:
		schedulerCfg = DefaultSchedulerConfig()
	default:
		return nil, fmt.Errorf("NewScheduler: %T invalid config type", cfg)
	}

	numWorkers := schedulerCfg.NumWorkers
	if numWorkers <= 0 {
		numWorkers = utils.AvailableCpusCount
	}
	if numWorkers > SCHEDULER_CONFIG_NUM_WORKERS_MAX {
		numWorkers = SCHEDULER_CONFIG_NUM_WORKERS_MAX
	}

	scheduler := &Scheduler{
		tasks:      make([]*Task, 0),
		taskQ:      make(chan *Task, SCHEDULER_TASK_Q_LEN),
		todoQ:      make(chan *Task, SCHEDULER_TODO_Q_LEN),
		numWorkers: numWorkers,
		stats:      make(SchedulerStats),
		state:      SchedulerStateCreated,
		mu:         &sync.Mutex{},
		wg:         &sync.WaitGroup{},
	}

	schedulerLog.Infof("num_workers=%d", scheduler.numWorkers)

	return scheduler, nil
}

func DefaultSchedulerConfig() *SchedulerConfig {
	return &SchedulerConfig{
		NumWorkers: SCHEDULER_CONFIG_NUM_WORKERS_DEFAULT,
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
			if scheduler.stats != nil {
				taskStats := scheduler.stats[task.id]
				if taskStats == nil {
					taskStats = NewTaskStats()
					scheduler.stats[task.id] = taskStats
				}
				taskStats.uint64Stats[TASK_STATS_SCHEDULED_COUNT] += 1
				if !queueIt {
					taskStats.uint64Stats[TASK_STATS_OVER_SCHEDULED_COUNT] += 1
				}
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
		startTs := time.Now()
		if task.action != nil {
			task.action.Execute()
		}
		taskRuntime := time.Since(startTs).Seconds()
		scheduler.mu.Lock()
		scheduler.stats[task.id].float64Stats[TASK_STATS_RUNTIME_SEC] += taskRuntime
		task.scheduled = false
		scheduler.mu.Unlock()
	}
}

func (scheduler *Scheduler) SnapStats(to SchedulerStats) SchedulerStats {
	if scheduler.stats == nil {
		return nil
	}
	if to == nil {
		to = make(SchedulerStats)
	}
	scheduler.mu.Lock()
	for taskId, taskStats := range scheduler.stats {
		toTaskStats := to[taskId]
		if toTaskStats == nil {
			toTaskStats = NewTaskStats()
			to[taskId] = toTaskStats
		}
		copy(toTaskStats.uint64Stats, taskStats.uint64Stats)
		copy(toTaskStats.float64Stats, taskStats.float64Stats)
	}
	scheduler.mu.Unlock()
	return to
}

func (scheduler *Scheduler) Start() {
	scheduler.mu.Lock()
	entryState := scheduler.state
	canStart := entryState == SchedulerStateCreated
	if canStart {
		scheduler.state = SchedulerStateRunning
	}
	scheduler.mu.Unlock()

	if !canStart {
		schedulerLog.Warnf(
			"scheduler can only be started from %q state, not from %q",
			SchedulerStateCreated, entryState,
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

	schedulerLog.Info("scheduler started")
}

func (scheduler *Scheduler) Shutdown() {
	scheduler.mu.Lock()
	stopped := scheduler.state == SchedulerStateStopped
	scheduler.state = SchedulerStateStopped
	scheduler.mu.Unlock()

	if stopped {
		schedulerLog.Warn("scheduler already stopped")
		return
	}

	schedulerLog.Info("stop scheduler")

	close(scheduler.taskQ)
	scheduler.wg.Wait()

	schedulerLog.Info("scheduler stopped")
}
