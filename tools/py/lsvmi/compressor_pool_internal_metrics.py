#! /usr/bin/env python3

# Generate test cases for lsvmi/compressor_pool_internal_metrics_test.go

import json
import os
import sys
import time
from typing import Any, Dict, List, Optional, Union

from . import (
    DEFAULT_TEST_HOSTNAME,
    DEFAULT_TEST_INSTANCE,
    HOSTNAME_LABEL_NAME,
    INSTANCE_LABEL_NAME,
    lsvmi_testcases_root,
)
from .internal_metrics import (
    TC_HOSTNAME_FIELD,
    TC_INSTANCE_FIELD,
    TC_NAME_FIELD,
    TC_PROM_TS_FIELD,
    TC_REPORT_EXTRA_FIELD,
    TC_STATS_FIELD,
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
    compressor_stats: CompressorStats,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    ts: Optional[float] = None,
) -> List[str]:
    if ts is None:
        ts = time.time()
    prom_ts = int(ts * 1000)
    metrics = []
    for i, metric_name in enumerate(compressor_stats_uint64_metric_names):
        if metric_name is None:
            continue
        metrics.append(
            f"{metric_name}{{"
            + ",".join(
                [
                    f'{INSTANCE_LABEL_NAME}="{instance}"',
                    f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                    f'{COMPRESSOR_ID_LABEL_NAME}="{compressor_id}"',
                ]
            )
            + f"}} {compressor_stats[UINT64_STATS_FIELD][i]} {prom_ts}"
        )
    for i, metric_name in enumerate(compressor_stats_float64_metric_names):
        if metric_name is None:
            continue
        metrics.append(
            f"{metric_name}{{"
            + ",".join(
                [
                    f'{INSTANCE_LABEL_NAME}="{instance}"',
                    f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                    f'{COMPRESSOR_ID_LABEL_NAME}="{compressor_id}"',
                ]
            )
            + f"}} {compressor_stats[FLOAT64_STATS_FIELD][i]:.3f} {prom_ts}"
        )
    return metrics


def generate_compressor_pool_internal_metrics_test_case(
    name: str,
    stats: CompressorPoolStats,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    report_extra: bool = True,
    ts: Optional[float] = None,
) -> Dict[str, Any]:
    if ts is None:
        ts = time.time()
    prom_ts = int(ts * 1000)
    metrics = []
    for compressor_id, compressor_stats in stats.items():
        metrics.extend(
            generate_compressor_metrics(
                compressor_id,
                compressor_stats,
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
        TC_STATS_FIELD: stats,
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

    stats_ref = {
        "0": {
            UINT64_STATS_FIELD: [1, 2, 3, 4, 5, 6, 7, 8],
            FLOAT64_STATS_FIELD: [3.0],
        },
        "1": {
            UINT64_STATS_FIELD: [11, 12, 13, 14, 15, 16, 17, 18],
            FLOAT64_STATS_FIELD: [3.1],
        },
    }

    test_cases = []
    tc_num = 0

    test_cases.append(
        generate_compressor_pool_internal_metrics_test_case(
            f"{tc_num:04d}",
            stats_ref,
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
