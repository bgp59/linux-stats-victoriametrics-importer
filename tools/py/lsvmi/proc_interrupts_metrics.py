#! /usr/bin/env python3

# Generate test cases for lsvmi/proc_interrupts_metrics_test.go

import json
import os
import re
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
    lsvmi_testcases_root,
    uint64_delta,
)

DEFAULT_PROC_INTERRUPTS_INTERVAL_SEC = 1
DEFAULT_PROC_INTERRUPTS_FULL_METRICS_FACTOR = 15

PROC_INTERRUPTS_DELTA_METRIC = "proc_interrupts_delta"
PROC_INTERRUPTS_IRQ_LABEL_NAME = "irq"
PROC_INTERRUPTS_CPU_LABEL_NAME = "cpu"

PROC_INTERRUPTS_INFO_METRIC = "proc_interrupts_info"
PROC_INTERRUPTS_INFO_IRQ_LABEL_NAME = PROC_INTERRUPTS_IRQ_LABEL_NAME
PROC_INTERRUPTS_INFO_CONTROLLER_LABEL_NAME = "controller"
PROC_INTERRUPTS_INFO_HW_INTERRUPT_LABEL_NAME = "hw_interrupt"
PROC_INTERRUPTS_INFO_DEV_LABEL_NAME = "dev"

PROC_INTERRUPTS_INTERVAL_METRIC_NAME = "proc_interrupts_metrics_delta_sec"


ZeroDeltaType = List[bool]
ZeroDeltaMapType = Dict[str, ZeroDeltaType]


@dataclass
class ProcInterruptsMetricsTestCase:
    Name: Optional[str] = None
    Instance: Optional[str] = None
    Hostname: Optional[str] = None
    CrtProcInterrupts: Optional[procfs.Interrupts] = None
    PrevProcInterrupts: Optional[procfs.Interrupts] = None
    CrtPromTs: Optional[int] = None
    PrevPromTs: Optional[int] = None
    CycleNum: int = 0
    FullMetricsFactor: int = DEFAULT_PROC_INTERRUPTS_FULL_METRICS_FACTOR
    ZeroDeltaMap: Optional[ZeroDeltaMapType] = None
    InfoMetricsCache: Optional[Dict[str, str]] = None
    WantMetricsCount: int = 0
    WantMetrics: Optional[List[str]] = None
    ReportExtra: Optional[bool] = None
    WantZeroDeltaMap: Optional[ZeroDeltaMapType] = None


testcases_file = "proc_interrupts.json"


def interrupts_info_metric(
    instance: str,
    hostname: str,
    irq: str,
    irq_info: procfs.InterruptsIrqInfo,
) -> str:
    controller = irq_info.Controller or ""
    hw_interrupt = irq_info.HWInterrupt or ""
    devices = irq_info.Devices or ""

    return (
        f"{PROC_INTERRUPTS_INFO_METRIC}{{"
        + ",".join(
            [
                f'{INSTANCE_LABEL_NAME}="{instance}"',
                f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                f'{PROC_INTERRUPTS_IRQ_LABEL_NAME}="{irq}"',
                f'{PROC_INTERRUPTS_INFO_CONTROLLER_LABEL_NAME}="{controller}"',
                f'{PROC_INTERRUPTS_INFO_HW_INTERRUPT_LABEL_NAME}="{hw_interrupt}"',
                f'{PROC_INTERRUPTS_INFO_DEV_LABEL_NAME}="{devices}"',
            ]
        )
        + "} "
    )


def generate_info_metrics_cache(
    instance: str,
    hostname: str,
    proc_interrupts: procfs.Interrupts,
) -> Dict[str, str]:
    if proc_interrupts.Info is not None and proc_interrupts.Info.IrqInfo is not None:
        return {
            irq: interrupts_info_metric(instance, hostname, irq, irq_info)
            for irq, irq_info in proc_interrupts.Info.IrqInfo.items()
        }
    else:
        return {}


