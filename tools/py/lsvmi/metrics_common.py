#! /usr/bin/env python3


INSTANCE_LABEL_NAME = "instance"
HOSTNAME_LABEL_NAME = "hostname"


def uint32_delta(curr: int, prev: int) -> int:
    delta = curr - prev
    while delta < 0:
        delta += 1 << 32
    return delta


def int32_to_uint32(i: int) -> int:
    return i if i >= 0 else ((1 << 32) + i)


def uint32_to_int32(i: int) -> int:
    return i if (i & (1 << 31)) == 0 else i - (1 << 32)


def uint64_delta(curr: int, prev: int) -> int:
    delta = curr - prev
    while delta < 0:
        delta += 1 << 64
    return delta
