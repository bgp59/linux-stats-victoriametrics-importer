#! python3

import os

# Should match internal/testutils/testdata.go:
TESTDATA_SUBDIR = "testdata"
LSVMI_TESTDATA_SUBDIR = f"{TESTDATA_SUBDIR}/lsvmi"
LSVMI_TESTCASES_SUBDIR = f"{LSVMI_TESTDATA_SUBDIR}/testcases"
LSVMI_PROC_SUBDIR = f"{LSVMI_TESTDATA_SUBDIR}/proc"
PROCFS_TESTDATA_SUBDIR = f"{TESTDATA_SUBDIR}/procfs"
PROCFS_TESTCASES_SUBDIR = f"{PROCFS_TESTDATA_SUBDIR}/testcases"
PROCFS_PROC_SUBDIR = f"{PROCFS_TESTDATA_SUBDIR}/proc"

DEFAULT_TEST_INSTANCE = "lsvmi_test"
DEFAULT_TEST_HOSTNAME = "lsvmi-test"

BENCHMARKS_SUBDIR = "benchmarks"

tools_py_root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
go_module_root = os.path.dirname(os.path.dirname((tools_py_root)))

lsvmi_proc_root_dir = os.path.join(go_module_root, LSVMI_PROC_SUBDIR)
lsvmi_test_cases_root_dir = os.path.join(go_module_root, LSVMI_TESTCASES_SUBDIR)

procfs_proc_root_dir = os.path.join(go_module_root, PROCFS_PROC_SUBDIR)
procfs_test_cases_root_dir = os.path.join(go_module_root, PROCFS_TESTCASES_SUBDIR)

benchmarks_root_dir = os.path.join(go_module_root, BENCHMARKS_SUBDIR)

procfs_testdata_root = os.path.join(go_module_root, PROCFS_TESTDATA_SUBDIR)

TEST_LINUX_CLKTCK_SEC = 0.01
TEST_BOOTTIME_SEC = 1_000_000_000  # FYI 2001-09-09 01:46:40 GMT, not that it matters
