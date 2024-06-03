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
    lsvmi_test_cases_root_dir,
)
from .internal_metrics import (
    TC_CURR_STATS_FIELD,
    TC_HOSTNAME_FIELD,
    TC_INSTANCE_FIELD,
    TC_NAME_FIELD,
    TC_PREV_STATS_FIELD,
    TC_PROM_TS_FIELD,
    TC_REPORT_EXTRA_FIELD,
    TC_WANT_METRICS_COUNT_FIELD,
    TC_WANT_METRICS_FIELD,
    test_cases_sub_dir,
)

TASK_STATS_TASK_ID_LABEL_NAME = "task_id"

TaskStats = Dict[str, Union[List[int], List[float]]]
SchedulerStats = Dict[str, TaskStats]

UINT64_STATS_FIELD = "Uint64Stats"
TASK_STATS_EXECUTED_COUNT_INDEX = 3

RUNTIME_TOTAL_FIELD = "RuntimeTotal"
# time.Duration unit is the nano-second
GO_TIME_MICROSECOND = 1_000
GO_TIME_MILLISECOND = 1_000_000
GO_TIME_SECOND = 1_000_000_000

task_stats_uint64_delta_metric_names = {
    0: "lsvmi_task_scheduled_delta",
    1: "lsvmi_task_delayed_delta",
    2: "lsvmi_task_overrun_delta",
    3: "lsvmi_task_executed_delta",
    4: "lsvmi_task_deadline_hack_delta",
}

task_stats_interval_avg_runtime_metric = "lsvmi_task_interval_avg_runtime_sec"

test_cases_file = "scheduler.json"


def generate_task_stats_metrics(
    task_id: str,
    curr_task_stats: TaskStats,
    prev_task_stats: Optional[TaskStats] = None,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    ts: Optional[float] = None,
) -> List[str]:
    if ts is None:
        ts = time.time()
    promTs = str(int(ts * 1000))
    metrics = []
    executed_delta = None
    for i, name in task_stats_uint64_delta_metric_names.items():
        if name is None:
            continue
        val = curr_task_stats[UINT64_STATS_FIELD][i]
        if prev_task_stats is not None:
            val -= prev_task_stats[UINT64_STATS_FIELD][i]
        if i == TASK_STATS_EXECUTED_COUNT_INDEX:
            executed_delta = val
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
    if executed_delta is None:
        val = curr_task_stats[UINT64_STATS_FIELD][TASK_STATS_EXECUTED_COUNT_INDEX]
        if prev_task_stats is not None:
            val -= prev_task_stats[UINT64_STATS_FIELD][TASK_STATS_EXECUTED_COUNT_INDEX]
    interval_runtime_average = 0
    if executed_delta > 0:
        runtime_delta = curr_task_stats[RUNTIME_TOTAL_FIELD]
        if prev_task_stats is not None:
            runtime_delta -= prev_task_stats[RUNTIME_TOTAL_FIELD]
        interval_runtime_average = runtime_delta / GO_TIME_SECOND / executed_delta
    metrics.append(
        f"{task_stats_interval_avg_runtime_metric}{{"
        + ",".join(
            [
                f'{INSTANCE_LABEL_NAME}="{instance}"',
                f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                f'{TASK_STATS_TASK_ID_LABEL_NAME}="{task_id}"',
            ]
        )
        + f"}} {interval_runtime_average:.6f} {promTs}"
    )
    return metrics


def generate_scheduler_internal_metrics_test_case(
    name: str,
    curr_stats: SchedulerStats,
    prev_stats: Optional[SchedulerStats] = None,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    report_extra: bool = True,
    ts: Optional[float] = None,
) -> Dict[str, Any]:
    if ts is None:
        ts = time.time()
    prom_ts = int(ts * 1000)
    metrics = []
    for task_id, curr_task_stats in curr_stats.items():
        prev_task_stats = prev_stats.get(task_id) if prev_stats is not None else None
        metrics.extend(
            generate_task_stats_metrics(
                task_id,
                curr_task_stats,
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
        TC_WANT_METRICS_COUNT_FIELD: len(metrics),
        TC_WANT_METRICS_FIELD: metrics,
        TC_REPORT_EXTRA_FIELD: report_extra,
        TC_CURR_STATS_FIELD: curr_stats,
        TC_PREV_STATS_FIELD: prev_stats,
    }


def generate_scheduler_internal_metrics_test_cases(
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    test_cases_root_dir: Optional[str] = lsvmi_test_cases_root_dir,
):
    ts = time.time()

    if test_cases_root_dir not in {None, "", "-"}:
        out_file = os.path.join(
            test_cases_root_dir, test_cases_sub_dir, test_cases_file
        )
        os.makedirs(os.path.dirname(out_file), exist_ok=True)
        fp = open(out_file, "wt")
    else:
        out_file = None
        fp = sys.stdout

    stats_ref = {
        "taskA": {
            UINT64_STATS_FIELD: [0, 1, 2, 3, 4],
            RUNTIME_TOTAL_FIELD: 100 * GO_TIME_MILLISECOND,
        },
        "taskB": {
            UINT64_STATS_FIELD: [10, 11, 12, 13, 14],
            RUNTIME_TOTAL_FIELD: 200 * GO_TIME_MILLISECOND,
        },
    }

    test_cases = []
    tc_num = 0

    test_cases.append(
        generate_scheduler_internal_metrics_test_case(
            f"{tc_num:04d}",
            stats_ref,
            instance=instance,
            hostname=hostname,
            ts=ts,
        )
    )
    tc_num += 1

    curr_stats = deepcopy(stats_ref)
    k = 0
    for task_id in curr_stats:
        k += 1
        for i in range(len(curr_stats[task_id][UINT64_STATS_FIELD])):
            curr_stats[task_id][UINT64_STATS_FIELD][i] += 100 * k + i
        curr_stats[task_id][RUNTIME_TOTAL_FIELD] += 1 * GO_TIME_SECOND
    test_cases.append(
        generate_scheduler_internal_metrics_test_case(
            f"{tc_num:04d}",
            curr_stats,
            prev_stats=stats_ref,
            instance=instance,
            hostname=hostname,
            ts=ts,
        )
    )
    tc_num += 1

    prev_stats = deepcopy(stats_ref)
    for task_id in list(prev_stats):
        del prev_stats[task_id]
        break
    test_cases.append(
        generate_scheduler_internal_metrics_test_case(
            f"{tc_num:04d}",
            stats_ref,
            prev_stats=prev_stats,
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
