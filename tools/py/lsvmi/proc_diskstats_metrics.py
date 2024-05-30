#! /usr/bin/env python3

# Generate test cases for lsvmi/proc_interrupts_metrics_test.go

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
    uint32_delta,
)

PROC_DISKSTATS_METRICS_CONFIG_INTERVAL_DEFAULT = 5
PROC_DISKSTATS_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT = 12
PROC_DISKSTATS_METRICS_CONFIG_MOUNTINFO_PID_DEFAULT = 0

PROC_DISKSTATS_METRICS_ID = "proc_diskstats_metrics"

PROC_DISKSTATS_NUM_READS_COMPLETED_DELTA_METRIC = (
    "proc_diskstats_num_reads_completed_delta"
)
PROC_DISKSTATS_NUM_READS_MERGED_DELTA_METRIC = "proc_diskstats_num_reads_merged_delta"
PROC_DISKSTATS_NUM_READ_SECTORS_DELTA_METRIC = "proc_diskstats_num_read_sectors_delta"
PROC_DISKSTATS_READ_PCT_METRIC = "proc_diskstats_read_pct"
PROC_DISKSTATS_NUM_WRITES_COMPLETED_DELTA_METRIC = (
    "proc_diskstats_num_writes_completed_delta"
)
PROC_DISKSTATS_NUM_WRITES_MERGED_DELTA_METRIC = "proc_diskstats_num_writes_merged_delta"
PROC_DISKSTATS_NUM_WRITE_SECTORS_DELTA_METRIC = "proc_diskstats_num_write_sectors_delta"
PROC_DISKSTATS_WRITE_PCT_METRIC = "proc_diskstats_write_pct"
PROC_DISKSTATS_NUM_IO_IN_PROGRESS_DELTA_METRIC = (
    "proc_diskstats_num_io_in_progress_delta"
)
PROC_DISKSTATS_IO_PCT_METRIC = "proc_diskstats_io_pct"
PROC_DISKSTATS_IO_WEIGTHED_PCT_METRIC = "proc_diskstats_io_weigthed_pct"
PROC_DISKSTATS_NUM_DISCARDS_COMPLETED_DELTA_METRIC = (
    "proc_diskstats_num_discards_completed_delta"
)
PROC_DISKSTATS_NUM_DISCARDS_MERGED_DELTA_METRIC = (
    "proc_diskstats_num_discards_merged_delta"
)
PROC_DISKSTATS_NUM_DISCARD_SECTORS_DELTA_METRIC = (
    "proc_diskstats_num_discard_sectors_delta"
)
PROC_DISKSTATS_DISCARD_PCT_METRIC = "proc_diskstats_discard_pct"
PROC_DISKSTATS_NUM_FLUSH_REQUESTS_DELTA_METRIC = (
    "proc_diskstats_num_flush_requests_delta"
)
PROC_DISKSTATS_FLUSH_PCT_METRIC = "proc_diskstats_flush_pct"

PROC_DISKSTATS_INFO_METRIC = "proc_diskstats_info"

PROC_DISKSTATS_MAJ_MIN_LABEL_NAME = "maj_min"
PROC_DISKSTATS_NAME_LABEL_NAME = "name"

PROC_MOUNTINFO_METRIC = "proc_mountinfo"
PROC_MOUNTINFO_PID_LABEL_NAME = "pid"
PROC_MOUNTINFO_MAJ_MIN_LABEL_NAME = "maj_min"
PROC_MOUNTINFO_ROOT_LABEL_NAME = "root"
PROC_MOUNTINFO_MOUNT_POINT_LABEL_NAME = "mount_point"
PROC_MOUNTINFO_FS_TYPE_LABEL_NAME = "fs_type"
PROC_MOUNTINFO_MOUNT_SOURCE_LABEL_NAME = "source"

PROC_DISKSTATS_INTERVAL_METRIC_NAME = "proc_diskstats_metrics_delta_sec"

