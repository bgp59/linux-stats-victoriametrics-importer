#! /usr/bin/env python3

# Generate test cases for lsvmi/proc_stat_metrics_test.go

import json
import os
import sys
import time
from copy import deepcopy
from dataclasses import asdict, dataclass
from typing import Dict, List, Optional, Tuple

import procfs

from . import (
    DEFAULT_TEST_HOSTNAME,
    DEFAULT_TEST_INSTANCE,
    HOSTNAME_LABEL_NAME,
    INSTANCE_LABEL_NAME,
    TEST_LINUX_CLKTCK_SEC,
    lsvmi_testcases_root,
    uint64_delta,
)

ZeroPcpu = List[bool]
ZeroPcpuMap = Dict[int, ZeroPcpu]


DEFAULT_PROC_STAT_INTERVAL_SEC = 0.2
DEFAULT_PROC_STAT_FULL_METRICS_FACTOR = 15
TEST_UPTIME_VALUE = 123.456789


# CPU metrics, must match lsvmi/proc_stat_metrics.go:

PROC_STAT_CPU_PCT_METRIC = "proc_stat_cpu_pct"

PROC_STAT_CPU_PCT_TYPE_LABEL_NAME = "type"
PROC_STAT_CPU_PCT_TYPE_USER = "user"
PROC_STAT_CPU_PCT_TYPE_NICE = "nice"
PROC_STAT_CPU_PCT_TYPE_SYSTEM = "system"
PROC_STAT_CPU_PCT_TYPE_IDLE = "idle"
PROC_STAT_CPU_PCT_TYPE_IOWAIT = "iowait"
PROC_STAT_CPU_PCT_TYPE_IRQ = "irq"
PROC_STAT_CPU_PCT_TYPE_SOFTIRQ = "softirq"
PROC_STAT_CPU_PCT_TYPE_STEAL = "steal"
PROC_STAT_CPU_PCT_TYPE_GUEST = "guest"
PROC_STAT_CPU_PCT_TYPE_GUEST_NICE = "guest_nice"

PROC_STAT_CPU_LABEL_NAME = "cpu"
PROC_STAT_CPU_ALL_LABEL_VALUE = "all"
PROC_STAT_CPU_AVG_LABEL_VALUE = "avg"

# Boot/up-time metrics:
PROC_STAT_BTIME_METRIC = "proc_stat_btime_sec"
PROC_STAT_UPTIME_METRIC = "proc_stat_uptime_sec"

# Other metrics:
PROC_STAT_PAGE_IN_DELTA_METRIC = "proc_stat_page_in_delta"
PROC_STAT_PAGE_OUT_DELTA_METRIC = "proc_stat_page_out_delta"
PROC_STAT_SWAP_IN_DELTA_METRIC = "proc_stat_swap_in_delta"
PROC_STAT_SWAP_OUT_DELTA_METRIC = "proc_stat_swap_out_delta"
PROC_STAT_CTXT_DELTA_METRIC = "proc_stat_ctxt_delta"
PROC_STAT_PROCESSES_DELTA_METRIC = "proc_stat_processes_delta"
PROC_STAT_PROCS_RUNNING_COUNT_METRIC = "proc_stat_procs_running_count"
PROC_STAT_PROCS_BLOCKED_COUNT_METRIC = "proc_stat_procs_blocked_count"

# Actual interval since last generation:
PROC_STAT_INTERVAL_METRIC_NAME = "proc_stat_metrics_delta_sec"


@dataclass
class ProcStatMetricsCpuInfoTestData:
    CycleNum: int = 0
    ZeroPcpu: Optional[List[bool]] = None


@dataclass
class ProcStatMetricsTestCase:
    Name: Optional[str] = None
    Description: Optional[str] = None
    Instance: Optional[str] = None
    Hostname: Optional[str] = None
    FullMetricsFactor: int = DEFAULT_PROC_STAT_FULL_METRICS_FACTOR
    CurrProcStat: Optional[procfs.Stat] = None
    PrevProcStat: Optional[procfs.Stat] = None
    CurrPromTs: int = 0
    PrevPromTs: int = 0
    CpuInfo: Optional[Dict[int, ProcStatMetricsCpuInfoTestData]] = None
    OtherCycleNum: int = 0
    OtherZeroDelta: Optional[List[bool]] = None
    WantMetricsCount: int = 0
    WantMetrics: Optional[List[str]] = None
    ReportExtra: bool = False
    WantZeroPcpuMap: Optional[Dict[int, List[bool]]] = None
    WantOtherZeroDelta: Optional[List[bool]] = None
    LinuxClktckSec: float = TEST_LINUX_CLKTCK_SEC
    TimeSinceBtime: float = TEST_UPTIME_VALUE


