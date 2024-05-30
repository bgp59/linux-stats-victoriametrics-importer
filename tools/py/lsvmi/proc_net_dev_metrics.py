#! /usr/bin/env python3

# Generate test cases for lsvmi/proc_net_dev_metrics_test.go

import json
import os
import sys
import time
from copy import deepcopy
from typing import Dict, List, Optional, Tuple

import procfs

from . import (
    DEFAULT_TEST_HOSTNAME,
    DEFAULT_TEST_INSTANCE,
    HOSTNAME_LABEL_NAME,
    INSTANCE_LABEL_NAME,
    lsvmi_test_cases_root_dir,
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


def generate_proc_net_dev_metrics(
    curr_proc_net_dev: procfs.NetDev,
    prev_proc_net_dev: procfs.NetDev,
    curr_prom_ts: int,
    interval: Optional[float] = DEFAULT_PROC_NET_DEV_INTERVAL_SEC,
    zero_delta_map: Optional[ZeroDeltaMap] = None,
    full_metrics: bool = False,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
) -> Tuple[List[str], Optional[ZeroDeltaMap]]:
    metrics = []
    new_zero_delta_map = {}

    for dev, curr_net_dev_stats in curr_proc_net_dev["DevStats"].items():
        prev_net_dev_stats = prev_proc_net_dev["DevStats"].get(dev)
        if prev_net_dev_stats is None:
            continue
        new_zero_delta_map[dev] = [False] * procfs.NET_DEV_NUM_STATS
        zero_delta = zero_delta_map.get(dev)
        for index, curr_val in enumerate(curr_net_dev_stats):
            val = curr_val - prev_net_dev_stats[index]
            new_zero_delta_map[dev][index] = val == 0
            if val != 0 or full_metrics or zero_delta is None or not zero_delta[index]:
                rate = proc_net_dev_index_rate.get(index)
                if rate is not None:
                    val = val / interval * rate[PROC_NET_DEV_RATE_FACTOR]
                    val = f"{val:.{rate[PROC_NET_DEV_RATE_PREC]}f}"
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
    cycle_num: int = 0,
    full_metrics_factor: int = DEFAULT_PROC_NET_DEV_FULL_METRICS_FACTOR,
    interval: Optional[float] = DEFAULT_PROC_NET_DEV_INTERVAL_SEC,
    zero_delta_map: Optional[ZeroDeltaMap] = None,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
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
        zero_delta_map=zero_delta_map,
        full_metrics=(cycle_num == 0),
        hostname=hostname,
        instance=instance,
    )
    return {
        "Name": name,
        "Instance": instance,
        "Hostname": hostname,
        "CurrProcNetDev": curr_proc_net_dev,
        "PrevProcNetDev": prev_proc_net_dev,
        "CurrPromTs": curr_prom_ts,
        "PrevPromTs": prev_prom_ts,
        "CycleNum": cycle_num,
        "FullMetricsFactor": full_metrics_factor,
        "ZeroDeltaMap": zero_delta_map,
        "WantMetricsCount": len(metrics),
        "WantMetrics": metrics,
        "ReportExtra": True,
        "WantZeroDeltaMap": want_zero_delta_map,
    }


def generate_proc_net_dev_metrics_test_cases(
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    test_cases_root_dir: Optional[str] = lsvmi_test_cases_root_dir,
):
    ts = time.time()

    if test_cases_root_dir not in {None, "", "-"}:
        out_file = os.path.join(test_cases_root_dir, test_cases_file)
        os.makedirs(os.path.dirname(out_file), exist_ok=True)
        fp = open(out_file, "wt")
    else:
        out_file = None
        fp = sys.stdout

    proc_net_dev_ref = {"DevStats": {}}
    num_dev = 2
    for n in range(num_dev):
        base = (n + 1) * 10000
        proc_net_dev_ref["DevStats"][f"dev{n}"] = [
            base + j for j in range(procfs.NET_DEV_NUM_STATS)
        ]

    test_cases = []
    tc_num = 0

    zero_delta_map_false = {}
    zero_delta_map_true = {}
    for dev in proc_net_dev_ref["DevStats"]:
        zero_delta_map_false[dev] = [False] * procfs.NET_DEV_NUM_STATS
        zero_delta_map_true[dev] = [True] * procfs.NET_DEV_NUM_STATS

    # No change:
    for cycle_num in [0, 1]:
        for zero_delta_map in [zero_delta_map_false, zero_delta_map_true]:
            test_cases.append(
                generate_proc_net_dev_metrics_test_case(
                    f"{tc_num:04d}",
                    proc_net_dev_ref,
                    proc_net_dev_ref,
                    ts=ts,
                    cycle_num=cycle_num,
                    zero_delta_map=zero_delta_map,
                    instance=instance,
                    hostname=hostname,
                )
            )
            tc_num += 1

    # One change at a time:
    for cycle_num in [0, 1]:
        for zero_delta_map in [zero_delta_map_false, zero_delta_map_true]:
            for dev in proc_net_dev_ref["DevStats"]:
                for index in range(procfs.NET_DEV_NUM_STATS):
                    curr_proc_net_dev = deepcopy(proc_net_dev_ref)
                    curr_proc_net_dev["DevStats"][dev][index] += 10000 * (index + 1)
                    test_cases.append(
                        generate_proc_net_dev_metrics_test_case(
                            f"{tc_num:04d}",
                            curr_proc_net_dev,
                            proc_net_dev_ref,
                            ts=ts,
                            cycle_num=cycle_num,
                            zero_delta_map=zero_delta_map,
                            instance=instance,
                            hostname=hostname,
                        )
                    )
                    tc_num += 1

    # New dev:
    for cycle_num in [0, 1]:
        for zero_delta_map in [zero_delta_map_false, zero_delta_map_true]:
            for dev in proc_net_dev_ref["DevStats"]:
                prev_proc_net_dev = deepcopy(proc_net_dev_ref)
                del prev_proc_net_dev["DevStats"][dev]
                zero_delta_map = deepcopy(zero_delta_map)
                del zero_delta_map[dev]
                test_cases.append(
                    generate_proc_net_dev_metrics_test_case(
                        f"{tc_num:04d}",
                        proc_net_dev_ref,
                        prev_proc_net_dev,
                        ts=ts,
                        cycle_num=cycle_num,
                        zero_delta_map=zero_delta_map,
                        instance=instance,
                        hostname=hostname,
                    )
                )
                tc_num += 1

    # Removed dev:
    for cycle_num in [0, 1]:
        for zero_delta_map in [zero_delta_map_false, zero_delta_map_true]:
            for dev in proc_net_dev_ref["DevStats"]:
                curr_proc_net_dev = deepcopy(proc_net_dev_ref)
                del curr_proc_net_dev["DevStats"][dev]
                test_cases.append(
                    generate_proc_net_dev_metrics_test_case(
                        f"{tc_num:04d}",
                        curr_proc_net_dev,
                        proc_net_dev_ref,
                        ts=ts,
                        cycle_num=cycle_num,
                        zero_delta_map=zero_delta_map,
                        instance=instance,
                        hostname=hostname,
                    )
                )
                tc_num += 1

    json.dump(test_cases, fp=fp, indent=2)
    fp.write("\n")
    if out_file is not None:
        fp.close()
        print(f"{out_file} generated", file=sys.stderr)
