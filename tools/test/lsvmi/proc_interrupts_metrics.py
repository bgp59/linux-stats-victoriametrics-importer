#! /usr/bin/env python3

# Generate test cases for lsvmi/proc_interrupts_metrics_test.go

import re
import time
from copy import deepcopy
from dataclasses import dataclass
from typing import Dict, List, Optional, Tuple, Union

import procfs

from . import (
    DEFAULT_TEST_HOSTNAME,
    DEFAULT_TEST_INSTANCE,
    HOSTNAME_LABEL_NAME,
    INSTANCE_LABEL_NAME,
    lsvmi_test_cases_root_dir,
    save_test_cases,
    uint64_delta,
)

DEFAULT_PROC_INTERRUPTS_INTERVAL_SEC = 1
DEFAULT_PROC_INTERRUPTS_FULL_METRICS_FACTOR = 15

PROC_INTERRUPTS_DELTA_METRIC = "proc_interrupts_delta"
PROC_INTERRUPTS_IRQ_LABEL_NAME = "irq"
PROC_INTERRUPTS_IRQ_DEV_LABEL_NAME = "dev"
PROC_INTERRUPTS_CPU_LABEL_NAME = "cpu"

PROC_INTERRUPTS_INFO_METRIC = "proc_interrupts_info"
PROC_INTERRUPTS_INFO_IRQ_LABEL_NAME = PROC_INTERRUPTS_IRQ_LABEL_NAME
PROC_INTERRUPTS_INFO_CONTROLLER_LABEL_NAME = "controller"
PROC_INTERRUPTS_INFO_HW_INTERRUPT_LABEL_NAME = "hw_interrupt"
PROC_INTERRUPTS_INFO_DEV_LABEL_NAME = PROC_INTERRUPTS_IRQ_DEV_LABEL_NAME

PROC_INTERRUPTS_INTERVAL_METRIC = "proc_interrupts_metrics_delta_sec"


ZeroDeltaMapType = Dict[str, List[bool]]


@dataclass
class ProcInterruptsMetricsIrqDataTest:
    CycleNum: int = 0
    DeltaMetricPrefix: Optional[str] = None
    InfoMetric: Optional[str] = None
    ZeroDelta: Optional[List[bool]] = None


@dataclass
class ProcInterruptsMetricsTestCase:
    Name: Optional[str] = None
    Description: Optional[str] = None
    Instance: Optional[str] = None
    Hostname: Optional[str] = None
    CurrProcInterrupts: Optional[procfs.Interrupts] = None
    PrevProcInterrupts: Optional[procfs.Interrupts] = None
    CurrPromTs: Optional[int] = None
    PrevPromTs: Optional[int] = None
    FullMetricsFactor: int = DEFAULT_PROC_INTERRUPTS_FULL_METRICS_FACTOR
    IrqDataCache: Optional[Dict[str, ProcInterruptsMetricsIrqDataTest]] = None
    WantMetricsCount: int = 0
    WantMetrics: Optional[List[str]] = None
    ReportExtra: Optional[bool] = None
    WantZeroDeltaMap: Optional[ZeroDeltaMapType] = None


test_cases_file = "proc_interrupts.json"


def interrupts_delta_metric_prefix(
    proc_interrupts: procfs.Interrupts,
    irq: str,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
) -> str:
    irq_info = proc_interrupts.Info.IrqInfo[irq]
    devices = irq_info.Devices or ""
    return f"{PROC_INTERRUPTS_DELTA_METRIC}{{" + ",".join(
        [
            f'{INSTANCE_LABEL_NAME}="{instance}"',
            f'{HOSTNAME_LABEL_NAME}="{hostname}"',
            f'{PROC_INTERRUPTS_IRQ_LABEL_NAME}="{irq}"',
            f'{PROC_INTERRUPTS_IRQ_DEV_LABEL_NAME}="{devices}"',
        ]
    )


def interrupts_delta_metric(
    proc_interrupts: procfs.Interrupts,
    irq: str,
    cpu: Union[int, str],
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
) -> str:
    return (
        interrupts_delta_metric_prefix(
            proc_interrupts, irq, instance=instance, hostname=hostname
        )
        + f',{PROC_INTERRUPTS_CPU_LABEL_NAME}="{cpu}"'
        + "} "
    )


def interrupts_info_metric(
    proc_interrupts: procfs.Interrupts,
    irq: str,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
) -> str:
    irq_info = proc_interrupts.Info.IrqInfo[irq]
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


