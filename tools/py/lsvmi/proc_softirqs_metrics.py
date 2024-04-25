#! /usr/bin/env python3

# Generate test cases for lsvmi/proc_softirqs_metrics_test.go

import json
import os
import sys
import time
from copy import deepcopy
from dataclasses import asdict, dataclass
from typing import Dict, List, Optional, Tuple, Union

import procfs

from . import (
    DEFAULT_TEST_HOSTNAME,
    DEFAULT_TEST_INSTANCE,
    HOSTNAME_LABEL_NAME,
    INSTANCE_LABEL_NAME,
    lsvmi_testcases_root,
    uint64_delta,
)

DEFAULT_PROC_SOFTIRQS_INTERVAL_SEC = 1
DEFAULT_PROC_SOFTIRQS_FULL_METRICS_FACTOR = 15

PROC_SOFTIRQS_METRICS_ID = "proc_softirqs_metrics"

PROC_SOFTIRQS_DELTA_METRIC = "proc_softirqs_delta"
PROC_SOFTIRQS_IRQ_LABEL_NAME = "irq"
PROC_SOFTIRQS_DEV_LABEL_NAME = "dev"
PROC_SOFTIRQS_CPU_LABEL_NAME = "cpu"

PROC_SOFTIRQS_INFO_METRIC = "proc_softirqs_info"
PROC_SOFTIRQS_INFO_IRQ_LABEL_NAME = PROC_SOFTIRQS_IRQ_LABEL_NAME

PROC_SOFTIRQS_INTERVAL_METRIC_NAME = "proc_softirqs_metrics_delta_sec"

ZeroDeltaType = List[bool]
ZeroDeltaMapType = Dict[str, ZeroDeltaType]


@dataclass
class ProcSoftirqsMetricsIrqDataTest:
    CycleNum: int = 0
    DeltaMetricPrefix: Optional[str] = None
    InfoMetric: Optional[str] = None
    ZeroDelta: Optional[ZeroDeltaType] = None


@dataclass
class ProcSoftirqsMetricsTestCase:
    Name: Optional[str] = None
    Description: Optional[str] = None
    Instance: Optional[str] = None
    Hostname: Optional[str] = None
    CurrProcSoftirqs: Optional[procfs.Softirqs] = None
    PrevProcSoftirqs: Optional[procfs.Softirqs] = None
    CurrPromTs: Optional[int] = None
    PrevPromTs: Optional[int] = None
    FullMetricsFactor: int = DEFAULT_PROC_SOFTIRQS_FULL_METRICS_FACTOR
    IrqDataCache: Optional[Dict[str, ProcSoftirqsMetricsIrqDataTest]] = None
    WantMetricsCount: int = 0
    WantMetrics: Optional[List[str]] = None
    ReportExtra: Optional[bool] = None
    WantZeroDeltaMap: Optional[ZeroDeltaMapType] = None


testcases_file = "proc_softirqs.json"


def softirqs_delta_metric_prefix(
    irq: str,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
) -> str:
    return f"{PROC_SOFTIRQS_DELTA_METRIC}{{" + ",".join(
        [
            f'{INSTANCE_LABEL_NAME}="{instance}"',
            f'{HOSTNAME_LABEL_NAME}="{hostname}"',
            f'{PROC_SOFTIRQS_IRQ_LABEL_NAME}="{irq}"',
        ]
    )


def softirqs_delta_metric(
    irq: str,
    cpu: Union[int, str],
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
) -> str:
    return (
        softirqs_delta_metric_prefix(irq, instance=instance, hostname=hostname)
        + f',{PROC_SOFTIRQS_CPU_LABEL_NAME}="{cpu}"'
        + "} "
    )


def softirqs_info_metric(
    irq: str,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
) -> str:

    return (
        f"{PROC_SOFTIRQS_INFO_METRIC}{{"
        + ",".join(
            [
                f'{INSTANCE_LABEL_NAME}="{instance}"',
                f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                f'{PROC_SOFTIRQS_IRQ_LABEL_NAME}="{irq}"',
            ]
        )
        + "} "
    )


