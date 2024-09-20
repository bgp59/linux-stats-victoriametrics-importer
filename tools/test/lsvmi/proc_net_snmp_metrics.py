#! /usr/bin/env python3

# Generate test cases for lsvmi/proc_net_snmp_metrics_test.go

import time
from copy import deepcopy
from dataclasses import dataclass
from typing import List, Optional, Tuple

import procfs

from . import (
    DEFAULT_TEST_HOSTNAME,
    DEFAULT_TEST_INSTANCE,
    HOSTNAME_LABEL_NAME,
    INSTANCE_LABEL_NAME,
    int32_to_uint32,
    lsvmi_test_cases_root_dir,
    save_test_cases,
    uint32_delta,
    uint32_to_int32,
)

DEFAULT_PROC_NET_SNMP_INTERVAL_SEC = 1
DEFAULT_PROC_NET_SNMP_FULL_METRICS_FACTOR = 15


# Metrics definitions, must match lsvmi/proc_net_snmp_metrics.go:
PROC_NET_SNMP_IP_FORWARDING_METRIC = "proc_net_snmp_ip_forwarding"
PROC_NET_SNMP_IP_DEFAULT_TTL_METRIC = "proc_net_snmp_ip_default_ttl"
PROC_NET_SNMP_IP_IN_RECEIVES_DELTA_METRIC = "proc_net_snmp_ip_in_receives_delta"
PROC_NET_SNMP_IP_IN_HDR_ERRORS_DELTA_METRIC = "proc_net_snmp_ip_in_hdr_errors_delta"
PROC_NET_SNMP_IP_IN_ADDR_ERRORS_DELTA_METRIC = "proc_net_snmp_ip_in_addr_errors_delta"
PROC_NET_SNMP_IP_FORW_DATAGRAMS_DELTA_METRIC = "proc_net_snmp_ip_forw_datagrams_delta"
PROC_NET_SNMP_IP_IN_UNKNOWN_PROTOS_DELTA_METRIC = (
    "proc_net_snmp_ip_in_unknown_protos_delta"
)
PROC_NET_SNMP_IP_IN_DISCARDS_DELTA_METRIC = "proc_net_snmp_ip_in_discards_delta"
PROC_NET_SNMP_IP_IN_DELIVERS_DELTA_METRIC = "proc_net_snmp_ip_in_delivers_delta"
PROC_NET_SNMP_IP_OUT_REQUESTS_DELTA_METRIC = "proc_net_snmp_ip_out_requests_delta"
PROC_NET_SNMP_IP_OUT_DISCARDS_DELTA_METRIC = "proc_net_snmp_ip_out_discards_delta"
PROC_NET_SNMP_IP_OUT_NO_ROUTES_DELTA_METRIC = "proc_net_snmp_ip_out_no_routes_delta"
PROC_NET_SNMP_IP_REASM_TIMEOUT_METRIC = "proc_net_snmp_ip_reasm_timeout"
PROC_NET_SNMP_IP_REASM_REQDS_DELTA_METRIC = "proc_net_snmp_ip_reasm_reqds_delta"
PROC_NET_SNMP_IP_REASM_OKS_DELTA_METRIC = "proc_net_snmp_ip_reasm_oks_delta"
PROC_NET_SNMP_IP_REASM_FAILS_DELTA_METRIC = "proc_net_snmp_ip_reasm_fails_delta"
PROC_NET_SNMP_IP_FRAG_OKS_DELTA_METRIC = "proc_net_snmp_ip_frag_oks_delta"
PROC_NET_SNMP_IP_FRAG_FAILS_DELTA_METRIC = "proc_net_snmp_ip_frag_fails_delta"
PROC_NET_SNMP_IP_FRAG_CREATES_DELTA_METRIC = "proc_net_snmp_ip_frag_creates_delta"
PROC_NET_SNMP_ICMP_IN_MSGS_DELTA_METRIC = "proc_net_snmp_icmp_in_msgs_delta"
PROC_NET_SNMP_ICMP_IN_ERRORS_DELTA_METRIC = "proc_net_snmp_icmp_in_errors_delta"
PROC_NET_SNMP_ICMP_IN_CSUM_ERRORS_DELTA_METRIC = (
    "proc_net_snmp_icmp_in_csum_errors_delta"
)
PROC_NET_SNMP_ICMP_IN_DEST_UNREACHS_DELTA_METRIC = (
    "proc_net_snmp_icmp_in_dest_unreachs_delta"
)
PROC_NET_SNMP_ICMP_IN_TIME_EXCDS_DELTA_METRIC = "proc_net_snmp_icmp_in_time_excds_delta"
PROC_NET_SNMP_ICMP_IN_PARM_PROBS_DELTA_METRIC = "proc_net_snmp_icmp_in_parm_probs_delta"
PROC_NET_SNMP_ICMP_IN_SRC_QUENCHS_DELTA_METRIC = (
    "proc_net_snmp_icmp_in_src_quenchs_delta"
)
PROC_NET_SNMP_ICMP_IN_REDIRECTS_DELTA_METRIC = "proc_net_snmp_icmp_in_redirects_delta"
PROC_NET_SNMP_ICMP_IN_ECHOS_DELTA_METRIC = "proc_net_snmp_icmp_in_echos_delta"
PROC_NET_SNMP_ICMP_IN_ECHO_REPS_DELTA_METRIC = "proc_net_snmp_icmp_in_echo_reps_delta"
PROC_NET_SNMP_ICMP_IN_TIMESTAMPS_DELTA_METRIC = "proc_net_snmp_icmp_in_timestamps_delta"
PROC_NET_SNMP_ICMP_IN_TIMESTAMP_REPS_DELTA_METRIC = (
    "proc_net_snmp_icmp_in_timestamp_reps_delta"
)
PROC_NET_SNMP_ICMP_IN_ADDR_MASKS_DELTA_METRIC = "proc_net_snmp_icmp_in_addr_masks_delta"
PROC_NET_SNMP_ICMP_IN_ADDR_MASK_REPS_DELTA_METRIC = (
    "proc_net_snmp_icmp_in_addr_mask_reps_delta"
)
PROC_NET_SNMP_ICMP_OUT_MSGS_DELTA_METRIC = "proc_net_snmp_icmp_out_msgs_delta"
PROC_NET_SNMP_ICMP_OUT_ERRORS_DELTA_METRIC = "proc_net_snmp_icmp_out_errors_delta"
PROC_NET_SNMP_ICMP_OUT_DEST_UNREACHS_DELTA_METRIC = (
    "proc_net_snmp_icmp_out_dest_unreachs_delta"
)
PROC_NET_SNMP_ICMP_OUT_TIME_EXCDS_DELTA_METRIC = (
    "proc_net_snmp_icmp_out_time_excds_delta"
)
PROC_NET_SNMP_ICMP_OUT_PARM_PROBS_DELTA_METRIC = (
    "proc_net_snmp_icmp_out_parm_probs_delta"
)
PROC_NET_SNMP_ICMP_OUT_SRC_QUENCHS_DELTA_METRIC = (
    "proc_net_snmp_icmp_out_src_quenchs_delta"
)
PROC_NET_SNMP_ICMP_OUT_REDIRECTS_DELTA_METRIC = "proc_net_snmp_icmp_out_redirects_delta"
PROC_NET_SNMP_ICMP_OUT_ECHOS_DELTA_METRIC = "proc_net_snmp_icmp_out_echos_delta"
PROC_NET_SNMP_ICMP_OUT_ECHO_REPS_DELTA_METRIC = "proc_net_snmp_icmp_out_echo_reps_delta"
PROC_NET_SNMP_ICMP_OUT_TIMESTAMPS_DELTA_METRIC = (
    "proc_net_snmp_icmp_out_timestamps_delta"
)
PROC_NET_SNMP_ICMP_OUT_TIMESTAMP_REPS_DELTA_METRIC = (
    "proc_net_snmp_icmp_out_timestamp_reps_delta"
)
PROC_NET_SNMP_ICMP_OUT_ADDR_MASKS_DELTA_METRIC = (
    "proc_net_snmp_icmp_out_addr_masks_delta"
)
PROC_NET_SNMP_ICMP_OUT_ADDR_MASK_REPS_DELTA_METRIC = (
    "proc_net_snmp_icmp_out_addr_mask_reps_delta"
)
PROC_NET_SNMP_ICMPMSG_IN_TYPE3_DELTA_METRIC = "proc_net_snmp_icmpmsg_in_type3_delta"
PROC_NET_SNMP_ICMPMSG_OUT_TYPE3_DELTA_METRIC = "proc_net_snmp_icmpmsg_out_type3_delta"
PROC_NET_SNMP_TCP_RTO_ALGORITHM_METRIC = "proc_net_snmp_tcp_rto_algorithm"
PROC_NET_SNMP_TCP_RTO_MIN_METRIC = "proc_net_snmp_tcp_rto_min"
PROC_NET_SNMP_TCP_RTO_MAX_METRIC = "proc_net_snmp_tcp_rto_max"
PROC_NET_SNMP_TCP_MAX_CONN_METRIC = "proc_net_snmp_tcp_max_conn"
PROC_NET_SNMP_TCP_ACTIVE_OPENS_DELTA_METRIC = "proc_net_snmp_tcp_active_opens_delta"
PROC_NET_SNMP_TCP_PASSIVE_OPENS_DELTA_METRIC = "proc_net_snmp_tcp_passive_opens_delta"
PROC_NET_SNMP_TCP_ATTEMPT_FAILS_DELTA_METRIC = "proc_net_snmp_tcp_attempt_fails_delta"
PROC_NET_SNMP_TCP_ESTAB_RESETS_DELTA_METRIC = "proc_net_snmp_tcp_estab_resets_delta"
PROC_NET_SNMP_TCP_CURR_ESTAB_METRIC = "proc_net_snmp_tcp_curr_estab"
PROC_NET_SNMP_TCP_IN_SEGS_DELTA_METRIC = "proc_net_snmp_tcp_in_segs_delta"
PROC_NET_SNMP_TCP_OUT_SEGS_DELTA_METRIC = "proc_net_snmp_tcp_out_segs_delta"
PROC_NET_SNMP_TCP_RETRANS_SEGS_DELTA_METRIC = "proc_net_snmp_tcp_retrans_segs_delta"
PROC_NET_SNMP_TCP_IN_ERRS_DELTA_METRIC = "proc_net_snmp_tcp_in_errs_delta"
PROC_NET_SNMP_TCP_OUT_RSTS_DELTA_METRIC = "proc_net_snmp_tcp_out_rsts_delta"
PROC_NET_SNMP_TCP_IN_CSUM_ERRORS_DELTA_METRIC = "proc_net_snmp_tcp_in_csum_errors_delta"
PROC_NET_SNMP_UDP_IN_DATAGRAMS_DELTA_METRIC = "proc_net_snmp_udp_in_datagrams_delta"
PROC_NET_SNMP_UDP_NO_PORTS_DELTA_METRIC = "proc_net_snmp_udp_no_ports_delta"
PROC_NET_SNMP_UDP_IN_ERRORS_DELTA_METRIC = "proc_net_snmp_udp_in_errors_delta"
PROC_NET_SNMP_UDP_OUT_DATAGRAMS_DELTA_METRIC = "proc_net_snmp_udp_out_datagrams_delta"
PROC_NET_SNMP_UDP_RCVBUF_ERRORS_DELTA_METRIC = "proc_net_snmp_udp_rcvbuf_errors_delta"
PROC_NET_SNMP_UDP_SNDBUF_ERRORS_DELTA_METRIC = "proc_net_snmp_udp_sndbuf_errors_delta"
PROC_NET_SNMP_UDP_IN_CSUM_ERRORS_DELTA_METRIC = "proc_net_snmp_udp_in_csum_errors_delta"
PROC_NET_SNMP_UDP_IGNORED_MULTI_DELTA_METRIC = "proc_net_snmp_udp_ignored_multi_delta"
PROC_NET_SNMP_UDP_MEM_ERRORS_DELTA_METRIC = "proc_net_snmp_udp_mem_errors_delta"
PROC_NET_SNMP_UDPLITE_IN_DATAGRAMS_DELTA_METRIC = (
    "proc_net_snmp_udplite_in_datagrams_delta"
)
PROC_NET_SNMP_UDPLITE_NO_PORTS_DELTA_METRIC = "proc_net_snmp_udplite_no_ports_delta"
PROC_NET_SNMP_UDPLITE_IN_ERRORS_DELTA_METRIC = "proc_net_snmp_udplite_in_errors_delta"
PROC_NET_SNMP_UDPLITE_OUT_DATAGRAMS_DELTA_METRIC = (
    "proc_net_snmp_udplite_out_datagrams_delta"
)
PROC_NET_SNMP_UDPLITE_RCVBUF_ERRORS_DELTA_METRIC = (
    "proc_net_snmp_udplite_rcvbuf_errors_delta"
)
PROC_NET_SNMP_UDPLITE_SNDBUF_ERRORS_DELTA_METRIC = (
    "proc_net_snmp_udplite_sndbuf_errors_delta"
)
PROC_NET_SNMP_UDPLITE_IN_CSUM_ERRORS_DELTA_METRIC = (
    "proc_net_snmp_udplite_in_csum_errors_delta"
)
PROC_NET_SNMP_UDPLITE_IGNORED_MULTI_DELTA_METRIC = (
    "proc_net_snmp_udplite_ignored_multi_delta"
)
PROC_NET_SNMP_UDPLITE_MEM_ERRORS_DELTA_METRIC = "proc_net_snmp_udplite_mem_errors_delta"