procDiskstatsIndexPctMetric = {
    procfs.DISKSTATS_READ_MILLISEC: (100.0 / 1000.0, 2),
    procfs.DISKSTATS_WRITE_MILLISEC: (100.0 / 1000.0, 2),
    procfs.DISKSTATS_IO_MILLISEC: (100.0 / 1000.0, 2),
    procfs.DISKSTATS_IO_WEIGTHED_MILLISEC: (100.0 / 1000.0, 2),
    procfs.DISKSTATS_DISCARD_MILLISEC: (100.0 / 1000.0, 2),
    procfs.DISKSTATS_FLUSH_MILLISEC: (100.0 / 1000.0, 2),
}

procDiskstatsIndexToMetricNameMap = {
    procfs.DISKSTATS_NUM_READS_COMPLETED: PROC_DISKSTATS_NUM_READS_COMPLETED_DELTA_METRIC,
    procfs.DISKSTATS_NUM_READS_MERGED: PROC_DISKSTATS_NUM_READS_MERGED_DELTA_METRIC,
    procfs.DISKSTATS_NUM_READ_SECTORS: PROC_DISKSTATS_NUM_READ_SECTORS_DELTA_METRIC,
    procfs.DISKSTATS_READ_MILLISEC: PROC_DISKSTATS_READ_PCT_METRIC,
    procfs.DISKSTATS_NUM_WRITES_COMPLETED: PROC_DISKSTATS_NUM_WRITES_COMPLETED_DELTA_METRIC,
    procfs.DISKSTATS_NUM_WRITES_MERGED: PROC_DISKSTATS_NUM_WRITES_MERGED_DELTA_METRIC,
    procfs.DISKSTATS_NUM_WRITE_SECTORS: PROC_DISKSTATS_NUM_WRITE_SECTORS_DELTA_METRIC,
    procfs.DISKSTATS_WRITE_MILLISEC: PROC_DISKSTATS_WRITE_PCT_METRIC,
    procfs.DISKSTATS_NUM_IO_IN_PROGRESS: PROC_DISKSTATS_NUM_IO_IN_PROGRESS_DELTA_METRIC,
    procfs.DISKSTATS_IO_MILLISEC: PROC_DISKSTATS_IO_PCT_METRIC,
    procfs.DISKSTATS_IO_WEIGTHED_MILLISEC: PROC_DISKSTATS_IO_WEIGTHED_PCT_METRIC,
    procfs.DISKSTATS_NUM_DISCARDS_COMPLETED: PROC_DISKSTATS_NUM_DISCARDS_COMPLETED_DELTA_METRIC,
    procfs.DISKSTATS_NUM_DISCARDS_MERGED: PROC_DISKSTATS_NUM_DISCARDS_MERGED_DELTA_METRIC,
    procfs.DISKSTATS_NUM_DISCARD_SECTORS: PROC_DISKSTATS_NUM_DISCARD_SECTORS_DELTA_METRIC,
    procfs.DISKSTATS_DISCARD_MILLISEC: PROC_DISKSTATS_DISCARD_PCT_METRIC,
    procfs.DISKSTATS_NUM_FLUSH_REQUESTS: PROC_DISKSTATS_NUM_FLUSH_REQUESTS_DELTA_METRIC,
    procfs.DISKSTATS_FLUSH_MILLISEC: PROC_DISKSTATS_FLUSH_PCT_METRIC,
}

procMountinfoIndexToMetricLabelList = [
    (procfs.MOUNTINFO_MAJOR_MINOR, PROC_MOUNTINFO_MAJ_MIN_LABEL_NAME),
    (procfs.MOUNTINFO_ROOT, PROC_MOUNTINFO_ROOT_LABEL_NAME),
    (procfs.MOUNTINFO_MOUNT_POINT, PROC_MOUNTINFO_MOUNT_POINT_LABEL_NAME),
    (procfs.MOUNTINFO_FS_TYPE, PROC_MOUNTINFO_FS_TYPE_LABEL_NAME),
    (procfs.MOUNTINFO_MOUNT_SOURCE, PROC_MOUNTINFO_MOUNT_SOURCE_LABEL_NAME),
]

ZeroDeltaType = List[bool]

test_cases_file = "proc_diskstats.json"


@dataclass
class ProcDiskstatsMetricsInfoTest:
    CycleNum: int = 0
    ZeroDelta: ZeroDeltaType = field(
        default_factory=lambda: [False] * procfs.DISKSTATS_VALUE_FIELDS_NUM
    )
    MetricsCache: Optional[List[str]] = None
    InfoMetric: Optional[str] = None


