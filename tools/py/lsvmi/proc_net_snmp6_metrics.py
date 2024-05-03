#! /usr/bin/env python3

# Generate test cases for lsvmi/proc_net_snmp6_metrics_test.go

import json
import os
import sys
import time
from copy import deepcopy
from dataclasses import asdict, dataclass
from typing import List, Optional, Tuple

import procfs

from . import (
    DEFAULT_TEST_HOSTNAME,
    DEFAULT_TEST_INSTANCE,
    HOSTNAME_LABEL_NAME,
    INSTANCE_LABEL_NAME,
    lsvmi_testcases_root,
    uint32_delta,
    uint64_delta,
)

DEFAULT_PROC_NET_SNMP6_INTERVAL_SEC = 1
DEFAULT_PROC_NET_SNMP6_FULL_METRICS_FACTOR = 15

ZeroDeltaType = List[bool]

# Metrics definitions, must match lsvmi/proc_net_snmp6_metrics.go:
PROC_NET_SNMP6_IP6_IN_RECEIVES_DELTA_METRIC = "proc_net_snmp6_ip6_in_receives_delta"
PROC_NET_SNMP6_IP6_IN_HDR_ERRORS_DELTA_METRIC = "proc_net_snmp6_ip6_in_hdr_errors_delta"
PROC_NET_SNMP6_IP6_IN_TOO_BIG_ERRORS_DELTA_METRIC = (
    "proc_net_snmp6_ip6_in_too_big_errors_delta"
)
PROC_NET_SNMP6_IP6_IN_NO_ROUTES_DELTA_METRIC = "proc_net_snmp6_ip6_in_no_routes_delta"
PROC_NET_SNMP6_IP6_IN_ADDR_ERRORS_DELTA_METRIC = (
    "proc_net_snmp6_ip6_in_addr_errors_delta"
)
PROC_NET_SNMP6_IP6_IN_UNKNOWN_PROTOS_DELTA_METRIC = (
    "proc_net_snmp6_ip6_in_unknown_protos_delta"
)
PROC_NET_SNMP6_IP6_IN_TRUNCATED_PKTS_DELTA_METRIC = (
    "proc_net_snmp6_ip6_in_truncated_pkts_delta"
)
PROC_NET_SNMP6_IP6_IN_DISCARDS_DELTA_METRIC = "proc_net_snmp6_ip6_in_discards_delta"
PROC_NET_SNMP6_IP6_IN_DELIVERS_DELTA_METRIC = "proc_net_snmp6_ip6_in_delivers_delta"
PROC_NET_SNMP6_IP6_OUT_FORW_DATAGRAMS_DELTA_METRIC = (
    "proc_net_snmp6_ip6_out_forw_datagrams_delta"
)
PROC_NET_SNMP6_IP6_OUT_REQUESTS_DELTA_METRIC = "proc_net_snmp6_ip6_out_requests_delta"
PROC_NET_SNMP6_IP6_OUT_DISCARDS_DELTA_METRIC = "proc_net_snmp6_ip6_out_discards_delta"
PROC_NET_SNMP6_IP6_OUT_NO_ROUTES_DELTA_METRIC = "proc_net_snmp6_ip6_out_no_routes_delta"
PROC_NET_SNMP6_IP6_REASM_TIMEOUT_DELTA_METRIC = "proc_net_snmp6_ip6_reasm_timeout_delta"
PROC_NET_SNMP6_IP6_REASM_REQDS_DELTA_METRIC = "proc_net_snmp6_ip6_reasm_reqds_delta"
PROC_NET_SNMP6_IP6_REASM_OKS_DELTA_METRIC = "proc_net_snmp6_ip6_reasm_oks_delta"
PROC_NET_SNMP6_IP6_REASM_FAILS_DELTA_METRIC = "proc_net_snmp6_ip6_reasm_fails_delta"
PROC_NET_SNMP6_IP6_FRAG_OKS_DELTA_METRIC = "proc_net_snmp6_ip6_frag_oks_delta"
PROC_NET_SNMP6_IP6_FRAG_FAILS_DELTA_METRIC = "proc_net_snmp6_ip6_frag_fails_delta"
PROC_NET_SNMP6_IP6_FRAG_CREATES_DELTA_METRIC = "proc_net_snmp6_ip6_frag_creates_delta"
PROC_NET_SNMP6_IP6_IN_MCAST_PKTS_DELTA_METRIC = "proc_net_snmp6_ip6_in_mcast_pkts_delta"
PROC_NET_SNMP6_IP6_OUT_MCAST_PKTS_DELTA_METRIC = (
    "proc_net_snmp6_ip6_out_mcast_pkts_delta"
)
PROC_NET_SNMP6_IP6_IN_KBPS_METRIC = "proc_net_snmp6_ip6_in_kbps"
PROC_NET_SNMP6_IP6_OUT_KBPS_METRIC = "proc_net_snmp6_ip6_out_kbps"
PROC_NET_SNMP6_IP6_IN_MCAST_KBPS_METRIC = "proc_net_snmp6_ip6_in_mcast_kbps"
PROC_NET_SNMP6_IP6_OUT_MCAST_KBPS_METRIC = "proc_net_snmp6_ip6_out_mcast_kbps"
PROC_NET_SNMP6_IP6_IN_BCAST_KBPS_METRIC = "proc_net_snmp6_ip6_in_bcast_kbps"
PROC_NET_SNMP6_IP6_OUT_BCAST_KBPS_METRIC = "proc_net_snmp6_ip6_out_bcast_kbps"
PROC_NET_SNMP6_IP6_IN_NO_ECT_PKTS_DELTA_METRIC = (
    "proc_net_snmp6_ip6_in_no_ect_pkts_delta"
)
PROC_NET_SNMP6_IP6_IN_ECT1_PKTS_DELTA_METRIC = "proc_net_snmp6_ip6_in_ect1_pkts_delta"
PROC_NET_SNMP6_IP6_IN_ECT0_PKTS_DELTA_METRIC = "proc_net_snmp6_ip6_in_ect0_pkts_delta"
PROC_NET_SNMP6_IP6_IN_CE_PKTS_DELTA_METRIC = "proc_net_snmp6_ip6_in_ce_pkts_delta"
PROC_NET_SNMP6_ICMP6_IN_MSGS_DELTA_METRIC = "proc_net_snmp6_icmp6_in_msgs_delta"
PROC_NET_SNMP6_ICMP6_IN_ERRORS_DELTA_METRIC = "proc_net_snmp6_icmp6_in_errors_delta"
PROC_NET_SNMP6_ICMP6_OUT_MSGS_DELTA_METRIC = "proc_net_snmp6_icmp6_out_msgs_delta"
PROC_NET_SNMP6_ICMP6_OUT_ERRORS_DELTA_METRIC = "proc_net_snmp6_icmp6_out_errors_delta"
PROC_NET_SNMP6_ICMP6_IN_CSUM_ERRORS_DELTA_METRIC = (
    "proc_net_snmp6_icmp6_in_csum_errors_delta"
)
PROC_NET_SNMP6_ICMP6_IN_DEST_UNREACHS_DELTA_METRIC = (
    "proc_net_snmp6_icmp6_in_dest_unreachs_delta"
)
PROC_NET_SNMP6_ICMP6_IN_PKT_TOO_BIGS_DELTA_METRIC = (
    "proc_net_snmp6_icmp6_in_pkt_too_bigs_delta"
)
PROC_NET_SNMP6_ICMP6_IN_TIME_EXCDS_DELTA_METRIC = (
    "proc_net_snmp6_icmp6_in_time_excds_delta"
)
PROC_NET_SNMP6_ICMP6_IN_PARM_PROBLEMS_DELTA_METRIC = (
    "proc_net_snmp6_icmp6_in_parm_problems_delta"
)
PROC_NET_SNMP6_ICMP6_IN_ECHOS_DELTA_METRIC = "proc_net_snmp6_icmp6_in_echos_delta"
PROC_NET_SNMP6_ICMP6_IN_ECHO_REPLIES_DELTA_METRIC = (
    "proc_net_snmp6_icmp6_in_echo_replies_delta"
)
PROC_NET_SNMP6_ICMP6_IN_GROUP_MEMB_QUERIES_DELTA_METRIC = (
    "proc_net_snmp6_icmp6_in_group_memb_queries_delta"
)
PROC_NET_SNMP6_ICMP6_IN_GROUP_MEMB_RESPONSES_DELTA_METRIC = (
    "proc_net_snmp6_icmp6_in_group_memb_responses_delta"
)
PROC_NET_SNMP6_ICMP6_IN_GROUP_MEMB_REDUCTIONS_DELTA_METRIC = (
    "proc_net_snmp6_icmp6_in_group_memb_reductions_delta"
)
PROC_NET_SNMP6_ICMP6_IN_ROUTER_SOLICITS_DELTA_METRIC = (
    "proc_net_snmp6_icmp6_in_router_solicits_delta"
)
PROC_NET_SNMP6_ICMP6_IN_ROUTER_ADVERTISEMENTS_DELTA_METRIC = (
    "proc_net_snmp6_icmp6_in_router_advertisements_delta"
)
PROC_NET_SNMP6_ICMP6_IN_NEIGHBOR_SOLICITS_DELTA_METRIC = (
    "proc_net_snmp6_icmp6_in_neighbor_solicits_delta"
)
PROC_NET_SNMP6_ICMP6_IN_NEIGHBOR_ADVERTISEMENTS_DELTA_METRIC = (
    "proc_net_snmp6_icmp6_in_neighbor_advertisements_delta"
)
PROC_NET_SNMP6_ICMP6_IN_REDIRECTS_DELTA_METRIC = (
    "proc_net_snmp6_icmp6_in_redirects_delta"
)
PROC_NET_SNMP6_ICMP6_IN_MLD_V2_REPORTS_DELTA_METRIC = (
    "proc_net_snmp6_icmp6_in_mld_v2_reports_delta"
)
PROC_NET_SNMP6_ICMP6_OUT_DEST_UNREACHS_DELTA_METRIC = (
    "proc_net_snmp6_icmp6_out_dest_unreachs_delta"
)
PROC_NET_SNMP6_ICMP6_OUT_PKT_TOO_BIGS_DELTA_METRIC = (
    "proc_net_snmp6_icmp6_out_pkt_too_bigs_delta"
)
PROC_NET_SNMP6_ICMP6_OUT_TIME_EXCDS_DELTA_METRIC = (
    "proc_net_snmp6_icmp6_out_time_excds_delta"
)
PROC_NET_SNMP6_ICMP6_OUT_PARM_PROBLEMS_DELTA_METRIC = (
    "proc_net_snmp6_icmp6_out_parm_problems_delta"
)
PROC_NET_SNMP6_ICMP6_OUT_ECHOS_DELTA_METRIC = "proc_net_snmp6_icmp6_out_echos_delta"
PROC_NET_SNMP6_ICMP6_OUT_ECHO_REPLIES_DELTA_METRIC = (
    "proc_net_snmp6_icmp6_out_echo_replies_delta"
)
PROC_NET_SNMP6_ICMP6_OUT_GROUP_MEMB_QUERIES_DELTA_METRIC = (
    "proc_net_snmp6_icmp6_out_group_memb_queries_delta"
)
PROC_NET_SNMP6_ICMP6_OUT_GROUP_MEMB_RESPONSES_DELTA_METRIC = (
    "proc_net_snmp6_icmp6_out_group_memb_responses_delta"
)
PROC_NET_SNMP6_ICMP6_OUT_GROUP_MEMB_REDUCTIONS_DELTA_METRIC = (
    "proc_net_snmp6_icmp6_out_group_memb_reductions_delta"
)
PROC_NET_SNMP6_ICMP6_OUT_ROUTER_SOLICITS_DELTA_METRIC = (
    "proc_net_snmp6_icmp6_out_router_solicits_delta"
)
PROC_NET_SNMP6_ICMP6_OUT_ROUTER_ADVERTISEMENTS_DELTA_METRIC = (
    "proc_net_snmp6_icmp6_out_router_advertisements_delta"
)
PROC_NET_SNMP6_ICMP6_OUT_NEIGHBOR_SOLICITS_DELTA_METRIC = (
    "proc_net_snmp6_icmp6_out_neighbor_solicits_delta"
)
PROC_NET_SNMP6_ICMP6_OUT_NEIGHBOR_ADVERTISEMENTS_DELTA_METRIC = (
    "proc_net_snmp6_icmp6_out_neighbor_advertisements_delta"
)
PROC_NET_SNMP6_ICMP6_OUT_REDIRECTS_DELTA_METRIC = (
    "proc_net_snmp6_icmp6_out_redirects_delta"
)
PROC_NET_SNMP6_ICMP6_OUT_MLD_V2_REPORTS_DELTA_METRIC = (
    "proc_net_snmp6_icmp6_out_mld_v2_reports_delta"
)
PROC_NET_SNMP6_ICMP6_OUT_TYPE133_DELTA_METRIC = "proc_net_snmp6_icmp6_out_type133_delta"
PROC_NET_SNMP6_ICMP6_OUT_TYPE135_DELTA_METRIC = "proc_net_snmp6_icmp6_out_type135_delta"
PROC_NET_SNMP6_ICMP6_OUT_TYPE143_DELTA_METRIC = "proc_net_snmp6_icmp6_out_type143_delta"
PROC_NET_SNMP6_UDP6_IN_DATAGRAMS_DELTA_METRIC = "proc_net_snmp6_udp6_in_datagrams_delta"
PROC_NET_SNMP6_UDP6_NO_PORTS_DELTA_METRIC = "proc_net_snmp6_udp6_no_ports_delta"
PROC_NET_SNMP6_UDP6_IN_ERRORS_DELTA_METRIC = "proc_net_snmp6_udp6_in_errors_delta"
PROC_NET_SNMP6_UDP6_OUT_DATAGRAMS_DELTA_METRIC = (
    "proc_net_snmp6_udp6_out_datagrams_delta"
)
PROC_NET_SNMP6_UDP6_RCVBUF_ERRORS_DELTA_METRIC = (
    "proc_net_snmp6_udp6_rcvbuf_errors_delta"
)
PROC_NET_SNMP6_UDP6_SNDBUF_ERRORS_DELTA_METRIC = (
    "proc_net_snmp6_udp6_sndbuf_errors_delta"
)
PROC_NET_SNMP6_UDP6_IN_CSUM_ERRORS_DELTA_METRIC = (
    "proc_net_snmp6_udp6_in_csum_errors_delta"
)
PROC_NET_SNMP6_UDP6_IGNORED_MULTI_DELTA_METRIC = (
    "proc_net_snmp6_udp6_ignored_multi_delta"
)
PROC_NET_SNMP6_UDP6_MEM_ERRORS_DELTA_METRIC = "proc_net_snmp6_udp6_mem_errors_delta"
PROC_NET_SNMP6_UDPLITE6_IN_DATAGRAMS_DELTA_METRIC = (
    "proc_net_snmp6_udplite6_in_datagrams_delta"
)
PROC_NET_SNMP6_UDPLITE6_NO_PORTS_DELTA_METRIC = "proc_net_snmp6_udplite6_no_ports_delta"
PROC_NET_SNMP6_UDPLITE6_IN_ERRORS_DELTA_METRIC = (
    "proc_net_snmp6_udplite6_in_errors_delta"
)
PROC_NET_SNMP6_UDPLITE6_OUT_DATAGRAMS_DELTA_METRIC = (
    "proc_net_snmp6_udplite6_out_datagrams_delta"
)
PROC_NET_SNMP6_UDPLITE6_RCVBUF_ERRORS_DELTA_METRIC = (
    "proc_net_snmp6_udplite6_rcvbuf_errors_delta"
)
PROC_NET_SNMP6_UDPLITE6_SNDBUF_ERRORS_DELTA_METRIC = (
    "proc_net_snmp6_udplite6_sndbuf_errors_delta"
)
PROC_NET_SNMP6_UDPLITE6_IN_CSUM_ERRORS_DELTA_METRIC = (
    "proc_net_snmp6_udplite6_in_csum_errors_delta"
)
PROC_NET_SNMP6_UDPLITE6_MEM_ERRORS_DELTA_METRIC = (
    "proc_net_snmp6_udplite6_mem_errors_delta"
)

