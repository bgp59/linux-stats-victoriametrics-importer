#! /usr/bin/env python3

# Generate test cases for lsvmi/qdisc_metrics_test.go

import time
from copy import deepcopy
from dataclasses import dataclass, field
from typing import List, Optional, Tuple

from qdisc import qdisc_parser as qdisc

from . import (
    DEFAULT_TEST_HOSTNAME,
    DEFAULT_TEST_INSTANCE,
    HOSTNAME_LABEL_NAME,
    INSTANCE_LABEL_NAME,
    lsvmi_test_cases_root_dir,
    save_test_cases,
    uint32_delta,
    uint64_delta,
)

DEFAULT_QDISC_INTERVAL_SEC = 1
DEFAULT_QDISC_FULL_METRICS_FACTOR = 15

QDISC_METRICS_ID = "qdisc_metrics"


QDISC_RATE_METRICS = "qdisc_rate_kbps"
QDISC_PACKETS_DELTA_METRIC = "qdisc_packets_delta"
QDISC_DROPS_DELTA_METRIC = "qdisc_drops_delta"
QDISC_REQUEUES_DELTA_METRIC = "qdisc_requeues_delta"
QDISC_OVERLIMITS_DELTA_METRIC = "qdisc_overlimits_delta"
QDISC_QLEN_METRIC = "qdisc_qlen"
QDISC_BACKLOG_METRIC = "qdisc_backlog"
QDISC_GCLOWS_DELTA_METRIC = "qdisc_gclows_delta"
QDISC_THROTTLED_DELTA_METRIC = "qdisc_throttled_delta"
QDISC_FLOWSPLIMIT_DELTA_METRIC = "qdisc_flowsplimit_delta"

QDISC_PRESENCE_METRIC = "qdisc_present"

QDISC_KIND_LABEL_NAME = "kind"
QDISC_HANDLE_LABEL_NAME = "handle"
QDISC_PARENT_LABEL_NAME = "parent"
QDISC_IF_LABEL_NAME = "if"  # interface

QDISC_INTERVAL_METRIC_NAME = "qdisc_metrics_delta_sec"

qdisc_uint32_index_to_delta_metric_name_map = {
    qdisc.QDISC_PACKETS: QDISC_PACKETS_DELTA_METRIC,
    qdisc.QDISC_DROPS: QDISC_DROPS_DELTA_METRIC,
    qdisc.QDISC_REQUEUES: QDISC_REQUEUES_DELTA_METRIC,
    qdisc.QDISC_OVERLIMITS: QDISC_OVERLIMITS_DELTA_METRIC,
}

qdisc_uint32_index_to_metric_name_map = {
    qdisc.QDISC_QLEN: QDISC_QLEN_METRIC,
    qdisc.QDISC_BACKLOG: QDISC_BACKLOG_METRIC,
}

qdisc_uint64_index_to_delta_metric_name_map = {
    qdisc.QDISC_BYTES: QDISC_RATE_METRICS,
    qdisc.QDISC_GCFLOWS: QDISC_GCLOWS_DELTA_METRIC,
    qdisc.QDISC_THROTTLED: QDISC_THROTTLED_DELTA_METRIC,
    qdisc.QDISC_FLOWSPLIMIT: QDISC_FLOWSPLIMIT_DELTA_METRIC,
}

qdisc_uint64_index_to_metric_name_map = {}

# Certain values are used to generate rates:
qdisc_uint32_index_rate = {}
qdisc_uint64_index_rate = {
    qdisc.QDISC_BYTES: (8.0 / 1000.0, 1),
}
QDISC_RATE_FACTOR = 0
QDISC_RATE_PREC = 1


@dataclass
class QdiscMetricsInfoTestData:
    QdiscInfoKey: qdisc.QdiscInfoKey
    Uint32ZeroDelta: List[bool] = field(
        default_factory=lambda: [False] * qdisc.QDISK_UINT32_NUM_STATS
    )
    Uint64ZeroDelta: List[bool] = field(
        default_factory=lambda: [False] * qdisc.QDISK_UINT64_NUM_STATS
    )
    CycleNum: int = 0


