#! /usr/bin/env python3

# Generate test cases for lsvmi/compressor_pool_internal_metrics_test.go

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
)
from .compressor_stats import (
    COMPRESSOR_STATS_COMPRESSION_FACTOR,
    COMPRESSOR_STATS_FLOAT64_LEN,
    COMPRESSOR_STATS_READ_BYTE_COUNT,
    COMPRESSOR_STATS_READ_COUNT,
    COMPRESSOR_STATS_SEND_BYTE_COUNT,
    COMPRESSOR_STATS_SEND_COUNT,
    COMPRESSOR_STATS_SEND_ERROR_COUNT,
    COMPRESSOR_STATS_TIMEOUT_FLUSH_COUNT,
    COMPRESSOR_STATS_UINT64_LEN,
    COMPRESSOR_STATS_WRITE_ERROR_COUNT,
    CompressorPoolStats,
    CompressorStats,
)
from .internal_metrics import InternalMetricsTestCase, test_cases_sub_dir

COMPRESSOR_STATS_READ_DELTA_METRIC = "lsvmi_compressor_read_delta"
COMPRESSOR_STATS_READ_BYTE_DELTA_METRIC = "lsvmi_compressor_read_byte_delta"
COMPRESSOR_STATS_SEND_DELTA_METRIC = "lsvmi_compressor_send_delta"
COMPRESSOR_STATS_SEND_BYTE_DELTA_METRIC = "lsvmi_compressor_send_byte_delta"
COMPRESSOR_STATS_TIMEOUT_FLUSH_DELTA_METRIC = "lsvmi_compressor_tout_flush_delta"
COMPRESSOR_STATS_SEND_ERROR_DELTA_METRIC = "lsvmi_compressor_send_error_delta"
COMPRESSOR_STATS_WRITE_ERROR_DELTA_METRIC = "lsvmi_compressor_write_error_delta"
COMPRESSOR_STATS_COMPRESSION_FACTOR_METRIC = "lsvmi_compressor_compression_factor"

COMPRESSOR_ID_LABEL_NAME = "compressor"


@dataclass
class CompressorPoolInternalMetricsTestCase(InternalMetricsTestCase):
    CurrStats: Optional[CompressorPoolStats] = None
    PrevStats: Optional[CompressorPoolStats] = None


compressor_stats_uint64_delta_metric_names = {
    COMPRESSOR_STATS_READ_COUNT: COMPRESSOR_STATS_READ_DELTA_METRIC,
    COMPRESSOR_STATS_READ_BYTE_COUNT: COMPRESSOR_STATS_READ_BYTE_DELTA_METRIC,
    COMPRESSOR_STATS_SEND_COUNT: COMPRESSOR_STATS_SEND_DELTA_METRIC,
    COMPRESSOR_STATS_SEND_BYTE_COUNT: COMPRESSOR_STATS_SEND_BYTE_DELTA_METRIC,
    COMPRESSOR_STATS_TIMEOUT_FLUSH_COUNT: COMPRESSOR_STATS_TIMEOUT_FLUSH_DELTA_METRIC,
    COMPRESSOR_STATS_SEND_ERROR_COUNT: COMPRESSOR_STATS_SEND_ERROR_DELTA_METRIC,
    COMPRESSOR_STATS_WRITE_ERROR_COUNT: COMPRESSOR_STATS_WRITE_ERROR_DELTA_METRIC,
}

compressor_stats_float64_metric_names = {
    COMPRESSOR_STATS_COMPRESSION_FACTOR: COMPRESSOR_STATS_COMPRESSION_FACTOR_METRIC,
}

test_cases_file = "compressor_pool.json"


