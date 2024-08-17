#!/usr/bin/env python3

import json
import os
import sys
from dataclasses import asdict, dataclass
from typing import List, Optional

import procfs
from testutils import go_module_root, procfs_proc_root_dir, procfs_test_cases_root_dir

tc_procfs_root = os.path.relpath(
    procfs_proc_root_dir, os.path.join(go_module_root, "procfs")
)


# Should match ../../procfs/pid_list_test.go
pid_tid_list_test_case_file = os.path.join(
    procfs_test_cases_root_dir, "pid_tid_list_test_case.json"
)


@dataclass
class PidTidListTestCase:
    ProcfsRoot: str = tc_procfs_root
    Flags: int = 0
    NPart: int = 0
    PidTidLists: Optional[List[List[procfs.PidTid]]] = None


if __name__ == "__main__":
    pid_tid_list = []
    for p in os.listdir(procfs_proc_root_dir):
        try:
            pid = int(p)
            if pid == 0:
                continue
            pid_tid_list.append(procfs.PidTid(pid, procfs.PID_ONLY_TID))
        except ValueError:
            continue
        for t in os.listdir(os.path.join(procfs_proc_root_dir, p, "task")):
            try:
                tid = int(t)
                pid_tid_list.append(procfs.PidTid(pid, tid))
            except ValueError:
                continue
    pid_tid_list.sort(key=lambda pt: (pt.Pid, pt.Tid))
    test_cases = []
    for flags in [
        procfs.PID_LIST_CACHE_PID_ENABLED,
        procfs.PID_LIST_CACHE_TID_ENABLED,
        procfs.PID_LIST_CACHE_ALL_ENABLED,
    ]:
        for n_part in range(1, 9):
            pid_tid_lists = [[] for _ in range(n_part)]
            for pid_tid in pid_tid_list:
                part = None
                if pid_tid.Tid == procfs.PID_ONLY_TID:
                    if flags & procfs.PID_LIST_CACHE_PID_ENABLED:
                        part = pid_tid.Pid % n_part
                elif flags & procfs.PID_LIST_CACHE_TID_ENABLED:
                    part = pid_tid.Tid % n_part
                if part is not None:
                    pid_tid_lists[part].append(pid_tid)
            tc = PidTidListTestCase(
                Flags=flags, NPart=n_part, PidTidLists=pid_tid_lists
            )
            test_cases.append(asdict(tc))
    with open(pid_tid_list_test_case_file, "wt") as f:
        json.dump(test_cases, f, indent=2)
        f.write("\n")
    print(f"{pid_tid_list_test_case_file} generated", file=sys.stderr)
