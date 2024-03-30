#! /usr/bin/env python3

import os
import sys

tools_py_root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
if tools_py_root not in sys.path:
    sys.path.append(tools_py_root)

from testutils import (
    DEFAULT_TEST_HOSTNAME,
    DEFAULT_TEST_INSTANCE,
    TEST_LINUX_CLKTCK_SEC,
    lsvmi_testcases_root,
)

from .metrics_common import (
    HOSTNAME_LABEL_NAME,
    INSTANCE_LABEL_NAME,
    b64encode_str,
    uint64_delta,
)
