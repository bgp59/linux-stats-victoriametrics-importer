#! /usr/bin/env python3

# JSON serialize-able TaskStats end SchedulerStats, matching lsvmi/scheduler.go

from dataclasses import dataclass, field
from typing import Dict, List

TASK_STATS_SCHEDULED_COUNT = 0
TASK_STATS_DELAYED_COUNT = 1
TASK_STATS_OVERRUN_COUNT = 2
TASK_STATS_EXECUTED_COUNT = 3
TASK_STATS_DEADLINE_HACK_COUNT = 4

TASK_STATS_UINT64_LEN = 5


@dataclass
class TaskStats:
    Uint64Stats: List[int] = field(default_factory=lambda: [0] * TASK_STATS_UINT64_LEN)
    RuntimeTotal: int = 0


SchedulerStats = Dict[str, TaskStats]
