#! python3

import os

TESTDATA_SUBDIR = "testdata"
LSVMI_TESTDATA_PROCFS_ROOT_SUBDIR = "testdata/lsvmi/proc"
PROCFS_TESTDATA_ROOT_SUBDIR = "testdata/procfs"

tools_py_root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
go_module_root = os.path.dirname(os.path.dirname((tools_py_root)))

lsvmi_testdata_procfs_root = os.path.join(
    go_module_root, LSVMI_TESTDATA_PROCFS_ROOT_SUBDIR
)
procfs_testdata_root = os.path.join(go_module_root, PROCFS_TESTDATA_ROOT_SUBDIR)
