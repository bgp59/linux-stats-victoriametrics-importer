#! /usr/bin/env python3

import argparse

from lsvmi.internal_metrics import generators as internal_metrics_generators
from lsvmi.proc_interrupts_metrics import generate_proc_interrupts_metrics_test_cases
from lsvmi.proc_net_dev_metrics import generate_proc_net_dev_metrics_test_cases
from lsvmi.proc_net_snmp_metrics import generate_proc_net_snmp_metrics_test_cases
from lsvmi.proc_softirqs_metrics import generate_proc_softirqs_metrics_test_cases
from lsvmi.proc_stat_metrics import generate_proc_stat_metrics_test_cases
from testutils import DEFAULT_TEST_HOSTNAME, DEFAULT_TEST_INSTANCE, lsvmi_testcases_root

testcase_generator_fn_map = {
    "proc_interrupts": generate_proc_interrupts_metrics_test_cases,
    "proc_net_dev": generate_proc_net_dev_metrics_test_cases,
    "proc_net_snmp": generate_proc_net_snmp_metrics_test_cases,
    "proc_softirqs": generate_proc_softirqs_metrics_test_cases,
    "proc_stat": generate_proc_stat_metrics_test_cases,
}

testcase_generator_fn_map.update(internal_metrics_generators)

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "-t",
        "--target-metrics",
        choices=testcase_generator_fn_map,
        action="append",
        help="Target for test cases, default all",
    )
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

    target_metrics = args.target_metrics
    if not target_metrics:
        target_metrics = testcase_generator_fn_map.keys()

    for t in target_metrics:
        testcase_generator_fn_map[t](
            hostname=args.hostname,
            testcases_root_dir=args.testcases_root_dir,
        )
