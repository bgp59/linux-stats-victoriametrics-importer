# LSVMI Stat (General OS) Metrics (id: `proc_stat_metrics`)

<!-- TOC tocDepth:2..3 chapterDepth:2..6 -->

- [General Information](#general-information)
- [Metrics](#metrics)
  - [proc_stat_cpu_pct](#proc_stat_cpu_pct)
  - [proc_stat_cpu_up](#proc_stat_cpu_up)
  - [proc_stat_btime_sec](#proc_stat_btime_sec)
  - [proc_stat_uptime_sec](#proc_stat_uptime_sec)
  - [proc_stat_page_in_delta](#proc_stat_page_in_delta)
  - [proc_stat_page_out_delta](#proc_stat_page_out_delta)
  - [proc_stat_swap_in_delta](#proc_stat_swap_in_delta)
  - [proc_stat_swap_out_delta](#proc_stat_swap_out_delta)
  - [proc_stat_ctxt_delta](#proc_stat_ctxt_delta)
  - [proc_stat_processes_delta](#proc_stat_processes_delta)
  - [proc_stat_procs_running_count](#proc_stat_procs_running_count)
  - [proc_stat_procs_blocked_count](#proc_stat_procs_blocked_count)
  - [proc_stat_metrics_delta_sec](#proc_stat_metrics_delta_sec)

<!-- /TOC -->

## General Information

Based on [/proc/stat](https://man7.org/linux/man-pages/man5/proc_stat.5.html).

## Metrics

Unless otherwise stated, the metrics in this section have the following label set:

| Label Name | Value(s)/Info |
| --- | --- |
| instance | _instance_ |
| hostname | _hostname_ |

### proc_stat_cpu_pct

%CPU per mode per CPU, over the interval since the last scan.

| Label Name | Value(s)/Info |
| --- | --- |
| instance | _instance_ |
| hostname | _hostname_ |
| mode | MODE, e.g. `user`, `system`, `idle`, etc/ |
| cpu | _CPU\#_, `all` or `avg`  |

%CPU for `cpu="avg"` is %CPU for `cpu="all"` / the number of CPUs found in that scan.

### proc_stat_cpu_up

[Pseudo-categorical](internals.md#pseudo-categorical-metrics ) metric containing information about the current CPU's. The list may change from one scan to another due to [CPU hotplug in the Kernel](https://docs.kernel.org/core-api/cpu_hotplug.html). This metric could be used for determining the list of available CPUs.

| Label Name | Value(s)/Info |
| --- | --- |
| instance | _instance_ |
| hostname | _hostname_ |
| cpu | _CPU\#_  |

### proc_stat_btime_sec

Boot time in seconds since the epoch.

### proc_stat_uptime_sec

Uptime in seconds (derived from boot time above).

### proc_stat_page_in_delta

### proc_stat_page_out_delta

Number of memory pages brought in and out since the last scan.

### proc_stat_swap_in_delta

### proc_stat_swap_out_delta

Number of swap pages brought in and out since the last scan.

### proc_stat_ctxt_delta

Number of context switches since the last scan.

### proc_stat_processes_delta

Number of forks since the last scan.

### proc_stat_procs_running_count

Number processes in runnable state.

### proc_stat_procs_blocked_count

Number processes of processes blocked waiting for I/O to complete.  

### proc_stat_metrics_delta_sec

Time in seconds since the last scan. The real life counterpart (i.e. measured value) to the desired (configured) `interval`.
