# LSVMI Network SNMP6 Metrics (id: `proc_net_snmp6_metrics`)

<!-- TOC tocDepth:2..3 chapterDepth:2..6 -->

- [General Information](#general-information)
- [Metrics](#metrics)
  - [proc_net_snmp6_ip6_in_receives_delta](#proc_net_snmp6_ip6_in_receives_delta)
  - [proc_net_snmp6_ip6_in_hdr_errors_delta](#proc_net_snmp6_ip6_in_hdr_errors_delta)
  - [proc_net_snmp6_ip6_in_too_big_errors_delta](#proc_net_snmp6_ip6_in_too_big_errors_delta)
  - [proc_net_snmp6_ip6_in_no_routes_delta](#proc_net_snmp6_ip6_in_no_routes_delta)
  - [proc_net_snmp6_ip6_in_addr_errors_delta](#proc_net_snmp6_ip6_in_addr_errors_delta)
  - [proc_net_snmp6_ip6_in_unknown_protos_delta](#proc_net_snmp6_ip6_in_unknown_protos_delta)
  - [proc_net_snmp6_ip6_in_truncated_pkts_delta](#proc_net_snmp6_ip6_in_truncated_pkts_delta)
  - [proc_net_snmp6_ip6_in_discards_delta](#proc_net_snmp6_ip6_in_discards_delta)
  - [proc_net_snmp6_ip6_in_delivers_delta](#proc_net_snmp6_ip6_in_delivers_delta)
  - [proc_net_snmp6_ip6_out_forw_datagrams_delta](#proc_net_snmp6_ip6_out_forw_datagrams_delta)
  - [proc_net_snmp6_ip6_out_requests_delta](#proc_net_snmp6_ip6_out_requests_delta)
  - [proc_net_snmp6_ip6_out_discards_delta](#proc_net_snmp6_ip6_out_discards_delta)
  - [proc_net_snmp6_ip6_out_no_routes_delta](#proc_net_snmp6_ip6_out_no_routes_delta)
  - [proc_net_snmp6_ip6_reasm_timeout_delta](#proc_net_snmp6_ip6_reasm_timeout_delta)
  - [proc_net_snmp6_ip6_reasm_reqds_delta](#proc_net_snmp6_ip6_reasm_reqds_delta)
  - [proc_net_snmp6_ip6_reasm_oks_delta](#proc_net_snmp6_ip6_reasm_oks_delta)
  - [proc_net_snmp6_ip6_reasm_fails_delta](#proc_net_snmp6_ip6_reasm_fails_delta)
  - [proc_net_snmp6_ip6_frag_oks_delta](#proc_net_snmp6_ip6_frag_oks_delta)
  - [proc_net_snmp6_ip6_frag_fails_delta](#proc_net_snmp6_ip6_frag_fails_delta)
  - [proc_net_snmp6_ip6_frag_creates_delta](#proc_net_snmp6_ip6_frag_creates_delta)
  - [proc_net_snmp6_ip6_in_mcast_pkts_delta](#proc_net_snmp6_ip6_in_mcast_pkts_delta)
  - [proc_net_snmp6_ip6_out_mcast_pkts_delta](#proc_net_snmp6_ip6_out_mcast_pkts_delta)
  - [proc_net_snmp6_ip6_in_kbps](#proc_net_snmp6_ip6_in_kbps)
  - [proc_net_snmp6_ip6_out_kbps](#proc_net_snmp6_ip6_out_kbps)
  - [proc_net_snmp6_ip6_in_mcast_kbps](#proc_net_snmp6_ip6_in_mcast_kbps)
  - [proc_net_snmp6_ip6_out_mcast_kbps](#proc_net_snmp6_ip6_out_mcast_kbps)
  - [proc_net_snmp6_ip6_in_bcast_kbps](#proc_net_snmp6_ip6_in_bcast_kbps)
  - [proc_net_snmp6_ip6_out_bcast_kbps](#proc_net_snmp6_ip6_out_bcast_kbps)
  - [proc_net_snmp6_ip6_in_no_ect_pkts_delta](#proc_net_snmp6_ip6_in_no_ect_pkts_delta)
  - [proc_net_snmp6_ip6_in_ect1_pkts_delta](#proc_net_snmp6_ip6_in_ect1_pkts_delta)
  - [proc_net_snmp6_ip6_in_ect0_pkts_delta](#proc_net_snmp6_ip6_in_ect0_pkts_delta)
  - [proc_net_snmp6_ip6_in_ce_pkts_delta](#proc_net_snmp6_ip6_in_ce_pkts_delta)
  - [proc_net_snmp6_icmp6_in_msgs_delta](#proc_net_snmp6_icmp6_in_msgs_delta)
  - [proc_net_snmp6_icmp6_in_errors_delta](#proc_net_snmp6_icmp6_in_errors_delta)
  - [proc_net_snmp6_icmp6_out_msgs_delta](#proc_net_snmp6_icmp6_out_msgs_delta)
  - [proc_net_snmp6_icmp6_out_errors_delta](#proc_net_snmp6_icmp6_out_errors_delta)
  - [proc_net_snmp6_icmp6_in_csum_errors_delta](#proc_net_snmp6_icmp6_in_csum_errors_delta)
  - [proc_net_snmp6_icmp6_in_dest_unreachs_delta](#proc_net_snmp6_icmp6_in_dest_unreachs_delta)
  - [proc_net_snmp6_icmp6_in_pkt_too_bigs_delta](#proc_net_snmp6_icmp6_in_pkt_too_bigs_delta)
  - [proc_net_snmp6_icmp6_in_time_excds_delta](#proc_net_snmp6_icmp6_in_time_excds_delta)
  - [proc_net_snmp6_icmp6_in_parm_problems_delta](#proc_net_snmp6_icmp6_in_parm_problems_delta)
  - [proc_net_snmp6_icmp6_in_echos_delta](#proc_net_snmp6_icmp6_in_echos_delta)
  - [proc_net_snmp6_icmp6_in_echo_replies_delta](#proc_net_snmp6_icmp6_in_echo_replies_delta)
  - [proc_net_snmp6_icmp6_in_group_memb_queries_delta](#proc_net_snmp6_icmp6_in_group_memb_queries_delta)
  - [proc_net_snmp6_icmp6_in_group_memb_responses_delta](#proc_net_snmp6_icmp6_in_group_memb_responses_delta)
  - [proc_net_snmp6_icmp6_in_group_memb_reductions_delta](#proc_net_snmp6_icmp6_in_group_memb_reductions_delta)
  - [proc_net_snmp6_icmp6_in_router_solicits_delta](#proc_net_snmp6_icmp6_in_router_solicits_delta)
  - [proc_net_snmp6_icmp6_in_router_advertisements_delta](#proc_net_snmp6_icmp6_in_router_advertisements_delta)
  - [proc_net_snmp6_icmp6_in_neighbor_solicits_delta](#proc_net_snmp6_icmp6_in_neighbor_solicits_delta)
  - [proc_net_snmp6_icmp6_in_neighbor_advertisements_delta](#proc_net_snmp6_icmp6_in_neighbor_advertisements_delta)
  - [proc_net_snmp6_icmp6_in_redirects_delta](#proc_net_snmp6_icmp6_in_redirects_delta)
  - [proc_net_snmp6_icmp6_in_mld_v2_reports_delta](#proc_net_snmp6_icmp6_in_mld_v2_reports_delta)
  - [proc_net_snmp6_icmp6_out_dest_unreachs_delta](#proc_net_snmp6_icmp6_out_dest_unreachs_delta)
  - [proc_net_snmp6_icmp6_out_pkt_too_bigs_delta](#proc_net_snmp6_icmp6_out_pkt_too_bigs_delta)
  - [proc_net_snmp6_icmp6_out_time_excds_delta](#proc_net_snmp6_icmp6_out_time_excds_delta)
  - [proc_net_snmp6_icmp6_out_parm_problems_delta](#proc_net_snmp6_icmp6_out_parm_problems_delta)
  - [proc_net_snmp6_icmp6_out_echos_delta](#proc_net_snmp6_icmp6_out_echos_delta)
  - [proc_net_snmp6_icmp6_out_echo_replies_delta](#proc_net_snmp6_icmp6_out_echo_replies_delta)
  - [proc_net_snmp6_icmp6_out_group_memb_queries_delta](#proc_net_snmp6_icmp6_out_group_memb_queries_delta)
  - [proc_net_snmp6_icmp6_out_group_memb_responses_delta](#proc_net_snmp6_icmp6_out_group_memb_responses_delta)
  - [proc_net_snmp6_icmp6_out_group_memb_reductions_delta](#proc_net_snmp6_icmp6_out_group_memb_reductions_delta)
  - [proc_net_snmp6_icmp6_out_router_solicits_delta](#proc_net_snmp6_icmp6_out_router_solicits_delta)
  - [proc_net_snmp6_icmp6_out_router_advertisements_delta](#proc_net_snmp6_icmp6_out_router_advertisements_delta)
  - [proc_net_snmp6_icmp6_out_neighbor_solicits_delta](#proc_net_snmp6_icmp6_out_neighbor_solicits_delta)
  - [proc_net_snmp6_icmp6_out_neighbor_advertisements_delta](#proc_net_snmp6_icmp6_out_neighbor_advertisements_delta)
  - [proc_net_snmp6_icmp6_out_redirects_delta](#proc_net_snmp6_icmp6_out_redirects_delta)
  - [proc_net_snmp6_icmp6_out_mld_v2_reports_delta](#proc_net_snmp6_icmp6_out_mld_v2_reports_delta)
  - [proc_net_snmp6_icmp6_out_type133_delta](#proc_net_snmp6_icmp6_out_type133_delta)
  - [proc_net_snmp6_icmp6_out_type135_delta](#proc_net_snmp6_icmp6_out_type135_delta)
  - [proc_net_snmp6_icmp6_out_type143_delta](#proc_net_snmp6_icmp6_out_type143_delta)
  - [proc_net_snmp6_udp6_in_datagrams_delta](#proc_net_snmp6_udp6_in_datagrams_delta)
  - [proc_net_snmp6_udp6_no_ports_delta](#proc_net_snmp6_udp6_no_ports_delta)
  - [proc_net_snmp6_udp6_in_errors_delta](#proc_net_snmp6_udp6_in_errors_delta)
  - [proc_net_snmp6_udp6_out_datagrams_delta](#proc_net_snmp6_udp6_out_datagrams_delta)
  - [proc_net_snmp6_udp6_rcvbuf_errors_delta](#proc_net_snmp6_udp6_rcvbuf_errors_delta)
  - [proc_net_snmp6_udp6_sndbuf_errors_delta](#proc_net_snmp6_udp6_sndbuf_errors_delta)
  - [proc_net_snmp6_udp6_in_csum_errors_delta](#proc_net_snmp6_udp6_in_csum_errors_delta)
  - [proc_net_snmp6_udp6_ignored_multi_delta](#proc_net_snmp6_udp6_ignored_multi_delta)
  - [proc_net_snmp6_udp6_mem_errors_delta](#proc_net_snmp6_udp6_mem_errors_delta)
  - [proc_net_snmp6_udplite6_in_datagrams_delta](#proc_net_snmp6_udplite6_in_datagrams_delta)
  - [proc_net_snmp6_udplite6_no_ports_delta](#proc_net_snmp6_udplite6_no_ports_delta)
  - [proc_net_snmp6_udplite6_in_errors_delta](#proc_net_snmp6_udplite6_in_errors_delta)
  - [proc_net_snmp6_udplite6_out_datagrams_delta](#proc_net_snmp6_udplite6_out_datagrams_delta)
  - [proc_net_snmp6_udplite6_rcvbuf_errors_delta](#proc_net_snmp6_udplite6_rcvbuf_errors_delta)
  - [proc_net_snmp6_udplite6_sndbuf_errors_delta](#proc_net_snmp6_udplite6_sndbuf_errors_delta)
  - [proc_net_snmp6_udplite6_in_csum_errors_delta](#proc_net_snmp6_udplite6_in_csum_errors_delta)
  - [proc_net_snmp6_udplite6_mem_errors_delta](#proc_net_snmp6_udplite6_mem_errors_delta)
  - [proc_net_snmp6_metrics_delta_sec](#proc_net_snmp6_metrics_delta_sec)

<!-- /TOC -->

## General Information

The `/proc/net/snmp` syntax is:

```text

ProtoCounterName Value
ProtoCounterName Value
...

```

e.g.

```text

Ip6InMcastPkts                   0
Ip6OutMcastPkts                  19
Ip6InOctets                      368
Ip6OutOctets                     1196
Ip6InMcastOctets                 0
Ip6OutMcastOctets                1196
Ip6InBcastOctets                 0

```

The metric name derivation schema:

`ProtoCounterName -> proc_net_snmp6_proto_counter_name[_delta] or ...[_kbps]`

with the following optional suffixes:

  | Suffix | Meaning |
  | ---    | ---     |
  | `_delta` | counter increase since the last scan |
  | `_kbps` | average throughput since the last scan in kbit/sec.<br>Based on byte#  delta / `proc_net_snmp6_metrics_delta_sec` |

References:

- [RFC2465](https://datatracker.ietf.org/doc/html/rfc2465)
- [RFC2466](https://datatracker.ietf.org/doc/html/rfc2466)
- [RFC4293](https://datatracker.ietf.org/doc/html/rfc4293), w/ the following comment from [linux/net/ipv6/proc.c](https://github.com/torvalds/linux/blob/master/net/ipv6/proc.c)

    ```c
    
    /* RFC 4293 v6 ICMPMsgStatsTable; named items for RFC 2466 compatibility */

    ```

- [linux/net/ipv6/proc.c](https://github.com/torvalds/linux/blob/master/net/ipv6/proc.c)

## Metrics

Unless otherwise specified, all the metrics have the following label set:

| Label Name | Value(s)/Info |
| --- | --- |
| instance | _instance_ |
| hostname | _hostname_ |

### proc_net_snmp6_ip6_in_receives_delta

### proc_net_snmp6_ip6_in_hdr_errors_delta

### proc_net_snmp6_ip6_in_too_big_errors_delta

### proc_net_snmp6_ip6_in_no_routes_delta

### proc_net_snmp6_ip6_in_addr_errors_delta

### proc_net_snmp6_ip6_in_unknown_protos_delta

### proc_net_snmp6_ip6_in_truncated_pkts_delta

### proc_net_snmp6_ip6_in_discards_delta

### proc_net_snmp6_ip6_in_delivers_delta

### proc_net_snmp6_ip6_out_forw_datagrams_delta

### proc_net_snmp6_ip6_out_requests_delta

### proc_net_snmp6_ip6_out_discards_delta

### proc_net_snmp6_ip6_out_no_routes_delta

### proc_net_snmp6_ip6_reasm_timeout_delta

### proc_net_snmp6_ip6_reasm_reqds_delta

### proc_net_snmp6_ip6_reasm_oks_delta

### proc_net_snmp6_ip6_reasm_fails_delta

### proc_net_snmp6_ip6_frag_oks_delta

### proc_net_snmp6_ip6_frag_fails_delta

### proc_net_snmp6_ip6_frag_creates_delta

### proc_net_snmp6_ip6_in_mcast_pkts_delta

### proc_net_snmp6_ip6_out_mcast_pkts_delta

### proc_net_snmp6_ip6_in_kbps

### proc_net_snmp6_ip6_out_kbps

### proc_net_snmp6_ip6_in_mcast_kbps

### proc_net_snmp6_ip6_out_mcast_kbps

### proc_net_snmp6_ip6_in_bcast_kbps

### proc_net_snmp6_ip6_out_bcast_kbps

### proc_net_snmp6_ip6_in_no_ect_pkts_delta

### proc_net_snmp6_ip6_in_ect1_pkts_delta

### proc_net_snmp6_ip6_in_ect0_pkts_delta

### proc_net_snmp6_ip6_in_ce_pkts_delta

### proc_net_snmp6_icmp6_in_msgs_delta

### proc_net_snmp6_icmp6_in_errors_delta

### proc_net_snmp6_icmp6_out_msgs_delta

### proc_net_snmp6_icmp6_out_errors_delta

### proc_net_snmp6_icmp6_in_csum_errors_delta

### proc_net_snmp6_icmp6_in_dest_unreachs_delta

### proc_net_snmp6_icmp6_in_pkt_too_bigs_delta

### proc_net_snmp6_icmp6_in_time_excds_delta

### proc_net_snmp6_icmp6_in_parm_problems_delta

### proc_net_snmp6_icmp6_in_echos_delta

### proc_net_snmp6_icmp6_in_echo_replies_delta

### proc_net_snmp6_icmp6_in_group_memb_queries_delta

### proc_net_snmp6_icmp6_in_group_memb_responses_delta

### proc_net_snmp6_icmp6_in_group_memb_reductions_delta

### proc_net_snmp6_icmp6_in_router_solicits_delta

### proc_net_snmp6_icmp6_in_router_advertisements_delta

### proc_net_snmp6_icmp6_in_neighbor_solicits_delta

### proc_net_snmp6_icmp6_in_neighbor_advertisements_delta

### proc_net_snmp6_icmp6_in_redirects_delta

### proc_net_snmp6_icmp6_in_mld_v2_reports_delta

### proc_net_snmp6_icmp6_out_dest_unreachs_delta

### proc_net_snmp6_icmp6_out_pkt_too_bigs_delta

### proc_net_snmp6_icmp6_out_time_excds_delta

### proc_net_snmp6_icmp6_out_parm_problems_delta

### proc_net_snmp6_icmp6_out_echos_delta

### proc_net_snmp6_icmp6_out_echo_replies_delta

### proc_net_snmp6_icmp6_out_group_memb_queries_delta

### proc_net_snmp6_icmp6_out_group_memb_responses_delta

### proc_net_snmp6_icmp6_out_group_memb_reductions_delta

### proc_net_snmp6_icmp6_out_router_solicits_delta

### proc_net_snmp6_icmp6_out_router_advertisements_delta

### proc_net_snmp6_icmp6_out_neighbor_solicits_delta

### proc_net_snmp6_icmp6_out_neighbor_advertisements_delta

### proc_net_snmp6_icmp6_out_redirects_delta

### proc_net_snmp6_icmp6_out_mld_v2_reports_delta

### proc_net_snmp6_icmp6_out_type133_delta

### proc_net_snmp6_icmp6_out_type135_delta

### proc_net_snmp6_icmp6_out_type143_delta

### proc_net_snmp6_udp6_in_datagrams_delta

### proc_net_snmp6_udp6_no_ports_delta

### proc_net_snmp6_udp6_in_errors_delta

### proc_net_snmp6_udp6_out_datagrams_delta

### proc_net_snmp6_udp6_rcvbuf_errors_delta

### proc_net_snmp6_udp6_sndbuf_errors_delta

### proc_net_snmp6_udp6_in_csum_errors_delta

### proc_net_snmp6_udp6_ignored_multi_delta

### proc_net_snmp6_udp6_mem_errors_delta

### proc_net_snmp6_udplite6_in_datagrams_delta

### proc_net_snmp6_udplite6_no_ports_delta

### proc_net_snmp6_udplite6_in_errors_delta

### proc_net_snmp6_udplite6_out_datagrams_delta

### proc_net_snmp6_udplite6_rcvbuf_errors_delta

### proc_net_snmp6_udplite6_sndbuf_errors_delta

### proc_net_snmp6_udplite6_in_csum_errors_delta

### proc_net_snmp6_udplite6_mem_errors_delta

### proc_net_snmp6_metrics_delta_sec

Time in seconds since the last scan. The real life counterpart (i.e. measured value) to the desired (configured) `interval`. This the basis for computing averages for `_kbps` suffix metrics.
