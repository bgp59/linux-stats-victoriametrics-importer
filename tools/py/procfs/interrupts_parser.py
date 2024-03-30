#! /usr/bin/env python3

from dataclasses import dataclass
from typing import Dict, List, Optional

# JSON serialize-able Interrupts et al, matching profcs/interrupts_parser.go:


@dataclass
class InterruptsIrqInfo:
    Controller: Optional[str] = None
    HWInterrupt: Optional[str] = None
    Devices: Optional[str] = None
    Changed: bool = False


@dataclass
class InterruptsInfo:
    IrqInfo: Optional[Dict[str, InterruptsIrqInfo]] = None
    IrqChanged: bool = False
    CpuListChanged: bool = False


@dataclass
class Interrupts:
    CpuList: Optional[List[int]] = None
    Counters: Optional[List[List[int]]] = None
    NumCounters: int = 0
    Info: Optional[InterruptsInfo] = None
