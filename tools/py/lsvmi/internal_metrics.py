#! /usr/bin/env python3

# Testcase generators for internal metrics:

generators = dict()

from .internal_metrics_common import testcases_sub_dir
from .http_endpoint_pool_internal_metrics import generate_http_endpoint_pool_internal_metrics_test_cases
generators["http_endpoint_pool"] = generate_http_endpoint_pool_internal_metrics_test_cases

