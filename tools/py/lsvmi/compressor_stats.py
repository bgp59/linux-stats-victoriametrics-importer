#! /usr/bin/env python3

# JSON serialize-able Compressor[Pool]Stats et al, matching lsvmi/compressor_pool.go

from dataclasses import dataclass, field
from typing import Dict, List

COMPRESSOR_STATS_READ_COUNT = 0
COMPRESSOR_STATS_READ_BYTE_COUNT = 1
COMPRESSOR_STATS_SEND_COUNT = 2
COMPRESSOR_STATS_SEND_BYTE_COUNT = 3
COMPRESSOR_STATS_TIMEOUT_FLUSH_COUNT = 4
COMPRESSOR_STATS_SEND_ERROR_COUNT = 5
COMPRESSOR_STATS_WRITE_ERROR_COUNT = 6
COMPRESSOR_STATS_UINT64_LEN = 7

COMPRESSOR_STATS_COMPRESSION_FACTOR = 0
COMPRESSOR_STATS_FLOAT64_LEN = 1


@dataclass
class CompressorStats:
    Uint64Stats: List[int] = field(
        default_factory=lambda: [0] * COMPRESSOR_STATS_UINT64_LEN
    )
    Float64Stats: List[int] = field(
        default_factory=lambda: [0.0] * COMPRESSOR_STATS_FLOAT64_LEN
    )


CompressorPoolStats = Dict[str, CompressorStats]
