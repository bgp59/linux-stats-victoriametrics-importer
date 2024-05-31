#! /usr/bin/env python3

# Generate test cases for lsvmi/proc_net_dev_metrics_test.go

import time
from copy import deepcopy
from dataclasses import dataclass, field
from typing import Dict, List, Optional, Tuple

import procfs

from . import (
    DEFAULT_TEST_HOSTNAME,
    DEFAULT_TEST_INSTANCE,
    HOSTNAME_LABEL_NAME,
    INSTANCE_LABEL_NAME,
    lsvmi_test_cases_root_dir,
    save_test_cases,
    uint64_delta,
)

DEFAULT_PROC_NET_DEV_INTERVAL_SEC = 1
DEFAULT_PROC_NET_DEV_FULL_METRICS_FACTOR = 15

ZeroDelta = List[bool]
ZeroDeltaMap = Dict[int, ZeroDelta]

# Metrics definitions, must match lsvmi/proc_net_dev_metrics.go:
PROC_NET_DEV_RX_RATE_METRIC = "proc_net_dev_rx_kbps"
PROC_NET_DEV_RX_PACKETS_DELTA_METRIC = "proc_net_dev_rx_pkts_delta"
PROC_NET_DEV_RX_ERRS_DELTA_METRIC = "proc_net_dev_rx_errs_delta"
PROC_NET_DEV_RX_DROP_DELTA_METRIC = "proc_net_dev_rx_drop_delta"
PROC_NET_DEV_RX_FIFO_DELTA_METRIC = "proc_net_dev_rx_fifo_delta"
PROC_NET_DEV_RX_FRAME_DELTA_METRIC = "proc_net_dev_rx_frame_delta"
PROC_NET_DEV_RX_COMPRESSED_DELTA_METRIC = "proc_net_dev_rx_compressed_delta"
PROC_NET_DEV_RX_MULTICAST_DELTA_METRIC = "proc_net_dev_rx_mcast_delta"
PROC_NET_DEV_TX_RATE_METRIC = "proc_net_dev_tx_kbps"
PROC_NET_DEV_TX_PACKETS_DELTA_METRIC = "proc_net_dev_tx_pkts_delta"
PROC_NET_DEV_TX_ERRS_DELTA_METRIC = "proc_net_dev_tx_errs_delta"
PROC_NET_DEV_TX_DROP_DELTA_METRIC = "proc_net_dev_tx_drop_delta"
PROC_NET_DEV_TX_FIFO_DELTA_METRIC = "proc_net_dev_tx_fifo_delta"
PROC_NET_DEV_TX_COLLS_DELTA_METRIC = "proc_net_dev_tx_colls_delta"
PROC_NET_DEV_TX_CARRIER_DELTA_METRIC = "proc_net_dev_tx_carrier_delta"
PROC_NET_DEV_TX_COMPRESSED_DELTA_METRIC = "proc_net_dev_tx_compressed_delta"

PROC_NET_DEV_PRESENCE_METRIC = "proc_net_dev_present"

PROC_NET_DEV_LABEL_NAME = "dev"

PROC_NET_DEV_INTERVAL_METRIC_NAME = "proc_net_dev_metrics_delta_sec"

test_cases_file = "proc_net_dev.json"

# Map stats index (see procfs/net_dev_parser.go) into metrics names:
proc_net_dev_index_delta_metric_name_map = {
    procfs.NET_DEV_RX_BYTES: PROC_NET_DEV_RX_RATE_METRIC,
    procfs.NET_DEV_RX_PACKETS: PROC_NET_DEV_RX_PACKETS_DELTA_METRIC,
    procfs.NET_DEV_RX_ERRS: PROC_NET_DEV_RX_ERRS_DELTA_METRIC,
    procfs.NET_DEV_RX_DROP: PROC_NET_DEV_RX_DROP_DELTA_METRIC,
    procfs.NET_DEV_RX_FIFO: PROC_NET_DEV_RX_FIFO_DELTA_METRIC,
    procfs.NET_DEV_RX_FRAME: PROC_NET_DEV_RX_FRAME_DELTA_METRIC,
    procfs.NET_DEV_RX_COMPRESSED: PROC_NET_DEV_RX_COMPRESSED_DELTA_METRIC,
    procfs.NET_DEV_RX_MULTICAST: PROC_NET_DEV_RX_MULTICAST_DELTA_METRIC,
    procfs.NET_DEV_TX_BYTES: PROC_NET_DEV_TX_RATE_METRIC,
    procfs.NET_DEV_TX_PACKETS: PROC_NET_DEV_TX_PACKETS_DELTA_METRIC,
    procfs.NET_DEV_TX_ERRS: PROC_NET_DEV_TX_ERRS_DELTA_METRIC,
    procfs.NET_DEV_TX_DROP: PROC_NET_DEV_TX_DROP_DELTA_METRIC,
    procfs.NET_DEV_TX_FIFO: PROC_NET_DEV_TX_FIFO_DELTA_METRIC,
    procfs.NET_DEV_TX_COLLS: PROC_NET_DEV_TX_COLLS_DELTA_METRIC,
    procfs.NET_DEV_TX_CARRIER: PROC_NET_DEV_TX_CARRIER_DELTA_METRIC,
    procfs.NET_DEV_TX_COMPRESSED: PROC_NET_DEV_TX_COMPRESSED_DELTA_METRIC,
}