PROC_NET_SNMP_INTERVAL_METRIC = "proc_net_snmp_metrics_delta_sec"

PROC_NET_SNMP_CYCLE_COUNTER_EXP = 4
PROC_NET_SNMP_CYCLE_COUNTER_NUM = 1 << PROC_NET_SNMP_CYCLE_COUNTER_EXP
PROC_NET_SNMP_CYCLE_COUNTER_MASK = PROC_NET_SNMP_CYCLE_COUNTER_NUM - 1

proc_net_snmp_index_to_metric_name = {
    procfs.NET_SNMP_IP_FORWARDING: PROC_NET_SNMP_IP_FORWARDING_METRIC,
    procfs.NET_SNMP_IP_DEFAULT_TTL: PROC_NET_SNMP_IP_DEFAULT_TTL_METRIC,
    procfs.NET_SNMP_IP_IN_RECEIVES: PROC_NET_SNMP_IP_IN_RECEIVES_DELTA_METRIC,
    procfs.NET_SNMP_IP_IN_HDR_ERRORS: PROC_NET_SNMP_IP_IN_HDR_ERRORS_DELTA_METRIC,
    procfs.NET_SNMP_IP_IN_ADDR_ERRORS: PROC_NET_SNMP_IP_IN_ADDR_ERRORS_DELTA_METRIC,
    procfs.NET_SNMP_IP_FORW_DATAGRAMS: PROC_NET_SNMP_IP_FORW_DATAGRAMS_DELTA_METRIC,
    procfs.NET_SNMP_IP_IN_UNKNOWN_PROTOS: PROC_NET_SNMP_IP_IN_UNKNOWN_PROTOS_DELTA_METRIC,
    procfs.NET_SNMP_IP_IN_DISCARDS: PROC_NET_SNMP_IP_IN_DISCARDS_DELTA_METRIC,
    procfs.NET_SNMP_IP_IN_DELIVERS: PROC_NET_SNMP_IP_IN_DELIVERS_DELTA_METRIC,
    procfs.NET_SNMP_IP_OUT_REQUESTS: PROC_NET_SNMP_IP_OUT_REQUESTS_DELTA_METRIC,
    procfs.NET_SNMP_IP_OUT_DISCARDS: PROC_NET_SNMP_IP_OUT_DISCARDS_DELTA_METRIC,
    procfs.NET_SNMP_IP_OUT_NO_ROUTES: PROC_NET_SNMP_IP_OUT_NO_ROUTES_DELTA_METRIC,
    procfs.NET_SNMP_IP_REASM_TIMEOUT: PROC_NET_SNMP_IP_REASM_TIMEOUT_METRIC,
    procfs.NET_SNMP_IP_REASM_REQDS: PROC_NET_SNMP_IP_REASM_REQDS_DELTA_METRIC,
    procfs.NET_SNMP_IP_REASM_OKS: PROC_NET_SNMP_IP_REASM_OKS_DELTA_METRIC,
    procfs.NET_SNMP_IP_REASM_FAILS: PROC_NET_SNMP_IP_REASM_FAILS_DELTA_METRIC,
    procfs.NET_SNMP_IP_FRAG_OKS: PROC_NET_SNMP_IP_FRAG_OKS_DELTA_METRIC,
    procfs.NET_SNMP_IP_FRAG_FAILS: PROC_NET_SNMP_IP_FRAG_FAILS_DELTA_METRIC,
    procfs.NET_SNMP_IP_FRAG_CREATES: PROC_NET_SNMP_IP_FRAG_CREATES_DELTA_METRIC,
    procfs.NET_SNMP_ICMP_IN_MSGS: PROC_NET_SNMP_ICMP_IN_MSGS_DELTA_METRIC,
    procfs.NET_SNMP_ICMP_IN_ERRORS: PROC_NET_SNMP_ICMP_IN_ERRORS_DELTA_METRIC,
    procfs.NET_SNMP_ICMP_IN_CSUM_ERRORS: PROC_NET_SNMP_ICMP_IN_CSUM_ERRORS_DELTA_METRIC,
    procfs.NET_SNMP_ICMP_IN_DEST_UNREACHS: PROC_NET_SNMP_ICMP_IN_DEST_UNREACHS_DELTA_METRIC,
    procfs.NET_SNMP_ICMP_IN_TIME_EXCDS: PROC_NET_SNMP_ICMP_IN_TIME_EXCDS_DELTA_METRIC,
    procfs.NET_SNMP_ICMP_IN_PARM_PROBS: PROC_NET_SNMP_ICMP_IN_PARM_PROBS_DELTA_METRIC,
    procfs.NET_SNMP_ICMP_IN_SRC_QUENCHS: PROC_NET_SNMP_ICMP_IN_SRC_QUENCHS_DELTA_METRIC,
    procfs.NET_SNMP_ICMP_IN_REDIRECTS: PROC_NET_SNMP_ICMP_IN_REDIRECTS_DELTA_METRIC,
    procfs.NET_SNMP_ICMP_IN_ECHOS: PROC_NET_SNMP_ICMP_IN_ECHOS_DELTA_METRIC,
    procfs.NET_SNMP_ICMP_IN_ECHO_REPS: PROC_NET_SNMP_ICMP_IN_ECHO_REPS_DELTA_METRIC,
    procfs.NET_SNMP_ICMP_IN_TIMESTAMPS: PROC_NET_SNMP_ICMP_IN_TIMESTAMPS_DELTA_METRIC,
    procfs.NET_SNMP_ICMP_IN_TIMESTAMP_REPS: PROC_NET_SNMP_ICMP_IN_TIMESTAMP_REPS_DELTA_METRIC,
    procfs.NET_SNMP_ICMP_IN_ADDR_MASKS: PROC_NET_SNMP_ICMP_IN_ADDR_MASKS_DELTA_METRIC,
    procfs.NET_SNMP_ICMP_IN_ADDR_MASK_REPS: PROC_NET_SNMP_ICMP_IN_ADDR_MASK_REPS_DELTA_METRIC,
    procfs.NET_SNMP_ICMP_OUT_MSGS: PROC_NET_SNMP_ICMP_OUT_MSGS_DELTA_METRIC,
    procfs.NET_SNMP_ICMP_OUT_ERRORS: PROC_NET_SNMP_ICMP_OUT_ERRORS_DELTA_METRIC,
    procfs.NET_SNMP_ICMP_OUT_DEST_UNREACHS: PROC_NET_SNMP_ICMP_OUT_DEST_UNREACHS_DELTA_METRIC,
    procfs.NET_SNMP_ICMP_OUT_TIME_EXCDS: PROC_NET_SNMP_ICMP_OUT_TIME_EXCDS_DELTA_METRIC,
    procfs.NET_SNMP_ICMP_OUT_PARM_PROBS: PROC_NET_SNMP_ICMP_OUT_PARM_PROBS_DELTA_METRIC,
    procfs.NET_SNMP_ICMP_OUT_SRC_QUENCHS: PROC_NET_SNMP_ICMP_OUT_SRC_QUENCHS_DELTA_METRIC,
    procfs.NET_SNMP_ICMP_OUT_REDIRECTS: PROC_NET_SNMP_ICMP_OUT_REDIRECTS_DELTA_METRIC,
    procfs.NET_SNMP_ICMP_OUT_ECHOS: PROC_NET_SNMP_ICMP_OUT_ECHOS_DELTA_METRIC,
    procfs.NET_SNMP_ICMP_OUT_ECHO_REPS: PROC_NET_SNMP_ICMP_OUT_ECHO_REPS_DELTA_METRIC,
    procfs.NET_SNMP_ICMP_OUT_TIMESTAMPS: PROC_NET_SNMP_ICMP_OUT_TIMESTAMPS_DELTA_METRIC,
    procfs.NET_SNMP_ICMP_OUT_TIMESTAMP_REPS: PROC_NET_SNMP_ICMP_OUT_TIMESTAMP_REPS_DELTA_METRIC,
    procfs.NET_SNMP_ICMP_OUT_ADDR_MASKS: PROC_NET_SNMP_ICMP_OUT_ADDR_MASKS_DELTA_METRIC,
    procfs.NET_SNMP_ICMP_OUT_ADDR_MASK_REPS: PROC_NET_SNMP_ICMP_OUT_ADDR_MASK_REPS_DELTA_METRIC,
    procfs.NET_SNMP_ICMPMSG_IN_TYPE3: PROC_NET_SNMP_ICMPMSG_IN_TYPE3_DELTA_METRIC,
    procfs.NET_SNMP_ICMPMSG_OUT_TYPE3: PROC_NET_SNMP_ICMPMSG_OUT_TYPE3_DELTA_METRIC,
    procfs.NET_SNMP_TCP_RTO_ALGORITHM: PROC_NET_SNMP_TCP_RTO_ALGORITHM_METRIC,
    procfs.NET_SNMP_TCP_RTO_MIN: PROC_NET_SNMP_TCP_RTO_MIN_METRIC,
    procfs.NET_SNMP_TCP_RTO_MAX: PROC_NET_SNMP_TCP_RTO_MAX_METRIC,
    procfs.NET_SNMP_TCP_MAX_CONN: PROC_NET_SNMP_TCP_MAX_CONN_METRIC,
    procfs.NET_SNMP_TCP_ACTIVE_OPENS: PROC_NET_SNMP_TCP_ACTIVE_OPENS_DELTA_METRIC,
    procfs.NET_SNMP_TCP_PASSIVE_OPENS: PROC_NET_SNMP_TCP_PASSIVE_OPENS_DELTA_METRIC,
    procfs.NET_SNMP_TCP_ATTEMPT_FAILS: PROC_NET_SNMP_TCP_ATTEMPT_FAILS_DELTA_METRIC,
    procfs.NET_SNMP_TCP_ESTAB_RESETS: PROC_NET_SNMP_TCP_ESTAB_RESETS_DELTA_METRIC,
    procfs.NET_SNMP_TCP_CURR_ESTAB: PROC_NET_SNMP_TCP_CURR_ESTAB_METRIC,
    procfs.NET_SNMP_TCP_IN_SEGS: PROC_NET_SNMP_TCP_IN_SEGS_DELTA_METRIC,
    procfs.NET_SNMP_TCP_OUT_SEGS: PROC_NET_SNMP_TCP_OUT_SEGS_DELTA_METRIC,
    procfs.NET_SNMP_TCP_RETRANS_SEGS: PROC_NET_SNMP_TCP_RETRANS_SEGS_DELTA_METRIC,
    procfs.NET_SNMP_TCP_IN_ERRS: PROC_NET_SNMP_TCP_IN_ERRS_DELTA_METRIC,
    procfs.NET_SNMP_TCP_OUT_RSTS: PROC_NET_SNMP_TCP_OUT_RSTS_DELTA_METRIC,
    procfs.NET_SNMP_TCP_IN_CSUM_ERRORS: PROC_NET_SNMP_TCP_IN_CSUM_ERRORS_DELTA_METRIC,
    procfs.NET_SNMP_UDP_IN_DATAGRAMS: PROC_NET_SNMP_UDP_IN_DATAGRAMS_DELTA_METRIC,
    procfs.NET_SNMP_UDP_NO_PORTS: PROC_NET_SNMP_UDP_NO_PORTS_DELTA_METRIC,
    procfs.NET_SNMP_UDP_IN_ERRORS: PROC_NET_SNMP_UDP_IN_ERRORS_DELTA_METRIC,
    procfs.NET_SNMP_UDP_OUT_DATAGRAMS: PROC_NET_SNMP_UDP_OUT_DATAGRAMS_DELTA_METRIC,
    procfs.NET_SNMP_UDP_RCVBUF_ERRORS: PROC_NET_SNMP_UDP_RCVBUF_ERRORS_DELTA_METRIC,
    procfs.NET_SNMP_UDP_SNDBUF_ERRORS: PROC_NET_SNMP_UDP_SNDBUF_ERRORS_DELTA_METRIC,
    procfs.NET_SNMP_UDP_IN_CSUM_ERRORS: PROC_NET_SNMP_UDP_IN_CSUM_ERRORS_DELTA_METRIC,
    procfs.NET_SNMP_UDP_IGNORED_MULTI: PROC_NET_SNMP_UDP_IGNORED_MULTI_DELTA_METRIC,
    procfs.NET_SNMP_UDP_MEM_ERRORS: PROC_NET_SNMP_UDP_MEM_ERRORS_DELTA_METRIC,
    procfs.NET_SNMP_UDPLITE_IN_DATAGRAMS: PROC_NET_SNMP_UDPLITE_IN_DATAGRAMS_DELTA_METRIC,
    procfs.NET_SNMP_UDPLITE_NO_PORTS: PROC_NET_SNMP_UDPLITE_NO_PORTS_DELTA_METRIC,
    procfs.NET_SNMP_UDPLITE_IN_ERRORS: PROC_NET_SNMP_UDPLITE_IN_ERRORS_DELTA_METRIC,
    procfs.NET_SNMP_UDPLITE_OUT_DATAGRAMS: PROC_NET_SNMP_UDPLITE_OUT_DATAGRAMS_DELTA_METRIC,
    procfs.NET_SNMP_UDPLITE_RCVBUF_ERRORS: PROC_NET_SNMP_UDPLITE_RCVBUF_ERRORS_DELTA_METRIC,
    procfs.NET_SNMP_UDPLITE_SNDBUF_ERRORS: PROC_NET_SNMP_UDPLITE_SNDBUF_ERRORS_DELTA_METRIC,
    procfs.NET_SNMP_UDPLITE_IN_CSUM_ERRORS: PROC_NET_SNMP_UDPLITE_IN_CSUM_ERRORS_DELTA_METRIC,
    procfs.NET_SNMP_UDPLITE_IGNORED_MULTI: PROC_NET_SNMP_UDPLITE_IGNORED_MULTI_DELTA_METRIC,
    procfs.NET_SNMP_UDPLITE_MEM_ERRORS: PROC_NET_SNMP_UDPLITE_MEM_ERRORS_DELTA_METRIC,
}

