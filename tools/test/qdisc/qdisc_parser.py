#! /usr/bin/env python3

from dataclasses import dataclass, field
from typing import List, Optional

# JSON serialize-able QdiscStats, matching qdisc/qdisc_parser.go:

# uint32 indexes:
QDISC_PARENT = 0
QDISC_HANDLE = 1
QDISC_PACKETS = 2
QDISC_DROPS = 3
QDISC_REQUEUES = 4
QDISC_OVERLIMITS = 5
QDISC_QLEN = 6
QDISC_BACKLOG = 7
QDISK_UINT32_NUM_STATS = 8

# uint64 indexes:
QDISC_BYTES = 0
QDISC_GCFLOWS = 1
QDISC_THROTTLED = 2
QDISC_FLOWSPLIMIT = 3
QDISK_UINT64_NUM_STATS = 4

QDISC_MAJ_NUM_BITS = 16
QDISC_MIN_NUM_BITS = 32 - QDISC_MAJ_NUM_BITS


def format_maj_min(val: int) -> str:
    return ":".join(
        [
            f"{val >> QDISC_MIN_NUM_BITS:0{(QDISC_MAJ_NUM_BITS + 3)//4}x}",
            f"{val & ((1 << QDISC_MIN_NUM_BITS) - 1):0{(QDISC_MIN_NUM_BITS + 3)//4}x}",
        ]
    )


@dataclass
class QdiscInfoKey:
    IfIndex: int = 0
    Handle: int = 0

    def __str__(self) -> str:
        return f"QdiscInfoKey(IfIndex={self.IfIndex}, Handle={self.Handle} ({format_maj_min(self.Handle)}))"

    def __hash__(self) -> int:
        return (self.IfIndex << 32) + self.Handle


@dataclass
class QdiscInfo:
    IfName: Optional[str] = None
    Kind: Optional[str] = None
    Uint32: List[int] = field(default_factory=lambda: [0] * QDISK_UINT32_NUM_STATS)
    Uint64: List[int] = field(default_factory=lambda: [0] * QDISK_UINT64_NUM_STATS)