def update_irq_data_cache(
    proc_softirqs: procfs.Softirqs,
    irq: str,
    irq_data_cache: Dict[str, ProcSoftirqsMetricsIrqDataTest],
    cycle_num: int = 0,
    zero_delta: bool = False,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
):
    irq_data_cache[irq] = ProcSoftirqsMetricsIrqDataTest(
        CycleNum=cycle_num,
        DeltaMetricPrefix=softirqs_delta_metric_prefix(
            irq,
            instance=instance,
            hostname=hostname,
        ),
        InfoMetric=softirqs_info_metric(
            irq,
            instance=instance,
            hostname=hostname,
        ),
        ZeroDelta=[zero_delta] * proc_softirqs.NumCounters,
    )


def generate_irq_data_cache(
    proc_softirqs: procfs.Softirqs,
    cycle_num: int = 0,
    zero_delta: bool = False,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
) -> Dict[str, ProcSoftirqsMetricsIrqDataTest]:
    irq_data_cache = {}
    if proc_softirqs.Counters is not None:
        for irq in proc_softirqs.Counters:
            update_irq_data_cache(
                proc_softirqs,
                irq,
                irq_data_cache,
                cycle_num=cycle_num,
                zero_delta=zero_delta,
                instance=instance,
                hostname=hostname,
            )
    return irq_data_cache