proc_net_snmp_non_delta_index = set(
    [
        procfs.NET_SNMP_IP_DEFAULT_TTL,
        procfs.NET_SNMP_IP_FORWARDING,
        procfs.NET_SNMP_IP_REASM_TIMEOUT,
        procfs.NET_SNMP_TCP_CURR_ESTAB,
        procfs.NET_SNMP_TCP_MAX_CONN,
        procfs.NET_SNMP_TCP_RTO_ALGORITHM,
        procfs.NET_SNMP_TCP_RTO_MAX,
        procfs.NET_SNMP_TCP_RTO_MIN,
    ]
)


@dataclass
class ProcNetSnmpMetricsTestCase:
    Name: Optional[str] = None
    Description: Optional[str] = None
    Instance: Optional[str] = None
    Hostname: Optional[str] = None
    CurrProcNetSnmp: Optional[procfs.NetSnmp] = None
    PrevProcNetSnmp: Optional[procfs.NetSnmp] = None
    CurrPromTs: int = 0
    PrevPromTs: int = 0
    CycleNum: Optional[List[int]] = None
    FullMetricsFactor: int = DEFAULT_PROC_NET_SNMP_FULL_METRICS_FACTOR
    ZeroDelta: Optional[List[bool]] = None
    WantMetricsCount: int = 0
    WantMetrics: Optional[List[str]] = None
    ReportExtra: bool = False
    WantZeroDelta: Optional[List[bool]] = None