def generate_proc_interrupts_metrics(
    crt_proc_interrupts: procfs.Interrupts,
    prev_proc_interrupts: procfs.Interrupts,
    crt_prom_ts: int,
    interval: Optional[float] = DEFAULT_PROC_INTERRUPTS_INTERVAL_SEC,
    zero_delta_map: Optional[ZeroDeltaMapType] = None,
    prev_info_metrics_cache: Optional[Dict[str, str]] = None,
    full_metrics: bool = False,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
) -> Tuple[List[str], Optional[ZeroDeltaMapType]]:
    metrics = []
    new_zero_delta_map = {}

    # Build the mapping between the crt and prev counter index# using the
    # CpuList; 2 counters have to refer to the same CPU#.
    crt_cpu_list = crt_proc_interrupts.CpuList
    if not crt_cpu_list:
        crt_cpu_list = [i for i in range(crt_proc_interrupts.NumCounters)]
    prev_cpu_list = prev_proc_interrupts.CpuList
    if not prev_cpu_list:
        prev_cpu_list = [i for i in range(crt_proc_interrupts.NumCounters)]
    prev_cpu_to_index_map = {cpu: i for i, cpu in enumerate(prev_cpu_list)}
    crt_to_prev_counter_index_map = {
        i: prev_cpu_to_index_map[cpu]
        for i, cpu in enumerate(crt_cpu_list)
        if cpu in prev_cpu_to_index_map
    }
    cpu_list_changed = crt_cpu_list != prev_cpu_list
    full_metrics_delta = full_metrics or cpu_list_changed

    for irq, crt_counters in crt_proc_interrupts.Counters.items():
        prev_counters = prev_proc_interrupts.Counters.get(irq)
        if prev_counters is None:
            continue
        zero_delta = zero_delta_map.get(irq)
        if cpu_list_changed or zero_delta is None:
            zero_delta = [False] * len(crt_counters)
        new_zero_delta = [False] * len(crt_counters)
        for crt_i, crt_counter in enumerate(crt_counters):
            prev_i = crt_to_prev_counter_index_map.get(crt_i)
            if prev_i is None:
                continue
            delta = uint64_delta(crt_counter, prev_counters[prev_i])
            if full_metrics_delta or delta > 0 or not zero_delta[crt_i]:
                metrics.append(
                    f"{PROC_INTERRUPTS_DELTA_METRIC}{{"
                    + ",".join(
                        [
                            f'{INSTANCE_LABEL_NAME}="{instance}"',
                            f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                            f'{PROC_INTERRUPTS_IRQ_LABEL_NAME}="{irq}"',
                            f'{PROC_INTERRUPTS_CPU_LABEL_NAME}="{crt_cpu_list[crt_i]}"',
                        ]
                    )
                    + f"}} {delta} {crt_prom_ts}"
                )
            new_zero_delta[crt_i] = delta == 0
        new_zero_delta_map[irq] = new_zero_delta

    if (
        crt_proc_interrupts.Info is not None
        and crt_proc_interrupts.Info.IrqInfo is not None
    ):
        if not prev_info_metrics_cache:
            prev_info_metrics_cache = generate_info_metrics_cache(
                instance,
                hostname,
                prev_proc_interrupts,
            )
        for irq, irq_info in crt_proc_interrupts.Info.IrqInfo.items():
            crt_info_metric = interrupts_info_metric(instance, hostname, irq, irq_info)
            prev_info_metric = prev_info_metrics_cache.get(irq)
            if prev_info_metric is not None and prev_info_metric != crt_info_metric:
                metrics.append(f"{prev_info_metric} 0 {crt_prom_ts}")
            if full_metrics or crt_info_metric != prev_info_metric:
                metrics.append(f"{crt_info_metric} 1 {crt_prom_ts}")

        for irq, prev_info_metric in prev_info_metrics_cache.items():
            if irq not in crt_proc_interrupts.Info.IrqInfo:
                metrics.append(f"{prev_info_metric} 0 {crt_prom_ts}")

    metrics.append(
        f"{PROC_INTERRUPTS_INTERVAL_METRIC_NAME}{{"
        + ",".join(
            [
                f'{INSTANCE_LABEL_NAME}="{instance}"',
                f'{HOSTNAME_LABEL_NAME}="{hostname}"',
            ]
        )
        + f"}} {interval} {crt_prom_ts}"
    )

    return metrics, new_zero_delta_map


