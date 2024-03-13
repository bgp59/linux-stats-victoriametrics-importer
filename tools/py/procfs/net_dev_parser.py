#! /usr/bin/env python3

from typing import Dict, List, Literal

# JSON serialize-able NetDev, matching profcs/net_dev_parser.go:

NET_DEV_RX_BYTES = 0
NET_DEV_RX_PACKETS = 1
NET_DEV_RX_ERRS = 2
NET_DEV_RX_DROP = 3
NET_DEV_RX_FIFO = 4
NET_DEV_RX_FRAME = 5
NET_DEV_RX_COMPRESSED = 6
NET_DEV_RX_MULTICAST = 7
NET_DEV_TX_BYTES = 8
NET_DEV_TX_PACKETS = 9
NET_DEV_TX_ERRS = 10
NET_DEV_TX_DROP = 11
NET_DEV_TX_FIFO = 12
NET_DEV_TX_COLLS = 13
NET_DEV_TX_CARRIER = 14
NET_DEV_TX_COMPRESSED = 15
NET_DEV_NUM_STATS = 16

NetDevStats = Dict[str, List[int]]

NetDev = Dict[Literal["DevStats"], NetDevStats]
