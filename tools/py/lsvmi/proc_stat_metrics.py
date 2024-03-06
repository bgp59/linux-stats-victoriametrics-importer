#! /usr/bin/env python3

# Generate test cases for lsvmi/proc_stat_metrics_test.go

import json
import os
import sys
import time
from copy import deepcopy
from typing import Dict, List, Optional, Tuple

import procfs

from . import (
    DEFAULT_TEST_HOSTNAME,
    DEFAULT_TEST_INSTANCE,
    HOSTNAME_LABEL_NAME,
    INSTANCE_LABEL_NAME,
    TEST_LINUX_CLKTCK_SEC,
    lsvmi_testcases_root,
)

ZeroPcpu = List[bool]
ZeroPcpuMap = Dict[int, ZeroPcpu]

DEFAULT_PROC_STAT_INTERVAL_SEC = 0.2
DEFAULT_PROC_STAT_FULL_METRICS_FACTOR = 15

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

# Map cpu into label value, must match lsvmi/proc_stat_metrics.go:
proc_stat_cpu_to_label_val = {
    procfs.STAT_CPU_ALL: "all",
}

testcases_file = "proc_stat.json"


def generate_proc_stat_metrics(
    crt_proc_stat: procfs.Stat,
    prev_proc_stat: Optional[procfs.Stat],
    crt_prom_ts: int,
    interval: Optional[float] = DEFAULT_PROC_STAT_INTERVAL_SEC,
    zero_pcpu_map: Optional[ZeroPcpuMap] = None,
    full_metrics: bool = False,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
) -> Tuple[List[str], Optional[ZeroPcpuMap]]:
    metrics = []
    pcpu_factor = TEST_LINUX_CLKTCK_SEC / interval * 100
    new_zero_pcpu_map = None
    if prev_proc_stat is not None:
        new_zero_pcpu_map = {}
        for cpu, crt_cpu_stats in crt_proc_stat["Cpu"].items():
            new_zero_pcpu_map[cpu] = [False] * procfs.STAT_CPU_NUM_STATS
            prev_cpu_stats = prev_proc_stat["Cpu"].get(cpu)
            if prev_cpu_stats is None:
                continue
            zero_pcpu = zero_pcpu_map.get(cpu) if zero_pcpu_map is not None else None
            if zero_pcpu is None:
                zero_pcpu = [False] * procfs.STAT_CPU_NUM_STATS
            cpu_label_val = proc_stat_cpu_to_label_val.get(cpu, cpu)
            for index, name in proc_stat_cpu_index_metric_name_map.items():
                delta_cpu_ticks = crt_cpu_stats[index] - prev_cpu_stats[index]
                if delta_cpu_ticks > 0 or full_metrics or not zero_pcpu[index]:
                    pcpu = delta_cpu_ticks * pcpu_factor
                    metrics.append(
                        f"{name}{{"
                        + ",".join(
                            [
                                f'{INSTANCE_LABEL_NAME}="{instance}"',
                                f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                                f'{PROC_STAT_CPU_LABEL_NAME}="{cpu_label_val}"',
                            ]
                        )
                        + f"}} {pcpu:.1f} {crt_prom_ts}"
                    )
                new_zero_pcpu_map[cpu][index] = delta_cpu_ticks == 0
    return metrics, new_zero_pcpu_map


def generate_proc_stat_metrics_test_case(
    name: str,
    crt_proc_stat: procfs.Stat,
    prev_proc_stat: Optional[procfs.Stat],
    ts: Optional[float] = None,
    cycle_num: int = 0,
    full_metrics_factor: int = DEFAULT_PROC_STAT_FULL_METRICS_FACTOR,
    interval: Optional[float] = DEFAULT_PROC_STAT_INTERVAL_SEC,
    zero_pcpu_map: Optional[ZeroPcpuMap] = None,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
) -> Dict:
    if ts is None:
        ts = time.time()
    crt_prom_ts = int(ts * 1000)
    prev_prom_ts = crt_prom_ts - int(interval * 1000)
    metrics, want_zero_pcpu_map = generate_proc_stat_metrics(
        crt_proc_stat,
        prev_proc_stat,
        crt_prom_ts,
        interval=interval,
        zero_pcpu_map=zero_pcpu_map,
        full_metrics=(cycle_num == 0),
        hostname=hostname,
        instance=instance,
    )
    return {
        "Name": name,
        "Instance": instance,
        "Hostname": hostname,
        "CrtProcStat": crt_proc_stat,
        "PrevProcStat": prev_proc_stat,
        "CrtPromTs": crt_prom_ts,
        "PrevPromTs": prev_prom_ts,
        "CycleNum": cycle_num,
        "FullMetricsFactor": full_metrics_factor,
        "ZeroPcpuMap": zero_pcpu_map,
        "WantMetricsCount": len(metrics),
        "WantMetrics": metrics,
        "ReportExtra": True,
        "WantZeroPcpuMap": want_zero_pcpu_map,
        "LinuxClktckSec": TEST_LINUX_CLKTCK_SEC,
    }


def generate_proc_stat_metrics_test_cases(
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    testcases_root_dir: Optional[str] = lsvmi_testcases_root,
):
    ts = time.time()

    if testcases_root_dir not in {None, "", "-"}:
        out_file = os.path.join(testcases_root_dir, testcases_file)
        os.makedirs(os.path.dirname(out_file), exist_ok=True)
        fp = open(out_file, "wt")
    else:
        out_file = None
        fp = sys.stdout

    proc_stat_ref = {
        "Cpu": {
            procfs.STAT_CPU_ALL: [20, 21, 22, 23, 24, 25, 26, 27, 28, 29],
        }
    }

    test_cases = []
    tc_num = 0

    crt_proc_stat = {
        "Cpu": {},
    }
    i = 0
    for cpu, cpu_stats in proc_stat_ref["Cpu"].items():
        i += 1
        crt_proc_stat["Cpu"][cpu] = [tck + 20 * i for tck in cpu_stats]

    test_cases.append(
        generate_proc_stat_metrics_test_case(
            f"{tc_num:04d}",
            crt_proc_stat,
            proc_stat_ref,
            ts=ts,
            instance=instance,
            hostname=hostname,
        )
    )
    tc_num += 1

    json.dump(test_cases, fp=fp, indent=2)
    fp.write("\n")
    if out_file is not None:
        fp.close()
        print(f"{out_file} generated", file=sys.stderr)