# Certain values are used to generate rates:
proc_net_dev_index_rate = {
    procfs.NET_DEV_RX_BYTES: (8.0 / 1000.0, 1),
    procfs.NET_DEV_TX_BYTES: (8.0 / 1000.0, 1),
}
PROC_NET_DEV_RATE_FACTOR = 0
PROC_NET_DEV_RATE_PREC = 1


@dataclass
class ProcNetDevInfoTestData:
    CycleNum: int = 0
    ZeroDelta: List[bool] = field(
        default_factory=lambda: [False] * procfs.NET_DEV_NUM_STATS
    )


@dataclass
class ProcNetDevMetricsTestCase:
    Name: Optional[str] = None
    Description: Optional[str] = None
    Instance: str = DEFAULT_TEST_INSTANCE
    Hostname: str = DEFAULT_TEST_HOSTNAME
    CurrProcNetDev: Optional[procfs.NetDev] = None
    PrevProcNetDev: Optional[procfs.NetDev] = None
    CurrPromTs: int = 0
    PrevPromTs: int = 0
    FullMetricsFactor: int = DEFAULT_PROC_NET_DEV_FULL_METRICS_FACTOR
    DevInfoMap: Optional[Dict[str, ProcNetDevInfoTestData]] = None
    WantMetricsCount: int = 0
    WantMetrics: Optional[List[str]] = None
    ReportExtra: bool = True
    WantZeroDeltaMap: Optional[Dict[str, List[bool]]] = None


def generate_proc_net_dev_metrics(
    curr_proc_net_dev: procfs.NetDev,
    prev_proc_net_dev: procfs.NetDev,
    curr_prom_ts: int,
    interval: Optional[float] = DEFAULT_PROC_NET_DEV_INTERVAL_SEC,
    dev_info_map: Optional[Dict[str, ProcNetDevInfoTestData]] = None,
    full_metrics: bool = False,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
) -> Tuple[List[str], Optional[ZeroDeltaMap]]:
    metrics = []
    new_zero_delta_map = {}

    for dev, curr_net_dev_stats in curr_proc_net_dev.DevStats.items():
        prev_net_dev_stats = prev_proc_net_dev.DevStats.get(dev)
        if prev_net_dev_stats is None:
            continue
        dev_info = dev_info_map.get(dev) if dev_info_map is not None else None
        full_metrics = dev_info is None or dev_info.CycleNum == 0
        new_zero_delta = [False] * procfs.NET_DEV_NUM_STATS
        for index, curr_val in enumerate(curr_net_dev_stats):
            delta = uint64_delta(curr_val, prev_net_dev_stats[index])
            if delta != 0 or full_metrics or not dev_info.ZeroDelta[index]:
                rate = proc_net_dev_index_rate.get(index)
                if rate is not None:
                    delta = delta / interval * rate[PROC_NET_DEV_RATE_FACTOR]
                    val = f"{delta:.{rate[PROC_NET_DEV_RATE_PREC]}f}"
                else:
                    val = delta
                metrics.append(
                    f"{proc_net_dev_index_delta_metric_name_map[index]}{{"
                    + ",".join(
                        [
                            f'{INSTANCE_LABEL_NAME}="{instance}"',
                            f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                            f'{PROC_NET_DEV_LABEL_NAME}="{dev}"',
                        ]
                    )
                    + f"}} {val} {curr_prom_ts}"
                )
            new_zero_delta[index] = delta == 0
        new_zero_delta_map[dev] = new_zero_delta
        if full_metrics:
            metrics.append(
                f"{PROC_NET_DEV_PRESENCE_METRIC}{{"
                + ",".join(
                    [
                        f'{INSTANCE_LABEL_NAME}="{instance}"',
                        f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                        f'{PROC_NET_DEV_LABEL_NAME}="{dev}"',
                    ]
                )
                + f"}} 1 {curr_prom_ts}"
            )

    if dev_info_map is not None:
        for dev in dev_info_map:
            if dev not in curr_proc_net_dev.DevStats:
                metrics.append(
                    f"{PROC_NET_DEV_PRESENCE_METRIC}{{"
                    + ",".join(
                        [
                            f'{INSTANCE_LABEL_NAME}="{instance}"',
                            f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                            f'{PROC_NET_DEV_LABEL_NAME}="{dev}"',
                        ]
                    )
                    + f"}} 0 {curr_prom_ts}"
                )

    metrics.append(
        f"{PROC_NET_DEV_INTERVAL_METRIC_NAME}{{"
        + ",".join(
            [
                f'{INSTANCE_LABEL_NAME}="{instance}"',
                f'{HOSTNAME_LABEL_NAME}="{hostname}"',
            ]
        )
        + f"}} {interval:.06f} {curr_prom_ts}"
    )

    return metrics, new_zero_delta_map


