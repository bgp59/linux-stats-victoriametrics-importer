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
from .internal_metrics_common import (
    TC_CRT_STATS_FIELD,
    TC_FULL_CYCLE_FIELD,
    TC_HOSTNAME_FIELD,
    TC_INSTANCE_FIELD,
    TC_NAME_FIELD,
    TC_PREV_STATS_FIELD,
    TC_PROM_TS_FIELD,
    TC_REPORT_EXTRA_FIELD,
    TC_WANT_METRICS_COUNT_FIELD,
    TC_WANT_METRICS_FIELD,
    testcases_sub_dir,
)

TASK_STATS_TASK_ID_LABEL_NAME = "task_id"

TaskStats = Dict[str, Union[List[int], List[float]]]
SchedulerStats = Dict[str, TaskStats]

UINT64_STATS_FIELD = "Uint64Stats"
FLOAT64_STATS_FIELD = "Float64Stats"

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

testcases_file = "scheduler.json"


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
        val = crt_task_stats[UINT64_STATS_FIELD][i]
        if prev_task_stats is None or val != prev_task_stats[UINT64_STATS_FIELD][i]:
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
        val = crt_task_stats[FLOAT64_STATS_FIELD][i]
        if prev_task_stats is None or val != prev_task_stats[FLOAT64_STATS_FIELD][i]:
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
    prom_ts = int(ts * 1000)
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
        TC_NAME_FIELD: name,
        TC_INSTANCE_FIELD: instance,
        TC_HOSTNAME_FIELD: hostname,
        TC_PROM_TS_FIELD: prom_ts,
        TC_FULL_CYCLE_FIELD: full_cycle,
        TC_WANT_METRICS_COUNT_FIELD: len(metrics),
        TC_WANT_METRICS_FIELD: metrics,
        TC_REPORT_EXTRA_FIELD: report_extra,
        TC_CRT_STATS_FIELD: crt_stats,
        TC_PREV_STATS_FIELD: prev_stats,
    }


def generate_scheduler_internal_metrics_test_cases(
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    testcases_root_dir: Optional[str] = lsvmi_testcases_root,
):
    ts = time.time()

    if testcases_root_dir not in {None, "", "-"}:
        out_file = os.path.join(testcases_root_dir, testcases_sub_dir, testcases_file)
        os.makedirs(os.path.dirname(out_file), exist_ok=True)
        fp = open(out_file, "wt")
    else:
        out_file = None
        fp = sys.stdout

    crt_stats = {
        "taskA": {
            UINT64_STATS_FIELD: [0, 1, 2, 3],
            FLOAT64_STATS_FIELD: [0.1, 0.2],
        },
        "taskB": {
            UINT64_STATS_FIELD: [10, 11, 12, 13],
            FLOAT64_STATS_FIELD: [0.11, 0.21],
        },
    }

    test_cases = []
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
            prev_stats[task_id][UINT64_STATS_FIELD][i] += 1000
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
            prev_stats[task_id][FLOAT64_STATS_FIELD][i] += 1000
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
    if out_file is not None:
        fp.close()
        print(f"{out_file} generated", file=sys.stderr)
