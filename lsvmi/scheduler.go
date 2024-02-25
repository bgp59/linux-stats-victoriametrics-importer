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
//            ^  ^              |
//   new task |  |              |
//   ---------+  |   +----------+--- ... ----+
//           +---+   | task     | task       | task
//           |       v          v            v
//           |  +--------+ +--------+   +--------+
//           |  | Worker | | Worker |...| Worker |
//           |  +--------+ +--------+   +--------+
//           |       | task     | task       | task
//           +-------+----------+--- ... ----+
//
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
// deadlines is updated and it is added to heap and it is placed into the TODO
// Queue.
//
// The TODO Queue feeds the Worker Pool; the number of workers in the pool
// controls the level of concurrency of task execution and it allows for short
// tasks to be executed without having to wait for a long one to complete.

import (
	"container/heap"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/eparparita/linux-stats-victoriametrics-importer/internal/utils"
)

const (
	SCHEDULER_CONFIG_NUM_WORKERS_DEFAULT = -1
)

const (
	SCHEDULER_TASK_Q_LEN = 64
	SCHEDULER_TODO_Q_LEN = 64
	// All intervals will be rounded to be a multiple of scheduler's granularity:
	SCHEDULER_GRANULARITY = 20 * time.Millisecond
	// The minimum pause between 2 executions:
	SCHEDULER_TASK_MIN_EXECUTION_PAUSE = 2 * SCHEDULER_GRANULARITY
)

const (
	// Indexes into Scheduler.stats.[id].Uint64Stats

	// How many times the task was scheduled:
	TASK_STATS_SCHEDULED_COUNT = iota

	// How many times the task was delayed because it was too close to its
	// previous execution:
	TASK_STATS_DELAYED_COUNT

	// How many times the task overran, i.e. its runtime >= interval:
	TASK_STATS_OVERRUN_COUNT

	// How many times the task was executed:
	TASK_STATS_EXECUTED_COUNT

	// Must be last:
	TASK_STATS_UINT64_LEN
)

const (
	// Indexes into Scheduler.stats.[id].Float64Stats

	// Total run time, in seconds:
	TASK_STATS_RUNTIME_SEC = iota

	// Average run time, in seconds:
	TASK_STATS_AVG_RUNTIME_SEC

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
	// Whether it was re-added by a worker or not (i.e. the logical complement
	// of new task). New tasks are scheduled for execution immediately whereas
	// re-added ones are scheduled according to the interval:
	addedByWorker bool
	// When last executed, used to protect long running tasks from being
	// scheduled back to back:
	lastExecuted time.Time
}

type TaskStats struct {
	Uint64Stats  []uint64
	Float64Stats []float64
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
	ctx      context.Context
	cancelFn context.CancelFunc
	wg       *sync.WaitGroup
}

type SchedulerConfig struct {
	// The number of workers. If set to -1 it will match the number of
	// available cores:
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
		id:            id,
		interval:      interval,
		action:        action,
		addedByWorker: false,
	}
}