@dataclass
class QdiscStatsTestData:
    QdiscInfoKey: qdisc.QdiscInfoKey
    QdiscInfo: qdisc.QdiscInfo


@dataclass
class QdiscMetricsTestCase:
    Name: Optional[str] = None
    Description: Optional[str] = None
    Instance: str = DEFAULT_TEST_INSTANCE
    Hostname: str = DEFAULT_TEST_HOSTNAME
    CurrQdiscStats: Optional[List[QdiscStatsTestData]] = None
    PrevQdiscStats: Optional[List[QdiscStatsTestData]] = None
    CurrPromTs: int = 0
    PrevPromTs: int = 0
    QdiscMetricsInfo: Optional[List[QdiscMetricsInfoTestData]] = None
    FullMetricsFactor: int = DEFAULT_QDISC_FULL_METRICS_FACTOR
    WantMetricsCount: int = 0
    WantMetrics: Optional[List[str]] = None
    ReportExtra: bool = True
    WantQdiscMetricsInfo: Optional[List[QdiscMetricsInfoTestData]] = None


test_cases_file = "qdisc.json"


def format_qdisc_maj_min_label_val(val: int) -> str:
    return ":".join(
        [
            f"{val >> qdisc.QDISC_MIN_NUM_BITS:0{(qdisc.QDISC_MAJ_NUM_BITS + 3)//4}x}",
            f"{val & ((1 << qdisc.QDISC_MIN_NUM_BITS) - 1):0{(qdisc.QDISC_MIN_NUM_BITS + 3)//4}x}",
        ]
    )


