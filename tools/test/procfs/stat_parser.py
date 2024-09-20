#! /usr/bin/env python3

from dataclasses import dataclass
from typing import Dict, List, Optional

# JSON serialize-able Stat, matching profcs/stat_parser.go:

STAT_CPU_USER_TICKS = 0
STAT_CPU_NICE_TICKS = 1
STAT_CPU_SYSTEM_TICKS = 2
STAT_CPU_IDLE_TICKS = 3
STAT_CPU_IOWAIT_TICKS = 4
STAT_CPU_IRQ_TICKS = 5
STAT_CPU_SOFTIRQ_TICKS = 6
STAT_CPU_STEAL_TICKS = 7
STAT_CPU_GUEST_TICKS = 8
STAT_CPU_GUEST_NICE_TICKS = 9
STAT_CPU_NUM_STATS = 10

STAT_CPU_ALL = -1

STAT_PAGE_IN = 0
STAT_PAGE_OUT = 1
STAT_SWAP_IN = 2
STAT_SWAP_OUT = 3
STAT_CTXT = 4
STAT_BTIME = 5
STAT_PROCESSES = 6
STAT_PROCS_RUNNING = 7
STAT_PROCS_BLOCKED = 8
STAT_NUMERIC_NUM_STATS = 9


@dataclass
class Stat:
    Cpu: Optional[Dict[int, List[int]]] = None
    NumCpus: int = 0
    NumericFields: Optional[List[int]] = None
