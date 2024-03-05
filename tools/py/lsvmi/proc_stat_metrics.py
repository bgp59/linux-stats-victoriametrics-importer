#! /usr/bin/env python3

# Generate test cases for lsvmi/proc_stat_metrics_test.go

import procfs

# CPU metrics, must match lsvmi/proc_stat_metrics.go:
PROC_STAT_CPU_USER_METRIC = "proc_stat_cpu_user_pct"
PROC_STAT_CPU_NICE_METRIC = "proc_stat_cpu_nice_pct"
PROC_STAT_CPU_SYSTEM_METRIC = "proc_stat_cpu_system_pct"
PROC_STAT_CPU_IDLE_METRIC = "proc_stat_cpu_idle_pct"
PROC_STAT_CPU_IOWAIT_METRIC = "proc_stat_cpu_iowait_pct"
PROC_STAT_CPU_IRQ_METRIC = "proc_stat_cpu_irq_pct"
PROC_STAT_CPU_SOFTIRQ_METRIC = "proc_stat_cpu_softirq_pct"
PROC_STAT_CPU_STEAL_METRIC = "proc_stat_cpu_steal_pct"
PROC_STAT_CPU_GUEST_METRIC = "proc_stat_cpu_guest_pct"
PROC_STAT_CPU_GUEST_NICE_METRIC = "proc_stat_cpu_guest_nice_pct"

PROC_STAT_CPU_LABEL_NAME = "cpu"

# Map procfs.Stat PROC_STAT_CPU_ index into metrics name, must match lsvmi/proc_stat_metrics.go:
proc_stat_cpu_index_metric_name_map = {
    procfs.STAT_CPU_USER_TICKS: PROC_STAT_CPU_USER_METRIC,
    procfs.STAT_CPU_NICE_TICKS: PROC_STAT_CPU_NICE_METRIC,
    procfs.STAT_CPU_SYSTEM_TICKS: PROC_STAT_CPU_SYSTEM_METRIC,
    procfs.STAT_CPU_IDLE_TICKS: PROC_STAT_CPU_IDLE_METRIC,
    procfs.STAT_CPU_IOWAIT_TICKS: PROC_STAT_CPU_IOWAIT_METRIC,
    procfs.STAT_CPU_IRQ_TICKS: PROC_STAT_CPU_IRQ_METRIC,
    procfs.STAT_CPU_SOFTIRQ_TICKS: PROC_STAT_CPU_SOFTIRQ_METRIC,
    procfs.STAT_CPU_STEAL_TICKS: PROC_STAT_CPU_STEAL_METRIC,
    procfs.STAT_CPU_GUEST_TICKS: PROC_STAT_CPU_GUEST_METRIC,
    procfs.STAT_CPU_GUEST_NICE_TICKS: PROC_STAT_CPU_GUEST_NICE_METRIC,
}