def generate_compressor_metrics(
    compressor_id: str,
    curr_compressor_stats: CompressorStats,
    prev_compressor_stats: Optional[CompressorStats] = None,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    ts: Optional[float] = None,
) -> List[str]:
    if ts is None:
        ts = time.time()
    prom_ts = int(ts * 1000)
    metrics = []
    for i, metric_name in compressor_stats_uint64_delta_metric_names.items():
        val = curr_compressor_stats.Uint64Stats[i]
        if prev_compressor_stats is not None:
            val -= prev_compressor_stats.Uint64Stats[i]
        metrics.append(
            f"{metric_name}{{"
            + ",".join(
                [
                    f'{INSTANCE_LABEL_NAME}="{instance}"',
                    f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                    f'{COMPRESSOR_ID_LABEL_NAME}="{compressor_id}"',
                ]
            )
            + f"}} {val} {prom_ts}"
        )
    for i, metric_name in compressor_stats_float64_metric_names.items():
        val = curr_compressor_stats.Float64Stats[i]
        metrics.append(
            f"{metric_name}{{"
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
    curr_stats: CompressorPoolStats,
    prev_stats: Optional[CompressorPoolStats] = None,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    ts: Optional[float] = None,
    description: Optional[str] = None,
) -> CompressorPoolInternalMetricsTestCase:
    if ts is None:
        ts = time.time()
    prom_ts = int(ts * 1000)
    metrics = []
    for compressor_id, curr_compressor_stats in curr_stats.items():
        prev_compressor_stats = (
            prev_stats.get(compressor_id) if prev_stats is not None else None
        )
        metrics.extend(
            generate_compressor_metrics(
                compressor_id,
                curr_compressor_stats,
                prev_compressor_stats=prev_compressor_stats,
                instance=instance,
                hostname=hostname,
                ts=ts,
            )
        )
    return CompressorPoolInternalMetricsTestCase(
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


def make_ref_compressor_pool_stats(num_compressors: int = 2) -> CompressorPoolStats:
    stats = {}
    for i in range(num_compressors):
        stats[str(i)] = CompressorStats(
            Uint64Stats=[
                i * 2 * COMPRESSOR_STATS_UINT64_LEN + j
                for j in range(COMPRESSOR_STATS_UINT64_LEN)
            ],
            Float64Stats=[
                (i * 2 * COMPRESSOR_STATS_FLOAT64_LEN + j) / 13
                for j in range(COMPRESSOR_STATS_FLOAT64_LEN)
            ],
        )
    return stats


def generate_compressor_pool_internal_metrics_test_cases(
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    test_cases_root_dir: Optional[str] = lsvmi_test_cases_root_dir,
):
    ts = time.time()

    num_compressors = 2
    stats_ref = make_ref_compressor_pool_stats(num_compressors=num_compressors)

    test_cases = []
    tc_num = 0

    name = "no_prev"
    test_cases.append(
        generate_compressor_pool_internal_metrics_test_case(
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
    for compressor_id in curr_stats:
        k += 1
        for i in range(COMPRESSOR_STATS_UINT64_LEN):
            curr_stats[compressor_id].Uint64Stats[i] += (
                k * 10 * COMPRESSOR_STATS_UINT64_LEN + 13 * i
            )
        for i in range(COMPRESSOR_STATS_FLOAT64_LEN):
            curr_stats[compressor_id].Float64Stats[i] += (
                k * COMPRESSOR_STATS_UINT64_LEN + 1.3 * i
            )
    test_cases.append(
        generate_compressor_pool_internal_metrics_test_case(
            f"{name}/{tc_num:04d}",
            curr_stats,
            prev_stats=stats_ref,
            instance=instance,
            hostname=hostname,
            ts=ts,
        )
    )
    tc_num += 1

    name = "new_compressor"
    curr_stats = stats_ref
    prev_stats = deepcopy(stats_ref)
    for compressor_id in curr_stats:
        prev_stats = deepcopy(stats_ref)
        del prev_stats[compressor_id]
        test_cases.append(
            generate_compressor_pool_internal_metrics_test_case(
                f"{name}/{tc_num:04d}",
                stats_ref,
                prev_stats=prev_stats,
                instance=instance,
                hostname=hostname,
                ts=ts,
                description=f"new_compressor_id={compressor_id}",
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