test_cases_file = "proc_net_snmp.json"


def generate_proc_net_snmp_metrics(
    curr_proc_net_snmp: procfs.NetSnmp,
    curr_prom_ts: int,
    prev_proc_net_snmp: Optional[procfs.NetSnmp] = None,
    cycle_num: Optional[List[int]] = None,
    zero_delta: Optional[List[bool]] = None,
    interval: float = DEFAULT_PROC_NET_SNMP_INTERVAL_SEC,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
) -> Tuple[List[str], Optional[List[bool]]]:
    metrics = []
    new_zero_delta = (
        None if prev_proc_net_snmp is None else [False] * procfs.NET_SNMP_NUM_VALUES
    )

    for i, curr_value in enumerate(curr_proc_net_snmp.Values):
        name = proc_net_snmp_index_to_metric_name.get(i)
        if name is None:
            continue
        full_metrics = (
            cycle_num is None or cycle_num[i & PROC_NET_SNMP_CYCLE_COUNTER_MASK] == 0
        )
        metric_val = None
        if i in proc_net_snmp_non_delta_index:
            if (
                full_metrics
                or prev_proc_net_snmp is None
                or curr_value != prev_proc_net_snmp.Values[i]
            ):
                metric_val = curr_value
                if i in procfs.NetSnmpValueMayBeNegative:
                    metric_val = uint32_to_int32(metric_val)

        elif prev_proc_net_snmp is not None:
            delta = uint32_delta(curr_value, prev_proc_net_snmp.Values[i])
            if full_metrics or delta != 0 or zero_delta is None or not zero_delta[i]:
                metric_val = delta
            if new_zero_delta is not None:
                new_zero_delta[i] = delta == 0
        if metric_val is not None:
            metrics.append(
                f"{name}{{"
                + ",".join(
                    [
                        f'{INSTANCE_LABEL_NAME}="{instance}"',
                        f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                    ]
                )
                + f"}} {metric_val} {curr_prom_ts}"
            )

    if prev_proc_net_snmp is not None:
        metrics.append(
            f"{PROC_NET_SNMP_INTERVAL_METRIC}{{"
            + ",".join(
                [
                    f'{INSTANCE_LABEL_NAME}="{instance}"',
                    f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                ]
            )
            + f"}} {interval:.06f} {curr_prom_ts}"
        )

    return metrics, new_zero_delta


