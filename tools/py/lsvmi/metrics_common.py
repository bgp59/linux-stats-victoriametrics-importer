#! /usr/bin/env python3

import json
import os
import sys
from dataclasses import asdict
from typing import List, Optional

INSTANCE_LABEL_NAME = "instance"
HOSTNAME_LABEL_NAME = "hostname"

from testutils import lsvmi_test_cases_root_dir


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


def save_test_cases(
    test_cases: List,
    test_cases_file: Optional[str] = None,
    test_cases_root_dir: Optional[str] = lsvmi_test_cases_root_dir,
):
    use_stdout = test_cases_file in [None, "-"] or test_cases_root_dir in [None, "-"]
    if use_stdout:
        fp = sys.stdout
    else:
        out_file = os.path.join(test_cases_root_dir, test_cases_file)
        out_dir = os.path.dirname(out_file)
        os.makedirs(out_dir, exist_ok=True)
        fp = open(out_file, "wt")
    json.dump(list(map(asdict, test_cases)), fp=fp, indent=2)
    if not use_stdout:
        fp.close()
        print(f"{out_file} generated", file=sys.stderr)
