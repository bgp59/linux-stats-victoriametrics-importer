# LSVMI Interrupts Metrics (id: `proc_interrupts_metrics`)

<!-- TOC tocDepth:2..3 chapterDepth:2..6 -->

- [Metrics](#metrics)
  - [proc_interrupts_delta](#proc_interrupts_delta)
  - [proc_interrupts_info](#proc_interrupts_info)
  - [proc_interrupts_metrics_delta_sec](#proc_interrupts_metrics_delta_sec)

<!-- /TOC -->

## Metrics

### proc_interrupts_delta

Number of interrupts since the last scan. Based on [/proc/interrupts](https://man7.org/linux/man-pages/man5/proc_interrupts.5.html) and [What is this column in /proc/interrupts?](https://serverfault.com/questions/896551/what-is-this-column-in-proc-interrupts).

| Label Name | Value(s)/Info |
| --- | --- |
| instance | _instance_ |
| hostname | _hostname_ |
| irq | _IRQ_ number or name, e.g. `123` or `NMI` |
| dev | _device\[,device,device,...,device\]_ where applicable, e.g. `i8042`|
| cpu | _CPU\#_ handling the interrupts |

### proc_interrupts_info

[Pseudo-categorical](internals.md#pseudo-categorical-metrics ) metric containing information about the interrupts. Applicable only to IO devices, i.e. number IRQs.

| Label Name | Value(s)/Info |
| --- | --- |
| instance | _instance_ |
| hostname | _hostname_ |
| irq | _IRQ_ number, e.g. `123` |
| controller | _controller_ AKA chip, e.g. `IR-IO-APIC` |
| hw_interrupt | H/W _IRQ-type_, e.g. `8-edge` |
| dev | _device\[,device,device,...,device\]_ , e.g. `i8042`|

### proc_interrupts_metrics_delta_sec

Time in seconds since the last scan. The actual value corresponding to the configured desired `interval`.

| Label Name | Value(s)/Info |
| --- | --- |
| instance | _instance_ |
| hostname | _hostname_ |