# Map procfs.Stat PROC_STAT_CPU_ indexes into type label value:
proc_stat_cpu_index_type_label_val_map = {
    procfs.STAT_CPU_USER_TICKS: PROC_STAT_CPU_PCT_TYPE_USER,
    procfs.STAT_CPU_NICE_TICKS: PROC_STAT_CPU_PCT_TYPE_NICE,
    procfs.STAT_CPU_SYSTEM_TICKS: PROC_STAT_CPU_PCT_TYPE_SYSTEM,
    procfs.STAT_CPU_IDLE_TICKS: PROC_STAT_CPU_PCT_TYPE_IDLE,
    procfs.STAT_CPU_IOWAIT_TICKS: PROC_STAT_CPU_PCT_TYPE_IOWAIT,
    procfs.STAT_CPU_IRQ_TICKS: PROC_STAT_CPU_PCT_TYPE_IRQ,
    procfs.STAT_CPU_SOFTIRQ_TICKS: PROC_STAT_CPU_PCT_TYPE_SOFTIRQ,
    procfs.STAT_CPU_STEAL_TICKS: PROC_STAT_CPU_PCT_TYPE_STEAL,
    procfs.STAT_CPU_GUEST_TICKS: PROC_STAT_CPU_PCT_TYPE_GUEST,
    procfs.STAT_CPU_GUEST_NICE_TICKS: PROC_STAT_CPU_PCT_TYPE_GUEST_NICE,
}

# Map procfs.NumericFields indexes into delta metrics name:
proc_stat_index_delta_metric_name_map = {
    procfs.STAT_PAGE_IN: PROC_STAT_PAGE_IN_DELTA_METRIC,
    procfs.STAT_PAGE_OUT: PROC_STAT_PAGE_OUT_DELTA_METRIC,
    procfs.STAT_SWAP_IN: PROC_STAT_SWAP_IN_DELTA_METRIC,
    procfs.STAT_SWAP_OUT: PROC_STAT_SWAP_OUT_DELTA_METRIC,
    procfs.STAT_CTXT: PROC_STAT_CTXT_DELTA_METRIC,
    procfs.STAT_PROCESSES: PROC_STAT_PROCESSES_DELTA_METRIC,
}

# Map procfs.NumericFields indexes into metrics name:
proc_stat_index_metric_name_map = {
    procfs.STAT_PROCS_RUNNING: PROC_STAT_PROCS_RUNNING_COUNT_METRIC,
    procfs.STAT_PROCS_BLOCKED: PROC_STAT_PROCS_BLOCKED_COUNT_METRIC,
}

proc_stat_relevant_numeric_field_indexes = list(
    proc_stat_index_delta_metric_name_map
) + list(proc_stat_index_metric_name_map)


testcases_file = "proc_stat.json"


