#! python3

import os

TESTDATA_SUBDIR = "testdata"
LSVMI_TESTDATA_SUBDIR = f"{TESTDATA_SUBDIR}/lsvmi"
LSVMI_TESTCASES_SUBDIR = f"{LSVMI_TESTDATA_SUBDIR}/testcases"
LSVMI_PROCFS_ROOT_SUBDIR = f"{LSVMI_TESTDATA_SUBDIR}/proc"

DEFAULT_TEST_INSTANCE = "lsvmi_test"
DEFAULT_TEST_HOSTNAME = "lsvmi-test"


PROCFS_TESTDATA_ROOT_SUBDIR = f"{TESTDATA_SUBDIR}/procfs"

BENCHMARKS_SUBDIR = "benchmarks"

tools_py_root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
go_module_root = os.path.dirname(os.path.dirname((tools_py_root)))
lsvmi_procfs_root = os.path.join(go_module_root, LSVMI_PROCFS_ROOT_SUBDIR)
lsvmi_test_cases_root_dir = os.path.join(go_module_root, LSVMI_TESTCASES_SUBDIR)
benchmarks_dir = os.path.join(go_module_root, BENCHMARKS_SUBDIR)

procfs_testdata_root = os.path.join(go_module_root, PROCFS_TESTDATA_ROOT_SUBDIR)

TEST_LINUX_CLKTCK_SEC = 0.01
