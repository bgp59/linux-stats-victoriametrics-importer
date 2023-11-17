#! python3

import os
import sys

tools_py_root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
if tools_py_root not in sys.path:
    sys.path.append(tools_py_root)
