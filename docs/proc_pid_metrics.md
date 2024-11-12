# LSVMI Process And Thread Metrics (id: `proc_pid_metrics#<part>`)

<!-- TOC tocDepth:2..4 chapterDepth:2..6 -->

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
  - [proc_pid_status_info](#proc_pid_status_info)
  - [proc_pid_status_vm_..., proc_pid_status_rss_..., proc_pid_status_...pages](#proc_pid_status_vm_-proc_pid_status_rss_-proc_pid_status_pages)
    - [proc_pid_status_vm_peak](#proc_pid_status_vm_peak)
    - [proc_pid_status_vm_size](#proc_pid_status_vm_size)
    - [proc_pid_status_vm_lck](#proc_pid_status_vm_lck)
    - [proc_pid_status_vm_pin](#proc_pid_status_vm_pin)
    - [proc_pid_status_vm_hwm](#proc_pid_status_vm_hwm)
    - [proc_pid_status_vm_rss](#proc_pid_status_vm_rss)
    - [proc_pid_status_rss_anon](#proc_pid_status_rss_anon)
    - [proc_pid_status_rss_file](#proc_pid_status_rss_file)
    - [proc_pid_status_rss_shmem](#proc_pid_status_rss_shmem)
    - [proc_pid_status_vm_data](#proc_pid_status_vm_data)
    - [proc_pid_status_vm_stk](#proc_pid_status_vm_stk)
    - [proc_pid_status_vm_exe](#proc_pid_status_vm_exe)
    - [proc_pid_status_vm_lib](#proc_pid_status_vm_lib)
    - [proc_pid_status_vm_pte](#proc_pid_status_vm_pte)
    - [proc_pid_status_vm_pmd](#proc_pid_status_vm_pmd)
    - [proc_pid_status_vm_swap](#proc_pid_status_vm_swap)
    - [proc_pid_status_hugetlbpages](#proc_pid_status_hugetlbpages)
  - [proc_pid_status_vol_ctx_switch_delta](#proc_pid_status_vol_ctx_switch_delta)
  - [proc_pid_status_nonvol_ctx_switch_delta](#proc_pid_status_nonvol_ctx_switch_delta)
- [`/proc/PID/cmdline` Metrics](#procpidcmdline-metrics)
  - [proc_pid_cmdline](#proc_pid_cmdline)
- [Additional Generator Metrics](#additional-generator-metrics)
  - [proc_pid_total_count](#proc_pid_total_count)
  - [proc_pid_parse_ok_count](#proc_pid_parse_ok_count)
  - [proc_pid_parse_err_count](#proc_pid_parse_err_count)
  - [proc_pid_active_count](#proc_pid_active_count)
  - [proc_pid_new_count](#proc_pid_new_count)
  - [proc_pid_del_count](#proc_pid_del_count)

<!-- /TOC -->

## General Information

Based on [/proc/PID/stat](https://man7.org/linux/man-pages/man5/proc_pid_stat.5.html), [/proc/PID/status](https://man7.org/linux/man-pages/man5/proc_pid_status.5.html) and [/proc/PID/cmdline](https://man7.org/linux/man-pages/man5/proc_pid_cmdline.5.html) info; thread level metrics use the `/proc/PID/task/TID/...` paths.

See the section about [Active Processes/Threads](internals.md#active-processesthreads) in [Reducing The Number Of Data Points](internals.md#reducing-the-number-of-data-points) internals doc.

Process and thread metrics are potentially the most numerous among all other types and for that reason they can be partitioned such that they are spread across multiple workers. The number of partitioned is controlled by `num_partitions` setting in the `proc_pid_metrics_config` section (see [lsvmi-config-reference.yaml](../lsvmi/lsvmi-config-reference.yaml)).

Since there can be multiple process/thread metrics generators, their ID in the common [Generator Metrics](internal_metrics.md#generator-metrics) is disambiguated by adding the `#<part>` suffix, i.e. the label will look like: `id="proc_pid_metrics#<part>"`.

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

### proc_pid_status_info

[Pseudo-categorical](internals.md#pseudo-categorical-metrics) metric with various information about the process, PID only!

| Label Name | Value(s)/Info |
| --- | --- |
| instance | _instance_ |
| hostname | _hostname_ |
| pid | _PID_ |
| uid | _UID_ |
| gid | _GID_ |
| cpus_allowed | Hexadecimal mask of CPUs on which this process may, a-la [cpuset](https://man7.org/linux/man-pages/man7/cpuset.7.html) |
| mems_allowed | Hexadecimal mask of memory nodes on which this process may, a-la [cpuset](https://man7.org/linux/man-pages/man7/cpuset.7.html) |

### proc_pid_status_vm_..., proc_pid_status_rss_..., proc_pid_status_...pages

Various virtual memory info. The metrics in this group are enabled individually based on `pid_status_memory_fields` setting in the `proc_pid_metrics_config` section (see [lsvmi-config-reference.yaml](../lsvmi/lsvmi-config-reference.yaml)).

They all have the same label set:

| Label Name | Value(s)/Info | Obs |
| --- | --- | --- |
| instance | _instance_ | |
| hostname | _hostname_ | |
| pid | _PID_ | |
| tid | _TID_ | Threads only, applicable to:<br>[proc_pid_status_vm_stk](#proc_pid_status_vm_stk) |
| unit | `kB` |

#### proc_pid_status_vm_peak

#### proc_pid_status_vm_size

#### proc_pid_status_vm_lck

#### proc_pid_status_vm_pin

#### proc_pid_status_vm_hwm

#### proc_pid_status_vm_rss

#### proc_pid_status_rss_anon

#### proc_pid_status_rss_file

#### proc_pid_status_rss_shmem

#### proc_pid_status_vm_data

#### proc_pid_status_vm_stk

#### proc_pid_status_vm_exe

#### proc_pid_status_vm_lib

#### proc_pid_status_vm_pte

#### proc_pid_status_vm_pmd

#### proc_pid_status_vm_swap

#### proc_pid_status_hugetlbpages

### proc_pid_status_vol_ctx_switch_delta

The number of voluntary context switches since the last scan.

| Label Name | Value(s)/Info | Obs |
| --- | --- | --- |
| instance | _instance_ | |
| hostname | _hostname_ | |
| pid | _PID_ | |
| tid | _TID_ | Threads only! |

### proc_pid_status_nonvol_ctx_switch_delta

The number of non-voluntary context switches since the last scan.

| Label Name | Value(s)/Info | Obs |
| --- | --- | --- |
| instance | _instance_ | |
| hostname | _hostname_ | |
| pid | _PID_ | |
| tid | _TID_ | Threads only! |

## `/proc/PID/cmdline` Metrics

### proc_pid_cmdline

[Pseudo-categorical](internals.md#pseudo-categorical-metrics) metric with information about the command line, PID only!

| Label Name | Value(s)/Info |
| --- | --- |
| instance | _instance_ |
| hostname | _hostname_ |
| pid | _PID_ |
| cmd_path | command (argv0) with full path |
| cmd | basename of command |
| args | the args, space separated |

## Additional Generator Metrics

Specific to [LSVMI Process And Thread Metrics](#lsvmi-process-and-thread-metrics-id-proc_pid_metrics), they are in addition to the common [Generator Metrics](internal_metrics.md#generator-metrics).

The all have the same label set:

| Label Name | Value(s)/Info |
| --- | --- |
| instance | _instance_ |
| hostname | _hostname_ |
| part | _partition#_ |

### proc_pid_total_count

Number of PID's/TID's found in the directory scan.

### proc_pid_parse_ok_count

Number of PID's/TID's parsed OK.

### proc_pid_parse_err_count

Number of PID's/TID's with parsing errors.

### proc_pid_active_count

Number of active PID's/TID's (i.e. they used CPU since the last scan). Inactive processes are excluded from partial metrics cycles.

### proc_pid_new_count

Number of PID's/TID's discovered in the current scan.

### proc_pid_del_count

Number of PID's/TID's found to be no longer valid in the current scan.