def generate_proc_stat_metrics(
    curr_proc_stat: procfs.Stat,
    prev_proc_stat: procfs.Stat,
    curr_prom_ts: int,
    interval: Optional[float] = DEFAULT_PROC_STAT_INTERVAL_SEC,
    cpu_info: Optional[Dict[int, ProcStatMetricsCpuInfoTestData]] = None,
    other_cycle_num: int = 0,
    other_zero_delta: Optional[List[bool]] = None,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
) -> Tuple[List[str], Dict[int, List[bool]], List[bool]]:
    metrics = []
    pcpu_factor = TEST_LINUX_CLKTCK_SEC / interval * 100

    new_zero_pcpu_map = {}
    new_other_zero_delta = [False] * procfs.STAT_NUMERIC_NUM_STATS

    # CPU stats:
    num_cpus = len(curr_proc_stat.Cpu)
    for cpu, curr_cpu_stats in curr_proc_stat.Cpu.items():
        new_zero_pcpu_map[cpu] = [False] * procfs.STAT_CPU_NUM_STATS
        prev_cpu_stats = prev_proc_stat.Cpu.get(cpu)
        if prev_cpu_stats is None:
            continue
        full_metrics = (
            cpu_info is None or cpu not in cpu_info or cpu_info[cpu].CycleNum == 0
        )
        cpu_label_val = (
            cpu if cpu != procfs.STAT_CPU_ALL else PROC_STAT_CPU_ALL_LABEL_VALUE
        )
        for index, type_label_val in proc_stat_cpu_index_type_label_val_map.items():
            delta_cpu_ticks = uint64_delta(curr_cpu_stats[index], prev_cpu_stats[index])
            if delta_cpu_ticks > 0 or full_metrics or not cpu_info[cpu].ZeroPcpu[index]:
                pcpu = delta_cpu_ticks * pcpu_factor
                metrics.append(
                    f"{PROC_STAT_CPU_PCT_METRIC}{{"
                    + ",".join(
                        [
                            f'{INSTANCE_LABEL_NAME}="{instance}"',
                            f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                            f'{PROC_STAT_CPU_PCT_TYPE_LABEL_NAME}="{type_label_val}"',
                            f'{PROC_STAT_CPU_LABEL_NAME}="{cpu_label_val}"',
                        ]
                    )
                    + f"}} {pcpu:.1f} {curr_prom_ts}"
                )
                if cpu == procfs.STAT_CPU_ALL and num_cpus > 0:
                    metrics.append(
                        f"{PROC_STAT_CPU_PCT_METRIC}{{"
                        + ",".join(
                            [
                                f'{INSTANCE_LABEL_NAME}="{instance}"',
                                f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                                f'{PROC_STAT_CPU_PCT_TYPE_LABEL_NAME}="{type_label_val}"',
                                f'{PROC_STAT_CPU_LABEL_NAME}="{PROC_STAT_CPU_AVG_LABEL_VALUE}"',
                            ]
                        )
                        + f"}} {pcpu/num_cpus:.1f} {curr_prom_ts}"
                    )

            new_zero_pcpu_map[cpu][index] = delta_cpu_ticks == 0

    # Other metrics:
    curr_numeric_fields, prev_numeric_fields = (
        curr_proc_stat.NumericFields,
        prev_proc_stat.NumericFields,
    )

    # Other metrics - deltas:
    for index, name in proc_stat_index_delta_metric_name_map.items():
        delta = uint64_delta(curr_numeric_fields[index], prev_numeric_fields[index])
        if (
            delta != 0
            or other_zero_delta is None
            or other_cycle_num == 0
            or not other_zero_delta[index]
        ):
            metrics.append(
                f"{name}{{"
                + ",".join(
                    [
                        f'{INSTANCE_LABEL_NAME}="{instance}"',
                        f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                    ]
                )
                + f"}} {delta} {curr_prom_ts}"
            )
        new_other_zero_delta[index] = delta == 0

    # Other metrics - non-deltas:
    for index, name in proc_stat_index_metric_name_map.items():
        val = curr_numeric_fields[index]
        if (
            val != prev_numeric_fields[index]
            or other_zero_delta is None
            or other_cycle_num == 0
        ):
            metrics.append(
                f"{name}{{"
                + ",".join(
                    [
                        f'{INSTANCE_LABEL_NAME}="{instance}"',
                        f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                    ]
                )
                + f"}} {val} {curr_prom_ts}"
            )

    # Boot/up-time metrics:
    if other_cycle_num == 0 or other_cycle_num == 0:
        btime = curr_numeric_fields[procfs.STAT_BTIME]
        metrics.append(
            f"{PROC_STAT_BTIME_METRIC}{{"
            + ",".join(
                [
                    f'{INSTANCE_LABEL_NAME}="{instance}"',
                    f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                ]
            )
            + f"}} {int(btime)} {curr_prom_ts}"
        )
        metrics.append(
            f"{PROC_STAT_UPTIME_METRIC}{{"
            + ",".join(
                [
                    f'{INSTANCE_LABEL_NAME}="{instance}"',
                    f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                ]
            )
            + f"}} {TEST_UPTIME_VALUE:.03f} {curr_prom_ts}"
        )

    # Interval:
    metrics.append(
        f"{PROC_STAT_INTERVAL_METRIC_NAME}{{"
        + ",".join(
            [
                f'{INSTANCE_LABEL_NAME}="{instance}"',
                f'{HOSTNAME_LABEL_NAME}="{hostname}"',
            ]
        )
        + f"}} {interval:.06f} {curr_prom_ts}"
    )

    return metrics, new_zero_pcpu_map, new_other_zero_delta