@dataclass
class ProcDiskstatsMetricsTestCase:
    Name: Optional[str] = None
    Description: Optional[str] = None
    Instance: str = DEFAULT_TEST_INSTANCE
    Hostname: str = DEFAULT_TEST_HOSTNAME
    MountinfoPid: int = PROC_DISKSTATS_METRICS_CONFIG_MOUNTINFO_PID_DEFAULT
    CurrProcDiskstats: Optional[procfs.Diskstats] = None
    PrevProcDiskstats: Optional[procfs.Diskstats] = None
    CurrPromTs: int = 0
    PrevPromTs: int = 0
    PrimeDiskstatsMetricsInfo: Optional[Dict[str, ProcDiskstatsMetricsInfoTest]] = None
    MountifoParsedLines: Optional[List[Dict[int, str]]] = None
    MountinfoChanged: bool = False
    PrimeMountinfoMetricsCache: Optional[List[str]] = None
    MountinfoCycleNum: int = 0
    FullMetricsFactor: int = PROC_DISKSTATS_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT
    WantMetricsCount: int = 0
    WantMetrics: Optional[List[str]] = None
    ReportExtra: bool = True
    WantZeroDeltaMap: Optional[Dict[str, ZeroDeltaType]] = None


def build_diskstats_metric(
    metric_name: str,
    maj_min: str,
    disk_name: str,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
) -> Optional[str]:
    return (
        metric_name
        + "{"
        + ",".join(
            [
                f'{INSTANCE_LABEL_NAME}="{instance}"',
                f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                f'{PROC_DISKSTATS_MAJ_MIN_LABEL_NAME}="{maj_min}"',
                f'{PROC_DISKSTATS_NAME_LABEL_NAME}="{disk_name}"',
            ]
        )
        + "} "
    )


def build_diskstats_metrics(
    maj_min: str,
    disk_name: str,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
) -> List[str]:
    return [
        build_diskstats_metric(
            metric_name, maj_min, disk_name, instance=instance, hostname=hostname
        )
        for metric_name in procDiskstatsIndexToMetricNameMap.values()
    ]


def build_mountinfo_metrics(
    mountinfo_parsed_lines: Optional[List[Dict[int, str]]] = None,
    proc_diskstats: Optional[procfs.Diskstats] = None,
    pid: int = PROC_DISKSTATS_METRICS_CONFIG_MOUNTINFO_PID_DEFAULT,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
) -> Optional[List[str]]:
    if mountinfo_parsed_lines is None:
        return None
    keep_maj_min = proc_diskstats.DevInfoMap if proc_diskstats is not None else None
    metrics = []
    for parsed_line in mountinfo_parsed_lines:
        maj_min = parsed_line[procfs.MOUNTINFO_MAJOR_MINOR]
        if keep_maj_min is not None and maj_min not in keep_maj_min:
            continue
        metrics.append(
            PROC_MOUNTINFO_METRIC
            + "{"
            + ",".join(
                [
                    f'{INSTANCE_LABEL_NAME}="{instance}"',
                    f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                    f'{PROC_MOUNTINFO_PID_LABEL_NAME}="{pid}"',
                ]
                + [
                    f'{label}="{parsed_line[i]}"'
                    for (i, label) in procMountinfoIndexToMetricLabelList
                ]
            )
            + "} "
        )
    return metrics


