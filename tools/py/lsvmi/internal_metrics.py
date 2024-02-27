#! /usr/bin/env python3

# Testcase generators for internal metrics:


from .compressor_pool_internal_metrics import (
    generate_compressor_pool_internal_metrics_test_cases,
)
from .http_endpoint_pool_internal_metrics import (
    generate_http_endpoint_pool_internal_metrics_test_cases,
)
from .scheduler_internal_metrics import generate_scheduler_internal_metrics_test_cases

testcases_sub_dir = "internal_metrics"

generators = {
    "compressor_pool": generate_compressor_pool_internal_metrics_test_cases,
    "http_endpoint_pool": generate_http_endpoint_pool_internal_metrics_test_cases,
    "scheduler": generate_scheduler_internal_metrics_test_cases,
}
