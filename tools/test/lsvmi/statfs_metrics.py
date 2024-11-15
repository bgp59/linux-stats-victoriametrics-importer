#! /usr/bin/env python3

# Generate test cases for lsvmi/statfs_metrics_test.go

import time
from copy import deepcopy
from dataclasses import dataclass
from typing import List, Optional

from statfs import statfs

from . import (
    DEFAULT_TEST_HOSTNAME,
    DEFAULT_TEST_INSTANCE,
    HOSTNAME_LABEL_NAME,
    INSTANCE_LABEL_NAME,
    lsvmi_test_cases_root_dir,
    save_test_cases,
)

# The following should match lsvmi/statfs_metrics.go:

DEFAULT_STATFS_INTERVAL_SEC = 5
DEFAULT_STATFS_FULL_METRICS_FACTOR = 12


STATFS_BSIZE_METRIC = "statfs_bsize"
STATFS_BLOCKS_METRIC = "statfs_blocks"
STATFS_BFREE_METRIC = "statfs_bfree"
STATFS_BAVAIL_METRIC = "statfs_bavail"
STATFS_FILES_METRIC = "statfs_files"
STATFS_FFREE_METRIC = "statfs_ffree"
STATFS_TOTAL_SIZE_METRIC = "statfs_total_size_kb"
STATFS_FREE_SIZE_METRIC = "statfs_free_size_kb"
STATFS_AVAIL_SIZE_METRIC = "statfs_avail_size_kb"
STATFS_FREE_PCT_METRIC = "statfs_free_pct"
STATFS_AVAIL_PCT_METRIC = "statfs_avail_pct"
STATFS_PRESENCE_METRIC = "statfs_present"

STATFS_MOUNTINFO_FS_LABEL_NAME = "fs"
STATFS_MOUNTINFO_FS_TYPE_LABEL_NAME = "fs_type"
STATFS_MOUNTINFO_MOUNT_POINT_LABEL_NAME = "mount_point"

STATFS_INTERVAL_METRIC = "statfs_metrics_delta_sec"

STATFS_FREE_PCT_METRIC_FMT = ".1f"
STATFS_AVAIL_PCT_METRIC_FMT = ".1f"

KBYTE = 1000  # not KiB, this is disk storage folks!

test_cases_file = "statfs.json"


@dataclass
class StatfsInfoTestData:
    Fs: Optional[str] = None
    FsType: Optional[str] = None
    MountPoint: Optional[str] = None
    Statfs: Optional[statfs.Statfs] = None
    CycleNum: int = 0


@dataclass
class StatfsMetricsTestCase:
    Name: Optional[str] = None
    Description: Optional[str] = None
    Instance: Optional[str] = None
    Hostname: Optional[str] = None
    CurrStatfsInfoList: Optional[List[StatfsInfoTestData]] = None
    PrevStatfsInfoList: Optional[List[StatfsInfoTestData]] = None
    CurrPromTs: int = 0
    PrevPromTs: int = 0
    WantMetricsCount: int = 0
    WantMetrics: Optional[List[str]] = None
    ReportExtra: bool = True