def generate_proc_interrupts_test_case(
    name: str,
    crt_proc_interrupts: procfs.Interrupts,
    prev_proc_interrupts: procfs.Interrupts,
    ts: Optional[float] = None,
    cycle_num: int = 0,
    full_metrics_factor: int = DEFAULT_PROC_INTERRUPTS_FULL_METRICS_FACTOR,
    interval: Optional[float] = DEFAULT_PROC_INTERRUPTS_INTERVAL_SEC,
    zero_delta_map: Optional[ZeroDeltaMapType] = None,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    empty_info_metrics_cache: bool = False,
) -> ProcInterruptsMetricsTestCase:
    if ts is None:
        ts = time.time()
    crt_prom_ts = int(ts * 1000)
    prev_prom_ts = crt_prom_ts - int(interval * 1000)
    prev_info_metrics_cache = generate_info_metrics_cache(
        instance, hostname, prev_proc_interrupts
    )
    metrics, want_zero_delta_map = generate_proc_interrupts_metrics(
        crt_proc_interrupts=crt_proc_interrupts,
        prev_proc_interrupts=prev_proc_interrupts,
        crt_prom_ts=crt_prom_ts,
        interval=interval,
        zero_delta_map=zero_delta_map,
        prev_info_metrics_cache=prev_info_metrics_cache,
        full_metrics=(cycle_num == 0),
        instance=instance,
        hostname=hostname,
    )
    crt_proc_interrupts = deepcopy(crt_proc_interrupts)
    crt_proc_interrupts.CpuListChanged = (
        crt_proc_interrupts.CpuList != prev_proc_interrupts.CpuList
    )
    crt_proc_interrupts.IrqChanged = False
    if (
        crt_proc_interrupts.Info is not None
        and crt_proc_interrupts.Info.IrqInfo is not None
    ):
        crt_info_metrics_cache = generate_info_metrics_cache(
            instance, hostname, crt_proc_interrupts
        )
        for irq, crt_irq_info in crt_proc_interrupts.Info.IrqInfo.items():
            crt_irq_info.Changed = crt_info_metrics_cache.get(
                irq
            ) != prev_info_metrics_cache.get(irq)
            if crt_irq_info.Changed:
                crt_proc_interrupts.IrqChanged = True
    return ProcInterruptsMetricsTestCase(
        Name=name,
        Instance=instance,
        Hostname=hostname,
        CrtProcInterrupts=crt_proc_interrupts,
        PrevProcInterrupts=prev_proc_interrupts,
        CrtPromTs=crt_prom_ts,
        PrevPromTs=prev_prom_ts,
        CycleNum=cycle_num,
        FullMetricsFactor=full_metrics_factor,
        ZeroDeltaMap=zero_delta_map,
        InfoMetricsCache=(
            prev_info_metrics_cache if not empty_info_metrics_cache else {}
        ),
        WantMetricsCount=len(metrics),
        WantMetrics=metrics,
        ReportExtra=True,
        WantZeroDeltaMap=want_zero_delta_map,
    )


def make_ref_interrupts_irq_info(irq: str) -> procfs.InterruptsIrqInfo:
    is_num_irq = isinstance(irq, int) or re.match(r"\s*\d+\s*", irq) is not None
    irq_info = procfs.InterruptsIrqInfo(Changed=False)
    if is_num_irq:
        num_irq = int(irq)
        irq_info.Controller = f"ctl-{num_irq}"
        irq_info.HWInterrupt = f"hw-int-{num_irq}"
        irq_info.Devices = ",".join(
            f"dev-{num_irq}-{i}" for i in range(num_irq % 3 + 1)
        )
    return irq_info


def make_ref_interrupts_info(irq_list: List[str]) -> procfs.InterruptsInfo:
    return procfs.InterruptsInfo(
        IrqInfo={str(irq): make_ref_interrupts_irq_info(irq) for irq in irq_list},
        IrqChanged=False,
        CpuListChanged=False,
    )


def make_ref_proc_interrupts(
    irq_list: List[str], num_counters: int
) -> procfs.Interrupts:
    return procfs.Interrupts(
        CpuList=None,
        Counters={
            str(irq): [i * num_counters + j for j in range(num_counters)]
            for i, irq in enumerate(irq_list)
        },
        NumCounters=num_counters,
        Info=make_ref_interrupts_info(irq_list),
    )


def generate_proc_interrupts_metrics_test_cases(
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

    test_cases = []
    tc_num = 0

    irq_list = [1, 2, 3, "nmi"]
    num_counters = 4

    ref_proc_interrupts = make_ref_proc_interrupts(irq_list, num_counters)
    zero_delta_map_false = {str(irq): [False] * num_counters for irq in irq_list}
    zero_delta_map_true = {str(irq): [True] * num_counters for irq in irq_list}

    # All counters changed:
    crt_proc_interrupts = ref_proc_interrupts
    prev_proc_interrupts = deepcopy(ref_proc_interrupts)
    delta = 1
    for irq in irq_list:
        counters = prev_proc_interrupts.Counters[str(irq)]
        for i in range(num_counters):
            counters[i] = uint64_delta(counters[i], delta)
            delta += 1
    for zero_delta_map in [zero_delta_map_false, zero_delta_map_true]:
        for empty_info_metrics_cache in [False, True]:
            test_cases.append(
                generate_proc_interrupts_test_case(
                    str(tc_num),
                    crt_proc_interrupts,
                    prev_proc_interrupts,
                    zero_delta_map=zero_delta_map,
                    instance=instance,
                    hostname=hostname,
                    empty_info_metrics_cache=empty_info_metrics_cache,
                )
            )
            tc_num += 1

    json.dump(list(map(asdict, test_cases)), fp=fp, indent=2)
    fp.write("\n")
    if out_file is not None:
        fp.close()
        print(f"{out_file} generated", file=sys.stderr)
