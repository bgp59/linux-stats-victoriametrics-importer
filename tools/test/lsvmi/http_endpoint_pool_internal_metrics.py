#! /usr/bin/env python3

# Generate test cases for lsvmi/http_endpoint_pool_internal_metrics_test.go

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
from .http_endpoint_pool_stats import (
    HTTP_ENDPOINT_POOL_STATS_HEALTHY_ROTATE_COUNT,
    HTTP_ENDPOINT_POOL_STATS_LEN,
    HTTP_ENDPOINT_POOL_STATS_NO_HEALTHY_EP_ERROR_COUNT,
    HTTP_ENDPOINT_STATS_HEALTH_CHECK_COUNT,
    HTTP_ENDPOINT_STATS_HEALTH_CHECK_ERROR_COUNT,
    HTTP_ENDPOINT_STATS_LEN,
    HTTP_ENDPOINT_STATS_SEND_BUFFER_BYTE_COUNT,
    HTTP_ENDPOINT_STATS_SEND_BUFFER_COUNT,
    HTTP_ENDPOINT_STATS_SEND_BUFFER_ERROR_COUNT,
    HTTP_ENDPOINT_STATS_STATE,
    HttpEndpointPoolStats,
    HttpEndpointStats,
    endpoint_state_name_map,
)
from .internal_metrics import InternalMetricsTestCase, test_cases_sub_dir


@dataclass
class HttpEndpointPoolInternalMetricsTestCase(InternalMetricsTestCase):
    CurrStats: Optional[HttpEndpointPoolStats] = None
    PrevStats: Optional[HttpEndpointPoolStats] = None


HTTP_ENDPOINT_STATS_SEND_BUFFER_DELTA_METRIC = "lsvmi_http_ep_send_buffer_delta"
HTTP_ENDPOINT_STATS_SEND_BUFFER_BYTE_DELTA_METRIC = (
    "lsvmi_http_ep_send_buffer_byte_delta"
)
HTTP_ENDPOINT_STATS_SEND_BUFFER_ERROR_DELTA_METRIC = (
    "lsvmi_http_ep_send_buffer_error_delta"
)
HTTP_ENDPOINT_STATS_HEALTH_CHECK_DELTA_METRIC = "lsvmi_http_ep_healthcheck_delta"
HTTP_ENDPOINT_STATS_HEALTH_CHECK_ERROR_DELTA_METRIC = (
    "lsvmi_http_ep_healthcheck_error_delta"
)
HTTP_ENDPOINT_STATS_STATE_METRIC = "lsvmi_http_ep_state"

HTTP_ENDPOINT_POOL_STATS_HEALTHY_ROTATE_COUNT_METRIC = (
    "lsvmi_http_ep_pool_healthy_rotate_count"
)
HTTP_ENDPOINT_POOL_STATS_NO_HEALTHY_EP_ERROR_DELTA_METRIC = (
    "lsvmi_http_ep_pool_no_healthy_ep_error_delta"
)

HTTP_ENDPOINT_URL_LABEL_NAME = "url"


http_endpoint_delta_metric_names = {
    HTTP_ENDPOINT_STATS_SEND_BUFFER_COUNT: HTTP_ENDPOINT_STATS_SEND_BUFFER_DELTA_METRIC,
    HTTP_ENDPOINT_STATS_SEND_BUFFER_BYTE_COUNT: HTTP_ENDPOINT_STATS_SEND_BUFFER_BYTE_DELTA_METRIC,
    HTTP_ENDPOINT_STATS_SEND_BUFFER_ERROR_COUNT: HTTP_ENDPOINT_STATS_SEND_BUFFER_ERROR_DELTA_METRIC,
    HTTP_ENDPOINT_STATS_HEALTH_CHECK_COUNT: HTTP_ENDPOINT_STATS_HEALTH_CHECK_DELTA_METRIC,
    HTTP_ENDPOINT_STATS_HEALTH_CHECK_ERROR_COUNT: HTTP_ENDPOINT_STATS_HEALTH_CHECK_ERROR_DELTA_METRIC,
}

http_endpoint_metric_names = {
    HTTP_ENDPOINT_STATS_STATE: HTTP_ENDPOINT_STATS_STATE_METRIC,
}

http_endpoint_pool_delta_metric_names = {
    HTTP_ENDPOINT_POOL_STATS_NO_HEALTHY_EP_ERROR_COUNT: HTTP_ENDPOINT_POOL_STATS_NO_HEALTHY_EP_ERROR_DELTA_METRIC,
}

http_endpoint_pool_metric_names = {
    HTTP_ENDPOINT_POOL_STATS_HEALTHY_ROTATE_COUNT: HTTP_ENDPOINT_POOL_STATS_HEALTHY_ROTATE_COUNT_METRIC,
}

test_cases_file = "http_endpoint_pool.json"