def generate_proc_diskstats_metrics(
    curr_proc_diskstats: procfs.Diskstats,
    prev_proc_diskstats: procfs.Diskstats,
    curr_prom_ts: int,
    interval: Optional[float] = PROC_DISKSTATS_METRICS_CONFIG_INTERVAL_DEFAULT,
    diskstats_metrics_info: Optional[Dict[str, ProcDiskstatsMetricsInfoTest]] = None,
    mountinfo_parsed_lines: Optional[List[Dict[int, str]]] = None,
    mountinfo_metrics_cache: Optional[List[str]] = None,
    pid: int = PROC_DISKSTATS_METRICS_CONFIG_MOUNTINFO_PID_DEFAULT,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
) -> Tuple[List[str], Optional[Dict[str, ZeroDeltaType]]]:
    metrics, zero_delta_map = [], {}

    if diskstats_metrics_info is None:
        diskstats_metrics_info = {}
    for maj_min, curr_dev_info in curr_proc_diskstats.DevInfoMap.items():
        prev_dev_info = prev_proc_diskstats.DevInfoMap.get(maj_min)
        if prev_dev_info is None:
            continue
        metrics_info = diskstats_metrics_info.get(maj_min)
        name_change = curr_dev_info.Name != prev_dev_info.Name
        full_data = metrics_info is None or metrics_info.CycleNum == 0 or name_change
        zero_delta = [False] * procfs.DISKSTATS_VALUE_FIELDS_NUM
        for index, metric_name in procDiskstatsIndexToMetricNameMap.items():
            delta = uint32_delta(curr_dev_info.Stats[index], prev_dev_info.Stats[index])
            if delta != 0 or full_data or not metrics_info.ZeroDelta[index]:
                if index in procDiskstatsIndexPctMetric:
                    factor, prec = procDiskstatsIndexPctMetric[index]
                    value = f"{delta * factor / interval:.{prec}f}"
                else:
                    value = str(delta)
                metrics.append(
                    build_diskstats_metric(
                        metric_name,
                        maj_min,
                        curr_dev_info.Name,
                        instance=instance,
                        hostname=hostname,
                    )
                    + f"{value} {curr_prom_ts}"
                )
            zero_delta[index] = delta == 0
        if metrics_info is not None and name_change:
            metrics.append(
                build_diskstats_metric(
                    PROC_DISKSTATS_INFO_METRIC,
                    maj_min,
                    prev_dev_info.Name,
                    instance=instance,
                    hostname=hostname,
                )
                + f"0 {curr_prom_ts}"
            )
        if full_data:
            metrics.append(
                build_diskstats_metric(
                    PROC_DISKSTATS_INFO_METRIC,
                    maj_min,
                    curr_dev_info.Name,
                    instance=instance,
                    hostname=hostname,
                )
                + f"1 {curr_prom_ts}"
            )
        zero_delta_map[maj_min] = zero_delta

    if mountinfo_parsed_lines is not None:
        mountinfo_metrics = build_mountinfo_metrics(
            mountinfo_parsed_lines,
            proc_diskstats=curr_proc_diskstats,
            pid=pid,
            instance=instance,
            hostname=hostname,
        )
        if mountinfo_metrics_cache is not None:
            out_of_scope_metrics = set(mountinfo_metrics_cache) - set(mountinfo_metrics)
            for metric in out_of_scope_metrics:
                metrics.append(f"{metric}0 {curr_prom_ts}")
        for metric in mountinfo_metrics:
            metrics.append(f"{metric}1 {curr_prom_ts}")

    metrics.append(
        f"{PROC_DISKSTATS_INTERVAL_METRIC_NAME}{{"
        + f'{INSTANCE_LABEL_NAME}="{instance}",{HOSTNAME_LABEL_NAME}="{hostname}"'
        + f"}} {interval:.06f} {curr_prom_ts}"
    )

    return metrics, zero_delta_map