def update_irq_data_cache(
    proc_interrupts: procfs.Interrupts,
    irq: str,
    irq_data_cache: Dict[str, ProcInterruptsMetricsIrqDataTest],
    cycle_num: int = 0,
    zero_delta: bool = False,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
):
    irq_data_cache[irq] = ProcInterruptsMetricsIrqDataTest(
        CycleNum=cycle_num,
        DeltaMetricPrefix=interrupts_delta_metric_prefix(
            proc_interrupts,
            irq,
            instance=instance,
            hostname=hostname,
        ),
        InfoMetric=interrupts_info_metric(
            proc_interrupts,
            irq,
            instance=instance,
            hostname=hostname,
        ),
        ZeroDelta=[zero_delta] * proc_interrupts.NumCounters,
    )


def generate_irq_data_cache(
    proc_interrupts: procfs.Interrupts,
    cycle_num: int = 0,
    zero_delta: bool = False,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
) -> Dict[str, ProcInterruptsMetricsIrqDataTest]:
    irq_data_cache = {}
    if proc_interrupts.Info is not None and proc_interrupts.Info.IrqInfo is not None:
        for irq in proc_interrupts.Counters:
            update_irq_data_cache(
                proc_interrupts,
                irq,
                irq_data_cache,
                cycle_num=cycle_num,
                zero_delta=zero_delta,
                instance=instance,
                hostname=hostname,
            )
    return irq_data_cache


def generate_proc_interrupts_metrics(
    curr_proc_interrupts: procfs.Interrupts,
    prev_proc_interrupts: procfs.Interrupts,
    curr_prom_ts: int,
    interval: Optional[float] = DEFAULT_PROC_INTERRUPTS_INTERVAL_SEC,
    irq_data_cache: Optional[Dict[str, ProcInterruptsMetricsIrqDataTest]] = None,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
) -> Tuple[List[str], Optional[ZeroDeltaMapType]]:
    metrics = []

    # Build the mapping between the curr and prev counter index# using the
    # CpuList; 2 counter indexes have to refer to the same CPU#.
    curr_cpu_list = curr_proc_interrupts.CpuList
    if not curr_cpu_list:
        curr_cpu_list = [i for i in range(curr_proc_interrupts.NumCounters)]
    prev_cpu_list = prev_proc_interrupts.CpuList
    if not prev_cpu_list:
        prev_cpu_list = [i for i in range(prev_proc_interrupts.NumCounters)]
    prev_cpu_to_index_map = {cpu: i for i, cpu in enumerate(prev_cpu_list)}
    curr_to_prev_counter_index_map = {
        i: prev_cpu_to_index_map[cpu]
        for i, cpu in enumerate(curr_cpu_list)
        if cpu in prev_cpu_to_index_map
    }
    cpu_list_changed = curr_cpu_list != prev_cpu_list

    # Peruse irq_data_cache for cycle# and zero delta, if one provided:
    if irq_data_cache is None:
        irq_data_cache = {}

    new_zero_delta_map = {}
    for irq, curr_counters in curr_proc_interrupts.Counters.items():
        prev_counters = prev_proc_interrupts.Counters.get(irq)
        if prev_counters is None:
            continue
        new_zero_delta_map[irq] = [False] * curr_proc_interrupts.NumCounters
        irq_data = irq_data_cache.get(irq)
        if irq_data is None or cpu_list_changed:
            zero_delta = [False] * curr_proc_interrupts.NumCounters
        else:
            zero_delta = irq_data.ZeroDelta
        full_metrics = irq_data is None or irq_data.CycleNum == 0
        if cpu_list_changed or zero_delta is None:
            zero_delta = [False] * curr_proc_interrupts.NumCounters
        for curr_i, curr_counter in enumerate(curr_counters):
            prev_i = curr_to_prev_counter_index_map.get(curr_i)
            if prev_i is None:
                continue
            delta = uint64_delta(curr_counter, prev_counters[prev_i])
            if full_metrics or delta > 0 or not zero_delta[curr_i]:
                metrics.append(
                    interrupts_delta_metric(
                        curr_proc_interrupts,
                        irq,
                        curr_cpu_list[curr_i],
                        instance=instance,
                        hostname=hostname,
                    )
                    + f"{delta} {curr_prom_ts}"
                )
            new_zero_delta_map[irq][curr_i] = delta == 0
        prev_info_metric = irq_data.InfoMetric if irq_data is not None else None
        curr_info_metric = interrupts_info_metric(
            curr_proc_interrupts,
            irq,
            instance=instance,
            hostname=hostname,
        )
        if prev_info_metric is not None and prev_info_metric != curr_info_metric:
            metrics.append(f"{prev_info_metric}0 {curr_prom_ts}")
        if prev_info_metric is None or full_metrics:
            metrics.append(f"{curr_info_metric}1 {curr_prom_ts}")

    # Handle removed IRQ's:
    for irq, irq_data in irq_data_cache.items():
        if irq not in curr_proc_interrupts.Counters:
            metrics.append(f"{irq_data.InfoMetric}0 {curr_prom_ts}")

    metrics.append(
        f"{PROC_INTERRUPTS_INTERVAL_METRIC}{{"
        + ",".join(
            [
                f'{INSTANCE_LABEL_NAME}="{instance}"',
                f'{HOSTNAME_LABEL_NAME}="{hostname}"',
            ]
        )
        + f"}} {interval:.06f} {curr_prom_ts}"
    )

    return metrics, new_zero_delta_map