PROC_NET_SNMP6_INTERVAL_METRIC_NAME = "proc_net_snmp6_metrics_delta_sec"

PROC_NET_SNMP6_CYCLE_COUNTER_EXP = 4
PROC_NET_SNMP6_CYCLE_COUNTER_NUM = 1 << PROC_NET_SNMP6_CYCLE_COUNTER_EXP
PROC_NET_SNMP6_CYCLE_COUNTER_MASK = PROC_NET_SNMP6_CYCLE_COUNTER_NUM - 1

proc_net_snmp6_index_rate = {
    procfs.NET_SNMP6_IP6_IN_OCTETS: (8.0 / 1000.0, 1),
    procfs.NET_SNMP6_IP6_OUT_OCTETS: (8.0 / 1000.0, 1),
    procfs.NET_SNMP6_IP6_IN_MCAST_OCTETS: (8.0 / 1000.0, 1),
    procfs.NET_SNMP6_IP6_OUT_MCAST_OCTETS: (8.0 / 1000.0, 1),
    procfs.NET_SNMP6_IP6_IN_BCAST_OCTETS: (8.0 / 1000.0, 1),
    procfs.NET_SNMP6_IP6_OUT_BCAST_OCTETS: (8.0 / 1000.0, 1),
}

proc_net_snmp6_index_to_metric_name = {
    procfs.NET_SNMP6_IP6_IN_RECEIVES: PROC_NET_SNMP6_IP6_IN_RECEIVES_DELTA_METRIC,
    procfs.NET_SNMP6_IP6_IN_HDR_ERRORS: PROC_NET_SNMP6_IP6_IN_HDR_ERRORS_DELTA_METRIC,
    procfs.NET_SNMP6_IP6_IN_TOO_BIG_ERRORS: PROC_NET_SNMP6_IP6_IN_TOO_BIG_ERRORS_DELTA_METRIC,
    procfs.NET_SNMP6_IP6_IN_NO_ROUTES: PROC_NET_SNMP6_IP6_IN_NO_ROUTES_DELTA_METRIC,
    procfs.NET_SNMP6_IP6_IN_ADDR_ERRORS: PROC_NET_SNMP6_IP6_IN_ADDR_ERRORS_DELTA_METRIC,
    procfs.NET_SNMP6_IP6_IN_UNKNOWN_PROTOS: PROC_NET_SNMP6_IP6_IN_UNKNOWN_PROTOS_DELTA_METRIC,
    procfs.NET_SNMP6_IP6_IN_TRUNCATED_PKTS: PROC_NET_SNMP6_IP6_IN_TRUNCATED_PKTS_DELTA_METRIC,
    procfs.NET_SNMP6_IP6_IN_DISCARDS: PROC_NET_SNMP6_IP6_IN_DISCARDS_DELTA_METRIC,
    procfs.NET_SNMP6_IP6_IN_DELIVERS: PROC_NET_SNMP6_IP6_IN_DELIVERS_DELTA_METRIC,
    procfs.NET_SNMP6_IP6_OUT_FORW_DATAGRAMS: PROC_NET_SNMP6_IP6_OUT_FORW_DATAGRAMS_DELTA_METRIC,
    procfs.NET_SNMP6_IP6_OUT_REQUESTS: PROC_NET_SNMP6_IP6_OUT_REQUESTS_DELTA_METRIC,
    procfs.NET_SNMP6_IP6_OUT_DISCARDS: PROC_NET_SNMP6_IP6_OUT_DISCARDS_DELTA_METRIC,
    procfs.NET_SNMP6_IP6_OUT_NO_ROUTES: PROC_NET_SNMP6_IP6_OUT_NO_ROUTES_DELTA_METRIC,
    procfs.NET_SNMP6_IP6_REASM_TIMEOUT: PROC_NET_SNMP6_IP6_REASM_TIMEOUT_DELTA_METRIC,
    procfs.NET_SNMP6_IP6_REASM_REQDS: PROC_NET_SNMP6_IP6_REASM_REQDS_DELTA_METRIC,
    procfs.NET_SNMP6_IP6_REASM_OKS: PROC_NET_SNMP6_IP6_REASM_OKS_DELTA_METRIC,
    procfs.NET_SNMP6_IP6_REASM_FAILS: PROC_NET_SNMP6_IP6_REASM_FAILS_DELTA_METRIC,
    procfs.NET_SNMP6_IP6_FRAG_OKS: PROC_NET_SNMP6_IP6_FRAG_OKS_DELTA_METRIC,
    procfs.NET_SNMP6_IP6_FRAG_FAILS: PROC_NET_SNMP6_IP6_FRAG_FAILS_DELTA_METRIC,
    procfs.NET_SNMP6_IP6_FRAG_CREATES: PROC_NET_SNMP6_IP6_FRAG_CREATES_DELTA_METRIC,
    procfs.NET_SNMP6_IP6_IN_MCAST_PKTS: PROC_NET_SNMP6_IP6_IN_MCAST_PKTS_DELTA_METRIC,
    procfs.NET_SNMP6_IP6_OUT_MCAST_PKTS: PROC_NET_SNMP6_IP6_OUT_MCAST_PKTS_DELTA_METRIC,
    procfs.NET_SNMP6_IP6_IN_OCTETS: PROC_NET_SNMP6_IP6_IN_KBPS_METRIC,
    procfs.NET_SNMP6_IP6_OUT_OCTETS: PROC_NET_SNMP6_IP6_OUT_KBPS_METRIC,
    procfs.NET_SNMP6_IP6_IN_MCAST_OCTETS: PROC_NET_SNMP6_IP6_IN_MCAST_KBPS_METRIC,
    procfs.NET_SNMP6_IP6_OUT_MCAST_OCTETS: PROC_NET_SNMP6_IP6_OUT_MCAST_KBPS_METRIC,
    procfs.NET_SNMP6_IP6_IN_BCAST_OCTETS: PROC_NET_SNMP6_IP6_IN_BCAST_KBPS_METRIC,
    procfs.NET_SNMP6_IP6_OUT_BCAST_OCTETS: PROC_NET_SNMP6_IP6_OUT_BCAST_KBPS_METRIC,
    procfs.NET_SNMP6_IP6_IN_NO_ECT_PKTS: PROC_NET_SNMP6_IP6_IN_NO_ECT_PKTS_DELTA_METRIC,
    procfs.NET_SNMP6_IP6_IN_ECT1_PKTS: PROC_NET_SNMP6_IP6_IN_ECT1_PKTS_DELTA_METRIC,
    procfs.NET_SNMP6_IP6_IN_ECT0_PKTS: PROC_NET_SNMP6_IP6_IN_ECT0_PKTS_DELTA_METRIC,
    procfs.NET_SNMP6_IP6_IN_CE_PKTS: PROC_NET_SNMP6_IP6_IN_CE_PKTS_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_IN_MSGS: PROC_NET_SNMP6_ICMP6_IN_MSGS_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_IN_ERRORS: PROC_NET_SNMP6_ICMP6_IN_ERRORS_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_OUT_MSGS: PROC_NET_SNMP6_ICMP6_OUT_MSGS_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_OUT_ERRORS: PROC_NET_SNMP6_ICMP6_OUT_ERRORS_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_IN_CSUM_ERRORS: PROC_NET_SNMP6_ICMP6_IN_CSUM_ERRORS_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_IN_DEST_UNREACHS: PROC_NET_SNMP6_ICMP6_IN_DEST_UNREACHS_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_IN_PKT_TOO_BIGS: PROC_NET_SNMP6_ICMP6_IN_PKT_TOO_BIGS_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_IN_TIME_EXCDS: PROC_NET_SNMP6_ICMP6_IN_TIME_EXCDS_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_IN_PARM_PROBLEMS: PROC_NET_SNMP6_ICMP6_IN_PARM_PROBLEMS_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_IN_ECHOS: PROC_NET_SNMP6_ICMP6_IN_ECHOS_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_IN_ECHO_REPLIES: PROC_NET_SNMP6_ICMP6_IN_ECHO_REPLIES_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_IN_GROUP_MEMB_QUERIES: PROC_NET_SNMP6_ICMP6_IN_GROUP_MEMB_QUERIES_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_IN_GROUP_MEMB_RESPONSES: PROC_NET_SNMP6_ICMP6_IN_GROUP_MEMB_RESPONSES_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_IN_GROUP_MEMB_REDUCTIONS: PROC_NET_SNMP6_ICMP6_IN_GROUP_MEMB_REDUCTIONS_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_IN_ROUTER_SOLICITS: PROC_NET_SNMP6_ICMP6_IN_ROUTER_SOLICITS_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_IN_ROUTER_ADVERTISEMENTS: PROC_NET_SNMP6_ICMP6_IN_ROUTER_ADVERTISEMENTS_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_IN_NEIGHBOR_SOLICITS: PROC_NET_SNMP6_ICMP6_IN_NEIGHBOR_SOLICITS_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_IN_NEIGHBOR_ADVERTISEMENTS: PROC_NET_SNMP6_ICMP6_IN_NEIGHBOR_ADVERTISEMENTS_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_IN_REDIRECTS: PROC_NET_SNMP6_ICMP6_IN_REDIRECTS_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_IN_MLD_V2_REPORTS: PROC_NET_SNMP6_ICMP6_IN_MLD_V2_REPORTS_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_OUT_DEST_UNREACHS: PROC_NET_SNMP6_ICMP6_OUT_DEST_UNREACHS_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_OUT_PKT_TOO_BIGS: PROC_NET_SNMP6_ICMP6_OUT_PKT_TOO_BIGS_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_OUT_TIME_EXCDS: PROC_NET_SNMP6_ICMP6_OUT_TIME_EXCDS_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_OUT_PARM_PROBLEMS: PROC_NET_SNMP6_ICMP6_OUT_PARM_PROBLEMS_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_OUT_ECHOS: PROC_NET_SNMP6_ICMP6_OUT_ECHOS_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_OUT_ECHO_REPLIES: PROC_NET_SNMP6_ICMP6_OUT_ECHO_REPLIES_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_OUT_GROUP_MEMB_QUERIES: PROC_NET_SNMP6_ICMP6_OUT_GROUP_MEMB_QUERIES_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_OUT_GROUP_MEMB_RESPONSES: PROC_NET_SNMP6_ICMP6_OUT_GROUP_MEMB_RESPONSES_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_OUT_GROUP_MEMB_REDUCTIONS: PROC_NET_SNMP6_ICMP6_OUT_GROUP_MEMB_REDUCTIONS_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_OUT_ROUTER_SOLICITS: PROC_NET_SNMP6_ICMP6_OUT_ROUTER_SOLICITS_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_OUT_ROUTER_ADVERTISEMENTS: PROC_NET_SNMP6_ICMP6_OUT_ROUTER_ADVERTISEMENTS_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_OUT_NEIGHBOR_SOLICITS: PROC_NET_SNMP6_ICMP6_OUT_NEIGHBOR_SOLICITS_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_OUT_NEIGHBOR_ADVERTISEMENTS: PROC_NET_SNMP6_ICMP6_OUT_NEIGHBOR_ADVERTISEMENTS_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_OUT_REDIRECTS: PROC_NET_SNMP6_ICMP6_OUT_REDIRECTS_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_OUT_MLD_V2_REPORTS: PROC_NET_SNMP6_ICMP6_OUT_MLD_V2_REPORTS_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_OUT_TYPE133: PROC_NET_SNMP6_ICMP6_OUT_TYPE133_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_OUT_TYPE135: PROC_NET_SNMP6_ICMP6_OUT_TYPE135_DELTA_METRIC,
    procfs.NET_SNMP6_ICMP6_OUT_TYPE143: PROC_NET_SNMP6_ICMP6_OUT_TYPE143_DELTA_METRIC,
    procfs.NET_SNMP6_UDP6_IN_DATAGRAMS: PROC_NET_SNMP6_UDP6_IN_DATAGRAMS_DELTA_METRIC,
    procfs.NET_SNMP6_UDP6_NO_PORTS: PROC_NET_SNMP6_UDP6_NO_PORTS_DELTA_METRIC,
    procfs.NET_SNMP6_UDP6_IN_ERRORS: PROC_NET_SNMP6_UDP6_IN_ERRORS_DELTA_METRIC,
    procfs.NET_SNMP6_UDP6_OUT_DATAGRAMS: PROC_NET_SNMP6_UDP6_OUT_DATAGRAMS_DELTA_METRIC,
    procfs.NET_SNMP6_UDP6_RCVBUF_ERRORS: PROC_NET_SNMP6_UDP6_RCVBUF_ERRORS_DELTA_METRIC,
    procfs.NET_SNMP6_UDP6_SNDBUF_ERRORS: PROC_NET_SNMP6_UDP6_SNDBUF_ERRORS_DELTA_METRIC,
    procfs.NET_SNMP6_UDP6_IN_CSUM_ERRORS: PROC_NET_SNMP6_UDP6_IN_CSUM_ERRORS_DELTA_METRIC,
    procfs.NET_SNMP6_UDP6_IGNORED_MULTI: PROC_NET_SNMP6_UDP6_IGNORED_MULTI_DELTA_METRIC,
    procfs.NET_SNMP6_UDP6_MEM_ERRORS: PROC_NET_SNMP6_UDP6_MEM_ERRORS_DELTA_METRIC,
    procfs.NET_SNMP6_UDPLITE6_IN_DATAGRAMS: PROC_NET_SNMP6_UDPLITE6_IN_DATAGRAMS_DELTA_METRIC,
    procfs.NET_SNMP6_UDPLITE6_NO_PORTS: PROC_NET_SNMP6_UDPLITE6_NO_PORTS_DELTA_METRIC,
    procfs.NET_SNMP6_UDPLITE6_IN_ERRORS: PROC_NET_SNMP6_UDPLITE6_IN_ERRORS_DELTA_METRIC,
    procfs.NET_SNMP6_UDPLITE6_OUT_DATAGRAMS: PROC_NET_SNMP6_UDPLITE6_OUT_DATAGRAMS_DELTA_METRIC,
    procfs.NET_SNMP6_UDPLITE6_RCVBUF_ERRORS: PROC_NET_SNMP6_UDPLITE6_RCVBUF_ERRORS_DELTA_METRIC,
    procfs.NET_SNMP6_UDPLITE6_SNDBUF_ERRORS: PROC_NET_SNMP6_UDPLITE6_SNDBUF_ERRORS_DELTA_METRIC,
    procfs.NET_SNMP6_UDPLITE6_IN_CSUM_ERRORS: PROC_NET_SNMP6_UDPLITE6_IN_CSUM_ERRORS_DELTA_METRIC,
    procfs.NET_SNMP6_UDPLITE6_MEM_ERRORS: PROC_NET_SNMP6_UDPLITE6_MEM_ERRORS_DELTA_METRIC,
}


