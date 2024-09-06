#! /usr/bin/env python3

# Must match procfs/pid_stat_parser.go:

PID_STAT_COMM = 0
PID_STAT_STATE = 1
PID_STAT_PPID = 2
PID_STAT_PGRP = 3
PID_STAT_SESSION = 4
PID_STAT_TTY_NR = 5
PID_STAT_TPGID = 6
PID_STAT_FLAGS = 7
PID_STAT_PRIORITY = 8
PID_STAT_NICE = 9
PID_STAT_NUM_THREADS = 10
PID_STAT_STARTTIME = 11
PID_STAT_VSIZE = 12
PID_STAT_RSSLIM = 13
PID_STAT_PROCESSOR = 14
PID_STAT_RT_PRIORITY = 15
PID_STAT_POLICY = 16
PID_STAT_BYTE_SLICE_NUM_FIELDS = 17

PID_STAT_MINFLT = 0
PID_STAT_MAJFLT = 1
PID_STAT_UTIME = 2
PID_STAT_STIME = 3
PID_STAT_RSS = 4
PID_STAT_ULONG_NUM_FIELDS = 5