def generate_proc_net_snmp_test_case(
    name: str,
    curr_proc_net_snmp: procfs.NetSnmp,
    ts: Optional[float] = None,
    prev_proc_net_snmp: Optional[procfs.NetSnmp] = None,
    cycle_num: Optional[List[int]] = None,
    zero_delta: Optional[List[bool]] = None,
    interval: float = DEFAULT_PROC_NET_SNMP_INTERVAL_SEC,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    full_metrics_factor: int = DEFAULT_PROC_NET_SNMP_FULL_METRICS_FACTOR,
    description: Optional[str] = None,
) -> ProcNetSnmpMetricsTestCase:
    if ts is None:
        ts = time.time()
    curr_prom_ts = int(ts * 1000)
    prev_prom_ts = curr_prom_ts - int(interval * 1000)
    metrics, want_zero_delta = generate_proc_net_snmp_metrics(
        curr_proc_net_snmp,
        curr_prom_ts=curr_prom_ts,
        prev_proc_net_snmp=prev_proc_net_snmp,
        cycle_num=cycle_num,
        zero_delta=zero_delta,
        interval=interval,
        instance=instance,
        hostname=hostname,
    )
    return ProcNetSnmpMetricsTestCase(
        Name=name,
        Description=description,
        Instance=instance,
        Hostname=hostname,
        CurrProcNetSnmp=curr_proc_net_snmp,
        PrevProcNetSnmp=prev_proc_net_snmp,
        CurrPromTs=curr_prom_ts,
        PrevPromTs=prev_prom_ts,
        CycleNum=cycle_num,
        FullMetricsFactor=full_metrics_factor,
        ZeroDelta=zero_delta,
        WantMetricsCount=len(metrics),
        WantMetrics=metrics,
        ReportExtra=True,
        WantZeroDelta=want_zero_delta,
    )