def generate_one_fstat_metrics_set(
    curr_statfs_info: StatfsInfoTestData,
    curr_prom_ts: int,
    prev_statfs_info: Optional[StatfsInfoTestData] = None,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
) -> List[str]:
    metrics = []

    curr_statfs = curr_statfs_info.Statfs
    prev_statfs = prev_statfs_info.Statfs if prev_statfs_info is not None else None
    all_metrics = (
        curr_statfs_info.CycleNum == 0
        or prev_statfs is None
        or curr_statfs.Bsize != prev_statfs.Bsize
    )
    update_avail_pct = False
    update_free_pct = False

    statfs_labels = ",".join(
        [
            f'{INSTANCE_LABEL_NAME}="{instance}"',
            f'{HOSTNAME_LABEL_NAME}="{hostname}"',
            f'{STATFS_MOUNTINFO_FS_LABEL_NAME}="{curr_statfs_info.Fs}"',
            f'{STATFS_MOUNTINFO_FS_TYPE_LABEL_NAME}="{curr_statfs_info.FsType}"',
            f'{STATFS_MOUNTINFO_MOUNT_POINT_LABEL_NAME}="{curr_statfs_info.MountPoint}"',
        ]
    )
    if all_metrics:
        metrics.append(
            f"{STATFS_BSIZE_METRIC}{{{statfs_labels}}} {curr_statfs.Bsize} {curr_prom_ts}"
        )
    if all_metrics or curr_statfs.Blocks != prev_statfs.Blocks:
        metrics.append(
            f"{STATFS_BLOCKS_METRIC}{{{statfs_labels}}} {curr_statfs.Blocks} {curr_prom_ts}"
        )
        metrics.append(
            f"{STATFS_TOTAL_SIZE_METRIC}{{{statfs_labels}}} {curr_statfs.Blocks*curr_statfs.Bsize//KBYTE} {curr_prom_ts}"
        )
        update_avail_pct = True
        update_free_pct = True
    if all_metrics or curr_statfs.Bfree != prev_statfs.Bfree:
        metrics.append(
            f"{STATFS_BFREE_METRIC}{{{statfs_labels}}} {curr_statfs.Bfree} {curr_prom_ts}"
        )
        metrics.append(
            f"{STATFS_FREE_SIZE_METRIC}{{{statfs_labels}}} {curr_statfs.Bfree*curr_statfs.Bsize//KBYTE} {curr_prom_ts}"
        )
        update_free_pct = True
    if all_metrics or curr_statfs.Bavail != prev_statfs.Bavail:
        metrics.append(
            f"{STATFS_BAVAIL_METRIC}{{{statfs_labels}}} {curr_statfs.Bavail} {curr_prom_ts}"
        )
        metrics.append(
            f"{STATFS_AVAIL_SIZE_METRIC}{{{statfs_labels}}} {curr_statfs.Bavail*curr_statfs.Bsize//KBYTE} {curr_prom_ts}"
        )
        update_avail_pct = True
    if all_metrics or update_free_pct:
        metrics.append(
            f"{STATFS_FREE_PCT_METRIC}{{{statfs_labels}}} {curr_statfs.Bfree/curr_statfs.Blocks*100:{STATFS_FREE_PCT_METRIC_FMT}} {curr_prom_ts}"
        )
    if all_metrics or update_avail_pct:
        metrics.append(
            f"{STATFS_AVAIL_PCT_METRIC}{{{statfs_labels}}} {curr_statfs.Bavail/curr_statfs.Blocks*100:{STATFS_AVAIL_PCT_METRIC_FMT}} {curr_prom_ts}"
        )
    if all_metrics or curr_statfs.Files != prev_statfs.Files:
        metrics.append(
            f"{STATFS_FILES_METRIC}{{{statfs_labels}}} {curr_statfs.Files} {curr_prom_ts}"
        )
    if all_metrics or curr_statfs.Ffree != prev_statfs.Ffree:
        metrics.append(
            f"{STATFS_FFREE_METRIC}{{{statfs_labels}}} {curr_statfs.Ffree} {curr_prom_ts}"
        )
    if all_metrics:
        metrics.append(f"{STATFS_PRESENCE_METRIC}{{{statfs_labels}}} 1 {curr_prom_ts}")
    return metrics


def generate_fstat_metrics(
    curr_statfs_info_list: List[StatfsInfoTestData],
    curr_prom_ts: int,
    prev_statfs_info_list: Optional[List[StatfsInfoTestData]] = None,
    prev_prom_ts: int = 0,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
) -> List[str]:
    metrics = []

    prev_statfs_info_by_mountinfo = {
        (sfsi.Fs, sfsi.FsType, sfsi.MountPoint): sfsi
        for sfsi in prev_statfs_info_list or []
    }

    curr_mountinfo = set()
    for sfsi in curr_statfs_info_list:
        mountinfo_key = (sfsi.Fs, sfsi.FsType, sfsi.MountPoint)
        metrics.extend(
            generate_one_fstat_metrics_set(
                sfsi,
                curr_prom_ts,
                prev_statfs_info=prev_statfs_info_by_mountinfo.get(mountinfo_key),
                instance=instance,
                hostname=hostname,
            )
        )
        curr_mountinfo.add(mountinfo_key)

    for mountinfo_key in prev_statfs_info_by_mountinfo:
        if mountinfo_key not in curr_mountinfo:
            fs, fs_type, mount_point = mountinfo_key
            statfs_labels = ",".join(
                [
                    f'{INSTANCE_LABEL_NAME}="{instance}"',
                    f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                    f'{STATFS_MOUNTINFO_FS_LABEL_NAME}="{fs}"',
                    f'{STATFS_MOUNTINFO_FS_TYPE_LABEL_NAME}="{fs_type}"',
                    f'{STATFS_MOUNTINFO_MOUNT_POINT_LABEL_NAME}="{mount_point}"',
                ]
            )
            metrics.append(
                f"{STATFS_PRESENCE_METRIC}{{{statfs_labels}}} 0 {curr_prom_ts}"
            )

    if prev_statfs_info_list:
        metrics.append(
            f'{STATFS_INTERVAL_METRIC}{{{INSTANCE_LABEL_NAME}="{instance}",{HOSTNAME_LABEL_NAME}="{hostname}"}}'
            + f" {(curr_prom_ts - prev_prom_ts) / 1000:.6f} {curr_prom_ts}"
        )

    return metrics


def generate_fstat_test_case(
    name: str,
    curr_statfs_info_list: List[StatfsInfoTestData],
    prev_statfs_info_list: Optional[List[StatfsInfoTestData]] = None,
    ts: Optional[float] = None,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    description: Optional[str] = None,
) -> StatfsMetricsTestCase:
    if ts is None:
        ts = time.time()
    curr_prom_ts = int(ts * 1000)
    prev_prom_ts = curr_prom_ts - int(DEFAULT_STATFS_INTERVAL_SEC * 1000)
    metrics = generate_fstat_metrics(
        curr_statfs_info_list,
        curr_prom_ts,
        prev_statfs_info_list=prev_statfs_info_list,
        prev_prom_ts=prev_prom_ts,
        instance=instance,
        hostname=hostname,
    )

    return StatfsMetricsTestCase(
        Name=name,
        Description=description,
        Instance=instance,
        Hostname=hostname,
        CurrStatfsInfoList=curr_statfs_info_list,
        PrevStatfsInfoList=prev_statfs_info_list,
        CurrPromTs=curr_prom_ts,
        PrevPromTs=prev_prom_ts,
        WantMetricsCount=len(metrics),
        WantMetrics=metrics,
        ReportExtra=True,
    )