def b64encode_interrupts(interrupts: procfs.Interrupts) -> procfs.Interrupts:
    interrupts = deepcopy(interrupts)
    interrupts.b64encode()
    return interrupts


def generate_proc_interrupts_test_case(
    name: str,
    curr_proc_interrupts: procfs.Interrupts,
    prev_proc_interrupts: procfs.Interrupts,
    ts: Optional[float] = None,
    cycle_num_map: Optional[Dict[str, int]] = None,
    zero_delta_map: Optional[ZeroDeltaMapType] = None,
    new_irq: Optional[str] = None,
    full_metrics_factor: int = DEFAULT_PROC_INTERRUPTS_FULL_METRICS_FACTOR,
    interval: Optional[float] = DEFAULT_PROC_INTERRUPTS_INTERVAL_SEC,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    empty_irq_data_cache: bool = False,
    description: Optional[str] = None,
) -> ProcInterruptsMetricsTestCase:
    if ts is None:
        ts = time.time()
    curr_prom_ts = int(ts * 1000)
    prev_prom_ts = curr_prom_ts - int(interval * 1000)

    if empty_irq_data_cache:
        irq_data_cache = {}
    else:
        irq_data_cache = generate_irq_data_cache(
            prev_proc_interrupts,
            instance=instance,
            hostname=hostname,
        )
        if new_irq is not None:
            del irq_data_cache[new_irq]

        if cycle_num_map is None:
            cycle_num_map = {}
        for irq, irq_data in irq_data_cache.items():
            irq_data.CycleNum = cycle_num_map.get(irq, 0)
            irq_data.ZeroDelta = zero_delta_map.get(
                irq, [False] * prev_proc_interrupts.NumCounters
            )

    metrics, want_zero_delta_map = generate_proc_interrupts_metrics(
        curr_proc_interrupts=curr_proc_interrupts,
        prev_proc_interrupts=prev_proc_interrupts,
        curr_prom_ts=curr_prom_ts,
        interval=interval,
        irq_data_cache=irq_data_cache,
        instance=instance,
        hostname=hostname,
    )
    curr_proc_interrupts = b64encode_interrupts(curr_proc_interrupts)
    prev_proc_interrupts = b64encode_interrupts(prev_proc_interrupts)
    curr_proc_interrupts.Info.CpuListChanged = (
        curr_proc_interrupts.CpuList != prev_proc_interrupts.CpuList
    )

    if (
        curr_proc_interrupts.Info is not None
        and curr_proc_interrupts.Info.IrqInfo is not None
    ):
        curr_irq_info_map = curr_proc_interrupts.Info.IrqInfo
        prev_irq_info_map = (
            prev_proc_interrupts.Info.IrqInfo
            if prev_proc_interrupts.Info is not None
            and prev_proc_interrupts.Info.IrqInfo is not None
            else {}
        )
        for irq, curr_irq_info in curr_irq_info_map.items():
            if curr_irq_info != prev_irq_info_map.get(irq):
                curr_irq_info.Changed = True
                curr_proc_interrupts.IrqChanged = True

    return ProcInterruptsMetricsTestCase(
        Name=name,
        Description=description,
        Instance=instance,
        Hostname=hostname,
        CurrProcInterrupts=curr_proc_interrupts,
        PrevProcInterrupts=prev_proc_interrupts,
        CurrPromTs=curr_prom_ts,
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
    test_cases_root_dir: Optional[str] = lsvmi_test_cases_root_dir,
):
    test_cases = []
    tc_num = 0

    irq_list = ["1", "2", "3", "nmi"]
    num_counters = 4

    ref_proc_interrupts = make_ref_proc_interrupts(irq_list, num_counters)
    # All counters changed:
    curr_proc_interrupts = ref_proc_interrupts
    prev_proc_interrupts = deepcopy(ref_proc_interrupts)
    delta = 1
    for irq in irq_list:
        counters = prev_proc_interrupts.Counters[irq]
        for i in range(num_counters):
            counters[i] = uint64_delta(counters[i], delta)
            delta += 1
    for zero_delta in [False, True]:
        zero_delta_map = {irq: [zero_delta] * num_counters for irq in irq_list}
        for cycle_num in [0, 1]:
            cycle_num_map = {irq: cycle_num for irq in irq_list}
            for empty_irq_data_cache in [False, True]:
                test_cases.append(
                    generate_proc_interrupts_test_case(
                        f"all_counters/{tc_num:04d}",
                        curr_proc_interrupts,
                        prev_proc_interrupts,
                        cycle_num_map=cycle_num_map,
                        zero_delta_map=zero_delta_map,
                        instance=instance,
                        hostname=hostname,
                        empty_irq_data_cache=empty_irq_data_cache,
                        description=", ".join(
                            [
                                f"zero_delta={zero_delta}",
                                f"cycle_num={cycle_num}",
                                f"empty_irq_data_cache={empty_irq_data_cache}",
                            ]
                        ),
                    )
                )
                tc_num += 1

    # Single counter change:
    curr_proc_interrupts = ref_proc_interrupts
    delta = 1
    for zero_delta in [False, True]:
        zero_delta_map = {irq: [zero_delta] * num_counters for irq in irq_list}
        for cycle_num in [0, 1]:
            cycle_num_map = {irq: cycle_num for irq in irq_list}
            for empty_irq_data_cache in [False, True]:
                for irq in irq_list:
                    for i in range(num_counters):
                        prev_proc_interrupts = deepcopy(ref_proc_interrupts)
                        counters = prev_proc_interrupts.Counters[irq]
                        counters[i] = uint64_delta(counters[i], delta)
                        test_cases.append(
                            generate_proc_interrupts_test_case(
                                f"single_counter/{tc_num:04d}",
                                curr_proc_interrupts,
                                prev_proc_interrupts,
                                cycle_num_map=cycle_num_map,
                                zero_delta_map=zero_delta_map,
                                instance=instance,
                                hostname=hostname,
                                empty_irq_data_cache=empty_irq_data_cache,
                                description=", ".join(
                                    [
                                        f"irq={irq}",
                                        f"counter#={i}",
                                        f"zero_delta={zero_delta}",
                                        f"cycle_num={cycle_num}",
                                        f"empty_irq_data_cache={empty_irq_data_cache}",
                                    ]
                                ),
                            )
                        )
                        tc_num += 1

    # New IRQ under 2 scenarios:
    #   - not in prev but in current
    #   - in both but not in cache
    curr_proc_interrupts = ref_proc_interrupts
    for zero_delta in [False, True]:
        for cycle_num in [0, 1]:
            for empty_irq_data_cache in [False, True]:
                for irq in irq_list:
                    for new_irq in [None, irq]:
                        prev_proc_interrupts = deepcopy(curr_proc_interrupts)
                        if new_irq is None:
                            del prev_proc_interrupts.Counters[irq]
                            del prev_proc_interrupts.Info.IrqInfo[irq]
                        cycle_num_map = {i: cycle_num for i in irq_list if i != irq}
                        zero_delta_map = {
                            i: [zero_delta] * num_counters for i in irq_list if i != irq
                        }
                        test_cases.append(
                            generate_proc_interrupts_test_case(
                                f"new_irq/{tc_num:04d}",
                                curr_proc_interrupts,
                                prev_proc_interrupts,
                                cycle_num_map=cycle_num_map,
                                zero_delta_map=zero_delta_map,
                                new_irq=new_irq,
                                instance=instance,
                                hostname=hostname,
                                empty_irq_data_cache=empty_irq_data_cache,
                                description=", ".join(
                                    [
                                        f"irq={irq}",
                                        f"new_irq={new_irq}",
                                        f"zero_delta={zero_delta}",
                                        f"cycle_num={cycle_num}",
                                        f"empty_irq_data_cache={empty_irq_data_cache}",
                                    ]
                                ),
                            )
                        )
                        tc_num += 1

    # Remove IRQ:
    prev_proc_interrupts = ref_proc_interrupts
    for zero_delta in [False, True]:
        zero_delta_map = {irq: [zero_delta] * num_counters for irq in irq_list}
        for cycle_num in [0, 1]:
            cycle_num_map = {irq: cycle_num for irq in irq_list}
            for empty_irq_data_cache in [False, True]:
                for irq in irq_list:
                    curr_proc_interrupts = deepcopy(prev_proc_interrupts)
                    del curr_proc_interrupts.Counters[irq]
                    del curr_proc_interrupts.Info.IrqInfo[irq]
                    test_cases.append(
                        generate_proc_interrupts_test_case(
                            f"remove_irq/{tc_num:04d}",
                            curr_proc_interrupts,
                            prev_proc_interrupts,
                            cycle_num_map=cycle_num_map,
                            zero_delta_map=zero_delta_map,
                            instance=instance,
                            hostname=hostname,
                            empty_irq_data_cache=empty_irq_data_cache,
                            description=", ".join(
                                [
                                    f"irq={irq}",
                                    f"zero_delta={zero_delta}",
                                    f"cycle_num={cycle_num}",
                                    f"empty_irq_data_cache={empty_irq_data_cache}",
                                ]
                            ),
                        )
                    )
                    tc_num += 1

    # Remove CPU:
    prev_proc_interrupts = ref_proc_interrupts
    for zero_delta in [False, True]:
        zero_delta_map = {irq: [zero_delta] * num_counters for irq in irq_list}
        for cycle_num in [0, 1]:
            cycle_num_map = {irq: cycle_num for irq in irq_list}
            for empty_irq_data_cache in [False, True]:
                for k in range(num_counters):
                    curr_proc_interrupts = deepcopy(prev_proc_interrupts)
                    for irq in irq_list:
                        del curr_proc_interrupts.Counters[irq][k : k + 1]
                    curr_proc_interrupts.NumCounters -= 1
                    curr_proc_interrupts.CpuList = [
                        i for i in range(num_counters) if i != k
                    ]
                    test_cases.append(
                        generate_proc_interrupts_test_case(
                            f"remove_cpu/{tc_num:04d}",
                            curr_proc_interrupts,
                            prev_proc_interrupts,
                            cycle_num_map=cycle_num_map,
                            zero_delta_map=zero_delta_map,
                            instance=instance,
                            hostname=hostname,
                            empty_irq_data_cache=empty_irq_data_cache,
                            description=", ".join(
                                [
                                    f"remove cpu={k}",
                                    f"zero_delta={zero_delta}",
                                    f"cycle_num={cycle_num}",
                                    f"empty_irq_data_cache={empty_irq_data_cache}",
                                ]
                            ),
                        )
                    )
                    tc_num += 1

    # New CPU:
    curr_proc_interrupts = ref_proc_interrupts
    for zero_delta in [False, True]:
        zero_delta_map = {irq: [zero_delta] * num_counters for irq in irq_list}
        for cycle_num in [0, 1]:
            cycle_num_map = {irq: cycle_num for irq in irq_list}
            for empty_irq_data_cache in [False, True]:
                for k in range(num_counters):
                    prev_proc_interrupts = deepcopy(curr_proc_interrupts)
                    for irq in irq_list:
                        del prev_proc_interrupts.Counters[irq][k : k + 1]
                    prev_proc_interrupts.NumCounters -= 1
                    prev_proc_interrupts.CpuList = [
                        i for i in range(num_counters) if i != k
                    ]
                    test_cases.append(
                        generate_proc_interrupts_test_case(
                            f"new_cpu/{tc_num:04d}",
                            curr_proc_interrupts,
                            prev_proc_interrupts,
                            cycle_num_map=cycle_num_map,
                            zero_delta_map=zero_delta_map,
                            instance=instance,
                            hostname=hostname,
                            empty_irq_data_cache=empty_irq_data_cache,
                            description=", ".join(
                                [
                                    f"new cpu={k}",
                                    f"zero_delta={zero_delta}",
                                    f"cycle_num={cycle_num}",
                                    f"empty_irq_data_cache={empty_irq_data_cache}",
                                ]
                            ),
                        )
                    )
                    tc_num += 1

    save_test_cases(
        test_cases, test_cases_file, test_cases_root_dir=test_cases_root_dir
    )
