#! /usr/bin/env python3

from dataclasses import dataclass
from typing import Dict, List, Optional

# JSON serialize-able Softirqs et al, matching profcs/softirqs_parser.go:


@dataclass
class Softirqs:
    Counters: Optional[Dict[str, List[int]]] = None
    CpuList: Optional[List[int]] = None
    CpuListChanged: bool = False
    NumCounters: int = 0