def make_ref_statfs_info(num: int = 0, cycle_num: int = 0) -> StatfsInfoTestData:
    bsize = (num + 1) * 512
    blocks = (num + 1) * 10000
    bfree = int(blocks * ((num % 10) + 1) / 10)
    bavail = int(bfree * 0.9)
    files = (num + 1) * 1000
    ffree = int(files * ((num % 10) + 1) / 10)
    return StatfsInfoTestData(
        Fs=f"fs{num}",
        FsType=f"fs_type{num}",
        MountPoint=f"/mount{num}",
        Statfs=statfs.Statfs(
            Bsize=bsize,
            Blocks=blocks,
            Bfree=bfree,
            Bavail=bavail,
            Files=files,
            Ffree=ffree,
        ),
        CycleNum=cycle_num % DEFAULT_STATFS_FULL_METRICS_FACTOR,
    )


def generate_statfs_metrics_test_cases(
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    test_cases_root_dir: Optional[str] = lsvmi_test_cases_root_dir,
):
    ts = time.time()

    test_cases = []
    tc_num = 0

    ref_statfs_info_list = [
        make_ref_statfs_info(num=num, cycle_num=num) for num in range(2)
    ]

    name = "all_new"
    test_cases.append(
        generate_fstat_test_case(
            f"{name}/{tc_num:04d}",
            curr_statfs_info_list=ref_statfs_info_list,
            ts=ts,
        )
    )
    tc_num += 1

    name = "no_change"
    for cycle_num in [0, 1]:
        curr_statfs_info_list = deepcopy(ref_statfs_info_list)
        for statfs_info in curr_statfs_info_list:
            statfs_info.CycleNum = cycle_num
        test_cases.append(
            generate_fstat_test_case(
                f"{name}/{tc_num:04d}",
                curr_statfs_info_list=curr_statfs_info_list,
                prev_statfs_info_list=curr_statfs_info_list,
                ts=ts,
                description=f"cycle_num={cycle_num}",
            )
        )
        tc_num += 1

    name = "one_change"
    prev_statfs_info_list = deepcopy(ref_statfs_info_list)
    for statfs_info in prev_statfs_info_list:
        statfs_info.CycleNum = 1
    for k in range(len(prev_statfs_info_list)):
        for statfs_attr in ["Bsize", "Blocks", "Bfree", "Bavail", "Files", "Ffree"]:
            curr_statfs_info_list = deepcopy(prev_statfs_info_list)
            statfs_info = curr_statfs_info_list[k]
            statfs = statfs_info.Statfs
            setattr(statfs, statfs_attr, int(getattr(statfs, statfs_attr) * 1.1))
            test_cases.append(
                generate_fstat_test_case(
                    f"{name}/{tc_num:04d}",
                    curr_statfs_info_list=curr_statfs_info_list,
                    prev_statfs_info_list=prev_statfs_info_list,
                    ts=ts,
                    description=f"fs={statfs_info.Fs},attr={statfs_attr}",
                )
            )
            tc_num += 1

    name = "new_fs"
    curr_statfs_info_list = deepcopy(ref_statfs_info_list)
    for statfs_info in curr_statfs_info_list:
        statfs_info.CycleNum = 1
    for k in range(len(curr_statfs_info_list)):
        prev_statfs_info_list = [
            statfs_info
            for (i, statfs_info) in enumerate(ref_statfs_info_list)
            if i != k
        ]
        test_cases.append(
            generate_fstat_test_case(
                f"{name}/{tc_num:04d}",
                curr_statfs_info_list=curr_statfs_info_list,
                prev_statfs_info_list=prev_statfs_info_list,
                ts=ts,
                description=f"fs={curr_statfs_info_list[k].Fs}",
            )
        )
        tc_num += 1

    name = "out_of_scope_mountinfo"
    for k in range(len(ref_statfs_info_list)):
        for mount_attr in ["Fs", "FsType", "MountPoint"]:
            prev_statfs_info_list = deepcopy(ref_statfs_info_list)
            fs_info = prev_statfs_info_list[k]
            setattr(fs_info, mount_attr, getattr(fs_info, mount_attr) + "-out")
            test_cases.append(
                generate_fstat_test_case(
                    f"{name}/{tc_num:04d}",
                    curr_statfs_info_list=ref_statfs_info_list,
                    prev_statfs_info_list=prev_statfs_info_list,
                    ts=ts,
                    description=f"fs={statfs_info.Fs},attr={mount_attr}",
                )
            )
            tc_num += 1

    save_test_cases(
        test_cases, test_cases_file, test_cases_root_dir=test_cases_root_dir
    )
