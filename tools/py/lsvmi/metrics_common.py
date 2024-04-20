#! /usr/bin/env python3


INSTANCE_LABEL_NAME = "instance"
HOSTNAME_LABEL_NAME = "hostname"


def uint64_delta(curr: int, prev: int) -> int:
    delta = curr - prev
    while delta < 0:
        delta += 1 << 64
    return delta
