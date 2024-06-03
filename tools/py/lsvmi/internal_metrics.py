#! /usr/bin/env python3

# Testcase generators for internal metrics:
from dataclasses import dataclass
from typing import List, Optional

from . import DEFAULT_TEST_HOSTNAME, DEFAULT_TEST_INSTANCE


@dataclass
class InternalMetricsTestCase:
    Name: Optional[str] = None
    Description: Optional[str] = None
    Instance: str = DEFAULT_TEST_INSTANCE
    Hostname: str = DEFAULT_TEST_HOSTNAME
    PromTs: int = 0
    WantMetricsCount: int = 0
    WantMetrics: Optional[List[str]] = None
    ReportExtra: bool = True


test_cases_sub_dir = "internal_metrics"

from .compressor_pool_internal_metrics import (
    generate_compressor_pool_internal_metrics_test_cases,
)
from .http_endpoint_pool_internal_metrics import (
    generate_http_endpoint_pool_internal_metrics_test_cases,
)

# from .scheduler_internal_metrics import generate_scheduler_internal_metrics_test_cases

generators = {
    "compressor_pool": generate_compressor_pool_internal_metrics_test_cases,
    "http_endpoint_pool": generate_http_endpoint_pool_internal_metrics_test_cases,
    # "scheduler": generate_scheduler_internal_metrics_test_cases,
}
