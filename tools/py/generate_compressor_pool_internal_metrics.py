#! /usr/bin/env python3

import argparse

from lsvmi import compressor_pool_internal_metrics as cpim
from testutils import DEFAULT_TEST_HOSTNAME, DEFAULT_TEST_INSTANCE

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
        "--out-file",
        default=cpim.default_out_file,
        help="Output file, use `-' for stdout. default: %(default)s",
    )
    args = parser.parse_args()
    cpim.generate_compressor_pool_internal_metrics_test_cases(
        instance=args.instance,
        hostname=args.hostname,
        out_file=args.out_file,
    )