def generate_proc_diskstats_test_case(
    name: str,
    curr_proc_diskstats: procfs.Diskstats,
    prev_proc_diskstats: procfs.Diskstats,
    ts: Optional[float] = None,
    interval: Optional[float] = PROC_DISKSTATS_METRICS_CONFIG_INTERVAL_DEFAULT,
    diskstats_metrics_info: Optional[Dict[str, ProcDiskstatsMetricsInfoTest]] = None,
    mountinfo_parsed_lines: Optional[List[Dict[int, str]]] = None,
    mountinfo_changed: bool = False,
    mountinfo_metrics_cache: Optional[List[str]] = None,
    mountinfo_cycle_num: int = 0,
    full_metrics_factor: int = PROC_DISKSTATS_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT,
    pid: int = PROC_DISKSTATS_METRICS_CONFIG_MOUNTINFO_PID_DEFAULT,
    description: Optional[str] = None,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
) -> ProcDiskstatsMetricsTestCase:
    if ts is None:
        ts = time.time()
    curr_prom_ts = int(ts * 1000)
    prev_prom_ts = curr_prom_ts - int(interval * 1000)

    metrics, want_zero_delta_map = generate_proc_diskstats_metrics(
        curr_proc_diskstats=curr_proc_diskstats,
        prev_proc_diskstats=prev_proc_diskstats,
        curr_prom_ts=curr_prom_ts,
        interval=interval,
        diskstats_metrics_info=diskstats_metrics_info,
        mountinfo_parsed_lines=mountinfo_parsed_lines,
        mountinfo_metrics_cache=mountinfo_metrics_cache,
        pid=pid,
        instance=instance,
        hostname=hostname,
    )

    return ProcDiskstatsMetricsTestCase(
        Name=name,
        Description=description,
        Instance=instance,
        Hostname=hostname,
        MountinfoPid=pid,
        CurrProcDiskstats=curr_proc_diskstats,
        PrevProcDiskstats=prev_proc_diskstats,
        CurrPromTs=curr_prom_ts,
        PrevPromTs=prev_prom_ts,
        PrimeDiskstatsMetricsInfo=diskstats_metrics_info,
        MountifoParsedLines=mountinfo_parsed_lines,
        MountinfoChanged=mountinfo_changed,
        PrimeMountinfoMetricsCache=mountinfo_metrics_cache,
        MountinfoCycleNum=mountinfo_cycle_num,
        FullMetricsFactor=full_metrics_factor,
        WantMetricsCount=len(metrics),
        WantMetrics=metrics,
        ReportExtra=True,
        WantZeroDeltaMap=want_zero_delta_map,
    )


def make_ref_proc_diskstats(num_dev: int = 2, changed: bool = True) -> procfs.Diskstats:
    dev_info_map = {}
    for k in range(num_dev):
        maj_min = f"{100*k}:{k % 3}"
        name = f"disk{100*k}{k % 3}"
        stats = [
            10 * (k + 1) * procfs.DISKSTATS_VALUE_FIELDS_NUM + i
            for i in range(procfs.DISKSTATS_VALUE_FIELDS_NUM)
        ]
        dev_info_map[maj_min] = procfs.DiskstatsDevInfo(
            Name=name,
            Stats=stats,
        )
    return procfs.Diskstats(
        DevInfoMap=dev_info_map,
        Changed=changed,
    )


def make_ref_mountinfo_parsed_lines(
    proc_diskstats: Optional[procfs.Diskstats] = None,
    num_extra_mounts: int = 2,
    max_mounts_per_dev: int = 3,
) -> List[Dict[int, str]]:
    mountinfo_parsed_lines = []
    if proc_diskstats is not None:
        i = 0
        for maj_min, dev_info in proc_diskstats.DevInfoMap.items():
            for j in range(i + 1):
                mountinfo_parsed_lines.append(
                    {
                        procfs.MOUNTINFO_MAJOR_MINOR: maj_min,
                        procfs.MOUNTINFO_ROOT: f"/root-{maj_min}-{j}",
                        procfs.MOUNTINFO_MOUNT_POINT: f"/mount-{maj_min}-{j}",
                        procfs.MOUNTINFO_FS_TYPE: f"/fstype-{maj_min}",
                        procfs.MOUNTINFO_MOUNT_SOURCE: f"/disk/{dev_info.Name}",
                    }
                )
            i = (i + 1) % max_mounts_per_dev
    for j in range(num_extra_mounts):
        maj_min = f"maj{(j + 1)*100}:min{j}"
        mountinfo_parsed_lines.append(
            {
                procfs.MOUNTINFO_MAJOR_MINOR: "",
                procfs.MOUNTINFO_ROOT: f"/root-{maj_min}-{j}",
                procfs.MOUNTINFO_MOUNT_POINT: f"/mount-{maj_min}-{j}",
                procfs.MOUNTINFO_FS_TYPE: f"/no-fstype-{maj_min}",
                procfs.MOUNTINFO_MOUNT_SOURCE: f"/no-disk/{dev_info.Name}",
            }
        )
    return mountinfo_parsed_lines


