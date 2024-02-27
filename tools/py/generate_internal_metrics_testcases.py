#! /usr/bin/env python3

import argparse

from testutils import (
    DEFAULT_TEST_HOSTNAME, 
    DEFAULT_TEST_INSTANCE, 
    lsvmi_testcases_root,
)

from lsvmi.internal_metrics import (
    testcases_sub_dir,
    generators,
)

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
            Generate files under {testcases_sub_dir!r} sub-dir
            of this dir. Use `-' for stdout. Default: %(default)s
        """,
    )
    parser.add_argument(
        "-t",
        "--testcase",
        choices=generators,
        help="Generate only a sub-set of testcases",
    )
    args = parser.parse_args()

    if args.testcase is None:
        g_list = generators.values()
    else:
        g_list = [generators[args.testcase]]
    for g in g_list:
        g(instance=args.instance, hostname=args.hostname, testcases_root_dir=args.testcases_root_dir)
