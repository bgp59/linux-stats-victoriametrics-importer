#! /usr/bin/env python3

# JSON serialize-able Compressor[Pool]Stats et al, matching lsvmi/http_endpoint_pool.go

from dataclasses import dataclass, field
from typing import Dict, List, Optional

HTTP_ENDPOINT_STATE_IN_HEALTH_CHECK = 0
HTTP_ENDPOINT_STATE_HEALTHY = 1
HTTP_ENDPOINT_STATE_AT_HEAD = 2

endpoint_state_name_map = {
    HTTP_ENDPOINT_STATE_IN_HEALTH_CHECK: "HealthCheck",
    HTTP_ENDPOINT_STATE_HEALTHY: "Healthy",
    HTTP_ENDPOINT_STATE_AT_HEAD: "AtHead",
}

HTTP_ENDPOINT_STATS_SEND_BUFFER_COUNT = 0
HTTP_ENDPOINT_STATS_SEND_BUFFER_BYTE_COUNT = 1
HTTP_ENDPOINT_STATS_SEND_BUFFER_ERROR_COUNT = 2
HTTP_ENDPOINT_STATS_HEALTH_CHECK_COUNT = 3
HTTP_ENDPOINT_STATS_HEALTH_CHECK_ERROR_COUNT = 4
HTTP_ENDPOINT_STATS_STATE = 5
HTTP_ENDPOINT_STATS_LEN = 6

HTTP_ENDPOINT_POOL_STATS_HEALTHY_ROTATE_COUNT = 0
HTTP_ENDPOINT_POOL_STATS_NO_HEALTHY_EP_ERROR_COUNT = 1
HTTP_ENDPOINT_POOL_STATS_LEN = 2

HttpEndpointStats = List[int]


@dataclass
class HttpEndpointPoolStats:
    PoolStats: List[int] = field(
        default_factory=lambda: [0] * HTTP_ENDPOINT_POOL_STATS_LEN
    )
    EndpointStats: Optional[Dict[str, HttpEndpointStats]] = None
