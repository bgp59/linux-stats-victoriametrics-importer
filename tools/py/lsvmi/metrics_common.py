#! /usr/bin/env python3

from base64 import b64encode
from codecs import utf_8_decode

INSTANCE_LABEL_NAME = "instance"
HOSTNAME_LABEL_NAME = "hostname"


def uint64_delta(crt: int, prev: int) -> int:
    delta = crt - prev
    while delta < 0:
        delta += 1 << 64
    return delta


def b64encode_str(s: str) -> str:
    return utf_8_decode(b64encode(bytes(s, "utf-8")))[0]
