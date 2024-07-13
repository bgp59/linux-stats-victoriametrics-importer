#!/usr/bin/env python3

import json
import os
import sys

from testutils import go_module_root, lsvmi_procfs_root, procfs_testdata_root

# Should match ../../procfs/pid_list_test.go
pidListTestCaseFile = os.path.join(procfs_testdata_root, "pid_list_test_case.json")
PID_STAT_PID_ONLY_TID = 0

if __name__ == "__main__":
    pid_tid_list = []
    for p in os.listdir(lsvmi_procfs_root):
        try:
            pid = int(p)
            if pid == 0:
                continue
            pid_tid_list.append([pid, PID_STAT_PID_ONLY_TID])
        except ValueError:
            continue
        for t in os.listdir(os.path.join(lsvmi_procfs_root, p, "task")):
            try:
                tid = int(t)
                pid_tid_list.append([pid, tid])
            except ValueError:
                continue
    pid_tid_list.sort()
    tc = dict(
        ProcfsRoot=os.path.relpath(
            lsvmi_procfs_root, os.path.join(go_module_root, "procfs")
        ),
        PidTidList=pid_tid_list,
    )
    with open(pidListTestCaseFile, "wt") as f:
        json.dump(tc, f, indent=2)
        f.write("\n")
    print(f"{pidListTestCaseFile} generated", file=sys.stderr)
