#! /usr/bin/env python3

import argparse

from lsvmi.proc_interrupts_metrics import generate_proc_interrupts_metrics_test_cases
from testutils import DEFAULT_TEST_HOSTNAME, DEFAULT_TEST_INSTANCE, lsvmi_testcases_root

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "-i",
        "--instance",
        default=DEFAULT_TEST_INSTANCE,
        help="Set test instance, default: %(default)s",
    )
    parser.add_argument(
        "-n",
        "--hostname",
        "--node",
        default=DEFAULT_TEST_HOSTNAME,
        help="Set test hostname, default: %(default)s",
    )
    parser.add_argument(
        "-o",
        "--testcases-root-dir",
        default=lsvmi_testcases_root,
        help=f"""
            Generate file(s) under this dir, use `-' for stdout. 
            Default: %(default)s
        """,
    )
    args = parser.parse_args()

    generate_proc_interrupts_metrics_test_cases(
        instance=args.instance,
        hostname=args.hostname,
        testcases_root_dir=args.testcases_root_dir,
    )
