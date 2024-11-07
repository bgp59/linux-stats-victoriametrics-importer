# LSVMI Softirqs Metrics (id: `proc_softirqs_metrics`)

<!-- TOC tocDepth:2..3 chapterDepth:2..6 -->

- [General Information](#general-information)
- [Metrics](#metrics)
  - [proc_softirqs_delta](#proc_softirqs_delta)
  - [proc_softirqs_info](#proc_softirqs_info)
  - [proc_softirqs_metrics_delta_sec](#proc_softirqs_metrics_delta_sec)

<!-- /TOC -->

## General Information

Based on [/proc/softirqs](https://docs.kernel.org/filesystems/proc.html#softirqs)

## Metrics

### proc_softirqs_delta

Number of interrupts per source, per CPU since the last scan.

| Label Name | Value(s)/Info |
| --- | --- |
| instance | _instance_ |
| hostname | _hostname_ |
| irq | _SOFTIRQ_, e.g. `TIMER` |
| cpu | _CPU\#_ handling the interrupts |

### proc_softirqs_info

[Pseudo-categorical](internals.md#pseudo-categorical-metrics ) metric containing information about the interrupts. Useful for determining the list of _SOFTIRQ_'s without need to use a specific CPU.

| Label Name | Value(s)/Info |
| --- | --- |
| instance | _instance_ |
| hostname | _hostname_ |
| irq | _SOFTIRQ_, e.g. `TIMER` |

### proc_softirqs_metrics_delta_sec

Time in seconds since the last scan. The real life counterpart (i.e. measured value) to the desired (configured) `interval`.

| Label Name | Value(s)/Info |
| --- | --- |
| instance | _instance_ |
| hostname | _hostname_ |
