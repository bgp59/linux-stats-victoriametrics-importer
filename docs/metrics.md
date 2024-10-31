# LSVMI Metrics

<!-- markdownlint-disable -->
<!-- TOC tocDepth:2..4 chapterDepth:2..6 -->

- [Information Applicable To All Metrics](#information-applicable-to-all-metrics)
  - [Common Labels](#common-labels)
- [Internal Metrics](#internal-metrics)
  - [Compressor Pool](#compressor-pool)
    - [lsvmi_compressor_read_delta](#lsvmi_compressor_read_delta)
    - [lsvmi_compressor_read_byte_delta](#lsvmi_compressor_read_byte_delta)
    - [lsvmi_compressor_send_delta](#lsvmi_compressor_send_delta)
    - [lsvmi_compressor_send_byte_delta](#lsvmi_compressor_send_byte_delta)
    - [lsvmi_compressor_send_error_delta](#lsvmi_compressor_send_error_delta)
    - [lsvmi_compressor_tout_flush_delta](#lsvmi_compressor_tout_flush_delta)
    - [lsvmi_compressor_write_error_delta](#lsvmi_compressor_write_error_delta)
    - [lsvmi_compressor_compression_factor](#lsvmi_compressor_compression_factor)
  - [Go Specific](#go-specific)
    - [lsvmi_go_mem_free_delta](#lsvmi_go_mem_free_delta)
    - [lsvmi_go_mem_gc_delta](#lsvmi_go_mem_gc_delta)
    - [lsvmi_go_mem_malloc_delta](#lsvmi_go_mem_malloc_delta)
    - [lsvmi_go_num_goroutine](#lsvmi_go_num_goroutine)
    - [lsvmi_go_mem_in_use_object_count](#lsvmi_go_mem_in_use_object_count)
    - [lsvmi_go_mem_heap_bytes](#lsvmi_go_mem_heap_bytes)
    - [lsvmi_go_mem_heap_sys_bytes](#lsvmi_go_mem_heap_sys_bytes)
    - [lsvmi_go_mem_sys_bytes](#lsvmi_go_mem_sys_bytes)
  - [Misc](#misc)
    - [lsvmi_internal_metrics_delta_sec](#lsvmi_internal_metrics_delta_sec)
    - [lsvmi_uptime_sec](#lsvmi_uptime_sec)
    - [os_info](#os_info)
    - [os_btime_sec](#os_btime_sec)
    - [os_uptime_sec](#os_uptime_sec)

<!-- /TOC -->
<!-- markdownlint-restore -->

## Information Applicable To All Metrics

### Common Labels

All metrics have the following labels:

- `instance` with the associated value identifying a specific [LSVMI](../README.md). The value, in decreasing order of precedence:
  - `-instance INSTANCE` command line arg
  - `global_config.instance` in config file
  - `lsvmi` built-in default
- `hostname` with the associated value identifying a host where [LSVMI](../README.md) runs. The value, in decreasing order of precedence:
  - `-hostname HOSTNAME` command line arg
  - the value returned by `hostname` syscall
  The value may be stripped of domain part, depending upon `global_config.use_short_hostname: true|false` config

## Internal Metrics

### Compressor Pool

  | Label Name | Value(s)/Info |
  | --- | --- |
  | instance | _instance_ |
  | hostname | _hostname_ |
  | compressor | _compressor#_ (0 .. num_compressor - 1) |

#### lsvmi_compressor_read_delta

The number of reads from the queue since last scan.

#### lsvmi_compressor_read_byte_delta

The number of read bytes from the queue since last scan.

#### lsvmi_compressor_send_delta

The number of sends since last scan.

#### lsvmi_compressor_send_byte_delta

The number of sent bytes since last scan.

#### lsvmi_compressor_send_error_delta

The number of send errors since last scan.

#### lsvmi_compressor_tout_flush_delta

The number of timeout (timed based, that is) flushes since last scan.

#### lsvmi_compressor_write_error_delta

The number of write (to compressor stream) errors since last scan.

#### lsvmi_compressor_compression_factor

The (exponentially decaying) compression factor average.

### Go Specific

  | Label Name | Value(s)/Info |
  | --- | --- |
  | instance | _instance_ |
  | hostname | _hostname_ |

#### lsvmi_go_mem_free_delta

The number of `free` calls since last scan.

#### lsvmi_go_mem_gc_delta

The number of garbage collector calls since last scan.

#### lsvmi_go_mem_malloc_delta

The number of `malloc` calls since last scan.

#### lsvmi_go_num_goroutine

The current number of goroutines.

#### lsvmi_go_mem_in_use_object_count

The current number of go objects in use.

#### lsvmi_go_mem_heap_bytes

#### lsvmi_go_mem_heap_sys_bytes

#### lsvmi_go_mem_sys_bytes

The size of various memory pools.

### Misc

#### lsvmi_internal_metrics_delta_sec

  The actual time delta, in seconds, since last internal metrics generation. This may be different than the scan `interval`, the latter is the desired, theoretical value.

  | Label Name | Value(s)/Info |
  | --- | --- |
  | instance | _instance_ |
  | hostname | _hostname_ |

#### lsvmi_uptime_sec

  Time, in seconds, since the agent was started.
  
  | Label Name | Value(s)/Info |
  | --- | --- |
  | instance | _instance_ |
  | hostname | _hostname_ |
  | version | semver of the agent |
  | gitinfo | git describe based |

#### os_info

  Categorical (constant `1`) with [uname](https://linux.die.net/man/1/uname) like info:

  | Label Name | Value(s)/Info |
  | --- | --- |
  | instance | _instance_ |
  | hostname | _hostname_ |
  | sys_name | \`uname -s\` |
  | sys_release | \`uname -r\` |
  | sys_version | \'uname -v\` |

#### os_btime_sec

  Boot time, in seconds.

  | Label Name | Value(s)/Info |
  | --- | --- |
  | instance | _instance_ |
  | hostname | _hostname_ |

#### os_uptime_sec

  Time, in seconds, since OS boot

  | Label Name | Value(s)/Info |
  | --- | --- |
  | instance | _instance_ |
  | hostname | _hostname_ |
