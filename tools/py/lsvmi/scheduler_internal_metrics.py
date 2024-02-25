#! /usr/bin/env python3

# Generate test cases for lsvmi/scheduler_internal_metrics_test.go

import json
import os
import sys
import time
from copy import deepcopy
from typing import Any, Dict, List, Optional, Union

from . import (
    DEFAULT_TEST_HOSTNAME,
    DEFAULT_TEST_INSTANCE,
    HOSTNAME_LABEL_NAME,
    INSTANCE_LABEL_NAME,
    lsvmi_testcases_root,
)

TASK_STATS_TASK_ID_LABEL_NAME = "task_id"

task_stats_uint64_metric_names = [
    "lsvmi_task_scheduled_count_delta",
    "lsvmi_task_delayed_count_delta",
    "lsvmi_task_overrun_count_delta",
    "lsvmi_task_executed_count_delta",
]

task_stats_float64_metric_names = [
    None,
    "lsvmi_task_avg_runtime_sec",
]

default_out_file = os.path.join(
    lsvmi_testcases_root, "internal_metrics", "scheduler.json"
)

TaskStats = Dict[str, Union[List[int], List[float]]]
SchedulerStats = Dict[str, TaskStats]


def generate_task_stats_metrics(
    task_id: str,
    crt_task_stats: TaskStats,
    prev_task_stats: Optional[TaskStats] = None,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    ts: Optional[float] = None,
) -> List[str]:
    if ts is None:
        ts = time.time()
    promTs = str(int(ts * 1000))
    metrics = []
    for i, name in enumerate(task_stats_uint64_metric_names):
        if name is None:
            continue
        val = crt_task_stats["Uint64Stats"][i]
        if prev_task_stats is None or val != prev_task_stats["Uint64Stats"][i]:
            metrics.append(
                f"{name}{{"
                + ",".join(
                    [
                        f'{INSTANCE_LABEL_NAME}="{instance}"',
                        f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                        f'{TASK_STATS_TASK_ID_LABEL_NAME}="{task_id}"',
                    ]
                )
                + f"}} {val} {promTs}"
            )
    for i, name in enumerate(task_stats_float64_metric_names):
        if name is None:
            continue
        val = crt_task_stats["Float64Stats"][i]
        if prev_task_stats is None or val != prev_task_stats["Float64Stats"][i]:
            metrics.append(
                f"{name}{{"
                + ",".join(
                    [
                        f'{INSTANCE_LABEL_NAME}="{instance}"',
                        f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                        f'{TASK_STATS_TASK_ID_LABEL_NAME}="{task_id}"',
                    ]
                )
                + f"}} {val:.6f} {promTs}"
            )
    return metrics


def generate_scheduler_internal_metrics_test_case(
    name: str,
    crt_stats: SchedulerStats,
    prev_stats: Optional[SchedulerStats] = None,
    full_cycle: bool = False,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    report_extra: bool = True,
    ts: Optional[float] = None,
) -> Dict[str, Any]:
    if ts is None:
        ts = time.time()
    metrics = []
    for task_id, crt_task_stats in crt_stats.items():
        if not full_cycle and prev_stats is not None:
            prev_task_stats = prev_stats.get(task_id)
        else:
            prev_task_stats = None
        metrics.extend(
            generate_task_stats_metrics(
                task_id,
                crt_task_stats,
                prev_task_stats=prev_task_stats,
                instance=instance,
                hostname=hostname,
                ts=ts,
            )
        )
    return {
        "Name": name,
        "Instance": instance,
        "Hostname": hostname,
        "PromTs": int(ts * 1000),
        "FullCycle": full_cycle,
        "WantMetricsCount": len(metrics),
        "WantMetrics": metrics,
        "ReportExtra": report_extra,
        "CrtStats": crt_stats,
        "PrevStats": prev_stats,
    }


def generate_scheduler_internal_metrics_test_cases(
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    out_file: str = default_out_file,
):
    ts = time.time()
    test_cases = []
    if out_file != "-":
        os.makedirs(os.path.dirname(out_file), exist_ok=True)
        fp = open(out_file, "wt")
    else:
        fp = sys.stdout

    crt_stats = {
        "taskA": {
            "Uint64Stats": [0, 1, 2, 3],
            "Float64Stats": [0.1, 0.2],
        },
        "taskB": {
            "Uint64Stats": [10, 11, 12, 13],
            "Float64Stats": [0.11, 0.21],
        },
    }

    tc_num = 0
    prev_stats = None
    test_cases.append(
        generate_scheduler_internal_metrics_test_case(
            f"{tc_num:04d}",
            crt_stats,
            prev_stats=prev_stats,
            full_cycle=False,
            instance=instance,
            hostname=hostname,
            ts=ts,
        )
    )
    tc_num += 1

    for task_id in crt_stats:
        for i in range(len(task_stats_uint64_metric_names)):
            prev_stats = deepcopy(crt_stats)
            prev_stats[task_id]["Uint64Stats"][i] += 1000
            for full_cycle in [False, True]:
                test_cases.append(
                    generate_scheduler_internal_metrics_test_case(
                        f"{tc_num:04d}",
                        crt_stats,
                        prev_stats=prev_stats,
                        full_cycle=full_cycle,
                        instance=instance,
                        hostname=hostname,
                        ts=ts,
                    )
                )
                tc_num += 1
        for i in range(len(task_stats_float64_metric_names)):
            prev_stats = deepcopy(crt_stats)
            prev_stats[task_id]["Float64Stats"][i] += 1000
            for full_cycle in [False, True]:
                test_cases.append(
                    generate_scheduler_internal_metrics_test_case(
                        f"{tc_num:04d}",
                        crt_stats,
                        prev_stats=prev_stats,
                        full_cycle=full_cycle,
                        instance=instance,
                        hostname=hostname,
                        ts=ts,
                    )
                )
                tc_num += 1

    for task_id in crt_stats:
        prev_stats = deepcopy(crt_stats)
        del prev_stats[task_id]
        test_cases.append(
            generate_scheduler_internal_metrics_test_case(
                f"{tc_num:04d}",
                crt_stats,
                prev_stats=prev_stats,
                full_cycle=False,
                instance=instance,
                hostname=hostname,
                ts=ts,
            )
        )
        tc_num += 1

    json.dump(test_cases, fp=fp, indent=2)
    fp.write("\n")
    if out_file != "-":
        fp.close()
        print(f"{out_file} generated", file=sys.stderr)