@dataclass
class ProcNetSnmp6MetricsTestCase:
    Name: Optional[str] = None
    Description: Optional[str] = None
    Instance: Optional[str] = None
    Hostname: Optional[str] = None
    CurrProcNetSnmp6: Optional[procfs.NetSnmp6] = None
    PrevProcNetSnmp6: Optional[procfs.NetSnmp6] = None
    CurrPromTs: int = 0
    PrevPromTs: int = 0
    CycleNum: Optional[List[int]] = None
    FullMetricsFactor: int = DEFAULT_PROC_NET_SNMP6_FULL_METRICS_FACTOR
    ZeroDelta: Optional[ZeroDeltaType] = None
    WantMetricsCount: int = 0
    WantMetrics: Optional[List[str]] = None
    ReportExtra: bool = False
    WantZeroDelta: Optional[ZeroDeltaType] = None


testcases_file = "proc_net_snmp6.json"


def generate_proc_net_snmp6_metrics(
    curr_proc_net_snmp6: procfs.NetSnmp6,
    curr_prom_ts: int,
    prev_proc_net_snmp6: Optional[procfs.NetSnmp6] = None,
    cycle_num: Optional[List[int]] = None,
    zero_delta: Optional[ZeroDeltaType] = None,
    interval: float = DEFAULT_PROC_NET_SNMP6_INTERVAL_SEC,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
) -> Tuple[List[str], Optional[ZeroDeltaType]]:
    metrics = []
    new_zero_delta = (
        None if prev_proc_net_snmp6 is None else [False] * procfs.NET_SNMP6_NUM_VALUES
    )
    if prev_proc_net_snmp6 is None:
        return metrics, new_zero_delta

    for i, curr_value in enumerate(curr_proc_net_snmp6.Values):
        name = proc_net_snmp6_index_to_metric_name.get(i)
        if name is None:
            continue
        full_metrics = (
            cycle_num is None or cycle_num[i & PROC_NET_SNMP6_CYCLE_COUNTER_MASK] == 0
        )
        if curr_proc_net_snmp6.IsUint32[i]:
            delta = uint32_delta(curr_value, prev_proc_net_snmp6.Values[i])
        else:
            delta = uint64_delta(curr_value, prev_proc_net_snmp6.Values[i])
        if full_metrics or delta != 0 or zero_delta is None or not zero_delta[i]:
            rate = proc_net_snmp6_index_rate.get(i)
            if rate is not None:
                factor, prec = rate
                val = f"{delta * factor / interval:.{prec}f}"
            else:
                val = str(delta)
            metrics.append(
                f"{name}{{"
                + ",".join(
                    [
                        f'{INSTANCE_LABEL_NAME}="{instance}"',
                        f'{HOSTNAME_LABEL_NAME}="{hostname}"',
                    ]
                )
                + f"}} {val} {curr_prom_ts}"
            )
        new_zero_delta[i] = delta == 0

    metrics.append(
        f"{PROC_NET_SNMP6_INTERVAL_METRIC_NAME}{{"
        + ",".join(
            [
                f'{INSTANCE_LABEL_NAME}="{instance}"',
                f'{HOSTNAME_LABEL_NAME}="{hostname}"',
            ]
        )
        + f"}} {interval:.06f} {curr_prom_ts}"
    )

    return metrics, new_zero_delta