def generate_qdisc_metrics(
    curr_qdisc_stats: QdiscStatsTestData,
    prev_qdisc_stats: QdiscStatsTestData,
    curr_prom_ts: int,
    interval: Optional[float] = DEFAULT_QDISC_INTERVAL_SEC,
    qdisc_metrics_info: Optional[List[QdiscMetricsInfoTestData]] = None,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
) -> Tuple[List[str], QdiscMetricsInfoTestData]:
    metrics = []
    want_qdisc_metrics_info = []

    prev_qdisc_stats_map = {
        qs_td.QdiscInfoKey: qs_td.QdiscInfo for qs_td in prev_qdisc_stats
    }
    qdisc_metrics_info_map = (
        {qmi_td.QdiscInfoKey: qmi_td for qmi_td in qdisc_metrics_info}
        if qdisc_metrics_info
        else {}
    )

    for curr_qs_ts in curr_qdisc_stats:
        qi_key, curr_qi = curr_qs_ts.QdiscInfoKey, curr_qs_ts.QdiscInfo
        prev_qi = prev_qdisc_stats_map.get(qi_key)
        if prev_qi is None:
            continue

        common_labels = ",".join(
            [
                f'{INSTANCE_LABEL_NAME}="{instance}"',
                f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                f'{QDISC_KIND_LABEL_NAME}="{curr_qi.Kind}"',
                f'{QDISC_HANDLE_LABEL_NAME}="{format_qdisc_maj_min_label_val(curr_qi.Uint32[qdisc.QDISC_HANDLE])}"',
                f'{QDISC_PARENT_LABEL_NAME}="{format_qdisc_maj_min_label_val(curr_qi.Uint32[qdisc.QDISC_PARENT])}"',
                f'{QDISC_IF_LABEL_NAME}="{curr_qi.IfName}"',
            ]
        )

        qmi = qdisc_metrics_info_map.get(qi_key)
        if qmi is not None and (
            curr_qi.Kind != prev_qi.Kind
            or curr_qi.Uint32[qdisc.QDISC_HANDLE] != prev_qi.Uint32[qdisc.QDISC_HANDLE]
            or curr_qi.Uint32[qdisc.QDISC_PARENT] != prev_qi.Uint32[qdisc.QDISC_PARENT]
        ):
            metrics.append(
                f"{QDISC_PRESENCE_METRIC}{{"
                + ",".join(
                    [
                        f'{INSTANCE_LABEL_NAME}="{instance}"',
                        f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                        f'{QDISC_KIND_LABEL_NAME}="{prev_qi.Kind}"',
                        f'{QDISC_HANDLE_LABEL_NAME}="{format_qdisc_maj_min_label_val(prev_qi.Uint32[qdisc.QDISC_HANDLE])}"',
                        f'{QDISC_PARENT_LABEL_NAME}="{format_qdisc_maj_min_label_val(prev_qi.Uint32[qdisc.QDISC_PARENT])}"',
                        f'{QDISC_IF_LABEL_NAME}="{prev_qi.IfName}"',
                    ]
                )
                + f"}} 0 {curr_prom_ts}"
            )
            qmi = None

        full_metrics = qmi is None or qmi.CycleNum == 0
        want_qmi = QdiscMetricsInfoTestData(QdiscInfoKey=qi_key)
        for i, name in qdisc_uint32_index_to_delta_metric_name_map.items():
            val = uint32_delta(curr_qi.Uint32[i], prev_qi.Uint32[i])
            if val > 0 or full_metrics or not qmi.Uint32ZeroDelta[i]:
                rate = qdisc_uint32_index_rate.get(i)
                if rate is not None:
                    val = f"{val * rate[QDISC_RATE_FACTOR] / interval:.{rate[QDISC_RATE_PREC]}f}"
                metrics.append(f"{name}{{{common_labels}}} {val} {curr_prom_ts}")
            want_qmi.Uint32ZeroDelta[i] = val == 0
        for i, name in qdisc_uint32_index_to_metric_name_map.items():
            val = curr_qi.Uint32[i]
            if val != prev_qi.Uint32[i] or full_metrics:
                metrics.append(f"{name}{{{common_labels}}} {val} {curr_prom_ts}")

        for i, name in qdisc_uint64_index_to_delta_metric_name_map.items():
            val = uint64_delta(curr_qi.Uint64[i], prev_qi.Uint64[i])
            if val > 0 or full_metrics or not qmi.Uint64ZeroDelta[i]:
                rate = qdisc_uint64_index_rate.get(i)
                if rate is not None:
                    val = f"{val * rate[QDISC_RATE_FACTOR] / interval:.{rate[QDISC_RATE_PREC]}f}"
                metrics.append(f"{name}{{{common_labels}}} {val} {curr_prom_ts}")
            want_qmi.Uint64ZeroDelta[i] = val == 0
        for i, name in qdisc_uint64_index_to_metric_name_map.items():
            val = curr_qi.Uint64[i]
            if val != prev_qi.Uint64[i] or full_metrics:
                metrics.append(f"{name}{{{common_labels}}} {val} {curr_prom_ts}")

        if full_metrics:
            metrics.append(
                f"{QDISC_PRESENCE_METRIC}{{{common_labels}}} 1 {curr_prom_ts}"
            )
        want_qdisc_metrics_info.append(want_qmi)

    metrics.append(
        f"{QDISC_INTERVAL_METRIC_NAME}{{"
        + ",".join(
            [
                f'{INSTANCE_LABEL_NAME}="{instance}"',
                f'{HOSTNAME_LABEL_NAME}="{hostname}"',
            ]
        )
        + f"}} {interval:.06f} {curr_prom_ts}"
    )
    return metrics, want_qdisc_metrics_info


def generate_qdisc_metrics_test_case(
    name: str,
    curr_qdisc_stats: QdiscStatsTestData,
    prev_qdisc_stats: QdiscStatsTestData,
    ts: Optional[float] = None,
    interval: Optional[float] = DEFAULT_QDISC_INTERVAL_SEC,
    qdisc_metrics_info: Optional[List[QdiscMetricsInfoTestData]] = None,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    description: Optional[str] = None,
) -> QdiscMetricsTestCase:
    if ts is None:
        ts = time.time()
    curr_prom_ts = int(ts * 1000)
    prev_prom_ts = curr_prom_ts - int(interval * 1000)
    metrics, want_qdisc_metrics_info = generate_qdisc_metrics(
        curr_qdisc_stats=curr_qdisc_stats,
        prev_qdisc_stats=prev_qdisc_stats,
        curr_prom_ts=curr_prom_ts,
        interval=interval,
        qdisc_metrics_info=qdisc_metrics_info,
        instance=instance,
        hostname=hostname,
    )
    return QdiscMetricsTestCase(
        Name=name,
        Description=description,
        Instance=instance,
        Hostname=hostname,
        CurrQdiscStats=curr_qdisc_stats,
        PrevQdiscStats=prev_qdisc_stats,
        CurrPromTs=curr_prom_ts,
        PrevPromTs=prev_prom_ts,
        QdiscMetricsInfo=qdisc_metrics_info,
        WantMetricsCount=len(metrics),
        WantMetrics=metrics,
        ReportExtra=True,
        WantQdiscMetricsInfo=want_qdisc_metrics_info,
    )