func NewTaskStats() *TaskStats {
	return &TaskStats{
		Uint64Stats:  make([]uint64, TASK_STATS_UINT64_LEN),
		Float64Stats: make([]float64, TASK_STATS_FLOAT64_LEN),
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

	ctx, cancelFn := context.WithCancel(context.Background())
	scheduler := &Scheduler{
		tasks:      make([]*Task, 0),
		taskQ:      make(chan *Task, SCHEDULER_TASK_Q_LEN),
		todoQ:      make(chan *Task, SCHEDULER_TODO_Q_LEN),
		numWorkers: numWorkers,
		stats:      make(SchedulerStats),
		state:      SchedulerStateCreated,
		mu:         &sync.Mutex{},
		ctx:        ctx,
		cancelFn:   cancelFn,
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

// Ensure that a task interval is scheduler compliant:
func CompliantTaskInterval(interval time.Duration) time.Duration {
	compliantInterval := interval.Truncate(SCHEDULER_GRANULARITY)
	if interval-compliantInterval >= SCHEDULER_GRANULARITY/2 {
		compliantInterval += SCHEDULER_GRANULARITY
	}
	if compliantInterval < SCHEDULER_TASK_MIN_EXECUTION_PAUSE {
		compliantInterval = SCHEDULER_TASK_MIN_EXECUTION_PAUSE
	}
	return compliantInterval
}

func (scheduler *Scheduler) AddNewTask(task *Task) {
	task.addedByWorker = false
	compliantInterval := CompliantTaskInterval(task.interval)
	if compliantInterval != task.interval {
		schedulerLog.Warnf(
			"task %s: interval: %s -> %s", task.id, task.interval, compliantInterval,
		)
		task.interval = compliantInterval
	}
	schedulerLog.Infof("add task %s: interval=%s", task.id, task.interval)
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
		if activeTimer && !timer.Stop() {
			<-timer.C
		}
		schedulerLog.Info("dispatcher stopped")
		scheduler.wg.Done()
	}()

	var (
		task            *Task
		currentDeadline time.Time
	)

	taskQ, todoQ := scheduler.taskQ, scheduler.todoQ
	stats, mu := scheduler.stats, scheduler.mu
	ctx := scheduler.ctx
	for {
		if !activeTimer && len(scheduler.tasks) > 0 {
			currentDeadline = scheduler.tasks[0].deadline
			timer.Reset(time.Until(currentDeadline))
			activeTimer = true
		}

		select {
		case <-ctx.Done():
			return
		case task = <-taskQ:
			// The desired next deadline is the nearest future multiple of
			// interval:
			timeNow := time.Now()
			nextDeadline := timeNow.Truncate(task.interval).Add(task.interval)

			// If this task was re-added by a worker or if the next deadline is
			// very close, then add it for later scheduling:
			if task.addedByWorker || nextDeadline.Sub(timeNow) < SCHEDULER_TASK_MIN_EXECUTION_PAUSE {
				if task.addedByWorker {
					// Check the pause since the last execution:
					taskNearestDeadline := task.lastExecuted.Add(SCHEDULER_TASK_MIN_EXECUTION_PAUSE)
					if nextDeadline.Before(taskNearestDeadline) {
						nextDeadline = taskNearestDeadline
						mu.Lock()
						stats[task.id].Uint64Stats[TASK_STATS_DELAYED_COUNT] += 1
						mu.Unlock()
					}
				}
				task.deadline = nextDeadline
				heap.Push(scheduler, task)

				// Cancel the timer if this new deadline is more recent than the
				// one currently pending:
				if activeTimer && nextDeadline.Before(currentDeadline) {
					if !timer.Stop() {
						<-timer.C
					}
					activeTimer = false
				}

				// Do not execute right away, wait for scheduling:
				task = nil
			} else {
				// New task, any other pending timer is no longer applicable:
				task.deadline = timeNow
				if activeTimer {
					if !timer.Stop() {
						<-timer.C
					}
					activeTimer = false
				}
			}

		case <-timer.C:
			activeTimer = false
			task = heap.Pop(scheduler).(*Task)
		}

		if task != nil {
			mu.Lock()
			if stats[task.id] == nil {
				stats[task.id] = NewTaskStats()
			}
			stats[task.id].Uint64Stats[TASK_STATS_SCHEDULED_COUNT] += 1
			mu.Unlock()
			todoQ <- task
		}
	}
}

func (scheduler *Scheduler) workerLoop(workerId int) {
	schedulerLog.Infof("start worker# %d", workerId)

	defer func() {
		schedulerLog.Infof("worker# %d stopped", workerId)
		scheduler.wg.Done()
	}()

	taskQ, todoQ := scheduler.taskQ, scheduler.todoQ
	stats, mu := scheduler.stats, scheduler.mu
	ctx := scheduler.ctx
	for {
		select {
		case <-ctx.Done():
			return
		case task := <-todoQ:
			startTs := time.Now()
			if task.action != nil {
				task.action.Execute()
			}
			endTs := time.Now()
			task.lastExecuted = endTs
			runtime := endTs.Sub(startTs)
			mu.Lock()
			taskStats := stats[task.id]
			if runtime >= task.interval {
				taskStats.Uint64Stats[TASK_STATS_OVERRUN_COUNT] += 1
			}
			taskStats.Uint64Stats[TASK_STATS_EXECUTED_COUNT] += 1
			taskStats.Float64Stats[TASK_STATS_RUNTIME_SEC] += runtime.Seconds()
			mu.Unlock()
			task.addedByWorker = true
			taskQ <- task
		}
	}
}

// Snap current stats. Optionally clear them, as a way of snapping deltas.
func (scheduler *Scheduler) SnapStats(to SchedulerStats, clearStats bool) SchedulerStats {
	if scheduler.stats == nil {
		return nil
	}
	if to == nil {
		to = make(SchedulerStats)
	}
	scheduler.mu.Lock()
	defer scheduler.mu.Unlock()
	for taskId, taskStats := range scheduler.stats {
		toTaskStats := to[taskId]
		if toTaskStats == nil {
			toTaskStats = NewTaskStats()
			to[taskId] = toTaskStats
		}
		copy(toTaskStats.Uint64Stats, taskStats.Uint64Stats)
		copy(toTaskStats.Float64Stats, taskStats.Float64Stats)
		n := toTaskStats.Uint64Stats[TASK_STATS_EXECUTED_COUNT]
		if n > 0 {
			toTaskStats.Float64Stats[TASK_STATS_AVG_RUNTIME_SEC] = taskStats.Float64Stats[TASK_STATS_RUNTIME_SEC] / float64(n)
		} else {
			toTaskStats.Float64Stats[TASK_STATS_AVG_RUNTIME_SEC] = 0
		}
		if clearStats {
			clear(taskStats.Uint64Stats)
			clear(taskStats.Float64Stats)
		}
	}
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
	scheduler.cancelFn()
	scheduler.wg.Wait()
	schedulerLog.Info("scheduler stopped")
}