def make_ref_proc_net_snmp() -> procfs.NetSnmp:
    proc_net_snmp = procfs.NetSnmp()
    for i in range(len(proc_net_snmp.Values)):
        v = i + 13
        if i in procfs.NetSnmpValueMayBeNegative:
            v = int32_to_uint32(-v)
        proc_net_snmp.Values[i] = v
    return proc_net_snmp


def make_zero_delta(val: bool = False) -> List[bool]:
    return [
        False if i in proc_net_snmp_non_delta_index else val
        for i in range(procfs.NET_SNMP_NUM_VALUES)
    ]


def generate_proc_net_snmp_metrics_test_cases(
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    test_cases_root_dir: Optional[str] = lsvmi_test_cases_root_dir,
):
    test_cases = []
    tc_num = 0

    ref_proc_net_snmp = make_ref_proc_net_snmp()

    name = "no_prev"
    for cycle_num_val in [0, 1]:
        for zero_delta_val in [None, False, True]:
            cycle_num = [cycle_num_val] * PROC_NET_SNMP_CYCLE_COUNTER_NUM
            zero_delta = (
                None if zero_delta_val is None else make_zero_delta(zero_delta_val)
            )
            test_cases.append(
                generate_proc_net_snmp_test_case(
                    f"{name}/{tc_num}",
                    curr_proc_net_snmp=deepcopy(ref_proc_net_snmp),
                    cycle_num=cycle_num,
                    zero_delta=zero_delta,
                    description=f"cycle_num={cycle_num_val}, zero_delta={zero_delta_val}",
                )
            )
            tc_num += 1

    name = "all_change"
    curr_proc_net_snmp = ref_proc_net_snmp
    max_val = max(
        curr_proc_net_snmp.Values[i]
        for i in range(procfs.NET_SNMP_NUM_VALUES)
        if i not in procfs.NetSnmpValueMayBeNegative
    )
    prev_proc_net_snmp = procfs.NetSnmp()
    for i, curr_val in enumerate(curr_proc_net_snmp.Values):
        if i in proc_net_snmp_non_delta_index:
            prev_proc_net_snmp.Values[i] = curr_val + 1
        else:
            prev_proc_net_snmp.Values[i] = uint32_delta(curr_val, max_val + 2 * i)
    for cycle_num_val in [0, 1]:
        for zero_delta_val in [None, False, True]:
            cycle_num = [cycle_num_val] * PROC_NET_SNMP_CYCLE_COUNTER_NUM
            zero_delta = (
                None if zero_delta_val is None else make_zero_delta(zero_delta_val)
            )
            test_cases.append(
                generate_proc_net_snmp_test_case(
                    f"{name}/{tc_num}",
                    curr_proc_net_snmp=curr_proc_net_snmp,
                    prev_proc_net_snmp=prev_proc_net_snmp,
                    cycle_num=cycle_num,
                    zero_delta=zero_delta,
                    description=f"cycle_num={cycle_num_val}, zero_delta={zero_delta_val}",
                )
            )
            tc_num += 1

    name = "no_change"
    for cycle_num_val in [0, 1]:
        for zero_delta_val in [None, False, True]:
            cycle_num = [cycle_num_val] * PROC_NET_SNMP_CYCLE_COUNTER_NUM
            zero_delta = (
                None if zero_delta_val is None else make_zero_delta(zero_delta_val)
            )
            test_cases.append(
                generate_proc_net_snmp_test_case(
                    f"{name}/{tc_num}",
                    curr_proc_net_snmp=ref_proc_net_snmp,
                    prev_proc_net_snmp=ref_proc_net_snmp,
                    cycle_num=cycle_num,
                    zero_delta=zero_delta,
                    description=f"cycle_num={cycle_num_val}, zero_delta={zero_delta_val}",
                )
            )
            tc_num += 1

    name = "single_change"
    curr_proc_net_snmp = ref_proc_net_snmp
    max_val = max(
        curr_proc_net_snmp.Values[i]
        for i in range(procfs.NET_SNMP_NUM_VALUES)
        if i not in procfs.NetSnmpValueMayBeNegative
    )
    for cycle_num_val in [0, 1]:
        for zero_delta_val in [None, False, True]:
            cycle_num = [cycle_num_val] * PROC_NET_SNMP_CYCLE_COUNTER_NUM
            zero_delta = (
                None if zero_delta_val is None else make_zero_delta(zero_delta_val)
            )
            for i in range(procfs.NET_SNMP_NUM_VALUES):
                prev_proc_net_snmp = deepcopy(curr_proc_net_snmp)
                if i in proc_net_snmp_non_delta_index:
                    prev_proc_net_snmp.Values[i] = curr_val + 1
                else:
                    prev_proc_net_snmp.Values[i] = uint32_delta(
                        curr_val, max_val + 2 * i
                    )
                test_cases.append(
                    generate_proc_net_snmp_test_case(
                        f"{name}/{tc_num}",
                        curr_proc_net_snmp=curr_proc_net_snmp,
                        prev_proc_net_snmp=prev_proc_net_snmp,
                        cycle_num=cycle_num,
                        zero_delta=zero_delta,
                        description=f"cycle_num={cycle_num_val}, zero_delta={zero_delta_val}, i={i}",
                    )
                )
                tc_num += 1

    save_test_cases(
        test_cases, test_cases_file, test_cases_root_dir=test_cases_root_dir
    )
