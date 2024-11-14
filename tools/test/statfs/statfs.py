#! /usr/bin/env python3

# JSON serialize-able Statfs, matching unix.Statfs_t:

from dataclasses import dataclass, field
from typing import List


@dataclass
class FsidT:
    Val: List[int] = field(default_factory=lambda: [0] * 2)


@dataclass
class Statfs:
    Type: int = 0
    Bsize: int = 0
    Blocks: int = 0
    Bfree: int = 0
    Bavail: int = 0
    Files: int = 0
    Ffree: int = 0
    Fsid: FsidT = field(default_factory=FsidT)
    Namelen: int = 0
    Frsize: int = 0
    Flags: int = 0
    Spare: List[int] = field(default_factory=lambda: [0] * 4)