def generate_proc_softirqs_metrics(
    curr_proc_softirqs: procfs.Softirqs,
    prev_proc_softirqs: procfs.Softirqs,
    curr_prom_ts: int,
    interval: Optional[float] = DEFAULT_PROC_SOFTIRQS_INTERVAL_SEC,
    irq_data_cache: Optional[Dict[str, ProcSoftirqsMetricsIrqDataTest]] = None,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
) -> Tuple[List[str], Optional[ZeroDeltaMapType]]:
    metrics = []

    # Build the mapping between the curr and prev counter index# using the
    # CpuList; 2 counter indexes have to refer to the same CPU#.
    curr_cpu_list = curr_proc_softirqs.CpuList
    if not curr_cpu_list:
        curr_cpu_list = [i for i in range(curr_proc_softirqs.NumCounters)]
    prev_cpu_list = prev_proc_softirqs.CpuList
    if not prev_cpu_list:
        prev_cpu_list = [i for i in range(prev_proc_softirqs.NumCounters)]
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
    for irq, curr_counters in curr_proc_softirqs.Counters.items():
        prev_counters = prev_proc_softirqs.Counters.get(irq)
        if prev_counters is None:
            continue
        new_zero_delta_map[irq] = [False] * curr_proc_softirqs.NumCounters
        irq_data = irq_data_cache.get(irq)
        if irq_data is None or cpu_list_changed:
            zero_delta = [False] * curr_proc_softirqs.NumCounters
        else:
            zero_delta = irq_data.ZeroDelta
        full_metrics = irq_data is None or irq_data.CycleNum == 0
        if cpu_list_changed or zero_delta is None:
            zero_delta = [False] * curr_proc_softirqs.NumCounters
        for curr_i, curr_counter in enumerate(curr_counters):
            prev_i = curr_to_prev_counter_index_map.get(curr_i)
            if prev_i is None:
                continue
            delta = uint64_delta(curr_counter, prev_counters[prev_i])
            if full_metrics or delta > 0 or not zero_delta[curr_i]:
                metrics.append(
                    softirqs_delta_metric(
                        irq,
                        curr_cpu_list[curr_i],
                        instance=instance,
                        hostname=hostname,
                    )
                    + f"{delta} {curr_prom_ts}"
                )
            new_zero_delta_map[irq][curr_i] = delta == 0
        prev_info_metric = irq_data.InfoMetric if irq_data is not None else None
        curr_info_metric = softirqs_info_metric(
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
        if irq not in curr_proc_softirqs.Counters:
            metrics.append(f"{irq_data.InfoMetric}0 {curr_prom_ts}")

    metrics.append(
        f"{PROC_SOFTIRQS_INTERVAL_METRIC_NAME}{{"
        + ",".join(
            [
                f'{INSTANCE_LABEL_NAME}="{instance}"',
                f'{HOSTNAME_LABEL_NAME}="{hostname}"',
            ]
        )
        + f"}} {interval:.06f} {curr_prom_ts}"
    )

    return metrics, new_zero_delta_map


def generate_proc_softirqs_test_case(
    name: str,
    curr_proc_softirqs: procfs.Softirqs,
    prev_proc_softirqs: procfs.Softirqs,
    ts: Optional[float] = None,
    cycle_num_map: Optional[Dict[str, int]] = None,
    zero_delta_map: Optional[ZeroDeltaMapType] = None,
    new_irq: Optional[str] = None,
    full_metrics_factor: int = DEFAULT_PROC_SOFTIRQS_FULL_METRICS_FACTOR,
    interval: Optional[float] = DEFAULT_PROC_SOFTIRQS_INTERVAL_SEC,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    empty_irq_data_cache: bool = False,
    description: Optional[str] = None,
) -> ProcSoftirqsMetricsTestCase:
    if ts is None:
        ts = time.time()
    curr_prom_ts = int(ts * 1000)
    prev_prom_ts = curr_prom_ts - int(interval * 1000)

    if empty_irq_data_cache:
        irq_data_cache = {}
    else:
        irq_data_cache = generate_irq_data_cache(
            prev_proc_softirqs,
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
                irq, [False] * prev_proc_softirqs.NumCounters
            )

    metrics, want_zero_delta_map = generate_proc_softirqs_metrics(
        curr_proc_softirqs=curr_proc_softirqs,
        prev_proc_softirqs=prev_proc_softirqs,
        curr_prom_ts=curr_prom_ts,
        interval=interval,
        irq_data_cache=irq_data_cache,
        instance=instance,
        hostname=hostname,
    )
    curr_proc_softirqs = deepcopy(curr_proc_softirqs)
    prev_proc_softirqs = deepcopy(prev_proc_softirqs)
    curr_proc_softirqs.CpuListChanged = (
        curr_proc_softirqs.CpuList != prev_proc_softirqs.CpuList
    )

    return ProcSoftirqsMetricsTestCase(
        Name=name,
        Description=description,
        Instance=instance,
        Hostname=hostname,
        CurrProcSoftirqs=curr_proc_softirqs,
        PrevProcSoftirqs=prev_proc_softirqs,
        CurrPromTs=curr_prom_ts,
        PrevPromTs=prev_prom_ts,
        FullMetricsFactor=full_metrics_factor,
        IrqDataCache=irq_data_cache,
        WantMetricsCount=len(metrics),
        WantMetrics=metrics,
        ReportExtra=True,
        WantZeroDeltaMap=want_zero_delta_map,
    )


def make_ref_proc_softirqs(irq_list: List[str], num_counters: int) -> procfs.Softirqs:
    return procfs.Softirqs(
        CpuList=None,
        Counters={
            irq: [i * num_counters + j for j in range(num_counters)]
            for i, irq in enumerate(irq_list)
        },
        NumCounters=num_counters,
    )


def generate_proc_softirqs_metrics_test_cases(
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

    irq_list = ["softirq-1", "softirq-2", "softirq-3", "softirq-4"]
    num_counters = 4

    ref_proc_softirqs = make_ref_proc_softirqs(irq_list, num_counters)
    # All counters changed:
    curr_proc_softirqs = ref_proc_softirqs
    prev_proc_softirqs = deepcopy(ref_proc_softirqs)
    delta = 1
    for irq in irq_list:
        counters = prev_proc_softirqs.Counters[irq]
        for i in range(num_counters):
            counters[i] = uint64_delta(counters[i], delta)
            delta += 1
    for zero_delta in [False, True]:
        zero_delta_map = {irq: [zero_delta] * num_counters for irq in irq_list}
        for cycle_num in [0, 1]:
            cycle_num_map = {irq: cycle_num for irq in irq_list}
            for empty_irq_data_cache in [False, True]:
                test_cases.append(
                    generate_proc_softirqs_test_case(
                        f"all_counters/{tc_num:04d}",
                        curr_proc_softirqs,
                        prev_proc_softirqs,
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
    curr_proc_softirqs = ref_proc_softirqs
    delta = 1
    for zero_delta in [False, True]:
        zero_delta_map = {irq: [zero_delta] * num_counters for irq in irq_list}
        for cycle_num in [0, 1]:
            cycle_num_map = {irq: cycle_num for irq in irq_list}
            for empty_irq_data_cache in [False, True]:
                for irq in irq_list:
                    for i in range(num_counters):
                        prev_proc_softirqs = deepcopy(ref_proc_softirqs)
                        counters = prev_proc_softirqs.Counters[irq]
                        counters[i] = uint64_delta(counters[i], delta)
                        test_cases.append(
                            generate_proc_softirqs_test_case(
                                f"single_counter/{tc_num:04d}",
                                curr_proc_softirqs,
                                prev_proc_softirqs,
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
    curr_proc_softirqs = ref_proc_softirqs
    for zero_delta in [False, True]:
        for cycle_num in [0, 1]:
            for empty_irq_data_cache in [False, True]:
                for irq in irq_list:
                    for new_irq in [None, irq]:
                        prev_proc_softirqs = deepcopy(curr_proc_softirqs)
                        if new_irq is None:
                            del prev_proc_softirqs.Counters[irq]
                        cycle_num_map = {i: cycle_num for i in irq_list if i != irq}
                        zero_delta_map = {
                            i: [zero_delta] * num_counters for i in irq_list if i != irq
                        }
                        test_cases.append(
                            generate_proc_softirqs_test_case(
                                f"new_irq/{tc_num:04d}",
                                curr_proc_softirqs,
                                prev_proc_softirqs,
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
    prev_proc_softirqs = ref_proc_softirqs
    for zero_delta in [False, True]:
        zero_delta_map = {irq: [zero_delta] * num_counters for irq in irq_list}
        for cycle_num in [0, 1]:
            cycle_num_map = {irq: cycle_num for irq in irq_list}
            for empty_irq_data_cache in [False, True]:
                for irq in irq_list:
                    curr_proc_softirqs = deepcopy(prev_proc_softirqs)
                    del curr_proc_softirqs.Counters[irq]
                    test_cases.append(
                        generate_proc_softirqs_test_case(
                            f"remove_irq/{tc_num:04d}",
                            curr_proc_softirqs,
                            prev_proc_softirqs,
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
    prev_proc_softirqs = ref_proc_softirqs
    for zero_delta in [False, True]:
        zero_delta_map = {irq: [zero_delta] * num_counters for irq in irq_list}
        for cycle_num in [0, 1]:
            cycle_num_map = {irq: cycle_num for irq in irq_list}
            for empty_irq_data_cache in [False, True]:
                for k in range(num_counters):
                    curr_proc_softirqs = deepcopy(prev_proc_softirqs)
                    for irq in irq_list:
                        del curr_proc_softirqs.Counters[irq][k : k + 1]
                    curr_proc_softirqs.NumCounters -= 1
                    curr_proc_softirqs.CpuList = [
                        i for i in range(num_counters) if i != k
                    ]
                    test_cases.append(
                        generate_proc_softirqs_test_case(
                            f"remove_cpu/{tc_num:04d}",
                            curr_proc_softirqs,
                            prev_proc_softirqs,
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
    curr_proc_softirqs = ref_proc_softirqs
    for zero_delta in [False, True]:
        zero_delta_map = {irq: [zero_delta] * num_counters for irq in irq_list}
        for cycle_num in [0, 1]:
            cycle_num_map = {irq: cycle_num for irq in irq_list}
            for empty_irq_data_cache in [False, True]:
                for k in range(num_counters):
                    prev_proc_softirqs = deepcopy(curr_proc_softirqs)
                    for irq in irq_list:
                        del prev_proc_softirqs.Counters[irq][k : k + 1]
                    prev_proc_softirqs.NumCounters -= 1
                    prev_proc_softirqs.CpuList = [
                        i for i in range(num_counters) if i != k
                    ]
                    test_cases.append(
                        generate_proc_softirqs_test_case(
                            f"new_cpu/{tc_num:04d}",
                            curr_proc_softirqs,
                            prev_proc_softirqs,
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

    json.dump(list(map(asdict, test_cases)), fp=fp, indent=2)
    fp.write("\n")
    if out_file is not None:
        fp.close()
        print(f"{out_file} generated", file=sys.stderr)