def make_ref_qdisc_stats(
    num_if: int = 2, num_qdisc_per_if: int = 2
) -> QdiscStatsTestData:
    qs_td = []
    maj_mask = (1 << qdisc.QDISC_MAJ_NUM_BITS) - 1
    min_mask = (1 << qdisc.QDISC_MIN_NUM_BITS) - 1
    for if_index in range(num_if):
        if_name = f"eth{if_index}"
        for j in range(num_qdisc_per_if):
            parent = ((if_index + 1) & maj_mask) << qdisc.QDISC_MIN_NUM_BITS
            handle = parent + ((j + 1) & min_mask)
            uint32 = [
                (2 + if_index) * qdisc.QDISK_UINT32_NUM_STATS + k
                for k in range(qdisc.QDISK_UINT32_NUM_STATS)
            ]
            uint32[qdisc.QDISC_PARENT] = parent
            uint32[qdisc.QDISC_HANDLE] = handle
            uint64 = [
                (20 + if_index) * qdisc.QDISK_UINT64_NUM_STATS + k
                for k in range(qdisc.QDISK_UINT64_NUM_STATS)
            ]
            qs_td.append(
                QdiscStatsTestData(
                    QdiscInfoKey=qdisc.QdiscInfoKey(
                        IfIndex=if_index,
                        Handle=handle,
                    ),
                    QdiscInfo=qdisc.QdiscInfo(
                        IfName=if_name,
                        Kind="qdisc",
                        Uint32=uint32,
                        Uint64=uint64,
                    ),
                )
            )
    return qs_td


def generate_qdisc_metrics_test_cases(
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    test_cases_root_dir: Optional[str] = lsvmi_test_cases_root_dir,
):
    num_if = 2
    num_qdisc_per_if = 2
    ts = time.time()
    qdisc_stats_ref = make_ref_qdisc_stats(
        num_if=num_if, num_qdisc_per_if=num_qdisc_per_if
    )
    max_uint32_val = max(max(qs_td.QdiscInfo.Uint32) for qs_td in qdisc_stats_ref)
    max_uint64_val = max(max(qs_td.QdiscInfo.Uint64) for qs_td in qdisc_stats_ref)
    parent_handle_index = {qdisc.QDISC_PARENT, qdisc.QDISC_HANDLE}

    test_cases = []
    tc_num = 0

    name = "all_new"
    curr_qdisc_stats = qdisc_stats_ref
    prev_qdisc_stats = deepcopy(curr_qdisc_stats)
    for i, qs_td in enumerate(prev_qdisc_stats):
        for j, val in enumerate(qs_td.QdiscInfo.Uint32):
            if j in parent_handle_index:
                continue
            qs_td.QdiscInfo.Uint32[j] = uint32_delta(
                val, (i + 1) * max_uint32_val + 7 * j
            )
        for j, val in enumerate(qs_td.QdiscInfo.Uint64):
            qs_td.QdiscInfo.Uint64[j] = uint64_delta(
                val, (i + 1) * max_uint64_val + 7 * j
            )
    test_cases.append(
        generate_qdisc_metrics_test_case(
            name=f"{name}/{tc_num:04d}",
            curr_qdisc_stats=curr_qdisc_stats,
            prev_qdisc_stats=prev_qdisc_stats,
            ts=ts,
            instance=instance,
            hostname=hostname,
        )
    )
    tc_num += 1

    save_test_cases(
        test_cases, test_cases_file, test_cases_root_dir=test_cases_root_dir
    )
