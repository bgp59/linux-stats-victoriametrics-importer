#! /usr/bin/env python3

from dataclasses import dataclass, field
from typing import Dict, List, Optional

# JSON serialize-able Diskstats et al, matching profcs/diskstats_parser.go:

DISKSTATS_NUM_READS_COMPLETED = 0
DISKSTATS_NUM_READS_MERGED = 1
DISKSTATS_NUM_READ_SECTORS = 2
DISKSTATS_READ_MILLISEC = 3
DISKSTATS_NUM_WRITES_COMPLETED = 4
DISKSTATS_NUM_WRITES_MERGED = 5
DISKSTATS_NUM_WRITE_SECTORS = 6
DISKSTATS_WRITE_MILLISEC = 7
DISKSTATS_NUM_IO_IN_PROGRESS = 8
DISKSTATS_IO_MILLISEC = 9
DISKSTATS_IO_WEIGTHED_MILLISEC = 10
DISKSTATS_NUM_DISCARDS_COMPLETED = 11
DISKSTATS_NUM_DISCARDS_MERGED = 12
DISKSTATS_NUM_DISCARD_SECTORS = 13
DISKSTATS_DISCARD_MILLISEC = 14
DISKSTATS_NUM_FLUSH_REQUESTS = 15
DISKSTATS_FLUSH_MILLISEC = 16
DISKSTATS_VALUE_FIELDS_NUM = 17


@dataclass
class DiskstatsDevInfo:
    Name: Optional[str] = None
    Stats: List[int] = field(default_factory=lambda: [0] * DISKSTATS_VALUE_FIELDS_NUM)


@dataclass
class Diskstats:
    DevInfoMap: Optional[Dict[str, DiskstatsDevInfo]] = None
    Changed: bool = False
