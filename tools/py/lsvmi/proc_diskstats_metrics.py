#! /usr/bin/env python3

# Generate test cases for lsvmi/proc_interrupts_metrics_test.go

from dataclasses import dataclass, field
from typing import Dict, List, Optional, Tuple

import procfs

from . import (
    DEFAULT_TEST_HOSTNAME,
    DEFAULT_TEST_INSTANCE,
    HOSTNAME_LABEL_NAME,
    INSTANCE_LABEL_NAME,
)

PROC_DISKSTATS_METRICS_CONFIG_INTERVAL_DEFAULT = "5s"
PROC_DISKSTATS_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT = 12
PROC_DISKSTATS_METRICS_CONFIG_MOUNTINFO_PID_DEFAULT = 1

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


@dataclass
class ProcDiskstatsMetricsInfoTest:
    CycleNum: int = 0
    ZeroDelta: ZeroDeltaType = field(
        default_factory=lambda: [False] * procfs.DISKSTATS_VALUE_FIELDS_NUM
    )
    MetricsCache: Optional[List[str]] = None


@dataclass
class ProcDiskstatsMetricsTestCase:
    Name: Optional[str] = None
    Description: Optional[str] = None
    Instance: str = DEFAULT_TEST_INSTANCE
    Hostname: str = DEFAULT_TEST_HOSTNAME
    MountinfoPid: int = PROC_DISKSTATS_METRICS_CONFIG_MOUNTINFO_PID_DEFAULT
    CurrProcDiskstats: Optional[procfs.Diskstats] = None
    PrevProcDiskstats: Optional[procfs.Diskstats] = None
    CurrPromTs: int
    PrevPromTs: int
    PrimeDiskstatsMetricsInfo: Optional[Dict[str, ProcDiskstatsMetricsInfoTest]] = None
    MountifoParsedLines: Optional[List[Dict[int, str]]] = None
    MountinfoChanged: bool = False
    PrimeMountinfoMetricsCache: Optional[List[str]] = None
    MountinfoCycleNum: int
    FullMetricsFactor: int = PROC_DISKSTATS_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT
    WantMetricsCount: int
    WantMetrics: Optional[List[str]] = None
    ReportExtra: bool = True
    WantZeroDeltaMap: Optional[Dict[str, ZeroDeltaType]] = None


def diskstats_metrics(
    maj_min: str,
    disk_name: str,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
) -> List[str]:
    return [
        (
            procDiskstatsIndexToMetricNameMap[i]
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
        for i in range(procfs.DISKSTATS_VALUE_FIELDS_NUM)
        if i in procDiskstatsIndexToMetricNameMap
    ]


def mountinfo_metrics(
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
            (
                PROC_MOUNTINFO_METRIC
                + "{"
                + ",".join(
                    [
                        f'{INSTANCE_LABEL_NAME}="{instance}"',
                        f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                        f'{PROC_MOUNTINFO_PID_LABEL_NAME}="pid"',
                    ]
                    + [
                        f'{label}="{parsed_line[i]}"'
                        for (i, label) in procMountinfoIndexToMetricLabelList
                    ]
                )
                + "} "
            )
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
    pass
