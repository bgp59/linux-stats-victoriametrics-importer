# LSVMI Internal Metrics (id: `internal_metrics`)

<!-- TOC tocDepth:2..6 chapterDepth:2..6 -->

- [General Information](#general-information)
- [Agent Metrics](#agent-metrics)
  - [lsvmi_internal_metrics_delta_sec](#lsvmi_internal_metrics_delta_sec)
  - [lsvmi_uptime_sec](#lsvmi_uptime_sec)
  - [lsvmi_proc_num_threads](#lsvmi_proc_num_threads)
  - [lsvmi_proc_pcpu](#lsvmi_proc_pcpu)
  - [lsvmi_proc_rss](#lsvmi_proc_rss)
  - [lsvmi_proc_vsize](#lsvmi_proc_vsize)
- [Compressor Pool Metrics](#compressor-pool-metrics)
  - [lsvmi_compressor_read_delta](#lsvmi_compressor_read_delta)
  - [lsvmi_compressor_read_byte_delta](#lsvmi_compressor_read_byte_delta)
  - [lsvmi_compressor_send_delta](#lsvmi_compressor_send_delta)
  - [lsvmi_compressor_send_byte_delta](#lsvmi_compressor_send_byte_delta)
  - [lsvmi_compressor_send_error_delta](#lsvmi_compressor_send_error_delta)
  - [lsvmi_compressor_tout_flush_delta](#lsvmi_compressor_tout_flush_delta)
  - [lsvmi_compressor_write_error_delta](#lsvmi_compressor_write_error_delta)
  - [lsvmi_compressor_compression_factor](#lsvmi_compressor_compression_factor)
- [Generator Metrics](#generator-metrics)
  - [lsvmi_metrics_gen_invocation_delta](#lsvmi_metrics_gen_invocation_delta)
  - [lsvmi_metrics_gen_actual_metrics_delta](#lsvmi_metrics_gen_actual_metrics_delta)
  - [lsvmi_metrics_gen_total_metrics_delta](#lsvmi_metrics_gen_total_metrics_delta)
  - [lsvmi_metrics_gen_bytes_delta](#lsvmi_metrics_gen_bytes_delta)
- [Go Specific Metrics](#go-specific-metrics)
  - [lsvmi_go_mem_free_delta](#lsvmi_go_mem_free_delta)
  - [lsvmi_go_mem_gc_delta](#lsvmi_go_mem_gc_delta)
  - [lsvmi_go_mem_malloc_delta](#lsvmi_go_mem_malloc_delta)
  - [lsvmi_go_num_goroutine](#lsvmi_go_num_goroutine)
  - [lsvmi_go_mem_in_use_object_count](#lsvmi_go_mem_in_use_object_count)
  - [lsvmi_go_mem_heap_bytes](#lsvmi_go_mem_heap_bytes)
  - [lsvmi_go_mem_heap_sys_bytes](#lsvmi_go_mem_heap_sys_bytes)
  - [lsvmi_go_mem_sys_bytes](#lsvmi_go_mem_sys_bytes)
- [HTTP Endpoint Pool Metrics](#http-endpoint-pool-metrics)
  - [Per Endpoint Metrics](#per-endpoint-metrics)
    - [lsvmi_http_ep_send_buffer_delta](#lsvmi_http_ep_send_buffer_delta)
    - [lsvmi_http_ep_send_buffer_byte_delta](#lsvmi_http_ep_send_buffer_byte_delta)
    - [lsvmi_http_ep_send_buffer_error_delta](#lsvmi_http_ep_send_buffer_error_delta)
    - [lsvmi_http_ep_healthcheck_delta](#lsvmi_http_ep_healthcheck_delta)
    - [lsvmi_http_ep_healthcheck_error_delta](#lsvmi_http_ep_healthcheck_error_delta)
    - [lsvmi_http_ep_state](#lsvmi_http_ep_state)
  - [Per Pool Metrics](#per-pool-metrics)
    - [lsvmi_http_ep_pool_healthy_rotate_count](#lsvmi_http_ep_pool_healthy_rotate_count)
    - [lsvmi_http_ep_pool_no_healthy_ep_error_delta](#lsvmi_http_ep_pool_no_healthy_ep_error_delta)
- [OS Metrics](#os-metrics)
  - [os_info](#os_info)
  - [os_btime_sec](#os_btime_sec)
  - [os_uptime_sec](#os_uptime_sec)
- [Scheduler Metrics](#scheduler-metrics)
  - [lsvmi_task_scheduled_delta](#lsvmi_task_scheduled_delta)
  - [lsvmi_task_delayed_delta](#lsvmi_task_delayed_delta)
  - [lsvmi_task_overrun_delta](#lsvmi_task_overrun_delta)
  - [lsvmi_task_executed_delta](#lsvmi_task_executed_delta)
  - [lsvmi_task_deadline_hack_delta](#lsvmi_task_deadline_hack_delta)
  - [lsvmi_task_interval_avg_runtime_sec](#lsvmi_task_interval_avg_runtime_sec)

<!-- /TOC -->

## General Information

These are metrics relating to the agent itself. There is no partial/full cycle approach for these metrics, the entire set is generated for every cycle.

## Agent Metrics

**NOTE!** Unless otherwise stated, the metrics in this paragraph have the following label set:

  | Label Name | Value(s)/Info |
  | --- | --- |
  | instance | _instance_ |
  | hostname | _hostname_ |

### lsvmi_internal_metrics_delta_sec

  The actual time delta, in seconds, since the last internal metrics generation. This may be different than the scan `interval`, the latter is the desired, theoretical value.

### lsvmi_uptime_sec

  Time, in seconds, since the agent was started.
  
  | Label Name | Value(s)/Info |
  | --- | --- |
  | instance | _instance_ |
  | hostname | _hostname_ |
  | version | semver of the agent |
  | gitinfo | git describe based |

### lsvmi_proc_num_threads

The number of threads.

### lsvmi_proc_pcpu

The %CPU for the scan interval.

### lsvmi_proc_rss

Resident memory size, in bytes.

### lsvmi_proc_vsize

Virtual memory resident size, in bytes.

## Compressor Pool Metrics

**NOTE!** Unless otherwise stated, the metrics in this paragraph have the following label set:

  | Label Name | Value(s)/Info |
  | --- | --- |
  | instance | _instance_ |
  | hostname | _hostname_ |
  | compressor | _compressor#_ (0 .. num_compressor - 1) |

### lsvmi_compressor_read_delta

The number of reads from the queue since the last scan.

### lsvmi_compressor_read_byte_delta

The number of read bytes from the queue since the last scan.

### lsvmi_compressor_send_delta

The number of sends since the last scan.

### lsvmi_compressor_send_byte_delta

The number of sent bytes since the last scan.

### lsvmi_compressor_send_error_delta

The number of send errors since the last scan.

### lsvmi_compressor_tout_flush_delta

The number of timeout (timed based, that is) flushes since the last scan.

### lsvmi_compressor_write_error_delta

The number of write (to compressor stream) errors since the last scan.

### lsvmi_compressor_compression_factor

The (exponentially decaying) compression factor average.

## Generator Metrics

[LSVMI](../README.md) comprises a number of metrics generators loosely organized by source of data and/or by the objects for which the metrics are generated, e.g. processes, network interfaces, interrupts, etc.

Each metrics generator maintains a standard set of stats, updated at the end of the generator's invocation. The stats are scanned periodically by the internal metrics generator for creating the set of metrics described in this section. All the deltas below are computed for the internal metrics interval and **not** for the generator's that they describe.

For instance the `proc_stats` metrics may be generated every 0.2 sec while the internal metrics are generated every 5 sec. The delta for `proc_stats` metrics count is for the 5 sec interval, i.e. the result of 25 runs.

This approach make the generator metrics comparable side-by-side, since they refer a common interval.

**NOTE!** Unless otherwise stated, the metrics in this paragraph have the following label set:

  | Label Name | Value(s)/Info |
  | --- | --- |
  | instance | _instance_ |
  | hostname | _hostname_ |
  | id | _id_ the unique generator ID e.g. `internal_metrics`, `proc_pid_metrics` |

### lsvmi_metrics_gen_invocation_delta

The number of invocations, since the last scan.

### lsvmi_metrics_gen_actual_metrics_delta

The actual number of metrics generated, since the last scan. This can be less than `lsvmi_metrics_gen_total_metrics_delta` due to [Reducing The Number Of Data Points](internals.md#reducing-the-number-of-data-points) techniques.

### lsvmi_metrics_gen_total_metrics_delta

The total (theoretical max) number of metrics that could have been generated, since the last scan. These 2 metrics can be used to assess the efficiency of the data points reduction.

### lsvmi_metrics_gen_bytes_delta

The number of bytes for all the generated metrics, since the last scan.

## Go Specific Metrics

**NOTE!** Unless otherwise stated, the metrics in this paragraph have the following label set:

  | Label Name | Value(s)/Info |
  | --- | --- |
  | instance | _instance_ |
  | hostname | _hostname_ |

### lsvmi_go_mem_free_delta

The number of `free` calls since the last scan.

### lsvmi_go_mem_gc_delta

The number of garbage collector calls since the last scan.

### lsvmi_go_mem_malloc_delta

The number of `malloc` calls since the last scan.

### lsvmi_go_num_goroutine

The current number of goroutines.

### lsvmi_go_mem_in_use_object_count

The current number of go objects in use.

### lsvmi_go_mem_heap_bytes

### lsvmi_go_mem_heap_sys_bytes

### lsvmi_go_mem_sys_bytes

The size of various memory pools, in bytes.

## HTTP Endpoint Pool Metrics

### Per Endpoint Metrics

**NOTE!** Unless otherwise stated, the metrics in this paragraph have the following label set:

  | Label Name | Value(s)/Info |
  | --- | --- |
  | instance | _instance_ |
  | hostname | _hostname_ |
  | url | _url_ |

#### lsvmi_http_ep_send_buffer_delta

The number of send calls against this URL, since the last scan.

#### lsvmi_http_ep_send_buffer_byte_delta

The number of bytes sent to this URL, since the last scan.

#### lsvmi_http_ep_send_buffer_error_delta

The number of send call errors against this URL, since the last scan.

#### lsvmi_http_ep_healthcheck_delta

The number of health checks for this URL, since the last scan.

#### lsvmi_http_ep_healthcheck_error_delta

The number of failed health checks for this URL, since the last scan.

#### lsvmi_http_ep_state

The state of this URL, `0`: `HealthCheck`, `1`: `Healthy`, `2`: `AtHead`

### Per Pool Metrics

**NOTE!** Unless otherwise stated, the metrics in this paragraph have the following label set:

  | Label Name | Value(s)/Info |
  | --- | --- |
  | instance | _instance_ |
  | hostname | _hostname_ |

#### lsvmi_http_ep_pool_healthy_rotate_count

The cumulative number of rotations.

#### lsvmi_http_ep_pool_no_healthy_ep_error_delta

The number of endpoint errors since the last scan.

## OS Metrics

**NOTE!** Unless otherwise stated, the metrics in this paragraph have the following label set:

  | Label Name | Value(s)/Info |
  | --- | --- |
  | instance | _instance_ |
  | hostname | _hostname_ |

### os_info

  Categorical metric (constant `1`) with [uname](https://linux.die.net/man/1/uname) and Linux [os-release](https://man7.org/linux/man-pages/man5/os-release.5.html) info:

  | Label Name | Value(s)/Info |
  | --- | --- |
  | instance | _instance_ |
  | hostname | _hostname_ |
  | sys_name | \`uname -s\` |
  | sys_release | \`uname -r\` |
  | sys_version | \`uname -v\` |
  | ID<br>NAME<br>PRETTY_NAME<br>VERSION<br>VERSION_CODENAME<br>VERSION_ID | See the eponymous fields in [os-release](https://man7.org/linux/man-pages/man5/os-release.5.html) |

### os_btime_sec

  Boot time, in seconds.

### os_uptime_sec

  Time, in seconds, since OS boot

## Scheduler Metrics

**NOTE!** They all have the same label set:

  | Label Name | Value(s)/Info |
  | --- | --- |
  | instance | _instance_ |
  | hostname | _hostname_ |
  | task_id | _id_, e.g. `internal_metrics`, `proc_pid_metrics`, etc |

### lsvmi_task_scheduled_delta

The number of times the task was scheduled, since previous scan.

### lsvmi_task_delayed_delta

The number of times the task was delayed because its next reschedule would have been too close to the deadline, since the last scan.

### lsvmi_task_overrun_delta

The number of times the task ran past the next  deadline, since the last scan.

### lsvmi_task_executed_delta

The number of times the task was executed, since the last scan.

### lsvmi_task_deadline_hack_delta

The number of times the deadline hack was applied for the task, since the last scan.

The hack is required for a rare condition observed when running Docker on a MacBook whereby the clock appears to move backwards and the next deadline results in being before the previous one. The hack consist in adding task intervals until the chronological order is restored.

### lsvmi_task_interval_avg_runtime_sec

The average time, in seconds, for all the runs of the task so far.