def generate_proc_stat_metrics_test_case(
    name: str,
    curr_proc_stat: procfs.Stat,
    prev_proc_stat: procfs.Stat,
    ts: Optional[float] = None,
    interval: Optional[float] = DEFAULT_PROC_STAT_INTERVAL_SEC,
    cpu_info: Optional[Dict[int, ProcStatMetricsCpuInfoTestData]] = None,
    other_cycle_num: int = 0,
    other_zero_delta: Optional[List[bool]] = None,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    description: Optional[str] = None,
    full_metrics_factor: int = DEFAULT_PROC_STAT_FULL_METRICS_FACTOR,
) -> Dict:
    if ts is None:
        ts = time.time()
    curr_prom_ts = int(ts * 1000)
    prev_prom_ts = curr_prom_ts - int(interval * 1000)
    metrics, want_zero_pcpu_map, want_other_zero_delta = generate_proc_stat_metrics(
        curr_proc_stat,
        prev_proc_stat,
        curr_prom_ts,
        interval=interval,
        cpu_info=cpu_info,
        other_cycle_num=other_cycle_num,
        other_zero_delta=other_zero_delta,
        instance=instance,
        hostname=hostname,
    )
    return ProcStatMetricsTestCase(
        Name=name,
        Description=description,
        Instance=instance,
        Hostname=hostname,
        FullMetricsFactor=full_metrics_factor,
        CurrProcStat=curr_proc_stat,
        PrevProcStat=prev_proc_stat,
        CurrPromTs=curr_prom_ts,
        PrevPromTs=prev_prom_ts,
        CpuInfo=cpu_info,
        OtherCycleNum=other_cycle_num,
        OtherZeroDelta=other_zero_delta,
        WantMetricsCount=len(metrics),
        WantMetrics=metrics,
        ReportExtra=True,
        WantZeroPcpuMap=want_zero_pcpu_map,
        WantOtherZeroDelta=want_other_zero_delta,
    )


def make_ref_proc_stat(num_cpus: int = 2, ts: Optional[float] = None) -> procfs.Stat:
    stat = procfs.Stat(
        Cpu={},
        NumericFields=[0] * procfs.STAT_NUMERIC_NUM_STATS,
    )
    cpu_ticks_all = [0] * procfs.STAT_CPU_NUM_STATS
    for cpu in range(num_cpus):
        cpu_ticks = [0] * procfs.STAT_CPU_NUM_STATS
        for i in range(procfs.STAT_CPU_NUM_STATS):
            cpu_ticks[i] = 2 * procfs.STAT_CPU_NUM_STATS * cpu + i
            cpu_ticks_all[i] += cpu_ticks[i]
        stat.Cpu[cpu] = cpu_ticks
    stat.Cpu[procfs.STAT_CPU_ALL] = cpu_ticks_all

    if ts is None:
        ts = time.time()
    stat.NumericFields[procfs.STAT_BTIME] = int(ts - TEST_UPTIME_VALUE)

    for i in proc_stat_relevant_numeric_field_indexes:
        stat.NumericFields[i] = 2 * procfs.STAT_CPU_NUM_STATS * i

    return stat


