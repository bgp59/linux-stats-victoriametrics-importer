#! /usr/bin/env python3

# Generate test cases for lsvmi/statfs_metrics_test.go

from dataclasses import dataclass
from typing import List, Optional

from statfs import statfs

# The following should match lsvmi/statfs_metrics.go:


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


@dataclass
class StatfsInfoTestData:
    Fs: Optional[str] = None
    MountPoint: Optional[str] = None
    FsType: Optional[str] = None
    Statfs: Optional[statfs.Statfs] = None
    CycleNum: int = 0
    ScanNum: int = 0


@dataclass
class StatfsMetricsTestCase:
    Name: Optional[str] = None
    Description: Optional[str] = None
    Instance: Optional[str] = None
    Hostname: Optional[str] = None
    CurrStatfsInfo: Optional[List[StatfsInfoTestData]] = None
    PrevStatfsInfo: Optional[List[StatfsInfoTestData]] = None
    WantMetricsCount: int = 0
    WantMetrics: Optional[List[str]] = None
    ReportExtra: bool = True
