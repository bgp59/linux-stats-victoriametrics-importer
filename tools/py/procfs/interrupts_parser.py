#! /usr/bin/env python3

from dataclasses import dataclass
from typing import Any, Dict, List, Optional

from .common import b64encode_str

# JSON serialize-able Interrupts et al, matching profcs/interrupts_parser.go:


@dataclass
class InterruptsIrqInfo:
    Controller: Optional[str] = None
    HWInterrupt: Optional[str] = None
    Devices: Optional[str] = None
    Changed: bool = False
    # The Go counterpart uses []byte fields and those have to be JSON serialized
    # base64 encoded, which renders them hard to read. Provide additional fields
    # to save the original content; these fields will not be deserialized
    # because they have no Go struct correspondent.
    rawController: Optional[str] = None
    rawHWInterrupt: Optional[str] = None
    rawDevices: Optional[str] = None

    def __eq__(self, other: Any = None) -> bool:
        return (
            isinstance(other, self.__class__)
            and self.Controller == other.Controller
            and self.HWInterrupt == other.HWInterrupt
            and self.Devices == other.Devices
        )


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

    def b64encode(self):
        if self.Info is None or self.Info.IrqInfo is None:
            return
        for irq_info in self.Info.IrqInfo.values():
            if irq_info.rawController is None and irq_info.Controller is not None:
                irq_info.rawController = irq_info.Controller
                irq_info.Controller = b64encode_str(irq_info.Controller)
            if irq_info.rawHWInterrupt is None and irq_info.HWInterrupt is not None:
                irq_info.rawHWInterrupt = irq_info.HWInterrupt
                irq_info.HWInterrupt = b64encode_str(irq_info.HWInterrupt)
            if irq_info.rawDevices is None and irq_info.Devices is not None:
                irq_info.rawDevices = irq_info.Devices
                irq_info.Devices = b64encode_str(irq_info.Devices)