def generate_proc_stat_metrics_test_cases(
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    testcases_root_dir: Optional[str] = lsvmi_testcases_root,
):

    if testcases_root_dir not in {None, "", "-"}:
        out_file = os.path.join(testcases_root_dir, testcases_file)
        os.makedirs(os.path.dirname(out_file), exist_ok=True)
        fp = open(out_file, "wt")
    else:
        out_file = None
        fp = sys.stdout

    num_cpus = 2
    ts = time.time()
    proc_stat_ref = make_ref_proc_stat(num_cpus=num_cpus, ts=ts)
    cpu_ticks_max = 0
    for cpu_ticks in proc_stat_ref.Cpu.values():
        cpu_ticks_max = max(cpu_ticks_max, *cpu_ticks)
    numeric_fields_max = max(
        proc_stat_ref.NumericFields[i] for i in proc_stat_relevant_numeric_field_indexes
    )

    test_cases = []
    tc_num = 0

    # First time:
    name = "first_time"
    curr_proc_stat = proc_stat_ref
    prev_proc_stat = deepcopy(proc_stat_ref)
    for cpu, cpu_ticks in prev_proc_stat.Cpu.items():
        for i in range(len(cpu_ticks)):
            cpu_ticks[i] = uint64_delta(
                cpu_ticks[i], procfs.STAT_CPU_NUM_STATS * (cpu_ticks_max + cpu) + i
            )
    for i in proc_stat_relevant_numeric_field_indexes:
        prev_proc_stat.NumericFields[i] = uint64_delta(
            prev_proc_stat.NumericFields[i], 2 * numeric_fields_max + i
        )
    test_cases.append(
        generate_proc_stat_metrics_test_case(
            name,
            curr_proc_stat,
            prev_proc_stat,
            ts=ts,
            instance=instance,
            hostname=hostname,
        )
    )
    tc_num += 1

    # proc_stat_ref = {
    #     "Cpu": {},
    #     "NumericFields": [0] * procfs.STAT_NUMERIC_NUM_STATS,
    # }
    # cpu_ticks_all = [0] * procfs.STAT_CPU_NUM_STATS
    # num_cpus = 2
    # for cpu in range(num_cpus):
    #     cpu_ticks = [0] * procfs.STAT_CPU_NUM_STATS
    #     for i in range(procfs.STAT_CPU_NUM_STATS):
    #         cpu_ticks[i] = 2 * procfs.STAT_CPU_NUM_STATS * cpu + i
    #         cpu_ticks_all[i] += cpu_ticks[i]
    #     proc_stat_ref["Cpu"][cpu] = cpu_ticks
    # proc_stat_ref["Cpu"][procfs.STAT_CPU_ALL] = cpu_ticks_all
    # proc_stat_ref["NumericFields"][procfs.STAT_BTIME] = int(ts)
    # for i in proc_stat_index_delta_metric_name_map:
    #     proc_stat_ref["NumericFields"][i] = 2 * procfs.STAT_CPU_NUM_STATS * i
    # for i in proc_stat_index_metric_name_map:
    #     proc_stat_ref["NumericFields"][i] = 2 * procfs.STAT_CPU_NUM_STATS * i

    # test_cases = []
    # tc_num = 0

    # # No previous stats:
    # for cycle_num in [0, 1]:
    #     for zero in [False, True]:
    #         zero_pcpu_map = {}
    #         for cpu in proc_stat_ref["Cpu"]:
    #             zero_pcpu_map[cpu] = [zero] * procfs.STAT_CPU_NUM_STATS
    #         test_cases.append(
    #             generate_proc_stat_metrics_test_case(
    #                 f"{tc_num:04d}",
    #                 proc_stat_ref,
    #                 None,
    #                 ts=ts,
    #                 cycle_num=cycle_num,
    #                 zero_pcpu_map=zero_pcpu_map,
    #                 instance=instance,
    #                 hostname=hostname,
    #             )
    #         )
    #         tc_num += 1

    # # No stats change:
    # for cycle_num in [0, 1]:
    #     for zero in [False, True]:
    #         zero_pcpu_map = {}
    #         for cpu in proc_stat_ref["Cpu"]:
    #             zero_pcpu_map[cpu] = [zero] * procfs.STAT_CPU_NUM_STATS
    #         test_cases.append(
    #             generate_proc_stat_metrics_test_case(
    #                 f"{tc_num:04d}",
    #                 proc_stat_ref,
    #                 proc_stat_ref,
    #                 ts=ts,
    #                 cycle_num=cycle_num,
    #                 zero_pcpu_map=zero_pcpu_map,
    #                 instance=instance,
    #                 hostname=hostname,
    #             )
    #         )
    #         tc_num += 1

    # # Single CPU stat change:
    # for cycle_num in [0, 1]:
    #     for zero in [False, True]:
    #         zero_pcpu_map = {}
    #         for cpu in proc_stat_ref["Cpu"]:
    #             zero_pcpu_map[cpu] = [zero] * procfs.STAT_CPU_NUM_STATS
    #         for cpu in proc_stat_ref["Cpu"]:
    #             for i in range(procfs.STAT_CPU_NUM_STATS):
    #                 curr_proc_stat = deepcopy(proc_stat_ref)
    #                 curr_proc_stat["Cpu"][cpu][i] += i + 13
    #                 test_cases.append(
    #                     generate_proc_stat_metrics_test_case(
    #                         f"{tc_num:04d}",
    #                         curr_proc_stat,
    #                         proc_stat_ref,
    #                         ts=ts,
    #                         cycle_num=cycle_num,
    #                         zero_pcpu_map=zero_pcpu_map,
    #                         instance=instance,
    #                         hostname=hostname,
    #                     )
    #                 )
    #                 tc_num += 1

    # # New CPU:
    # curr_proc_stat = deepcopy(proc_stat_ref)
    # curr_proc_stat["Cpu"][num_cpus] = [i * 10 for i in range(procfs.STAT_CPU_NUM_STATS)]
    # for cycle_num in [0, 1]:
    #     for zero in [False, True]:
    #         zero_pcpu_map = {}
    #         for cpu in proc_stat_ref["Cpu"]:
    #             zero_pcpu_map[cpu] = [zero] * procfs.STAT_CPU_NUM_STATS
    #         test_cases.append(
    #             generate_proc_stat_metrics_test_case(
    #                 f"{tc_num:04d}",
    #                 curr_proc_stat,
    #                 proc_stat_ref,
    #                 ts=ts,
    #                 cycle_num=cycle_num,
    #                 zero_pcpu_map=zero_pcpu_map,
    #                 instance=instance,
    #                 hostname=hostname,
    #             )
    #         )
    #         tc_num += 1

    # # Vanishing CPU:
    # for cycle_num in [0, 1]:
    #     for zero in [False, True]:
    #         zero_pcpu_map = {}
    #         for cpu in proc_stat_ref["Cpu"]:
    #             zero_pcpu_map[cpu] = [zero] * procfs.STAT_CPU_NUM_STATS
    #         for cpu in proc_stat_ref["Cpu"]:
    #             if cpu == procfs.STAT_CPU_ALL:
    #                 continue
    #             curr_proc_stat = deepcopy(proc_stat_ref)
    #             del curr_proc_stat["Cpu"][cpu]
    #             test_cases.append(
    #                 generate_proc_stat_metrics_test_case(
    #                     f"{tc_num:04d}",
    #                     curr_proc_stat,
    #                     proc_stat_ref,
    #                     ts=ts,
    #                     cycle_num=cycle_num,
    #                     zero_pcpu_map=zero_pcpu_map,
    #                     instance=instance,
    #                     hostname=hostname,
    #                 )
    #             )
    #             tc_num += 1

    # # Other stats change:
    # for cycle_num in [0, 1]:
    #     for i in list(proc_stat_index_delta_metric_name_map) + list(
    #         proc_stat_index_metric_name_map
    #     ):
    #         curr_proc_stat = deepcopy(proc_stat_ref)
    #         curr_proc_stat["NumericFields"][i] += 1000 * (i + 1)
    #         test_cases.append(
    #             generate_proc_stat_metrics_test_case(
    #                 f"{tc_num:04d}",
    #                 curr_proc_stat,
    #                 proc_stat_ref,
    #                 ts=ts,
    #                 cycle_num=cycle_num,
    #                 zero_pcpu_map=zero_pcpu_map,
    #                 instance=instance,
    #                 hostname=hostname,
    #             )
    #         )
    #         tc_num += 1

    json.dump(list(map(asdict, test_cases)), fp=fp, indent=2)
    fp.write("\n")
    if out_file is not None:
        fp.close()
        print(f"{out_file} generated", file=sys.stderr)
