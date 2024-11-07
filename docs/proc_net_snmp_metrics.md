# LSVMI Network SNMP Metrics (id: `proc_net_snmp_metrics`)

<!-- TOC tocDepth:2..3 chapterDepth:2..6 -->

- [General Information](#general-information)
- [Metrics](#metrics)
  - [proc_net_snmp_ip_forwarding](#proc_net_snmp_ip_forwarding)
  - [proc_net_snmp_ip_default_ttl](#proc_net_snmp_ip_default_ttl)
  - [proc_net_snmp_ip_in_receives_delta](#proc_net_snmp_ip_in_receives_delta)
  - [proc_net_snmp_ip_in_hdr_errors_delta](#proc_net_snmp_ip_in_hdr_errors_delta)
  - [proc_net_snmp_ip_in_addr_errors_delta](#proc_net_snmp_ip_in_addr_errors_delta)
  - [proc_net_snmp_ip_forw_datagrams_delta](#proc_net_snmp_ip_forw_datagrams_delta)
  - [proc_net_snmp_ip_in_unknown_protos_delta](#proc_net_snmp_ip_in_unknown_protos_delta)
  - [proc_net_snmp_ip_in_discards_delta](#proc_net_snmp_ip_in_discards_delta)
  - [proc_net_snmp_ip_in_delivers_delta](#proc_net_snmp_ip_in_delivers_delta)
  - [proc_net_snmp_ip_out_requests_delta](#proc_net_snmp_ip_out_requests_delta)
  - [proc_net_snmp_ip_out_discards_delta](#proc_net_snmp_ip_out_discards_delta)
  - [proc_net_snmp_ip_out_no_routes_delta](#proc_net_snmp_ip_out_no_routes_delta)
  - [proc_net_snmp_ip_reasm_timeout](#proc_net_snmp_ip_reasm_timeout)
  - [proc_net_snmp_ip_reasm_reqds_delta](#proc_net_snmp_ip_reasm_reqds_delta)
  - [proc_net_snmp_ip_reasm_oks_delta](#proc_net_snmp_ip_reasm_oks_delta)
  - [proc_net_snmp_ip_reasm_fails_delta](#proc_net_snmp_ip_reasm_fails_delta)
  - [proc_net_snmp_ip_frag_oks_delta](#proc_net_snmp_ip_frag_oks_delta)
  - [proc_net_snmp_ip_frag_fails_delta](#proc_net_snmp_ip_frag_fails_delta)
  - [proc_net_snmp_ip_frag_creates_delta](#proc_net_snmp_ip_frag_creates_delta)
  - [proc_net_snmp_icmp_in_msgs_delta](#proc_net_snmp_icmp_in_msgs_delta)
  - [proc_net_snmp_icmp_in_errors_delta](#proc_net_snmp_icmp_in_errors_delta)
  - [proc_net_snmp_icmp_in_csum_errors_delta](#proc_net_snmp_icmp_in_csum_errors_delta)
  - [proc_net_snmp_icmp_in_dest_unreachs_delta](#proc_net_snmp_icmp_in_dest_unreachs_delta)
  - [proc_net_snmp_icmp_in_time_excds_delta](#proc_net_snmp_icmp_in_time_excds_delta)
  - [proc_net_snmp_icmp_in_parm_probs_delta](#proc_net_snmp_icmp_in_parm_probs_delta)
  - [proc_net_snmp_icmp_in_src_quenchs_delta](#proc_net_snmp_icmp_in_src_quenchs_delta)
  - [proc_net_snmp_icmp_in_redirects_delta](#proc_net_snmp_icmp_in_redirects_delta)
  - [proc_net_snmp_icmp_in_echos_delta](#proc_net_snmp_icmp_in_echos_delta)
  - [proc_net_snmp_icmp_in_echo_reps_delta](#proc_net_snmp_icmp_in_echo_reps_delta)
  - [proc_net_snmp_icmp_in_timestamps_delta](#proc_net_snmp_icmp_in_timestamps_delta)
  - [proc_net_snmp_icmp_in_timestamp_reps_delta](#proc_net_snmp_icmp_in_timestamp_reps_delta)
  - [proc_net_snmp_icmp_in_addr_masks_delta](#proc_net_snmp_icmp_in_addr_masks_delta)
  - [proc_net_snmp_icmp_in_addr_mask_reps_delta](#proc_net_snmp_icmp_in_addr_mask_reps_delta)
  - [proc_net_snmp_icmp_out_msgs_delta](#proc_net_snmp_icmp_out_msgs_delta)
  - [proc_net_snmp_icmp_out_errors_delta](#proc_net_snmp_icmp_out_errors_delta)
  - [proc_net_snmp_icmp_out_dest_unreachs_delta](#proc_net_snmp_icmp_out_dest_unreachs_delta)
  - [proc_net_snmp_icmp_out_time_excds_delta](#proc_net_snmp_icmp_out_time_excds_delta)
  - [proc_net_snmp_icmp_out_parm_probs_delta](#proc_net_snmp_icmp_out_parm_probs_delta)
  - [proc_net_snmp_icmp_out_src_quenchs_delta](#proc_net_snmp_icmp_out_src_quenchs_delta)
  - [proc_net_snmp_icmp_out_redirects_delta](#proc_net_snmp_icmp_out_redirects_delta)
  - [proc_net_snmp_icmp_out_echos_delta](#proc_net_snmp_icmp_out_echos_delta)
  - [proc_net_snmp_icmp_out_echo_reps_delta](#proc_net_snmp_icmp_out_echo_reps_delta)
  - [proc_net_snmp_icmp_out_timestamps_delta](#proc_net_snmp_icmp_out_timestamps_delta)
  - [proc_net_snmp_icmp_out_timestamp_reps_delta](#proc_net_snmp_icmp_out_timestamp_reps_delta)
  - [proc_net_snmp_icmp_out_addr_masks_delta](#proc_net_snmp_icmp_out_addr_masks_delta)
  - [proc_net_snmp_icmp_out_addr_mask_reps_delta](#proc_net_snmp_icmp_out_addr_mask_reps_delta)
  - [proc_net_snmp_icmpmsg_in_type3_delta](#proc_net_snmp_icmpmsg_in_type3_delta)
  - [proc_net_snmp_icmpmsg_out_type3_delta](#proc_net_snmp_icmpmsg_out_type3_delta)
  - [proc_net_snmp_tcp_rto_algorithm](#proc_net_snmp_tcp_rto_algorithm)
  - [proc_net_snmp_tcp_rto_min](#proc_net_snmp_tcp_rto_min)
  - [proc_net_snmp_tcp_rto_max](#proc_net_snmp_tcp_rto_max)
  - [proc_net_snmp_tcp_max_conn](#proc_net_snmp_tcp_max_conn)
  - [proc_net_snmp_tcp_active_opens_delta](#proc_net_snmp_tcp_active_opens_delta)
  - [proc_net_snmp_tcp_passive_opens_delta](#proc_net_snmp_tcp_passive_opens_delta)
  - [proc_net_snmp_tcp_attempt_fails_delta](#proc_net_snmp_tcp_attempt_fails_delta)
  - [proc_net_snmp_tcp_estab_resets_delta](#proc_net_snmp_tcp_estab_resets_delta)
  - [proc_net_snmp_tcp_curr_estab](#proc_net_snmp_tcp_curr_estab)
  - [proc_net_snmp_tcp_in_segs_delta](#proc_net_snmp_tcp_in_segs_delta)
  - [proc_net_snmp_tcp_out_segs_delta](#proc_net_snmp_tcp_out_segs_delta)
  - [proc_net_snmp_tcp_retrans_segs_delta](#proc_net_snmp_tcp_retrans_segs_delta)
  - [proc_net_snmp_tcp_in_errs_delta](#proc_net_snmp_tcp_in_errs_delta)
  - [proc_net_snmp_tcp_out_rsts_delta](#proc_net_snmp_tcp_out_rsts_delta)
  - [proc_net_snmp_tcp_in_csum_errors_delta](#proc_net_snmp_tcp_in_csum_errors_delta)
  - [proc_net_snmp_udp_in_datagrams_delta](#proc_net_snmp_udp_in_datagrams_delta)
  - [proc_net_snmp_udp_no_ports_delta](#proc_net_snmp_udp_no_ports_delta)
  - [proc_net_snmp_udp_in_errors_delta](#proc_net_snmp_udp_in_errors_delta)
  - [proc_net_snmp_udp_out_datagrams_delta](#proc_net_snmp_udp_out_datagrams_delta)
  - [proc_net_snmp_udp_rcvbuf_errors_delta](#proc_net_snmp_udp_rcvbuf_errors_delta)
  - [proc_net_snmp_udp_sndbuf_errors_delta](#proc_net_snmp_udp_sndbuf_errors_delta)
  - [proc_net_snmp_udp_in_csum_errors_delta](#proc_net_snmp_udp_in_csum_errors_delta)
  - [proc_net_snmp_udp_ignored_multi_delta](#proc_net_snmp_udp_ignored_multi_delta)
  - [proc_net_snmp_udp_mem_errors_delta](#proc_net_snmp_udp_mem_errors_delta)
  - [proc_net_snmp_udplite_in_datagrams_delta](#proc_net_snmp_udplite_in_datagrams_delta)
  - [proc_net_snmp_udplite_no_ports_delta](#proc_net_snmp_udplite_no_ports_delta)
  - [proc_net_snmp_udplite_in_errors_delta](#proc_net_snmp_udplite_in_errors_delta)
  - [proc_net_snmp_udplite_out_datagrams_delta](#proc_net_snmp_udplite_out_datagrams_delta)
  - [proc_net_snmp_udplite_rcvbuf_errors_delta](#proc_net_snmp_udplite_rcvbuf_errors_delta)
  - [proc_net_snmp_udplite_sndbuf_errors_delta](#proc_net_snmp_udplite_sndbuf_errors_delta)
  - [proc_net_snmp_udplite_in_csum_errors_delta](#proc_net_snmp_udplite_in_csum_errors_delta)
  - [proc_net_snmp_udplite_ignored_multi_delta](#proc_net_snmp_udplite_ignored_multi_delta)
  - [proc_net_snmp_udplite_mem_errors_delta](#proc_net_snmp_udplite_mem_errors_delta)
  - [proc_net_snmp_metrics_delta_sec](#proc_net_snmp_metrics_delta_sec)

<!-- /TOC -->

## General Information

The `/proc/net/snmp` syntax is:

```text
Proto: CounterName ...
Proto: CounterValue ...

```

e.g.

```text

Tcp: RtoAlgorithm RtoMin RtoMax MaxConn ActiveOpens PassiveOpens AttemptFails EstabResets CurrEstab InSegs OutSegs RetransSegs InErrs OutRsts InCsumErrors
Tcp: 1 200 120000 -1 98 63 4 1 1 5708 14228 35 0 15 0
Udp: InDatagrams NoPorts InErrors OutDatagrams RcvbufErrors SndbufErrors InCsumErrors IgnoredMulti MemErrors
Udp: 1006 16 0 1023 0 0 0 0 0
```

The metric name derivation schema:

`Proto: CounterName` -> `proc_net_snmp_proto_counter_name[_delta]`

with the suffix delta indicating the counter increase since the last scan.

References:

- [RFC2011](https://datatracker.ietf.org/doc/html/rfc2011)
- [RFC5097](https://datatracker.ietf.org/doc/html/rfc5097)
- [linux/include/uapi/linux/snmp.h](https://github.com/torvalds/linux/tree/master/include/uapi/linux/snmp.h)
- [SNMP counter](https://github.com/torvalds/linux/tree/master/Documentation/networking/snmp_counter.rst)

## Metrics

Unless otherwise specified, all the metrics have the following label set:

| Label Name | Value(s)/Info |
| --- | --- |
| instance | _instance_ |
| hostname | _hostname_ |

### proc_net_snmp_ip_forwarding

### proc_net_snmp_ip_default_ttl

### proc_net_snmp_ip_in_receives_delta

### proc_net_snmp_ip_in_hdr_errors_delta

### proc_net_snmp_ip_in_addr_errors_delta

### proc_net_snmp_ip_forw_datagrams_delta

### proc_net_snmp_ip_in_unknown_protos_delta

### proc_net_snmp_ip_in_discards_delta

### proc_net_snmp_ip_in_delivers_delta

### proc_net_snmp_ip_out_requests_delta

### proc_net_snmp_ip_out_discards_delta

### proc_net_snmp_ip_out_no_routes_delta

### proc_net_snmp_ip_reasm_timeout

### proc_net_snmp_ip_reasm_reqds_delta

### proc_net_snmp_ip_reasm_oks_delta

### proc_net_snmp_ip_reasm_fails_delta

### proc_net_snmp_ip_frag_oks_delta

### proc_net_snmp_ip_frag_fails_delta

### proc_net_snmp_ip_frag_creates_delta

### proc_net_snmp_icmp_in_msgs_delta

### proc_net_snmp_icmp_in_errors_delta

### proc_net_snmp_icmp_in_csum_errors_delta

### proc_net_snmp_icmp_in_dest_unreachs_delta

### proc_net_snmp_icmp_in_time_excds_delta

### proc_net_snmp_icmp_in_parm_probs_delta

### proc_net_snmp_icmp_in_src_quenchs_delta

### proc_net_snmp_icmp_in_redirects_delta

### proc_net_snmp_icmp_in_echos_delta

### proc_net_snmp_icmp_in_echo_reps_delta

### proc_net_snmp_icmp_in_timestamps_delta

### proc_net_snmp_icmp_in_timestamp_reps_delta

### proc_net_snmp_icmp_in_addr_masks_delta

### proc_net_snmp_icmp_in_addr_mask_reps_delta

### proc_net_snmp_icmp_out_msgs_delta

### proc_net_snmp_icmp_out_errors_delta

### proc_net_snmp_icmp_out_dest_unreachs_delta

### proc_net_snmp_icmp_out_time_excds_delta

### proc_net_snmp_icmp_out_parm_probs_delta

### proc_net_snmp_icmp_out_src_quenchs_delta

### proc_net_snmp_icmp_out_redirects_delta

### proc_net_snmp_icmp_out_echos_delta

### proc_net_snmp_icmp_out_echo_reps_delta

### proc_net_snmp_icmp_out_timestamps_delta

### proc_net_snmp_icmp_out_timestamp_reps_delta

### proc_net_snmp_icmp_out_addr_masks_delta

### proc_net_snmp_icmp_out_addr_mask_reps_delta

### proc_net_snmp_icmpmsg_in_type3_delta

### proc_net_snmp_icmpmsg_out_type3_delta

### proc_net_snmp_tcp_rto_algorithm

### proc_net_snmp_tcp_rto_min

### proc_net_snmp_tcp_rto_max

### proc_net_snmp_tcp_max_conn

### proc_net_snmp_tcp_active_opens_delta

### proc_net_snmp_tcp_passive_opens_delta

### proc_net_snmp_tcp_attempt_fails_delta

### proc_net_snmp_tcp_estab_resets_delta

### proc_net_snmp_tcp_curr_estab

### proc_net_snmp_tcp_in_segs_delta

### proc_net_snmp_tcp_out_segs_delta

### proc_net_snmp_tcp_retrans_segs_delta

### proc_net_snmp_tcp_in_errs_delta

### proc_net_snmp_tcp_out_rsts_delta

### proc_net_snmp_tcp_in_csum_errors_delta

### proc_net_snmp_udp_in_datagrams_delta

### proc_net_snmp_udp_no_ports_delta

### proc_net_snmp_udp_in_errors_delta

### proc_net_snmp_udp_out_datagrams_delta

### proc_net_snmp_udp_rcvbuf_errors_delta

### proc_net_snmp_udp_sndbuf_errors_delta

### proc_net_snmp_udp_in_csum_errors_delta

### proc_net_snmp_udp_ignored_multi_delta

### proc_net_snmp_udp_mem_errors_delta

### proc_net_snmp_udplite_in_datagrams_delta

### proc_net_snmp_udplite_no_ports_delta

### proc_net_snmp_udplite_in_errors_delta

### proc_net_snmp_udplite_out_datagrams_delta

### proc_net_snmp_udplite_rcvbuf_errors_delta

### proc_net_snmp_udplite_sndbuf_errors_delta

### proc_net_snmp_udplite_in_csum_errors_delta

### proc_net_snmp_udplite_ignored_multi_delta

### proc_net_snmp_udplite_mem_errors_delta

### proc_net_snmp_metrics_delta_sec

Time in seconds since the last scan. The actual value corresponding to the configured desired `interval`.
