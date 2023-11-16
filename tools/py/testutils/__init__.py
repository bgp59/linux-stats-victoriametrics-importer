#! python3

import os

# The following sub-dirs are relative to module root, they should match
# testutils/common.go:
TESTDATA_SUBDIR = "testdata"
TESTDATA_PROC_SUBDIR = TESTDATA_SUBDIR + "/proc"
TESTDATA_PROCFS_SUBDIR = TESTDATA_SUBDIR + "/procfs"

TESTCASES_SUBDIR = TESTDATA_SUBDIR + "/testcases"

tools_py_root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
go_module_root = os.path.dirname(os.path.dirname((tools_py_root)))

testdata_proc_dir = os.path.join(go_module_root, TESTDATA_PROC_SUBDIR)
testdata_procfs_dir = os.path.join(go_module_root, TESTDATA_PROCFS_SUBDIR)
testcases_dir = os.path.join(go_module_root, TESTCASES_SUBDIR)
