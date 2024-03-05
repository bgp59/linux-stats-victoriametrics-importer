#! /usr/bin/env python3

from typing import Dict, List, Literal, Union

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

StatCpuStats = Dict[int, List[int]]

Stat = Dict[
    Union[Literal["Cpu"], Literal["NumericFields"]],
    Union[StatCpuStats, List[int]],
]
