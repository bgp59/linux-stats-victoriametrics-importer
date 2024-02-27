#! /usr/bin/env python3

# Generate test cases for lsvmi/compressor_pool_internal_metrics_test.go

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
from .internal_metrics import (
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

COMPRESSOR_ID_LABEL_NAME = "compressor"

CompressorStats = Dict[str, Union[List[int], List[float]]]
CompressorPoolStats = List[CompressorStats]
UINT64_STATS_FIELD = "Uint64Stats"
FLOAT64_STATS_FIELD = "Float64Stats"


compressor_stats_uint64_metric_names = [
    "lsvmi_compressor_read_count_delta",
    "lsvmi_compressor_read_byte_count_delta",
    "lsvmi_compressor_send_count_delta",
    "lsvmi_compressor_send_byte_count_delta",
    "lsvmi_compressor_tout_flush_count_delta",
    "lsvmi_compressor_send_error_count_delta",
    "lsvmi_compressor_write_error_count_delta",
]

compressor_stats_float64_metric_names = [
    "lsvmi_compressor_compression_factor",
]

testcases_file = "compressor_pool.json"


def generate_compressor_metrics(
    compressor_id: str,
    crt_compressor_stats: CompressorStats,
    prev_compressor_stats: Optional[CompressorStats] = None,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    ts: Optional[float] = None,
) -> List[str]:
    if ts is None:
        ts = time.time()
    prom_ts = int(ts * 1000)
    metrics = []
    for i, name in enumerate(compressor_stats_uint64_metric_names):
        if name is None:
            continue
        val = crt_compressor_stats[UINT64_STATS_FIELD][i]
        if (
            prev_compressor_stats is None
            or val != prev_compressor_stats[UINT64_STATS_FIELD][i]
        ):
            metrics.append(
                f"{name}{{"
                + ",".join(
                    [
                        f'{INSTANCE_LABEL_NAME}="{instance}"',
                        f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                        f'{COMPRESSOR_ID_LABEL_NAME}="{compressor_id}"',
                    ]
                )
                + f"}} {val} {prom_ts}"
            )
    for i, name in enumerate(compressor_stats_float64_metric_names):
        if name is None:
            continue
        val = crt_compressor_stats[FLOAT64_STATS_FIELD][i]
        if (
            prev_compressor_stats is None
            or val != prev_compressor_stats[FLOAT64_STATS_FIELD][i]
        ):
            metrics.append(
                f"{name}{{"
                + ",".join(
                    [
                        f'{INSTANCE_LABEL_NAME}="{instance}"',
                        f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                        f'{COMPRESSOR_ID_LABEL_NAME}="{compressor_id}"',
                    ]
                )
                + f"}} {val:.3f} {prom_ts}"
            )
    return metrics


def generate_compressor_pool_internal_metrics_test_case(
    name: str,
    crt_stats: CompressorPoolStats,
    prev_stats: Optional[CompressorPoolStats] = None,
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
    for compressor_d, crt_compressor_stats in crt_stats.items():
        if not full_cycle and prev_stats is not None:
            prev_compressor_stats = prev_stats.get(compressor_d)
        else:
            prev_compressor_stats = None
        metrics.extend(
            generate_compressor_metrics(
                compressor_d,
                crt_compressor_stats,
                prev_compressor_stats=prev_compressor_stats,
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


def generate_compressor_pool_internal_metrics_test_cases(
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
        "0": {
            UINT64_STATS_FIELD: [0, 1, 2, 3, 4, 5, 6, 7],
            FLOAT64_STATS_FIELD: [3.0],
        },
        "1": {
            UINT64_STATS_FIELD: [10, 11, 12, 13, 14, 15, 16, 17],
            FLOAT64_STATS_FIELD: [3.1],
        },
    }

    test_cases = []
    tc_num = 0

    prev_stats = None
    test_cases.append(
        generate_compressor_pool_internal_metrics_test_case(
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

    for compressor_d in crt_stats:
        for i in range(len(compressor_stats_uint64_metric_names)):
            prev_stats = deepcopy(crt_stats)
            prev_stats[compressor_d][UINT64_STATS_FIELD][i] += 1000
            for full_cycle in [False, True]:
                test_cases.append(
                    generate_compressor_pool_internal_metrics_test_case(
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
        for i in range(len(compressor_stats_float64_metric_names)):
            prev_stats = deepcopy(crt_stats)
            prev_stats[compressor_d][FLOAT64_STATS_FIELD][i] += 1000
            for full_cycle in [False, True]:
                test_cases.append(
                    generate_compressor_pool_internal_metrics_test_case(
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

    for compressor_id in crt_stats:
        prev_stats = deepcopy(crt_stats)
        del prev_stats[compressor_id]
        test_cases.append(
            generate_compressor_pool_internal_metrics_test_case(
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
