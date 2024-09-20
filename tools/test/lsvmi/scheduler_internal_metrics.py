#! /usr/bin/env python3

# Generate test cases for lsvmi/scheduler_internal_metrics_test.go

import os
import time
from copy import deepcopy
from dataclasses import dataclass
from typing import List, Optional

from . import (
    DEFAULT_TEST_HOSTNAME,
    DEFAULT_TEST_INSTANCE,
    HOSTNAME_LABEL_NAME,
    INSTANCE_LABEL_NAME,
    lsvmi_test_cases_root_dir,
    save_test_cases,
    uint64_delta,
)
from .internal_metrics import InternalMetricsTestCase, test_cases_sub_dir
from .scheduler_stats import (
    TASK_STATS_DEADLINE_HACK_COUNT,
    TASK_STATS_DELAYED_COUNT,
    TASK_STATS_EXECUTED_COUNT,
    TASK_STATS_OVERRUN_COUNT,
    TASK_STATS_SCHEDULED_COUNT,
    TASK_STATS_UINT64_LEN,
    SchedulerStats,
    TaskStats,
)

TASK_STATS_SCHEDULED_DELTA_METRIC = "lsvmi_task_scheduled_delta"
TASK_STATS_DELAYED_DELTA_METRIC = "lsvmi_task_delayed_delta"
TASK_STATS_OVERRUN_DELTA_METRIC = "lsvmi_task_overrun_delta"
TASK_STATS_EXECUTED_DELTA_METRIC = "lsvmi_task_executed_delta"
TASK_STATS_DEADLINE_HACK_DELTA_METRIC = "lsvmi_task_deadline_hack_delta"
TASK_STATS_INTERVAL_AVG_RUNTIME_METRIC = "lsvmi_task_interval_avg_runtime_sec"

TASK_STATS_TASK_ID_LABEL_NAME = "task_id"

# time.Duration unit is the nano-second
GO_TIME_MICROSECOND = 1_000
GO_TIME_MILLISECOND = 1_000_000
GO_TIME_SECOND = 1_000_000_000


@dataclass
class SchedulerInternalMetricsTestCase(InternalMetricsTestCase):
    CurrStats: Optional[SchedulerStats] = None
    PrevStats: Optional[SchedulerStats] = None


task_stats_uint64_delta_metric_names = {
    TASK_STATS_SCHEDULED_COUNT: TASK_STATS_SCHEDULED_DELTA_METRIC,
    TASK_STATS_DELAYED_COUNT: TASK_STATS_DELAYED_DELTA_METRIC,
    TASK_STATS_OVERRUN_COUNT: TASK_STATS_OVERRUN_DELTA_METRIC,
    TASK_STATS_EXECUTED_COUNT: TASK_STATS_EXECUTED_DELTA_METRIC,
    TASK_STATS_DEADLINE_HACK_COUNT: TASK_STATS_DEADLINE_HACK_DELTA_METRIC,
}

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
    for i, name in task_stats_uint64_delta_metric_names.items():
        if name is None:
            continue
        val = curr_task_stats.Uint64Stats[i]
        if prev_task_stats is not None:
            val -= prev_task_stats.Uint64Stats[i]
        if i == TASK_STATS_EXECUTED_COUNT:
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
    executed_delta = curr_task_stats.Uint64Stats[TASK_STATS_EXECUTED_COUNT]
    if prev_task_stats is not None:
        executed_delta = uint64_delta(
            executed_delta,
            prev_task_stats.Uint64Stats[TASK_STATS_EXECUTED_COUNT],
        )
    if executed_delta > 0:
        runtime_delta = curr_task_stats.RuntimeTotal
        if prev_task_stats is not None:
            runtime_delta -= prev_task_stats.RuntimeTotal
        interval_runtime_average = runtime_delta / GO_TIME_SECOND / executed_delta
    else:
        interval_runtime_average = 0
    metrics.append(
        f"{TASK_STATS_INTERVAL_AVG_RUNTIME_METRIC}{{"
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
    ts: Optional[float] = None,
    description: Optional[str] = None,
) -> SchedulerInternalMetricsTestCase:
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
    return SchedulerInternalMetricsTestCase(
        Name=name,
        Description=description,
        Instance=instance,
        Hostname=hostname,
        PromTs=prom_ts,
        WantMetricsCount=len(metrics),
        WantMetrics=metrics,
        ReportExtra=True,
        CurrStats=curr_stats,
        PrevStats=prev_stats,
    )


def make_ref_scheduler_stats(num_tasks: int = 2) -> SchedulerStats:
    stats = {}
    for i in range(num_tasks):
        stats[f"task{i}"] = TaskStats(
            Uint64Stats=[
                2 * TASK_STATS_UINT64_LEN * i + j for j in range(TASK_STATS_UINT64_LEN)
            ],
            RuntimeTotal=(i + 1) * 100 * GO_TIME_MILLISECOND,
        )
    return stats


def generate_scheduler_internal_metrics_test_cases(
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    test_cases_root_dir: Optional[str] = lsvmi_test_cases_root_dir,
):
    ts = time.time()

    num_tasks = 2
    stats_ref = make_ref_scheduler_stats(num_tasks=num_tasks)

    test_cases = []
    tc_num = 0

    name = "no_prev"
    test_cases.append(
        generate_scheduler_internal_metrics_test_case(
            f"{name}/{tc_num:04d}",
            stats_ref,
            instance=instance,
            hostname=hostname,
            ts=ts,
        )
    )
    tc_num += 1

    name = "all_change"
    curr_stats = deepcopy(stats_ref)
    k = 0
    for task_stats in curr_stats.values():
        k += 1
        for i in range(TASK_STATS_UINT64_LEN):
            task_stats.Uint64Stats[i] += 100 * k + i
        task_stats.RuntimeTotal += 1 * GO_TIME_SECOND
    test_cases.append(
        generate_scheduler_internal_metrics_test_case(
            f"{name}/{tc_num:04d}",
            curr_stats,
            prev_stats=stats_ref,
            instance=instance,
            hostname=hostname,
            ts=ts,
        )
    )
    tc_num += 1

    name = "new_task"
    curr_stats = stats_ref
    for new_task_id in curr_stats:
        prev_stats = deepcopy(stats_ref)
        del prev_stats[new_task_id]
        test_cases.append(
            generate_scheduler_internal_metrics_test_case(
                f"{tc_num:04d}",
                stats_ref,
                prev_stats=prev_stats,
                instance=instance,
                hostname=hostname,
                ts=ts,
                description=f"new_task_id={new_task_id}",
            )
        )
        tc_num += 1

    save_test_cases(
        test_cases,
        test_cases_file=test_cases_file,
        test_cases_root_dir=os.path.join(
            lsvmi_test_cases_root_dir,
            test_cases_sub_dir,
        ),
    )