def generate_http_endpoint_metrics(
    url: str,
    curr_ep_stats: HttpEndpointStats,
    prev_ep_stats: Optional[HttpEndpointStats] = None,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    ts: Optional[float] = None,
) -> List[str]:
    if ts is None:
        ts = time.time()
    prom_ts = int(ts * 1000)
    metrics = []

    for i, metric_name in http_endpoint_delta_metric_names.items():
        val = curr_ep_stats[i]
        if prev_ep_stats is not None:
            val -= prev_ep_stats[i]
        metrics.append(
            f"{metric_name}{{"
            + ",".join(
                [
                    f'{INSTANCE_LABEL_NAME}="{instance}"',
                    f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                    f'{HTTP_ENDPOINT_URL_LABEL_NAME}="{url}"',
                ]
            )
            + f"}} {val} {prom_ts}"
        )
    for i, metric_name in http_endpoint_metric_names.items():
        val = curr_ep_stats[i]
        metrics.append(
            f"{metric_name}{{"
            + ",".join(
                [
                    f'{INSTANCE_LABEL_NAME}="{instance}"',
                    f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                    f'{HTTP_ENDPOINT_URL_LABEL_NAME}="{url}"',
                ]
            )
            + f"}} {val} {prom_ts}"
        )
    return metrics


def generate_http_endpoint_pool_internal_metrics_test_case(
    name: str,
    curr_stats: HttpEndpointPoolStats,
    prev_stats: Optional[HttpEndpointPoolStats] = None,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    report_extra: bool = True,
    ts: Optional[float] = None,
    description: Optional[str] = None,
) -> HttpEndpointPoolInternalMetricsTestCase:
    if ts is None:
        ts = time.time()
    prom_ts = int(ts * 1000)

    metrics = []

    curr_pool_stats = curr_stats.PoolStats
    prev_pool_stats = prev_stats.PoolStats if prev_stats is not None else None

    for i, metric_name in http_endpoint_pool_metric_names.items():
        val = curr_pool_stats[i]
        metrics.append(
            f"{metric_name}{{"
            + ",".join(
                [
                    f'{INSTANCE_LABEL_NAME}="{instance}"',
                    f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                ]
            )
            + f"}} {val} {prom_ts}"
        )
    for i, metric_name in http_endpoint_pool_delta_metric_names.items():
        val = curr_pool_stats[i]
        if prev_pool_stats is not None:
            val -= prev_pool_stats[i]
        metrics.append(
            f"{metric_name}{{"
            + ",".join(
                [
                    f'{INSTANCE_LABEL_NAME}="{instance}"',
                    f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                ]
            )
            + f"}} {val} {prom_ts}"
        )

    for url, curr_ep_stats in curr_stats.EndpointStats.items():
        prev_ep_stats = (
            prev_stats.EndpointStats.get(url) if prev_stats is not None else None
        )
        metrics.extend(
            generate_http_endpoint_metrics(
                url,
                curr_ep_stats,
                prev_ep_stats=prev_ep_stats,
                instance=instance,
                hostname=hostname,
                ts=ts,
            )
        )
    return HttpEndpointPoolInternalMetricsTestCase(
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


def make_ref_http_endpoint_pool_stats(
    num_ep: int = len(endpoint_state_name_map),
) -> HttpEndpointPoolStats:
    pool_ep_stats = {}
    ep_states = sorted(endpoint_state_name_map)
    for i in range(num_ep):
        ep_stats = [
            i * 2 * HTTP_ENDPOINT_STATS_LEN + j for j in range(HTTP_ENDPOINT_STATS_LEN)
        ]
        ep_stats[HTTP_ENDPOINT_STATS_STATE] = ep_states[i % len(ep_states)]
        pool_ep_stats[f"http://test{i}"] = ep_stats
    return HttpEndpointPoolStats(
        PoolStats=[
            1000 * HTTP_ENDPOINT_POOL_STATS_LEN + i
            for i in range(HTTP_ENDPOINT_POOL_STATS_LEN)
        ],
        EndpointStats=pool_ep_stats,
    )


def generate_http_endpoint_pool_internal_metrics_test_cases(
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    test_cases_root_dir: Optional[str] = lsvmi_test_cases_root_dir,
):
    ts = time.time()

    num_ep = len(endpoint_state_name_map)
    stats_ref = make_ref_http_endpoint_pool_stats(num_ep=num_ep)

    test_cases = []
    tc_num = 0

    name = "no_prev"
    test_cases.append(
        generate_http_endpoint_pool_internal_metrics_test_case(
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
    for i in range(HTTP_ENDPOINT_POOL_STATS_LEN):
        curr_stats.PoolStats[i] += HTTP_ENDPOINT_POOL_STATS_LEN * (i + 1)
    k = 0
    for url, ep_stats in curr_stats.EndpointStats.items():
        k += 1
        for i in range(HTTP_ENDPOINT_STATS_LEN):
            if i != HTTP_ENDPOINT_STATS_STATE:
                ep_stats[i] += k * HTTP_ENDPOINT_STATS_LEN + i
            else:
                states = [s for s in endpoint_state_name_map if s != ep_stats[i]]
                ep_stats[i] = states[k % len(states)]
    test_cases.append(
        generate_http_endpoint_pool_internal_metrics_test_case(
            f"{name}/{tc_num:04d}",
            curr_stats,
            prev_stats=stats_ref,
            instance=instance,
            hostname=hostname,
            ts=ts,
        )
    )
    tc_num += 1

    name = "new_endpoint"
    curr_stats = stats_ref
    for url in curr_stats.EndpointStats:
        prev_stats = deepcopy(curr_stats)
        del prev_stats.EndpointStats[url]
        test_cases.append(
            generate_http_endpoint_pool_internal_metrics_test_case(
                f"{name}/{tc_num:04d}",
                curr_stats,
                prev_stats=prev_stats,
                instance=instance,
                hostname=hostname,
                ts=ts,
                description=f"new_endpoint={url}",
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