def generate_proc_net_snmp6_test_case(
    name: str,
    curr_proc_net_snmp6: procfs.NetSnmp6,
    ts: Optional[float] = None,
    prev_proc_net_snmp6: Optional[procfs.NetSnmp6] = None,
    cycle_num: Optional[List[int]] = None,
    zero_delta: Optional[ZeroDeltaType] = None,
    interval: float = DEFAULT_PROC_NET_SNMP6_INTERVAL_SEC,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    full_metrics_factor: int = DEFAULT_PROC_NET_SNMP6_FULL_METRICS_FACTOR,
    description: Optional[str] = None,
) -> ProcNetSnmp6MetricsTestCase:
    if ts is None:
        ts = time.time()
    curr_prom_ts = int(ts * 1000)
    prev_prom_ts = curr_prom_ts - int(interval * 1000)
    metrics, want_zero_delta = generate_proc_net_snmp6_metrics(
        curr_proc_net_snmp6,
        curr_prom_ts=curr_prom_ts,
        prev_proc_net_snmp6=prev_proc_net_snmp6,
        cycle_num=cycle_num,
        zero_delta=zero_delta,
        interval=interval,
        instance=instance,
        hostname=hostname,
    )
    return ProcNetSnmp6MetricsTestCase(
        Name=name,
        Description=description,
        Instance=instance,
        Hostname=hostname,
        CurrProcNetSnmp6=curr_proc_net_snmp6,
        PrevProcNetSnmp6=prev_proc_net_snmp6,
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


def make_ref_proc_net_snmp6() -> procfs.NetSnmp6:
    return procfs.NetSnmp6(Values=[i + 13 for i in range(procfs.NET_SNMP6_NUM_VALUES)])


def make_zero_delta(val: bool = False) -> ZeroDeltaType:
    return [val for i in range(procfs.NET_SNMP6_NUM_VALUES)]


def generate_proc_net_snmp6_metrics_test_cases(
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    testcases_root_dir: Optional[str] = lsvmi_testcases_root,
):
    ts = time.time()
    interval = DEFAULT_PROC_NET_SNMP6_INTERVAL_SEC

    if testcases_root_dir not in {None, "", "-"}:
        out_file = os.path.join(testcases_root_dir, testcases_file)
        os.makedirs(os.path.dirname(out_file), exist_ok=True)
        fp = open(out_file, "wt")
    else:
        out_file = None
        fp = sys.stdout

    test_cases = []
    tc_num = 0

    ref_proc_net_snmp6 = make_ref_proc_net_snmp6()
    max_val = max(ref_proc_net_snmp6.Values)

    name = "all_change"
    curr_proc_net_snmp6 = ref_proc_net_snmp6
    prev_proc_net_snmp6 = procfs.NetSnmp6()
    for i, curr_val in enumerate(curr_proc_net_snmp6.Values):
        rate = proc_net_snmp6_index_rate.get(i)
        delta = max_val + 2 * i
        if rate is not None:
            factor = rate[0]
            delta = int(delta / factor * interval)
        if curr_proc_net_snmp6.IsUint32[i]:
            prev_proc_net_snmp6.Values[i] = uint32_delta(curr_val, delta)
        else:
            prev_proc_net_snmp6.Values[i] = uint64_delta(curr_val, delta)
    for cycle_num_val in [0, 1]:
        for zero_delta_val in [None, False, True]:
            cycle_num = [cycle_num_val] * PROC_NET_SNMP6_CYCLE_COUNTER_NUM
            zero_delta = (
                None if zero_delta_val is None else make_zero_delta(zero_delta_val)
            )
            test_cases.append(
                generate_proc_net_snmp6_test_case(
                    f"{name}/{tc_num}",
                    curr_proc_net_snmp6=curr_proc_net_snmp6,
                    prev_proc_net_snmp6=prev_proc_net_snmp6,
                    cycle_num=cycle_num,
                    zero_delta=zero_delta,
                    interval=interval,
                    description=f"cycle_num={cycle_num_val}, zero_delta={zero_delta_val}",
                )
            )
            tc_num += 1

    name = "no_change"
    for cycle_num_val in [0, 1]:
        for zero_delta_val in [None, False, True]:
            cycle_num = [cycle_num_val] * PROC_NET_SNMP6_CYCLE_COUNTER_NUM
            zero_delta = (
                None if zero_delta_val is None else make_zero_delta(zero_delta_val)
            )
            test_cases.append(
                generate_proc_net_snmp6_test_case(
                    f"{name}/{tc_num}",
                    curr_proc_net_snmp6=ref_proc_net_snmp6,
                    prev_proc_net_snmp6=ref_proc_net_snmp6,
                    cycle_num=cycle_num,
                    zero_delta=zero_delta,
                    interval=interval,
                    description=f"cycle_num={cycle_num_val}, zero_delta={zero_delta_val}",
                )
            )
            tc_num += 1

    name = "single_change"
    curr_proc_net_snmp6 = ref_proc_net_snmp6
    for cycle_num_val in [0, 1]:
        for zero_delta_val in [None, False, True]:
            cycle_num = [cycle_num_val] * PROC_NET_SNMP6_CYCLE_COUNTER_NUM
            zero_delta = (
                None if zero_delta_val is None else make_zero_delta(zero_delta_val)
            )
            for i in range(procfs.NET_SNMP6_NUM_VALUES):
                prev_proc_net_snmp6 = deepcopy(curr_proc_net_snmp6)
                rate = proc_net_snmp6_index_rate.get(i)
                delta = max_val + 2 * i
                if rate is not None:
                    factor = rate[0]
                    delta = int(delta / factor * interval)
                if curr_proc_net_snmp6.IsUint32[i]:
                    prev_proc_net_snmp6.Values[i] = uint32_delta(curr_val, delta)
                else:
                    prev_proc_net_snmp6.Values[i] = uint64_delta(curr_val, delta)
                test_cases.append(
                    generate_proc_net_snmp6_test_case(
                        f"{name}/{tc_num}",
                        curr_proc_net_snmp6=curr_proc_net_snmp6,
                        prev_proc_net_snmp6=prev_proc_net_snmp6,
                        cycle_num=cycle_num,
                        zero_delta=zero_delta,
                        interval=interval,
                        description=f"cycle_num={cycle_num_val}, zero_delta={zero_delta_val}, i={i}",
                    )
                )
                tc_num += 1

    json.dump(list(map(asdict, test_cases)), fp=fp, indent=2)
    fp.write("\n")
    if out_file is not None:
        fp.close()
        print(f"{out_file} generated", file=sys.stderr)
