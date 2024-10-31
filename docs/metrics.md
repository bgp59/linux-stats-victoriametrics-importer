# LSVMI Metrics

<!-- TOC tocDepth:2..4 chapterDepth:2..6 -->

- [Information Applicable To All Metrics](#information-applicable-to-all-metrics)
  - [Common Labels](#common-labels)

<!-- /TOC -->

<!-- internal_metrics.md -->
- [LSVMI Internal Metrics](internal_metrics.md#lsvmi-internal-metrics)
  - [Agent Metrics](internal_metrics.md#agent-metrics)
    - [lsvmi_internal_metrics_delta_sec](internal_metrics.md#lsvmi_internal_metrics_delta_sec)
    - [lsvmi_uptime_sec](internal_metrics.md#lsvmi_uptime_sec)
    - [lsvmi_proc_num_threads](internal_metrics.md#lsvmi_proc_num_threads)
    - [lsvmi_proc_pcpu](internal_metrics.md#lsvmi_proc_pcpu)
    - [lsvmi_proc_rss](internal_metrics.md#lsvmi_proc_rss)
    - [lsvmi_proc_vsize](internal_metrics.md#lsvmi_proc_vsize)
  - [Compressor Pool Metrics](internal_metrics.md#compressor-pool-metrics)
    - [lsvmi_compressor_read_delta](internal_metrics.md#lsvmi_compressor_read_delta)
    - [lsvmi_compressor_read_byte_delta](internal_metrics.md#lsvmi_compressor_read_byte_delta)
    - [lsvmi_compressor_send_delta](internal_metrics.md#lsvmi_compressor_send_delta)
    - [lsvmi_compressor_send_byte_delta](internal_metrics.md#lsvmi_compressor_send_byte_delta)
    - [lsvmi_compressor_send_error_delta](internal_metrics.md#lsvmi_compressor_send_error_delta)
    - [lsvmi_compressor_tout_flush_delta](internal_metrics.md#lsvmi_compressor_tout_flush_delta)
    - [lsvmi_compressor_write_error_delta](internal_metrics.md#lsvmi_compressor_write_error_delta)
    - [lsvmi_compressor_compression_factor](internal_metrics.md#lsvmi_compressor_compression_factor)
  - [Generator Metrics](internal_metrics.md#generator-metrics)
    - [lsvmi_metrics_gen_invocation_delta](internal_metrics.md#lsvmi_metrics_gen_invocation_delta)
    - [lsvmi_metrics_gen_actual_metrics_delta](internal_metrics.md#lsvmi_metrics_gen_actual_metrics_delta)
    - [lsvmi_metrics_gen_total_metrics_delta](internal_metrics.md#lsvmi_metrics_gen_total_metrics_delta)
    - [lsvmi_metrics_gen_bytes_delta](internal_metrics.md#lsvmi_metrics_gen_bytes_delta)
  - [Go Specific Metrics](internal_metrics.md#go-specific-metrics)
    - [lsvmi_go_mem_free_delta](internal_metrics.md#lsvmi_go_mem_free_delta)
    - [lsvmi_go_mem_gc_delta](internal_metrics.md#lsvmi_go_mem_gc_delta)
    - [lsvmi_go_mem_malloc_delta](internal_metrics.md#lsvmi_go_mem_malloc_delta)
    - [lsvmi_go_num_goroutine](internal_metrics.md#lsvmi_go_num_goroutine)
    - [lsvmi_go_mem_in_use_object_count](internal_metrics.md#lsvmi_go_mem_in_use_object_count)
    - [lsvmi_go_mem_heap_bytes](internal_metrics.md#lsvmi_go_mem_heap_bytes)
    - [lsvmi_go_mem_heap_sys_bytes](internal_metrics.md#lsvmi_go_mem_heap_sys_bytes)
    - [lsvmi_go_mem_sys_bytes](internal_metrics.md#lsvmi_go_mem_sys_bytes)
  - [HTTP Endpoint Pool Metrics](internal_metrics.md#http-endpoint-pool-metrics)
    - [Per Endpoint Metrics](internal_metrics.md#per-endpoint-metrics)
      - [lsvmi_http_ep_send_buffer_delta](internal_metrics.md#lsvmi_http_ep_send_buffer_delta)
      - [lsvmi_http_ep_send_buffer_byte_delta](internal_metrics.md#lsvmi_http_ep_send_buffer_byte_delta)
      - [lsvmi_http_ep_send_buffer_error_delta](internal_metrics.md#lsvmi_http_ep_send_buffer_error_delta)
      - [lsvmi_http_ep_healthcheck_delta](internal_metrics.md#lsvmi_http_ep_healthcheck_delta)
      - [lsvmi_http_ep_healthcheck_error_delta](internal_metrics.md#lsvmi_http_ep_healthcheck_error_delta)
      - [lsvmi_http_ep_state](internal_metrics.md#lsvmi_http_ep_state)
    - [Per Pool Metrics](internal_metrics.md#per-pool-metrics)
      - [lsvmi_http_ep_pool_healthy_rotate_count](internal_metrics.md#lsvmi_http_ep_pool_healthy_rotate_count)
      - [lsvmi_http_ep_pool_no_healthy_ep_error_delta](internal_metrics.md#lsvmi_http_ep_pool_no_healthy_ep_error_delta)
  - [OS Metrics](internal_metrics.md#os-metrics)
    - [os_info](internal_metrics.md#os_info)
    - [os_btime_sec](internal_metrics.md#os_btime_sec)
    - [os_uptime_sec](internal_metrics.md#os_uptime_sec)
  - [Scheduler Metrics](internal_metrics.md#scheduler-metrics)
    - [lsvmi_task_scheduled_delta](internal_metrics.md#lsvmi_task_scheduled_delta)
    - [lsvmi_task_delayed_delta](internal_metrics.md#lsvmi_task_delayed_delta)
    - [lsvmi_task_overrun_delta](internal_metrics.md#lsvmi_task_overrun_delta)
    - [lsvmi_task_executed_delta](internal_metrics.md#lsvmi_task_executed_delta)
    - [lsvmi_task_deadline_hack_delta](internal_metrics.md#lsvmi_task_deadline_hack_delta)
    - [lsvmi_task_interval_avg_runtime_sec](internal_metrics.md#lsvmi_task_interval_avg_runtime_sec)

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