def generate_proc_net_dev_metrics_test_case(
    name: str,
    curr_proc_net_dev: procfs.NetDev,
    prev_proc_net_dev: procfs.NetDev,
    ts: Optional[float] = None,
    full_metrics_factor: int = DEFAULT_PROC_NET_DEV_FULL_METRICS_FACTOR,
    interval: Optional[float] = DEFAULT_PROC_NET_DEV_INTERVAL_SEC,
    dev_info_map: Optional[Dict[str, ProcNetDevInfoTestData]] = None,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    description: Optional[str] = None,
) -> Dict:
    if ts is None:
        ts = time.time()
    curr_prom_ts = int(ts * 1000)
    prev_prom_ts = curr_prom_ts - int(interval * 1000)
    metrics, want_zero_delta_map = generate_proc_net_dev_metrics(
        curr_proc_net_dev,
        prev_proc_net_dev,
        curr_prom_ts,
        interval=interval,
        dev_info_map=dev_info_map,
        hostname=hostname,
        instance=instance,
    )
    return ProcNetDevMetricsTestCase(
        Name=name,
        Description=description,
        Instance=instance,
        Hostname=hostname,
        CurrProcNetDev=curr_proc_net_dev,
        PrevProcNetDev=prev_proc_net_dev,
        CurrPromTs=curr_prom_ts,
        PrevPromTs=prev_prom_ts,
        FullMetricsFactor=full_metrics_factor,
        DevInfoMap=dev_info_map,
        WantMetricsCount=len(metrics),
        WantMetrics=metrics,
        ReportExtra=True,
        WantZeroDeltaMap=want_zero_delta_map,
    )


def make_ref_proc_net_dev(num_dev: int = 2) -> procfs.NetDev:
    dev_stats = {}
    for i in range(num_dev):
        base = (i + 1) * 10 * procfs.NET_DEV_NUM_STATS
        dev_stats[f"dev{i}"] = [base + j for j in range(procfs.NET_DEV_NUM_STATS)]
    return procfs.NetDev(DevStats=dev_stats)


