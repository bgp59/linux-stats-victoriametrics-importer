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
class ProcInterruptsMetricsIrqDataTest:
    CycleNum: int = 0
    DeltaMetricPrefix: Optional[str] = None
    InfoMetric: Optional[str] = None
    ZeroDelta: Optional[ZeroDeltaType] = None


@dataclass
class ProcInterruptsMetricsTestCase:
    Name: Optional[str] = None
    Instance: Optional[str] = None
    Hostname: Optional[str] = None
    CrtProcInterrupts: Optional[procfs.Interrupts] = None
    PrevProcInterrupts: Optional[procfs.Interrupts] = None
    CrtPromTs: Optional[int] = None
    PrevPromTs: Optional[int] = None
    FullMetricsFactor: int = DEFAULT_PROC_INTERRUPTS_FULL_METRICS_FACTOR
    IrqDataCache: Optional[Dict[str, ProcInterruptsMetricsIrqDataTest]] = None
    WantMetricsCount: int = 0
    WantMetrics: Optional[List[str]] = None
    ReportExtra: Optional[bool] = None
    WantZeroDeltaMap: Optional[ZeroDeltaMapType] = None


testcases_file = "proc_interrupts.json"


def interrupts_info_metric(
    irq: str,
    irq_info: procfs.InterruptsIrqInfo,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
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


def generate_irq_data_cache(
    proc_interrupts: procfs.Interrupts,
    seed_prefix: bool = False,
    seed_zero_delta: bool = False,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
) -> Dict[str, ProcInterruptsMetricsIrqDataTest]:
    irq_data_cache = {}
    if proc_interrupts.Info is not None and proc_interrupts.Info.IrqInfo is not None:
        for irq, irq_info in proc_interrupts.Info.IrqInfo.items():
            controller = irq_info.Controller or ""
            hw_interrupt = irq_info.HWInterrupt or ""
            devices = irq_info.Devices or ""
            irq_data_cache[irq] = ProcInterruptsMetricsIrqDataTest(
                InfoMetric=(
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
            )
            if seed_prefix:
                irq_data_cache[
                    irq
                ].DeltaMetricPrefix = f"{PROC_INTERRUPTS_DELTA_METRIC}{{" + ",".join(
                    [
                        f'{INSTANCE_LABEL_NAME}="{instance}"',
                        f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                        f'{PROC_INTERRUPTS_IRQ_LABEL_NAME}="{irq}"',
                    ]
                )
            if seed_zero_delta:
                irq_data_cache[irq].ZeroDelta = [False] * proc_interrupts.NumCounters
    return irq_data_cache


def generate_proc_interrupts_metrics(
    crt_proc_interrupts: procfs.Interrupts,
    prev_proc_interrupts: procfs.Interrupts,
    crt_prom_ts: int,
    interval: Optional[float] = DEFAULT_PROC_INTERRUPTS_INTERVAL_SEC,
    irq_data_cache: Optional[Dict[str, ProcInterruptsMetricsIrqDataTest]] = None,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
) -> Tuple[List[str], Optional[ZeroDeltaMapType]]:
    metrics = []

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

    if irq_data_cache is None:
        irq_data_cache = generate_irq_data_cache(
            prev_proc_interrupts,
            seed_zero_delta=True,
            instance=instance,
            hostname=hostname,
        )
    new_irq_data_cache = generate_irq_data_cache(
        crt_proc_interrupts,
        seed_zero_delta=True,
        instance=instance,
        hostname=hostname,
    )

    irq_ifo_map = crt_proc_interrupts.Info.IrqInfo
    for irq, crt_counters in crt_proc_interrupts.Counters.items():
        prev_counters = prev_proc_interrupts.Counters.get(irq)
        if prev_counters is None:
            continue
        irq_data = irq_data_cache.get(
            irq,
            ProcInterruptsMetricsIrqDataTest(
                InfoMetric=interrupts_info_metric(
                    irq, irq_ifo_map[irq], instance=instance, hostname=hostname
                ),
                ZeroDelta=[False] * prev_proc_interrupts.NumCounters,
            ),
        )
        full_metrics = irq_data.CycleNum == 0
        zero_delta = irq_data.ZeroDelta
        if cpu_list_changed or zero_delta is None:
            zero_delta = [False] * crt_proc_interrupts.NumCounters
        for crt_i, crt_counter in enumerate(crt_counters):
            prev_i = crt_to_prev_counter_index_map.get(crt_i)
            if prev_i is None:
                continue
            delta = uint64_delta(crt_counter, prev_counters[prev_i])
            if full_metrics or delta > 0 or not zero_delta[crt_i]:
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
            new_irq_data_cache[irq].ZeroDelta[crt_i] = delta == 0
        prev_info_metric = (
            irq_data_cache[irq].InfoMetric if irq in irq_data_cache else None
        )
        crt_info_metric = new_irq_data_cache[irq].InfoMetric
        if prev_info_metric is not None and prev_info_metric != crt_info_metric:
            metrics.append(f"{prev_info_metric}0 {crt_prom_ts}")
        if prev_info_metric is None or full_metrics:
            metrics.append(f"{crt_info_metric}1 {crt_prom_ts}")

    metrics.append(
        f"{PROC_INTERRUPTS_INTERVAL_METRIC_NAME}{{"
        + ",".join(
            [
                f'{INSTANCE_LABEL_NAME}="{instance}"',
                f'{HOSTNAME_LABEL_NAME}="{hostname}"',
            ]
        )
        + f"}} {interval:.06f} {crt_prom_ts}"
    )

    return metrics, {
        irq: irq_data.ZeroDelta for irq, irq_data in new_irq_data_cache.items()
    }


def b64encode_interrupts(interrupts: procfs.Interrupts) -> procfs.Interrupts:
    interrupts = deepcopy(interrupts)
    interrupts.b64encode()
    return interrupts


def generate_proc_interrupts_test_case(
    name: str,
    crt_proc_interrupts: procfs.Interrupts,
    prev_proc_interrupts: procfs.Interrupts,
    ts: Optional[float] = None,
    cycle_num_map: Optional[Dict[str, int]] = None,
    zero_delta_map: Optional[ZeroDeltaMapType] = None,
    full_metrics_factor: int = DEFAULT_PROC_INTERRUPTS_FULL_METRICS_FACTOR,
    interval: Optional[float] = DEFAULT_PROC_INTERRUPTS_INTERVAL_SEC,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    empty_irq_data_cache: bool = False,
) -> ProcInterruptsMetricsTestCase:
    if ts is None:
        ts = time.time()
    crt_prom_ts = int(ts * 1000)
    prev_prom_ts = crt_prom_ts - int(interval * 1000)

    if empty_irq_data_cache:
        irq_data_cache = {}
    else:
        irq_data_cache = generate_irq_data_cache(
            prev_proc_interrupts,
            seed_prefix=True,
            seed_zero_delta=True,
            instance=instance,
            hostname=hostname,
        )

        if cycle_num_map is None:
            cycle_num_map = {}
        for irq, irq_data in irq_data_cache.items():
            irq_data.CycleNum = cycle_num_map.get(irq, 0)
            irq_data.ZeroDelta = zero_delta_map.get(
                irq, [False] * prev_proc_interrupts.NumCounters
            )

    metrics, want_zero_delta_map = generate_proc_interrupts_metrics(
        crt_proc_interrupts=crt_proc_interrupts,
        prev_proc_interrupts=prev_proc_interrupts,
        crt_prom_ts=crt_prom_ts,
        interval=interval,
        irq_data_cache=irq_data_cache,
        instance=instance,
        hostname=hostname,
    )
    crt_proc_interrupts = b64encode_interrupts(crt_proc_interrupts)
    prev_proc_interrupts = b64encode_interrupts(prev_proc_interrupts)
    crt_proc_interrupts.CpuListChanged = (
        crt_proc_interrupts.CpuList != prev_proc_interrupts.CpuList
    )
    crt_proc_interrupts.IrqChanged = False
    if (
        crt_proc_interrupts.Info is not None
        and crt_proc_interrupts.Info.IrqInfo is not None
    ):
        crt_irq_info_map = crt_proc_interrupts.Info.IrqInfo
        prev_irq_info_map = (
            prev_proc_interrupts.Info.IrqInfo
            if prev_proc_interrupts.Info is not None
            and prev_proc_interrupts.Info.IrqInfo is not None
            else {}
        )
        for irq, crt_irq_info in crt_irq_info_map.items():
            if crt_irq_info != prev_irq_info_map.get(irq):
                crt_irq_info.Changed = True
                crt_proc_interrupts.IrqChanged = True

    return ProcInterruptsMetricsTestCase(
        Name=name,
        Instance=instance,
        Hostname=hostname,
        CrtProcInterrupts=crt_proc_interrupts,
        PrevProcInterrupts=prev_proc_interrupts,
        CrtPromTs=crt_prom_ts,
        PrevPromTs=prev_prom_ts,
        FullMetricsFactor=full_metrics_factor,
        IrqDataCache=irq_data_cache,
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
        IrqInfo={irq: make_ref_interrupts_irq_info(irq) for irq in irq_list},
        IrqChanged=False,
        CpuListChanged=False,
    )


def make_ref_proc_interrupts(
    irq_list: List[str], num_counters: int
) -> procfs.Interrupts:
    return procfs.Interrupts(
        CpuList=None,
        Counters={
            irq: [i * num_counters + j for j in range(num_counters)]
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

    irq_list = ["1", "2", "3", "nmi"]
    num_counters = 4

    ref_proc_interrupts = make_ref_proc_interrupts(irq_list, num_counters)
    zero_delta_map_false = {irq: [False] * num_counters for irq in irq_list}
    zero_delta_map_true = {irq: [True] * num_counters for irq in irq_list}

    # All counters changed:
    crt_proc_interrupts = ref_proc_interrupts
    prev_proc_interrupts = deepcopy(ref_proc_interrupts)
    delta = 1
    for irq in irq_list:
        counters = prev_proc_interrupts.Counters[irq]
        for i in range(num_counters):
            counters[i] = uint64_delta(counters[i], delta)
            delta += 1
    for zero_delta_map in [zero_delta_map_false, zero_delta_map_true]:
        for cycle_num in [0, 1]:
            cycle_num_map = {irq: cycle_num for irq in irq_list}
            for empty_irq_data_cache in [False, True]:
                test_cases.append(
                    generate_proc_interrupts_test_case(
                        f"all_counters/{tc_num:04d}",
                        crt_proc_interrupts,
                        prev_proc_interrupts,
                        cycle_num_map=cycle_num_map,
                        zero_delta_map=zero_delta_map,
                        instance=instance,
                        hostname=hostname,
                        empty_irq_data_cache=empty_irq_data_cache,
                    )
                )
                tc_num += 1

    # Single counter change:
    crt_proc_interrupts = ref_proc_interrupts
    delta = 1
    for irq in irq_list:
        for i in range(num_counters):
            prev_proc_interrupts = deepcopy(ref_proc_interrupts)
            counters = prev_proc_interrupts.Counters[irq]
            counters[i] = uint64_delta(counters[i], delta)
            for zero_delta_map in [zero_delta_map_false, zero_delta_map_true]:
                for cycle_num in [0, 1]:
                    cycle_num_map = {irq: cycle_num for irq in irq_list}
                    test_cases.append(
                        generate_proc_interrupts_test_case(
                            f"single_counter/{tc_num:04d}",
                            crt_proc_interrupts,
                            prev_proc_interrupts,
                            cycle_num_map=cycle_num_map,
                            zero_delta_map=zero_delta_map,
                            instance=instance,
                            hostname=hostname,
                            empty_irq_data_cache=False,
                        )
                    )
                    tc_num += 1

    # # New IRQ:
    # crt_proc_interrupts = ref_proc_interrupts
    # for irq in irq_list:
    #     prev_proc_interrupts = deepcopy(crt_proc_interrupts)
    #     del prev_proc_interrupts.Counters[irq]
    #     del prev_proc_interrupts.Info.IrqInfo[irq]
    #     for zero_delta_map in [zero_delta_map_false, zero_delta_map_true]:
    #         for cycle_num in [0, 1]:
    #             test_cases.append(
    #                 generate_proc_interrupts_test_case(
    #                     f"new_irq/{tc_num:04d}",
    #                     crt_proc_interrupts,
    #                     prev_proc_interrupts,
    #                     cycle_num=cycle_num,
    #                     zero_delta_map=zero_delta_map,
    #                     instance=instance,
    #                     hostname=hostname,
    #                     empty_irq_data_cache=False,
    #                 )
    #             )
    #             tc_num += 1

    json.dump(list(map(asdict, test_cases)), fp=fp, indent=2)
    fp.write("\n")
    if out_file is not None:
        fp.close()
        print(f"{out_file} generated", file=sys.stderr)
