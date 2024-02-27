#! /usr/bin/env python3

# Testcase generators for internal metrics:

# InternaMetricsTestCase fields:
TC_NAME_FIELD = "Name"
TC_INSTANCE_FIELD = "Instance"
TC_HOSTNAME_FIELD = "Hostname"
TC_PROM_TS_FIELD = "PromTs"
TC_FULL_CYCLE_FIELD = "FullCycle"
TC_WANT_METRICS_COUNT_FIELD = "WantMetricsCount"
TC_WANT_METRICS_FIELD = "WantMetrics"
TC_REPORT_EXTRA_FIELD = "ReportExtra"
# ... InternaMetricsTestCase fields:
TC_STATS_FIELD = "Stats"

testcases_sub_dir = "internal_metrics"

from .compressor_pool_internal_metrics import (
    generate_compressor_pool_internal_metrics_test_cases,
)
from .http_endpoint_pool_internal_metrics import (
    generate_http_endpoint_pool_internal_metrics_test_cases,
)
from .scheduler_internal_metrics import generate_scheduler_internal_metrics_test_cases

generators = {
    "compressor_pool": generate_compressor_pool_internal_metrics_test_cases,
    "http_endpoint_pool": generate_http_endpoint_pool_internal_metrics_test_cases,
    "scheduler": generate_scheduler_internal_metrics_test_cases,
}
