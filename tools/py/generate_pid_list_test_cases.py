#!/usr/bin/env python3

import json
import os
import sys
from dataclasses import asdict, dataclass
from typing import List, Optional

from testutils import go_module_root, lsvmi_procfs_root, procfs_testdata_root

tc_procfs_root = os.path.relpath(
    lsvmi_procfs_root, os.path.join(go_module_root, "procfs")
)

# Should match ../../procfs/pid_list.go
PID_LIST_CACHE_PID_ENABLED = 1 << 0
PID_LIST_CACHE_TID_ENABLED = 1 << 1
PID_STAT_PID_ONLY_TID = 0


@dataclass
class PidTid:
    Pid: int = 0
    Tid: int = 0


# Should match ../../procfs/pid_list_test.go
pidListTestCaseFile = os.path.join(procfs_testdata_root, "pid_list_test_case.json")


@dataclass
class PidListTestCase:
    ProcfsRoot: str = tc_procfs_root
    Flags: int = 0
    NPart: int = 0
    PidTidLists: Optional[List[List[PidTid]]] = None


if __name__ == "__main__":
    pid_tid_list = []
    for p in os.listdir(lsvmi_procfs_root):
        try:
            pid = int(p)
            if pid == 0:
                continue
            pid_tid_list.append(PidTid(pid, PID_STAT_PID_ONLY_TID))
        except ValueError:
            continue
        for t in os.listdir(os.path.join(lsvmi_procfs_root, p, "task")):
            try:
                tid = int(t)
                pid_tid_list.append(PidTid(pid, tid))
            except ValueError:
                continue
    pid_tid_list.sort(key=lambda pt: (pt.Pid, pt.Tid))
    test_cases = []
    for flags in [
        PID_LIST_CACHE_PID_ENABLED,
        PID_LIST_CACHE_TID_ENABLED,
        PID_LIST_CACHE_PID_ENABLED | PID_LIST_CACHE_TID_ENABLED,
    ]:
        for n_part in range(1, 9):
            pid_tid_lists = [[] for _ in range(n_part)]
            for pid_tid in pid_tid_list:
                part = None
                if pid_tid.Tid == PID_STAT_PID_ONLY_TID:
                    if flags & PID_LIST_CACHE_PID_ENABLED:
                        part = pid_tid.Pid % n_part
                elif flags & PID_LIST_CACHE_TID_ENABLED:
                    part = pid_tid.Tid % n_part
                if part is not None:
                    pid_tid_lists[part].append(pid_tid)
            tc = PidListTestCase(Flags=flags, NPart=n_part, PidTidLists=pid_tid_lists)
            test_cases.append(asdict(tc))
    with open(pidListTestCaseFile, "wt") as f:
        json.dump(test_cases, f, indent=2)
        f.write("\n")
    print(f"{pidListTestCaseFile} generated", file=sys.stderr)