def generate_proc_net_dev_metrics_test_cases(
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    test_cases_root_dir: Optional[str] = lsvmi_test_cases_root_dir,
):
    num_dev = 2
    proc_net_dev_ref = make_ref_proc_net_dev(num_dev=num_dev)
    max_val = max(map(max, proc_net_dev_ref.DevStats.values()))

    ts = time.time()

    test_cases = []
    tc_num = 0

    name = "all_new"
    curr_proc_net_dev = proc_net_dev_ref
    prev_proc_net_dev = deepcopy(proc_net_dev_ref)
    for i, dev_stats in enumerate(prev_proc_net_dev.DevStats.values()):
        for j in range(len(dev_stats)):
            dev_stats[j] = uint64_delta(
                dev_stats[j],
                (i + 1) * max_val + j,
            )
    test_cases.append(
        generate_proc_net_dev_metrics_test_case(
            f"{name}/{tc_num:04d}",
            curr_proc_net_dev,
            prev_proc_net_dev,
            ts=ts,
            instance=instance,
            hostname=hostname,
        )
    )
    tc_num += 1

    name = "all_change"
    curr_proc_net_dev = deepcopy(proc_net_dev_ref)
    prev_proc_net_dev = deepcopy(proc_net_dev_ref)
    for zero_delta in [True, False]:
        for cycle_num in [0, 1]:
            test_cases.append(
                generate_proc_net_dev_metrics_test_case(
                    f"{name}/{tc_num:04d}",
                    curr_proc_net_dev,
                    prev_proc_net_dev,
                    ts=ts,
                    dev_info_map={
                        dev: ProcNetDevInfoTestData(
                            CycleNum=cycle_num,
                            ZeroDelta=[zero_delta] * procfs.NET_DEV_NUM_STATS,
                        )
                        for dev in curr_proc_net_dev.DevStats
                    },
                    instance=instance,
                    hostname=hostname,
                    description=f"zero_delta={zero_delta},cycle_num={cycle_num}",
                )
            )
            tc_num += 1

    name = "no_change"
    curr_proc_net_dev = deepcopy(proc_net_dev_ref)
    for zero_delta in [True, False]:
        for cycle_num in [0, 1]:
            test_cases.append(
                generate_proc_net_dev_metrics_test_case(
                    f"{name}/{tc_num:04d}",
                    curr_proc_net_dev,
                    curr_proc_net_dev,
                    ts=ts,
                    dev_info_map={
                        dev: ProcNetDevInfoTestData(
                            CycleNum=cycle_num,
                            ZeroDelta=[zero_delta] * procfs.NET_DEV_NUM_STATS,
                        )
                        for dev in curr_proc_net_dev.DevStats
                    },
                    instance=instance,
                    hostname=hostname,
                    description=f"zero_delta={zero_delta},cycle_num={cycle_num}",
                )
            )
            tc_num += 1

    name = "one_change"
    curr_proc_net_dev = deepcopy(proc_net_dev_ref)
    for zero_delta in [True, False]:
        for cycle_num in [0, 1]:
            for n, dev in enumerate(curr_proc_net_dev.DevStats):
                for i in range(procfs.NET_DEV_NUM_STATS):
                    prev_proc_net_dev = deepcopy(proc_net_dev_ref)
                    prev_proc_net_dev.DevStats[dev][j] = uint64_delta(
                        prev_proc_net_dev.DevStats[dev][i],
                        (n + 1) * max_val + i,
                    )
                    test_cases.append(
                        generate_proc_net_dev_metrics_test_case(
                            f"{name}/{tc_num:04d}",
                            curr_proc_net_dev,
                            prev_proc_net_dev,
                            ts=ts,
                            dev_info_map={
                                dev: ProcNetDevInfoTestData(
                                    CycleNum=cycle_num,
                                    ZeroDelta=[zero_delta] * procfs.NET_DEV_NUM_STATS,
                                )
                                for dev in curr_proc_net_dev.DevStats
                            },
                            instance=instance,
                            hostname=hostname,
                            description=f"zero_delta={zero_delta},cycle_num={cycle_num},dev={dev},i={i}",
                        )
                    )
                    tc_num += 1

    name = "new_dev"
    curr_proc_net_dev = deepcopy(proc_net_dev_ref)
    for new_dev in curr_proc_net_dev.DevStats:
        for prev_present in [False, True]:
            prev_proc_net_dev = deepcopy(curr_proc_net_dev)
            if not prev_present:
                del prev_proc_net_dev.DevStats[new_dev]
            for zero_delta in [True, False]:
                for cycle_num in [0, 1]:
                    test_cases.append(
                        generate_proc_net_dev_metrics_test_case(
                            f"{name}/{tc_num:04d}",
                            curr_proc_net_dev,
                            prev_proc_net_dev,
                            ts=ts,
                            dev_info_map={
                                dev: ProcNetDevInfoTestData(
                                    CycleNum=cycle_num,
                                    ZeroDelta=[zero_delta] * procfs.NET_DEV_NUM_STATS,
                                )
                                for dev in curr_proc_net_dev.DevStats
                                if dev != new_dev
                            },
                            instance=instance,
                            hostname=hostname,
                            description=f"zero_delta={zero_delta},cycle_num={cycle_num},new_dev={new_dev},prev_present={prev_present}",
                        )
                    )
                    tc_num += 1

    name = "remove_dev"
    prev_proc_net_dev = deepcopy(proc_net_dev_ref)
    for rm_dev in prev_proc_net_dev.DevStats:
        curr_proc_net_dev = deepcopy(prev_proc_net_dev)
        del curr_proc_net_dev.DevStats[rm_dev]
        for prev_present in [False, True]:
            for zero_delta in [True, False]:
                for cycle_num in [0, 1]:
                    test_cases.append(
                        generate_proc_net_dev_metrics_test_case(
                            f"{name}/{tc_num:04d}",
                            curr_proc_net_dev,
                            prev_proc_net_dev,
                            ts=ts,
                            dev_info_map={
                                dev: ProcNetDevInfoTestData(
                                    CycleNum=cycle_num,
                                    ZeroDelta=[zero_delta] * procfs.NET_DEV_NUM_STATS,
                                )
                                for dev in prev_proc_net_dev.DevStats
                                if dev != rm_dev or prev_present
                            },
                            instance=instance,
                            hostname=hostname,
                            description=f"zero_delta={zero_delta},cycle_num={cycle_num},rm_dev={rm_dev},prev_present={prev_present}",
                        )
                    )
                    tc_num += 1

    # zero_delta_map_false = {}
    # zero_delta_map_true = {}
    # for dev in proc_net_dev_ref.DevStats:
    #     zero_delta_map_false[dev] = [False] * procfs.NET_DEV_NUM_STATS
    #     zero_delta_map_true[dev] = [True] * procfs.NET_DEV_NUM_STATS

    # # No change:
    # for cycle_num in [0, 1]:
    #     for zero_delta_map in [zero_delta_map_false, zero_delta_map_true]:
    #         test_cases.append(
    #             generate_proc_net_dev_metrics_test_case(
    #                 f"{tc_num:04d}",
    #                 proc_net_dev_ref,
    #                 proc_net_dev_ref,
    #                 ts=ts,
    #                 cycle_num=cycle_num,
    #                 zero_delta_map=zero_delta_map,
    #                 instance=instance,
    #                 hostname=hostname,
    #             )
    #         )
    #         tc_num += 1

    # # One change at a time:
    # for cycle_num in [0, 1]:
    #     for zero_delta_map in [zero_delta_map_false, zero_delta_map_true]:
    #         for dev in proc_net_dev_ref.DevStats:
    #             for index in range(procfs.NET_DEV_NUM_STATS):
    #                 curr_proc_net_dev = deepcopy(proc_net_dev_ref)
    #                 curr_proc_net_dev.DevStats[dev][index] += 10000 * (index + 1)
    #                 test_cases.append(
    #                     generate_proc_net_dev_metrics_test_case(
    #                         f"{tc_num:04d}",
    #                         curr_proc_net_dev,
    #                         proc_net_dev_ref,
    #                         ts=ts,
    #                         cycle_num=cycle_num,
    #                         zero_delta_map=zero_delta_map,
    #                         instance=instance,
    #                         hostname=hostname,
    #                     )
    #                 )
    #                 tc_num += 1

    # # New dev:
    # for cycle_num in [0, 1]:
    #     for zero_delta_map in [zero_delta_map_false, zero_delta_map_true]:
    #         for dev in proc_net_dev_ref.DevStats:
    #             prev_proc_net_dev = deepcopy(proc_net_dev_ref)
    #             del prev_proc_net_dev.DevStats[dev]
    #             zero_delta_map = deepcopy(zero_delta_map)
    #             del zero_delta_map[dev]
    #             test_cases.append(
    #                 generate_proc_net_dev_metrics_test_case(
    #                     f"{tc_num:04d}",
    #                     proc_net_dev_ref,
    #                     prev_proc_net_dev,
    #                     ts=ts,
    #                     cycle_num=cycle_num,
    #                     zero_delta_map=zero_delta_map,
    #                     instance=instance,
    #                     hostname=hostname,
    #                 )
    #             )
    #             tc_num += 1

    # # Removed dev:
    # for cycle_num in [0, 1]:
    #     for zero_delta_map in [zero_delta_map_false, zero_delta_map_true]:
    #         for dev in proc_net_dev_ref.DevStats:
    #             curr_proc_net_dev = deepcopy(proc_net_dev_ref)
    #             del curr_proc_net_dev.DevStats[dev]
    #             test_cases.append(
    #                 generate_proc_net_dev_metrics_test_case(
    #                     f"{tc_num:04d}",
    #                     curr_proc_net_dev,
    #                     proc_net_dev_ref,
    #                     ts=ts,
    #                     cycle_num=cycle_num,
    #                     zero_delta_map=zero_delta_map,
    #                     instance=instance,
    #                     hostname=hostname,
    #                 )
    #             )
    #             tc_num += 1

    save_test_cases(
        test_cases, test_cases_file, test_cases_root_dir=test_cases_root_dir
    )
