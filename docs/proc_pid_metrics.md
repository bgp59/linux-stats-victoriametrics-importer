# LSVMI Process And Thread Metrics (id: `proc_pid_metrics`)

<!-- TOC tocDepth:2..3 chapterDepth:2..6 -->

- [General Information](#general-information)
- [`/proc/PID/stat` Metrics](#procpidstat-metrics)
  - [proc_pid_stat_state](#proc_pid_stat_state)
  - [proc_pid_stat_comm](#proc_pid_stat_comm)
  - [proc_pid_stat_info](#proc_pid_stat_info)
  - [proc_pid_stat_num_threads](#proc_pid_stat_num_threads)
  - [proc_pid_stat_prio](#proc_pid_stat_prio)
  - [proc_pid_stat_vsize_bytes](#proc_pid_stat_vsize_bytes)
  - [proc_pid_stat_rss_bytes](#proc_pid_stat_rss_bytes)
  - [proc_pid_stat_rsslim_bytes](#proc_pid_stat_rsslim_bytes)
  - [proc_pid_stat_minflt_delta](#proc_pid_stat_minflt_delta)
  - [proc_pid_stat_majflt_delta](#proc_pid_stat_majflt_delta)
  - [proc_pid_stat_utime_pcpu](#proc_pid_stat_utime_pcpu)
  - [proc_pid_stat_stime_pcpu](#proc_pid_stat_stime_pcpu)
  - [proc_pid_stat_pcpu](#proc_pid_stat_pcpu)
  - [proc_pid_cpu_num](#proc_pid_cpu_num)
- [`/proc/PID/status` Metrics](#procpidstatus-metrics)

<!-- /TOC -->

## General Information

Based on [/proc/PID/stat](https://man7.org/linux/man-pages/man5/proc_pid_stat.5.html), [/proc/PID/status](https://man7.org/linux/man-pages/man5/proc_pid_status.5.html) and [/proc/PID/cmdline](https://man7.org/linux/man-pages/man5/proc_pid_cmdline.5.html) info; thread level metrics use the `/proc/PID/task/TID/...` paths.

See the section about [Active Processes/Threads](internals.md#active-processesthreads) in [Reducing The Number Of Data Points](internals.md#reducing-the-number-of-data-points) internals doc.

## `/proc/PID/stat` Metrics

### proc_pid_stat_state

[Pseudo-categorical](internals.md#pseudo-categorical-metrics) metric for the state of the process

| Label Name | Value(s)/Info | Obs |
| --- | --- | --- |
| instance | _instance_ | |
| hostname | _hostname_ | |
| pid | _PID_ | |
| tid | _TID_ | Threads only! |
| state | `R`, `S`, `D`, `Z`, `I`, etc. | |

### proc_pid_stat_comm

[Pseudo-categorical](internals.md#pseudo-categorical-metrics) metric with the short `comm`(and) and start time, in milliseconds.

| Label Name | Value(s)/Info | Obs |
| --- | --- | --- |
| instance | _instance_ | |
| hostname | _hostname_ | |
| pid | _PID_ | |
| tid | _TID_ | Threads only! |
| comm | `comm` field |
| starttime_msec | start time, in milliseconds |

### proc_pid_stat_info

[Pseudo-categorical](internals.md#pseudo-categorical-metrics) metric with various information about the process, PID only!

| Label Name | Value(s)/Info |
| --- | --- |
| instance | _instance_ |
| hostname | _hostname_ |
| pid | _PID_ |
| ppid | Parent PID |
| pgrp | The process group ID |
| session | The session ID |
| tty | _tty_nr_ field |
| tpgid | The ID of the foreground process group of the controlling terminal |
| flags | The kernel flags word |

### proc_pid_stat_num_threads

The number of threads, PID only!

| Label Name | Value(s)/Info |
| --- | --- |
| instance | _instance_ |
| hostname | _hostname_ |
| pid | _PID_ |

### proc_pid_stat_prio

[Pseudo-categorical](internals.md#pseudo-categorical-metrics) metric with priority & scheduling info; PID only!

| Label Name | Value(s)/Info |
| --- | --- |
| instance | _instance_ |
| hostname | _hostname_ |
| pid | _PID_ |
| prio | _prority_ |
| nice | _nice_ value |
| rt_prio | _rt_priority_ field |
| policy | Scheduling policy |

### proc_pid_stat_vsize_bytes

Virtual memory size in bytes; PID only!

| Label Name | Value(s)/Info |
| --- | --- |
| instance | _instance_ |
| hostname | _hostname_ |
| pid | _PID_ |

### proc_pid_stat_rss_bytes

Resident Set Size in bytes; PID only!

| Label Name | Value(s)/Info |
| --- | --- |
| instance | _instance_ |
| hostname | _hostname_ |
| pid | _PID_ |

### proc_pid_stat_rsslim_bytes

Current soft limit in bytes on the rss; PID only!

| Label Name | Value(s)/Info |
| --- | --- |
| instance | _instance_ |
| hostname | _hostname_ |
| pid | _PID_ |

### proc_pid_stat_minflt_delta

The number of minor faults since the last scan.

| Label Name | Value(s)/Info | Obs |
| --- | --- | --- |
| instance | _instance_ | |
| hostname | _hostname_ | |
| pid | _PID_ | |
| tid | _TID_ | Threads only! |

### proc_pid_stat_majflt_delta

The number of major faults since the last scan.

| Label Name | Value(s)/Info | Obs |
| --- | --- | --- |
| instance | _instance_ | |
| hostname | _hostname_ | |
| pid | _PID_ | |
| tid | _TID_ | Threads only! |

### proc_pid_stat_utime_pcpu

The percent of time in user mode over the interval since the last scan.

| Label Name | Value(s)/Info | Obs |
| --- | --- | --- |
| instance | _instance_ | |
| hostname | _hostname_ | |
| pid | _PID_ | |
| tid | _TID_ | Threads only! |

### proc_pid_stat_stime_pcpu

The percent of time in system mode over the interval since the last scan.

| Label Name | Value(s)/Info | Obs |
| --- | --- | --- |
| instance | _instance_ | |
| hostname | _hostname_ | |
| pid | _PID_ | |
| tid | _TID_ | Threads only! |

### proc_pid_stat_pcpu

The %CPU over the interval since the last scan.

| Label Name | Value(s)/Info | Obs |
| --- | --- | --- |
| instance | _instance_ | |
| hostname | _hostname_ | |
| pid | _PID_ | |
| tid | _TID_ | Threads only! |

### proc_pid_cpu_num

The cpu number last executed on.

| Label Name | Value(s)/Info | Obs |
| --- | --- | --- |
| instance | _instance_ | |
| hostname | _hostname_ | |
| pid | _PID_ | |
| tid | _TID_ | Threads only! |

## `/proc/PID/status` Metrics

**NOTE!** These metrics are optional, they are controlled by `use_pid_status` setting in the `proc_pid_metrics_config` section (see [lsvmi-config-reference.yaml](../lsvmi/lsvmi-config-reference.yaml))
