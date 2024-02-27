#! /usr/bin/env python3

# Generate test cases for lsvmi/http_endpoint_pool_internal_metrics_test.go

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

HttpEndpointStats = Dict[str, List[int]]
HttpEndpointPoolStats = Dict[str, Union[List[int], HttpEndpointStats]]
POOL_STATS_FIELD = "Stats"
ENDPOINT_STATS_FIELD = "EndpointStats"

HTTP_ENDPOINT_URL_LABEL_NAME = "url"

http_endpoint_metric_names = [
    "lsvmi_http_ep_send_buffer_count_delta",
    "lsvmi_http_ep_send_buffer_byte_count_delta",
    "lsvmi_http_ep_send_buffer_error_count_delta",
    "lsvmi_http_ep_healthcheck_count_delta",
    "lsvmi_http_ep_healthcheck_error_count_delta",
]

http_endpoint_pool_metric_names = [
    "lsvmi_http_ep_pool_healthy_rotate_count_delta",
    "lsvmi_http_ep_pool_no_healthy_ep_error_count_delta",
]

testcases_file = "http_endpoint_pool.json"


def generate_http_endpoint_metrics(
    url: str,
    ep_stats: HttpEndpointStats,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    ts: Optional[float] = None,
) -> List[str]:
    if ts is None:
        ts = time.time()
    prom_ts = int(ts * 1000)
    metrics = []

    for i, metric_name in enumerate(http_endpoint_metric_names):
        if metric_name is None:
            continue
        metrics.append(
            f"{metric_name}{{"
            + ",".join(
                [
                    f'{INSTANCE_LABEL_NAME}="{instance}"',
                    f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                    f'{HTTP_ENDPOINT_URL_LABEL_NAME}="{url}"',
                ]
            )
            + f"}} {ep_stats[i]} {prom_ts}"
        )
    return metrics


def generate_http_endpoint_pool_internal_metrics_test_case(
    name: str,
    stats: HttpEndpointPoolStats,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    report_extra: bool = True,
    ts: Optional[float] = None,
) -> Dict[str, Any]:
    if ts is None:
        ts = time.time()
    prom_ts = int(ts * 1000)

    metrics = []

    pool_stats = stats[POOL_STATS_FIELD]

    for i, metric_name in enumerate(http_endpoint_pool_metric_names):
        if metric_name is None:
            continue
        metrics.append(
            f"{metric_name}{{"
            + ",".join(
                [
                    f'{INSTANCE_LABEL_NAME}="{instance}"',
                    f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                ]
            )
            + f"}} {pool_stats[i]} {prom_ts}"
        )

    for url, ep_stats in stats[ENDPOINT_STATS_FIELD].items():
        metrics.extend(
            generate_http_endpoint_metrics(
                url,
                ep_stats,
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


def generate_http_endpoint_pool_internal_metrics_test_cases(
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
        POOL_STATS_FIELD: [1000, 1001],
        ENDPOINT_STATS_FIELD: {
            "http://test1": [10, 11, 12, 13, 14],
            "http://test2": [20, 21, 22, 23, 24],
        },
    }

    test_cases = []
    tc_num = 0

    test_cases.append(
        generate_http_endpoint_pool_internal_metrics_test_case(
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