def generate_proc_diskstats_metrics_test_cases(
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    test_cases_root_dir: Optional[str] = lsvmi_test_cases_root_dir,
):
    test_cases = []
    tc_num = 0

    num_dev = 3

    ref_proc_diskstats = make_ref_proc_diskstats(num_dev=num_dev)
    max_val = 0
    for dev_info in ref_proc_diskstats.DevInfoMap.values():
        max_val = max(max_val, max(dev_info.Stats))
    ref_mountinfo_parsed_lines = make_ref_mountinfo_parsed_lines(
        proc_diskstats=ref_proc_diskstats
    )
    ref_mountinfo_metrics_cache = build_mountinfo_metrics(
        ref_mountinfo_parsed_lines,
        ref_proc_diskstats,
        instance=instance,
        hostname=hostname,
    )

    name = "all_changed"
    curr_proc_diskstats = ref_proc_diskstats
    prev_proc_diskstats = deepcopy(curr_proc_diskstats)
    for n, dev_info in enumerate(prev_proc_diskstats.DevInfoMap.values()):
        for i in range(len(dev_info.Stats)):
            delta = max_val + (n + 1) * procfs.DISKSTATS_VALUE_FIELDS_NUM + i
            dev_info.Stats[i] = uint32_delta(dev_info.Stats[i], delta)

    for null_mountinfo in [True, False]:
        description = f"null_mountinfo={null_mountinfo}"
        test_cases.append(
            generate_proc_diskstats_test_case(
                f"{name}/{tc_num:04d}",
                curr_proc_diskstats,
                prev_proc_diskstats,
                mountinfo_parsed_lines=(
                    ref_mountinfo_parsed_lines if not null_mountinfo else None
                ),
                mountinfo_changed=not null_mountinfo,
                description=description,
                instance=instance,
                hostname=hostname,
            )
        )
        tc_num += 1

        for cycle_num in [0, 1]:
            for zero_delta in [False, True]:
                diskstats_metrics_info = {
                    maj_min: ProcDiskstatsMetricsInfoTest(
                        CycleNum=cycle_num,
                        ZeroDelta=[zero_delta] * procfs.DISKSTATS_VALUE_FIELDS_NUM,
                        MetricsCache=build_diskstats_metrics(
                            maj_min, dev_info.Name, instance=instance, hostname=hostname
                        ),
                        InfoMetric=build_diskstats_metric(
                            PROC_DISKSTATS_INFO_METRIC,
                            maj_min,
                            dev_info.Name,
                            instance=instance,
                            hostname=hostname,
                        ),
                    )
                    for maj_min, dev_info in curr_proc_diskstats.DevInfoMap.items()
                }
                for null_diskstats_metrics_info in [True, False]:
                    description = ",".join(
                        [
                            f"cycle_num={cycle_num}",
                            f"zero_delta={zero_delta}",
                            f"null_diskstats_metrics_info={null_diskstats_metrics_info}",
                            f"null_mountinfo={null_mountinfo}",
                        ]
                    )
                    test_cases.append(
                        generate_proc_diskstats_test_case(
                            f"{name}/{tc_num:04d}",
                            curr_proc_diskstats,
                            prev_proc_diskstats,
                            diskstats_metrics_info=(
                                diskstats_metrics_info
                                if not null_diskstats_metrics_info
                                else None
                            ),
                            mountinfo_parsed_lines=(
                                ref_mountinfo_parsed_lines
                                if not null_mountinfo
                                else None
                            ),
                            mountinfo_changed=not null_mountinfo,
                            description=description,
                            instance=instance,
                            hostname=hostname,
                        )
                    )
                    tc_num += 1

    name = "none_changed"
    curr_proc_diskstats = ref_proc_diskstats
    prev_proc_diskstats = curr_proc_diskstats
    for cycle_num in [0, 1]:
        for zero_delta in [False, True]:
            diskstats_metrics_info = {
                maj_min: ProcDiskstatsMetricsInfoTest(
                    CycleNum=cycle_num,
                    ZeroDelta=[zero_delta] * procfs.DISKSTATS_VALUE_FIELDS_NUM,
                    MetricsCache=build_diskstats_metrics(
                        maj_min, dev_info.Name, instance=instance, hostname=hostname
                    ),
                    InfoMetric=build_diskstats_metric(
                        PROC_DISKSTATS_INFO_METRIC,
                        maj_min,
                        dev_info.Name,
                        instance=instance,
                        hostname=hostname,
                    ),
                )
                for maj_min, dev_info in curr_proc_diskstats.DevInfoMap.items()
            }
            description = ",".join(
                [
                    f"cycle_num={cycle_num}",
                    f"zero_delta={zero_delta}",
                ]
            )
            test_cases.append(
                generate_proc_diskstats_test_case(
                    f"{name}/{tc_num:04d}",
                    curr_proc_diskstats,
                    prev_proc_diskstats,
                    diskstats_metrics_info=diskstats_metrics_info,
                    description=description,
                    instance=instance,
                    hostname=hostname,
                )
            )
            tc_num += 1

    name = "single_change"
    curr_proc_diskstats = ref_proc_diskstats
    diskstats_metrics_info = {
        maj_min: ProcDiskstatsMetricsInfoTest(
            CycleNum=1,
            ZeroDelta=[True] * procfs.DISKSTATS_VALUE_FIELDS_NUM,
            MetricsCache=build_diskstats_metrics(
                maj_min, dev_info.Name, instance=instance, hostname=hostname
            ),
            InfoMetric=build_diskstats_metric(
                PROC_DISKSTATS_INFO_METRIC,
                maj_min,
                dev_info.Name,
                instance=instance,
                hostname=hostname,
            ),
        )
        for maj_min, dev_info in curr_proc_diskstats.DevInfoMap.items()
    }
    for n, maj_min in enumerate(curr_proc_diskstats.DevInfoMap):
        for i in range(procfs.DISKSTATS_VALUE_FIELDS_NUM):
            prev_proc_diskstats = deepcopy(curr_proc_diskstats)
            dev_info = prev_proc_diskstats.DevInfoMap[maj_min]
            delta = max_val + (n + 1) * procfs.DISKSTATS_VALUE_FIELDS_NUM + i
            dev_info.Stats[i] = uint32_delta(dev_info.Stats[i], delta)
            description = f"maj_min={maj_min},i={i}"
            test_cases.append(
                generate_proc_diskstats_test_case(
                    f"{name}/{tc_num:04d}",
                    curr_proc_diskstats,
                    prev_proc_diskstats,
                    diskstats_metrics_info=diskstats_metrics_info,
                    description=description,
                    instance=instance,
                    hostname=hostname,
                )
            )
            tc_num += 1

    name = "1st_time_dev"
    curr_proc_diskstats = ref_proc_diskstats
    for maj_min in curr_proc_diskstats.DevInfoMap:
        prev_proc_diskstats = deepcopy(curr_proc_diskstats)
        del prev_proc_diskstats.DevInfoMap[maj_min]
        diskstats_metrics_info = {
            maj_min: ProcDiskstatsMetricsInfoTest(
                CycleNum=1,
                ZeroDelta=[True] * procfs.DISKSTATS_VALUE_FIELDS_NUM,
                MetricsCache=build_diskstats_metrics(
                    maj_min, dev_info.Name, instance=instance, hostname=hostname
                ),
                InfoMetric=build_diskstats_metric(
                    PROC_DISKSTATS_INFO_METRIC,
                    maj_min,
                    dev_info.Name,
                    instance=instance,
                    hostname=hostname,
                ),
            )
            for maj_min, dev_info in prev_proc_diskstats.DevInfoMap.items()
        }
        description = f"maj_min={maj_min}"
        test_cases.append(
            generate_proc_diskstats_test_case(
                f"{name}/{tc_num:04d}",
                curr_proc_diskstats,
                prev_proc_diskstats,
                diskstats_metrics_info=diskstats_metrics_info,
                description=description,
                instance=instance,
                hostname=hostname,
            )
        )
        tc_num += 1

    name = "new_dev"
    curr_proc_diskstats = ref_proc_diskstats
    for n, maj_min in enumerate(curr_proc_diskstats.DevInfoMap):
        prev_proc_diskstats = deepcopy(curr_proc_diskstats)
        dev_info = prev_proc_diskstats.DevInfoMap[maj_min]
        for i in range(procfs.DISKSTATS_VALUE_FIELDS_NUM):
            delta = max_val + (n + 1) * procfs.DISKSTATS_VALUE_FIELDS_NUM + i
            dev_info.Stats[i] = uint32_delta(dev_info.Stats[i], delta)
        diskstats_metrics_info = {
            maj_min1: ProcDiskstatsMetricsInfoTest(
                CycleNum=1,
                ZeroDelta=[True] * procfs.DISKSTATS_VALUE_FIELDS_NUM,
                MetricsCache=build_diskstats_metrics(
                    maj_min1, dev_info.Name, instance=instance, hostname=hostname
                ),
                InfoMetric=build_diskstats_metric(
                    PROC_DISKSTATS_INFO_METRIC,
                    maj_min1,
                    dev_info.Name,
                    instance=instance,
                    hostname=hostname,
                ),
            )
            for maj_min1, dev_info in curr_proc_diskstats.DevInfoMap.items()
            if maj_min1 != maj_min
        }
        description = f"maj_min={maj_min}"
        test_cases.append(
            generate_proc_diskstats_test_case(
                f"{name}/{tc_num:04d}",
                curr_proc_diskstats,
                prev_proc_diskstats,
                diskstats_metrics_info=diskstats_metrics_info,
                description=description,
                instance=instance,
                hostname=hostname,
            )
        )
        tc_num += 1
        description = f"maj_min={maj_min},all_zero"
        test_cases.append(
            generate_proc_diskstats_test_case(
                f"{name}/{tc_num:04d}",
                curr_proc_diskstats,
                curr_proc_diskstats,
                diskstats_metrics_info=diskstats_metrics_info,
                description=description,
                instance=instance,
                hostname=hostname,
            )
        )
        tc_num += 1

    name = "remove_dev"
    prev_proc_diskstats = ref_proc_diskstats
    mountinfo_metrics_cache = ref_mountinfo_metrics_cache
    diskstats_metrics_info = {
        maj_min: ProcDiskstatsMetricsInfoTest(
            CycleNum=1,
            ZeroDelta=[True] * procfs.DISKSTATS_VALUE_FIELDS_NUM,
            MetricsCache=build_diskstats_metrics(
                maj_min, dev_info.Name, instance=instance, hostname=hostname
            ),
            InfoMetric=build_diskstats_metric(
                PROC_DISKSTATS_INFO_METRIC,
                maj_min,
                dev_info.Name,
                instance=instance,
                hostname=hostname,
            ),
        )
        for maj_min, dev_info in prev_proc_diskstats.DevInfoMap.items()
    }
    for maj_min in curr_proc_diskstats.DevInfoMap:
        curr_proc_diskstats = deepcopy(curr_proc_diskstats)
        del curr_proc_diskstats.DevInfoMap[maj_min]
        description = f"maj_min={maj_min},null_mountinfo={True}"
        test_cases.append(
            generate_proc_diskstats_test_case(
                f"{name}/{tc_num:04d}",
                curr_proc_diskstats,
                prev_proc_diskstats,
                diskstats_metrics_info=diskstats_metrics_info,
                description=description,
                instance=instance,
                hostname=hostname,
            )
        )
        tc_num += 1

        description = f"maj_min={maj_min},null_mountinfo={False}"
        mountinfo_parsed_lines = [
            parsed_line
            for parsed_line in ref_mountinfo_parsed_lines
            if parsed_line[procfs.MOUNTINFO_MAJOR_MINOR] != maj_min
        ]
        test_cases.append(
            generate_proc_diskstats_test_case(
                f"{name}/{tc_num:04d}",
                curr_proc_diskstats,
                prev_proc_diskstats,
                diskstats_metrics_info=diskstats_metrics_info,
                mountinfo_parsed_lines=mountinfo_parsed_lines,
                mountinfo_changed=True,
                mountinfo_metrics_cache=mountinfo_metrics_cache,
                description=description,
                instance=instance,
                hostname=hostname,
            )
        )
        tc_num += 1

    save_test_cases(
        test_cases, test_cases_file, test_cases_root_dir=test_cases_root_dir
    )
