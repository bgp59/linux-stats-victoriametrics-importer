#! /usr/bin/env python3

from dataclasses import dataclass

# Should match procfs/pid_tid_list.go:

PID_LIST_CACHE_PID_ENABLED = 1 << 0
PID_LIST_CACHE_TID_ENABLED = 1 << 1
PID_LIST_CACHE_ALL_ENABLED = PID_LIST_CACHE_PID_ENABLED | PID_LIST_CACHE_TID_ENABLED

# Special TID to indicate that the stats are for PID only:
PID_ONLY_TID = 0


@dataclass
class PidTid:
    Pid: int = 0
    Tid: int = PID_ONLY_TID
