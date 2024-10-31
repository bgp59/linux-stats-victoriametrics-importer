# LSVMI Metrics

<!-- markdownlint-disable -->
<!-- TOC tocDepth:2..4 chapterDepth:2..6 -->

- [Information Applicable To All Metrics](#information-applicable-to-all-metrics)
  - [Common Labels](#common-labels)
- [Internal Metrics](internal_metrics.md)
  - [Compressor Pool](internal_metrics.md#compressor-pool)
    - [lsvmi_compressor_read_delta](internal_metrics.md#lsvmi_compressor_read_delta)
    - [lsvmi_compressor_read_byte_delta](internal_metrics.md#lsvmi_compressor_read_byte_delta)
    - [lsvmi_compressor_send_delta](internal_metrics.md#lsvmi_compressor_send_delta)
    - [lsvmi_compressor_send_byte_delta](internal_metrics.md#lsvmi_compressor_send_byte_delta)
    - [lsvmi_compressor_send_error_delta](internal_metrics.md#lsvmi_compressor_send_error_delta)
    - [lsvmi_compressor_tout_flush_delta](internal_metrics.md#lsvmi_compressor_tout_flush_delta)
    - [lsvmi_compressor_write_error_delta](internal_metrics.md#lsvmi_compressor_write_error_delta)
    - [lsvmi_compressor_compression_factor](internal_metrics.md#lsvmi_compressor_compression_factor)
  - [Go Specific](internal_metrics.md#go-specific)
    - [lsvmi_go_mem_free_delta](internal_metrics.md#lsvmi_go_mem_free_delta)
    - [lsvmi_go_mem_gc_delta](internal_metrics.md#lsvmi_go_mem_gc_delta)
    - [lsvmi_go_mem_malloc_delta](internal_metrics.md#lsvmi_go_mem_malloc_delta)
    - [lsvmi_go_num_goroutine](internal_metrics.md#lsvmi_go_num_goroutine)
    - [lsvmi_go_mem_in_use_object_count](internal_metrics.md#lsvmi_go_mem_in_use_object_count)
    - [lsvmi_go_mem_heap_bytes](internal_metrics.md#lsvmi_go_mem_heap_bytes)
    - [lsvmi_go_mem_heap_sys_bytes](internal_metrics.md#lsvmi_go_mem_heap_sys_bytes)
    - [lsvmi_go_mem_sys_bytes](internal_metrics.md#lsvmi_go_mem_sys_bytes)
  - [Misc](internal_metrics.md#misc)
    - [lsvmi_internal_metrics_delta_sec](internal_metrics.md#lsvmi_internal_metrics_delta_sec)
    - [lsvmi_uptime_sec](internal_metrics.md#lsvmi_uptime_sec)
    - [os_info](internal_metrics.md#os_info)
    - [os_btime_sec](internal_metrics.md#os_btime_sec)
    - [os_uptime_sec](internal_metrics.md#os_uptime_sec)

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
