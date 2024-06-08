// /proc/net/snmp metrics

package lsvmi

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/emypar/linux-stats-victoriametrics-importer/procfs"
)

const (
	PROC_NET_SNMP_METRICS_CONFIG_INTERVAL_DEFAULT            = "1s"
	PROC_NET_SNMP_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT = 15

	// This generator id:
	PROC_NET_SNMP_METRICS_ID = "proc_net_snmp_metrics"
)

// Metrics definitions:
const (
	PROC_NET_SNMP_IP_FORWARDING_METRIC                 = "proc_net_snmp_ip_forwarding"
	PROC_NET_SNMP_IP_DEFAULT_TTL_METRIC                = "proc_net_snmp_ip_default_ttl"
	PROC_NET_SNMP_IP_IN_RECEIVES_DELTA_METRIC          = "proc_net_snmp_ip_in_receives_delta"
	PROC_NET_SNMP_IP_IN_HDR_ERRORS_DELTA_METRIC        = "proc_net_snmp_ip_in_hdr_errors_delta"
	PROC_NET_SNMP_IP_IN_ADDR_ERRORS_DELTA_METRIC       = "proc_net_snmp_ip_in_addr_errors_delta"
	PROC_NET_SNMP_IP_FORW_DATAGRAMS_DELTA_METRIC       = "proc_net_snmp_ip_forw_datagrams_delta"
	PROC_NET_SNMP_IP_IN_UNKNOWN_PROTOS_DELTA_METRIC    = "proc_net_snmp_ip_in_unknown_protos_delta"
	PROC_NET_SNMP_IP_IN_DISCARDS_DELTA_METRIC          = "proc_net_snmp_ip_in_discards_delta"
	PROC_NET_SNMP_IP_IN_DELIVERS_DELTA_METRIC          = "proc_net_snmp_ip_in_delivers_delta"
	PROC_NET_SNMP_IP_OUT_REQUESTS_DELTA_METRIC         = "proc_net_snmp_ip_out_requests_delta"
	PROC_NET_SNMP_IP_OUT_DISCARDS_DELTA_METRIC         = "proc_net_snmp_ip_out_discards_delta"
	PROC_NET_SNMP_IP_OUT_NO_ROUTES_DELTA_METRIC        = "proc_net_snmp_ip_out_no_routes_delta"
	PROC_NET_SNMP_IP_REASM_TIMEOUT_METRIC              = "proc_net_snmp_ip_reasm_timeout"
	PROC_NET_SNMP_IP_REASM_REQDS_DELTA_METRIC          = "proc_net_snmp_ip_reasm_reqds_delta"
	PROC_NET_SNMP_IP_REASM_OKS_DELTA_METRIC            = "proc_net_snmp_ip_reasm_oks_delta"
	PROC_NET_SNMP_IP_REASM_FAILS_DELTA_METRIC          = "proc_net_snmp_ip_reasm_fails_delta"
	PROC_NET_SNMP_IP_FRAG_OKS_DELTA_METRIC             = "proc_net_snmp_ip_frag_oks_delta"
	PROC_NET_SNMP_IP_FRAG_FAILS_DELTA_METRIC           = "proc_net_snmp_ip_frag_fails_delta"
	PROC_NET_SNMP_IP_FRAG_CREATES_DELTA_METRIC         = "proc_net_snmp_ip_frag_creates_delta"
	PROC_NET_SNMP_ICMP_IN_MSGS_DELTA_METRIC            = "proc_net_snmp_icmp_in_msgs_delta"
	PROC_NET_SNMP_ICMP_IN_ERRORS_DELTA_METRIC          = "proc_net_snmp_icmp_in_errors_delta"
	PROC_NET_SNMP_ICMP_IN_CSUM_ERRORS_DELTA_METRIC     = "proc_net_snmp_icmp_in_csum_errors_delta"
	PROC_NET_SNMP_ICMP_IN_DEST_UNREACHS_DELTA_METRIC   = "proc_net_snmp_icmp_in_dest_unreachs_delta"
	PROC_NET_SNMP_ICMP_IN_TIME_EXCDS_DELTA_METRIC      = "proc_net_snmp_icmp_in_time_excds_delta"
	PROC_NET_SNMP_ICMP_IN_PARM_PROBS_DELTA_METRIC      = "proc_net_snmp_icmp_in_parm_probs_delta"
	PROC_NET_SNMP_ICMP_IN_SRC_QUENCHS_DELTA_METRIC     = "proc_net_snmp_icmp_in_src_quenchs_delta"
	PROC_NET_SNMP_ICMP_IN_REDIRECTS_DELTA_METRIC       = "proc_net_snmp_icmp_in_redirects_delta"
	PROC_NET_SNMP_ICMP_IN_ECHOS_DELTA_METRIC           = "proc_net_snmp_icmp_in_echos_delta"
	PROC_NET_SNMP_ICMP_IN_ECHO_REPS_DELTA_METRIC       = "proc_net_snmp_icmp_in_echo_reps_delta"
	PROC_NET_SNMP_ICMP_IN_TIMESTAMPS_DELTA_METRIC      = "proc_net_snmp_icmp_in_timestamps_delta"
	PROC_NET_SNMP_ICMP_IN_TIMESTAMP_REPS_DELTA_METRIC  = "proc_net_snmp_icmp_in_timestamp_reps_delta"
	PROC_NET_SNMP_ICMP_IN_ADDR_MASKS_DELTA_METRIC      = "proc_net_snmp_icmp_in_addr_masks_delta"
	PROC_NET_SNMP_ICMP_IN_ADDR_MASK_REPS_DELTA_METRIC  = "proc_net_snmp_icmp_in_addr_mask_reps_delta"
	PROC_NET_SNMP_ICMP_OUT_MSGS_DELTA_METRIC           = "proc_net_snmp_icmp_out_msgs_delta"
	PROC_NET_SNMP_ICMP_OUT_ERRORS_DELTA_METRIC         = "proc_net_snmp_icmp_out_errors_delta"
	PROC_NET_SNMP_ICMP_OUT_DEST_UNREACHS_DELTA_METRIC  = "proc_net_snmp_icmp_out_dest_unreachs_delta"
	PROC_NET_SNMP_ICMP_OUT_TIME_EXCDS_DELTA_METRIC     = "proc_net_snmp_icmp_out_time_excds_delta"
	PROC_NET_SNMP_ICMP_OUT_PARM_PROBS_DELTA_METRIC     = "proc_net_snmp_icmp_out_parm_probs_delta"
	PROC_NET_SNMP_ICMP_OUT_SRC_QUENCHS_DELTA_METRIC    = "proc_net_snmp_icmp_out_src_quenchs_delta"
	PROC_NET_SNMP_ICMP_OUT_REDIRECTS_DELTA_METRIC      = "proc_net_snmp_icmp_out_redirects_delta"
	PROC_NET_SNMP_ICMP_OUT_ECHOS_DELTA_METRIC          = "proc_net_snmp_icmp_out_echos_delta"
	PROC_NET_SNMP_ICMP_OUT_ECHO_REPS_DELTA_METRIC      = "proc_net_snmp_icmp_out_echo_reps_delta"
	PROC_NET_SNMP_ICMP_OUT_TIMESTAMPS_DELTA_METRIC     = "proc_net_snmp_icmp_out_timestamps_delta"
	PROC_NET_SNMP_ICMP_OUT_TIMESTAMP_REPS_DELTA_METRIC = "proc_net_snmp_icmp_out_timestamp_reps_delta"
	PROC_NET_SNMP_ICMP_OUT_ADDR_MASKS_DELTA_METRIC     = "proc_net_snmp_icmp_out_addr_masks_delta"
	PROC_NET_SNMP_ICMP_OUT_ADDR_MASK_REPS_DELTA_METRIC = "proc_net_snmp_icmp_out_addr_mask_reps_delta"
	PROC_NET_SNMP_ICMPMSG_IN_TYPE3_DELTA_METRIC        = "proc_net_snmp_icmpmsg_in_type3_delta"
	PROC_NET_SNMP_ICMPMSG_OUT_TYPE3_DELTA_METRIC       = "proc_net_snmp_icmpmsg_out_type3_delta"
	PROC_NET_SNMP_TCP_RTO_ALGORITHM_METRIC             = "proc_net_snmp_tcp_rto_algorithm"
	PROC_NET_SNMP_TCP_RTO_MIN_METRIC                   = "proc_net_snmp_tcp_rto_min"
	PROC_NET_SNMP_TCP_RTO_MAX_METRIC                   = "proc_net_snmp_tcp_rto_max"
	PROC_NET_SNMP_TCP_MAX_CONN_METRIC                  = "proc_net_snmp_tcp_max_conn"
	PROC_NET_SNMP_TCP_ACTIVE_OPENS_DELTA_METRIC        = "proc_net_snmp_tcp_active_opens_delta"
	PROC_NET_SNMP_TCP_PASSIVE_OPENS_DELTA_METRIC       = "proc_net_snmp_tcp_passive_opens_delta"
	PROC_NET_SNMP_TCP_ATTEMPT_FAILS_DELTA_METRIC       = "proc_net_snmp_tcp_attempt_fails_delta"
	PROC_NET_SNMP_TCP_ESTAB_RESETS_DELTA_METRIC        = "proc_net_snmp_tcp_estab_resets_delta"
	PROC_NET_SNMP_TCP_CURR_ESTAB_METRIC                = "proc_net_snmp_tcp_curr_estab"
	PROC_NET_SNMP_TCP_IN_SEGS_DELTA_METRIC             = "proc_net_snmp_tcp_in_segs_delta"
	PROC_NET_SNMP_TCP_OUT_SEGS_DELTA_METRIC            = "proc_net_snmp_tcp_out_segs_delta"
	PROC_NET_SNMP_TCP_RETRANS_SEGS_DELTA_METRIC        = "proc_net_snmp_tcp_retrans_segs_delta"
	PROC_NET_SNMP_TCP_IN_ERRS_DELTA_METRIC             = "proc_net_snmp_tcp_in_errs_delta"
	PROC_NET_SNMP_TCP_OUT_RSTS_DELTA_METRIC            = "proc_net_snmp_tcp_out_rsts_delta"
	PROC_NET_SNMP_TCP_IN_CSUM_ERRORS_DELTA_METRIC      = "proc_net_snmp_tcp_in_csum_errors_delta"
	PROC_NET_SNMP_UDP_IN_DATAGRAMS_DELTA_METRIC        = "proc_net_snmp_udp_in_datagrams_delta"
	PROC_NET_SNMP_UDP_NO_PORTS_DELTA_METRIC            = "proc_net_snmp_udp_no_ports_delta"
	PROC_NET_SNMP_UDP_IN_ERRORS_DELTA_METRIC           = "proc_net_snmp_udp_in_errors_delta"
	PROC_NET_SNMP_UDP_OUT_DATAGRAMS_DELTA_METRIC       = "proc_net_snmp_udp_out_datagrams_delta"
	PROC_NET_SNMP_UDP_RCVBUF_ERRORS_DELTA_METRIC       = "proc_net_snmp_udp_rcvbuf_errors_delta"
	PROC_NET_SNMP_UDP_SNDBUF_ERRORS_DELTA_METRIC       = "proc_net_snmp_udp_sndbuf_errors_delta"
	PROC_NET_SNMP_UDP_IN_CSUM_ERRORS_DELTA_METRIC      = "proc_net_snmp_udp_in_csum_errors_delta"
	PROC_NET_SNMP_UDP_IGNORED_MULTI_DELTA_METRIC       = "proc_net_snmp_udp_ignored_multi_delta"
	PROC_NET_SNMP_UDP_MEM_ERRORS_DELTA_METRIC          = "proc_net_snmp_udp_mem_errors_delta"
	PROC_NET_SNMP_UDPLITE_IN_DATAGRAMS_DELTA_METRIC    = "proc_net_snmp_udplite_in_datagrams_delta"
	PROC_NET_SNMP_UDPLITE_NO_PORTS_DELTA_METRIC        = "proc_net_snmp_udplite_no_ports_delta"
	PROC_NET_SNMP_UDPLITE_IN_ERRORS_DELTA_METRIC       = "proc_net_snmp_udplite_in_errors_delta"
	PROC_NET_SNMP_UDPLITE_OUT_DATAGRAMS_DELTA_METRIC   = "proc_net_snmp_udplite_out_datagrams_delta"
	PROC_NET_SNMP_UDPLITE_RCVBUF_ERRORS_DELTA_METRIC   = "proc_net_snmp_udplite_rcvbuf_errors_delta"
	PROC_NET_SNMP_UDPLITE_SNDBUF_ERRORS_DELTA_METRIC   = "proc_net_snmp_udplite_sndbuf_errors_delta"
	PROC_NET_SNMP_UDPLITE_IN_CSUM_ERRORS_DELTA_METRIC  = "proc_net_snmp_udplite_in_csum_errors_delta"
	PROC_NET_SNMP_UDPLITE_IGNORED_MULTI_DELTA_METRIC   = "proc_net_snmp_udplite_ignored_multi_delta"
	PROC_NET_SNMP_UDPLITE_MEM_ERRORS_DELTA_METRIC      = "proc_net_snmp_udplite_mem_errors_delta"

	// Interval since last generation, i.e. the interval underlying the deltas.
	// Normally this should be close to scan interval, but this is the actual
	// value, rather than the desired one:
	PROC_NET_SNMP_INTERVAL_METRIC_NAME = "proc_net_snmp_metrics_delta_sec"
)

// Rather than having individual metric cycle counter, employ N < number of
// metrics whereby the metric generated from index i will use (i % N) counter.
// This grouping will slightly increase the efficiency, especially if N is a
// power of 2, for fast modulo (%) evaluation.
const (
	PROC_NET_SNMP_CYCLE_COUNTER_EXP  = 4
	PROC_NET_SNMP_CYCLE_COUNTER_NUM  = 1 << PROC_NET_SNMP_CYCLE_COUNTER_EXP
	PROC_NET_SNMP_CYCLE_COUNTER_MASK = PROC_NET_SNMP_CYCLE_COUNTER_NUM - 1
)

// The following stats indexes are not associated w/ delta metrics, their value
// should be published as is:
var procNetSnmpNonDeltaIndex = map[int]bool{
	procfs.NET_SNMP_IP_DEFAULT_TTL:    true,
	procfs.NET_SNMP_IP_FORWARDING:     true,
	procfs.NET_SNMP_IP_REASM_TIMEOUT:  true,
	procfs.NET_SNMP_TCP_CURR_ESTAB:    true,
	procfs.NET_SNMP_TCP_MAX_CONN:      true,
	procfs.NET_SNMP_TCP_RTO_ALGORITHM: true,
	procfs.NET_SNMP_TCP_RTO_MAX:       true,
	procfs.NET_SNMP_TCP_RTO_MIN:       true,
}

// Stats index to metrics name map; indexes not in the map will be ignored:
var procNetSnmpIndexToMetricNameMap = map[int]string{
	procfs.NET_SNMP_IP_FORWARDING:           PROC_NET_SNMP_IP_FORWARDING_METRIC,
	procfs.NET_SNMP_IP_DEFAULT_TTL:          PROC_NET_SNMP_IP_DEFAULT_TTL_METRIC,
	procfs.NET_SNMP_IP_IN_RECEIVES:          PROC_NET_SNMP_IP_IN_RECEIVES_DELTA_METRIC,
	procfs.NET_SNMP_IP_IN_HDR_ERRORS:        PROC_NET_SNMP_IP_IN_HDR_ERRORS_DELTA_METRIC,
	procfs.NET_SNMP_IP_IN_ADDR_ERRORS:       PROC_NET_SNMP_IP_IN_ADDR_ERRORS_DELTA_METRIC,
	procfs.NET_SNMP_IP_FORW_DATAGRAMS:       PROC_NET_SNMP_IP_FORW_DATAGRAMS_DELTA_METRIC,
	procfs.NET_SNMP_IP_IN_UNKNOWN_PROTOS:    PROC_NET_SNMP_IP_IN_UNKNOWN_PROTOS_DELTA_METRIC,
	procfs.NET_SNMP_IP_IN_DISCARDS:          PROC_NET_SNMP_IP_IN_DISCARDS_DELTA_METRIC,
	procfs.NET_SNMP_IP_IN_DELIVERS:          PROC_NET_SNMP_IP_IN_DELIVERS_DELTA_METRIC,
	procfs.NET_SNMP_IP_OUT_REQUESTS:         PROC_NET_SNMP_IP_OUT_REQUESTS_DELTA_METRIC,
	procfs.NET_SNMP_IP_OUT_DISCARDS:         PROC_NET_SNMP_IP_OUT_DISCARDS_DELTA_METRIC,
	procfs.NET_SNMP_IP_OUT_NO_ROUTES:        PROC_NET_SNMP_IP_OUT_NO_ROUTES_DELTA_METRIC,
	procfs.NET_SNMP_IP_REASM_TIMEOUT:        PROC_NET_SNMP_IP_REASM_TIMEOUT_METRIC,
	procfs.NET_SNMP_IP_REASM_REQDS:          PROC_NET_SNMP_IP_REASM_REQDS_DELTA_METRIC,
	procfs.NET_SNMP_IP_REASM_OKS:            PROC_NET_SNMP_IP_REASM_OKS_DELTA_METRIC,
	procfs.NET_SNMP_IP_REASM_FAILS:          PROC_NET_SNMP_IP_REASM_FAILS_DELTA_METRIC,
	procfs.NET_SNMP_IP_FRAG_OKS:             PROC_NET_SNMP_IP_FRAG_OKS_DELTA_METRIC,
	procfs.NET_SNMP_IP_FRAG_FAILS:           PROC_NET_SNMP_IP_FRAG_FAILS_DELTA_METRIC,
	procfs.NET_SNMP_IP_FRAG_CREATES:         PROC_NET_SNMP_IP_FRAG_CREATES_DELTA_METRIC,
	procfs.NET_SNMP_ICMP_IN_MSGS:            PROC_NET_SNMP_ICMP_IN_MSGS_DELTA_METRIC,
	procfs.NET_SNMP_ICMP_IN_ERRORS:          PROC_NET_SNMP_ICMP_IN_ERRORS_DELTA_METRIC,
	procfs.NET_SNMP_ICMP_IN_CSUM_ERRORS:     PROC_NET_SNMP_ICMP_IN_CSUM_ERRORS_DELTA_METRIC,
	procfs.NET_SNMP_ICMP_IN_DEST_UNREACHS:   PROC_NET_SNMP_ICMP_IN_DEST_UNREACHS_DELTA_METRIC,
	procfs.NET_SNMP_ICMP_IN_TIME_EXCDS:      PROC_NET_SNMP_ICMP_IN_TIME_EXCDS_DELTA_METRIC,
	procfs.NET_SNMP_ICMP_IN_PARM_PROBS:      PROC_NET_SNMP_ICMP_IN_PARM_PROBS_DELTA_METRIC,
	procfs.NET_SNMP_ICMP_IN_SRC_QUENCHS:     PROC_NET_SNMP_ICMP_IN_SRC_QUENCHS_DELTA_METRIC,
	procfs.NET_SNMP_ICMP_IN_REDIRECTS:       PROC_NET_SNMP_ICMP_IN_REDIRECTS_DELTA_METRIC,
	procfs.NET_SNMP_ICMP_IN_ECHOS:           PROC_NET_SNMP_ICMP_IN_ECHOS_DELTA_METRIC,
	procfs.NET_SNMP_ICMP_IN_ECHO_REPS:       PROC_NET_SNMP_ICMP_IN_ECHO_REPS_DELTA_METRIC,
	procfs.NET_SNMP_ICMP_IN_TIMESTAMPS:      PROC_NET_SNMP_ICMP_IN_TIMESTAMPS_DELTA_METRIC,
	procfs.NET_SNMP_ICMP_IN_TIMESTAMP_REPS:  PROC_NET_SNMP_ICMP_IN_TIMESTAMP_REPS_DELTA_METRIC,
	procfs.NET_SNMP_ICMP_IN_ADDR_MASKS:      PROC_NET_SNMP_ICMP_IN_ADDR_MASKS_DELTA_METRIC,
	procfs.NET_SNMP_ICMP_IN_ADDR_MASK_REPS:  PROC_NET_SNMP_ICMP_IN_ADDR_MASK_REPS_DELTA_METRIC,
	procfs.NET_SNMP_ICMP_OUT_MSGS:           PROC_NET_SNMP_ICMP_OUT_MSGS_DELTA_METRIC,
	procfs.NET_SNMP_ICMP_OUT_ERRORS:         PROC_NET_SNMP_ICMP_OUT_ERRORS_DELTA_METRIC,
	procfs.NET_SNMP_ICMP_OUT_DEST_UNREACHS:  PROC_NET_SNMP_ICMP_OUT_DEST_UNREACHS_DELTA_METRIC,
	procfs.NET_SNMP_ICMP_OUT_TIME_EXCDS:     PROC_NET_SNMP_ICMP_OUT_TIME_EXCDS_DELTA_METRIC,
	procfs.NET_SNMP_ICMP_OUT_PARM_PROBS:     PROC_NET_SNMP_ICMP_OUT_PARM_PROBS_DELTA_METRIC,
	procfs.NET_SNMP_ICMP_OUT_SRC_QUENCHS:    PROC_NET_SNMP_ICMP_OUT_SRC_QUENCHS_DELTA_METRIC,
	procfs.NET_SNMP_ICMP_OUT_REDIRECTS:      PROC_NET_SNMP_ICMP_OUT_REDIRECTS_DELTA_METRIC,
	procfs.NET_SNMP_ICMP_OUT_ECHOS:          PROC_NET_SNMP_ICMP_OUT_ECHOS_DELTA_METRIC,
	procfs.NET_SNMP_ICMP_OUT_ECHO_REPS:      PROC_NET_SNMP_ICMP_OUT_ECHO_REPS_DELTA_METRIC,
	procfs.NET_SNMP_ICMP_OUT_TIMESTAMPS:     PROC_NET_SNMP_ICMP_OUT_TIMESTAMPS_DELTA_METRIC,
	procfs.NET_SNMP_ICMP_OUT_TIMESTAMP_REPS: PROC_NET_SNMP_ICMP_OUT_TIMESTAMP_REPS_DELTA_METRIC,
	procfs.NET_SNMP_ICMP_OUT_ADDR_MASKS:     PROC_NET_SNMP_ICMP_OUT_ADDR_MASKS_DELTA_METRIC,
	procfs.NET_SNMP_ICMP_OUT_ADDR_MASK_REPS: PROC_NET_SNMP_ICMP_OUT_ADDR_MASK_REPS_DELTA_METRIC,
	procfs.NET_SNMP_ICMPMSG_IN_TYPE3:        PROC_NET_SNMP_ICMPMSG_IN_TYPE3_DELTA_METRIC,
	procfs.NET_SNMP_ICMPMSG_OUT_TYPE3:       PROC_NET_SNMP_ICMPMSG_OUT_TYPE3_DELTA_METRIC,
	procfs.NET_SNMP_TCP_RTO_ALGORITHM:       PROC_NET_SNMP_TCP_RTO_ALGORITHM_METRIC,
	procfs.NET_SNMP_TCP_RTO_MIN:             PROC_NET_SNMP_TCP_RTO_MIN_METRIC,
	procfs.NET_SNMP_TCP_RTO_MAX:             PROC_NET_SNMP_TCP_RTO_MAX_METRIC,
	procfs.NET_SNMP_TCP_MAX_CONN:            PROC_NET_SNMP_TCP_MAX_CONN_METRIC,
	procfs.NET_SNMP_TCP_ACTIVE_OPENS:        PROC_NET_SNMP_TCP_ACTIVE_OPENS_DELTA_METRIC,
	procfs.NET_SNMP_TCP_PASSIVE_OPENS:       PROC_NET_SNMP_TCP_PASSIVE_OPENS_DELTA_METRIC,
	procfs.NET_SNMP_TCP_ATTEMPT_FAILS:       PROC_NET_SNMP_TCP_ATTEMPT_FAILS_DELTA_METRIC,
	procfs.NET_SNMP_TCP_ESTAB_RESETS:        PROC_NET_SNMP_TCP_ESTAB_RESETS_DELTA_METRIC,
	procfs.NET_SNMP_TCP_CURR_ESTAB:          PROC_NET_SNMP_TCP_CURR_ESTAB_METRIC,
	procfs.NET_SNMP_TCP_IN_SEGS:             PROC_NET_SNMP_TCP_IN_SEGS_DELTA_METRIC,
	procfs.NET_SNMP_TCP_OUT_SEGS:            PROC_NET_SNMP_TCP_OUT_SEGS_DELTA_METRIC,
	procfs.NET_SNMP_TCP_RETRANS_SEGS:        PROC_NET_SNMP_TCP_RETRANS_SEGS_DELTA_METRIC,
	procfs.NET_SNMP_TCP_IN_ERRS:             PROC_NET_SNMP_TCP_IN_ERRS_DELTA_METRIC,
	procfs.NET_SNMP_TCP_OUT_RSTS:            PROC_NET_SNMP_TCP_OUT_RSTS_DELTA_METRIC,
	procfs.NET_SNMP_TCP_IN_CSUM_ERRORS:      PROC_NET_SNMP_TCP_IN_CSUM_ERRORS_DELTA_METRIC,
	procfs.NET_SNMP_UDP_IN_DATAGRAMS:        PROC_NET_SNMP_UDP_IN_DATAGRAMS_DELTA_METRIC,
	procfs.NET_SNMP_UDP_NO_PORTS:            PROC_NET_SNMP_UDP_NO_PORTS_DELTA_METRIC,
	procfs.NET_SNMP_UDP_IN_ERRORS:           PROC_NET_SNMP_UDP_IN_ERRORS_DELTA_METRIC,
	procfs.NET_SNMP_UDP_OUT_DATAGRAMS:       PROC_NET_SNMP_UDP_OUT_DATAGRAMS_DELTA_METRIC,
	procfs.NET_SNMP_UDP_RCVBUF_ERRORS:       PROC_NET_SNMP_UDP_RCVBUF_ERRORS_DELTA_METRIC,
	procfs.NET_SNMP_UDP_SNDBUF_ERRORS:       PROC_NET_SNMP_UDP_SNDBUF_ERRORS_DELTA_METRIC,
	procfs.NET_SNMP_UDP_IN_CSUM_ERRORS:      PROC_NET_SNMP_UDP_IN_CSUM_ERRORS_DELTA_METRIC,
	procfs.NET_SNMP_UDP_IGNORED_MULTI:       PROC_NET_SNMP_UDP_IGNORED_MULTI_DELTA_METRIC,
	procfs.NET_SNMP_UDP_MEM_ERRORS:          PROC_NET_SNMP_UDP_MEM_ERRORS_DELTA_METRIC,
	procfs.NET_SNMP_UDPLITE_IN_DATAGRAMS:    PROC_NET_SNMP_UDPLITE_IN_DATAGRAMS_DELTA_METRIC,
	procfs.NET_SNMP_UDPLITE_NO_PORTS:        PROC_NET_SNMP_UDPLITE_NO_PORTS_DELTA_METRIC,
	procfs.NET_SNMP_UDPLITE_IN_ERRORS:       PROC_NET_SNMP_UDPLITE_IN_ERRORS_DELTA_METRIC,
	procfs.NET_SNMP_UDPLITE_OUT_DATAGRAMS:   PROC_NET_SNMP_UDPLITE_OUT_DATAGRAMS_DELTA_METRIC,
	procfs.NET_SNMP_UDPLITE_RCVBUF_ERRORS:   PROC_NET_SNMP_UDPLITE_RCVBUF_ERRORS_DELTA_METRIC,
	procfs.NET_SNMP_UDPLITE_SNDBUF_ERRORS:   PROC_NET_SNMP_UDPLITE_SNDBUF_ERRORS_DELTA_METRIC,
	procfs.NET_SNMP_UDPLITE_IN_CSUM_ERRORS:  PROC_NET_SNMP_UDPLITE_IN_CSUM_ERRORS_DELTA_METRIC,
	procfs.NET_SNMP_UDPLITE_IGNORED_MULTI:   PROC_NET_SNMP_UDPLITE_IGNORED_MULTI_DELTA_METRIC,
	procfs.NET_SNMP_UDPLITE_MEM_ERRORS:      PROC_NET_SNMP_UDPLITE_MEM_ERRORS_DELTA_METRIC,
}

var procNetSnmpMetricsLog = NewCompLogger(PROC_NET_SNMP_METRICS_ID)

type ProcNetSnmpMetricsConfig struct {
	// How often to generate the metrics in time.ParseDuration() format:
	Interval string `yaml:"interval"`
	// Normally metrics are generated only if there is a change in value from
	// the previous scan. However every N cycles the full set is generated. Use
	// 0 to generate full metrics every cycle.
	FullMetricsFactor int `yaml:"full_metrics_factor"`
}

func DefaultProcNetSnmpMetricsConfig() *ProcNetSnmpMetricsConfig {
	return &ProcNetSnmpMetricsConfig{
		Interval:          PROC_NET_SNMP_METRICS_CONFIG_INTERVAL_DEFAULT,
		FullMetricsFactor: PROC_NET_SNMP_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT,
	}
}

type ProcNetSnmpMetrics struct {
	// id/task_id:
	id string
	// Scan interval:
	interval time.Duration
	// Dual storage for parsed stats used as previous, current:
	procNetSnmp [2]*procfs.NetSnmp
	// Timestamp when the stats were collected:
	procNetSnmpTs [2]time.Time
	// Index for current stats, toggled after each use:
	currIndex int
	// Full metric factor:
	fullMetricsFactor int
	// Cycle counters:
	cycleNum []int

	// Metrics cache by stats index:
	metricsCache [][]byte

	// Interval metric:
	intervalMetric []byte

	// Total number of metrics:
	totalMetricsCount int

	// A buffer for the timestamp suffix:
	tsSuffixBuf *bytes.Buffer

	// Delta metrics are generated with skip-zero-after-zero rule, i.e. if the
	// current and previous deltas are both zero, then the current metric is
	// skipped, save for full cycles. Keep track of zero deltas, indexed by
	// counter index (see procfs.NetSnmp.Values)
	zeroDelta []bool

	// The following are needed for testing only. Left to their default values,
	// the usual objects will be used.
	instance, hostname string
	timeNowFn          func() time.Time
	metricsQueue       MetricsQueue
	procfsRoot         string
}

func NewProcNetSnmpMetrics(cfg any) (*ProcNetSnmpMetrics, error) {
	var (
		err                   error
		procNetSnmpMetricsCfg *ProcNetSnmpMetricsConfig
	)

	switch cfg := cfg.(type) {
	case *LsvmiConfig:
		procNetSnmpMetricsCfg = cfg.ProcNetSnmpMetricsConfig
	case *ProcNetSnmpMetricsConfig:
		procNetSnmpMetricsCfg = cfg
	case nil:
		procNetSnmpMetricsCfg = DefaultProcNetSnmpMetricsConfig()
	default:
		return nil, fmt.Errorf("NewProcNetSnmpMetrics: %T invalid config type", cfg)
	}

	interval, err := time.ParseDuration(procNetSnmpMetricsCfg.Interval)
	if err != nil {
		return nil, err
	}
	procNetSnmpMetrics := &ProcNetSnmpMetrics{
		id:                PROC_NET_SNMP_METRICS_ID,
		interval:          interval,
		fullMetricsFactor: procNetSnmpMetricsCfg.FullMetricsFactor,
		cycleNum:          make([]int, PROC_NET_SNMP_CYCLE_COUNTER_NUM),
		zeroDelta:         make([]bool, procfs.NET_SNMP_NUM_VALUES),
		tsSuffixBuf:       &bytes.Buffer{},
	}

	for i := 0; i < len(procNetSnmpMetrics.cycleNum); i++ {
		procNetSnmpMetrics.cycleNum[i] = initialCycleNum.Get(procNetSnmpMetrics.fullMetricsFactor)
	}

	procNetSnmpMetricsLog.Infof("id=%s", procNetSnmpMetrics.id)
	procNetSnmpMetricsLog.Infof("interval=%s", procNetSnmpMetrics.interval)
	procNetSnmpMetricsLog.Infof("full_metrics_factor=%d", procNetSnmpMetrics.fullMetricsFactor)
	return procNetSnmpMetrics, nil
}

func (pnsm *ProcNetSnmpMetrics) updateMetricsCache() {
	instance, hostname := GlobalInstance, GlobalHostname
	if pnsm.instance != "" {
		instance = pnsm.instance
	}
	if pnsm.hostname != "" {
		hostname = pnsm.hostname
	}

	pnsm.metricsCache = make([][]byte, procfs.NET_SNMP_NUM_VALUES)
	pnsm.totalMetricsCount = 1 // for interval metric
	for i := 0; i < len(pnsm.metricsCache); i++ {
		name, ok := procNetSnmpIndexToMetricNameMap[i]
		if ok {
			pnsm.metricsCache[i] = []byte(fmt.Sprintf(
				`%s{%s="%s",%s="%s"} `, // N.B. include whitespace before value!
				name,
				INSTANCE_LABEL_NAME, instance,
				HOSTNAME_LABEL_NAME, hostname,
			))
			pnsm.totalMetricsCount++
		}
	}
}

func (pnsm *ProcNetSnmpMetrics) updateIntervalMetricsCache() {
	instance, hostname := GlobalInstance, GlobalHostname
	if pnsm.instance != "" {
		instance = pnsm.instance
	}
	if pnsm.hostname != "" {
		hostname = pnsm.hostname
	}
	pnsm.intervalMetric = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s"} `, // N.B. include space before val
		PROC_NET_SNMP_INTERVAL_METRIC_NAME,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
	))
}

func (pnsm *ProcNetSnmpMetrics) generateMetrics(buf *bytes.Buffer) (int, int) {
	actualMetricsCount := 0
	currProcNetSnmp, prevProcNetSnmp := pnsm.procNetSnmp[pnsm.currIndex], pnsm.procNetSnmp[1-pnsm.currIndex]

	currValues := currProcNetSnmp.Values
	var prevValues []uint32 = nil
	if prevProcNetSnmp != nil {
		prevValues = prevProcNetSnmp.Values
	}

	currTs := pnsm.procNetSnmpTs[pnsm.currIndex]
	pnsm.tsSuffixBuf.Reset()
	fmt.Fprintf(
		pnsm.tsSuffixBuf, " %d\n", currTs.UnixMilli(),
	)
	promTs := pnsm.tsSuffixBuf.Bytes()

	metricsCache := pnsm.metricsCache
	if metricsCache == nil {
		pnsm.updateMetricsCache()
		metricsCache = pnsm.metricsCache
	}

	zeroDelta := pnsm.zeroDelta
	for index, value := range currValues {
		metric := metricsCache[index]
		if metric == nil {
			// This value is ignored
			continue
		}

		fullCycle := pnsm.cycleNum[index&PROC_NET_SNMP_CYCLE_COUNTER_MASK] == 0

		if procNetSnmpNonDeltaIndex[index] {
			// As-is value:
			if fullCycle || prevValues == nil || value != prevValues[index] {
				buf.Write(metricsCache[index])
				// Some values are in fact signed:
				if procfs.NetSnmpValueMayBeNegative[index] {
					buf.WriteString(strconv.FormatInt(int64(int32(value)), 10))
				} else {
					buf.WriteString(strconv.FormatUint(uint64(value), 10))
				}
				buf.Write(promTs)
				actualMetricsCount++
			}
			continue
		} else if prevValues == nil {
			continue
		}

		// Delta value:
		delta := value - prevValues[index]
		if delta != 0 || fullCycle || !zeroDelta[index] {
			buf.Write(metricsCache[index])
			buf.WriteString(strconv.FormatUint(uint64(delta), 10))
			buf.Write(promTs)
			actualMetricsCount++
		}
		zeroDelta[index] = delta == 0
	}

	if prevProcNetSnmp != nil {
		prevTs := pnsm.procNetSnmpTs[1-pnsm.currIndex]
		deltaSec := currTs.Sub(prevTs).Seconds()

		if pnsm.intervalMetric == nil {
			pnsm.updateIntervalMetricsCache()
		}
		buf.Write(pnsm.intervalMetric)
		buf.WriteString(strconv.FormatFloat(deltaSec, 'f', 6, 64))
		buf.Write(promTs)
		actualMetricsCount++
	}

	// Update cycle counters:
	for i := 0; i < PROC_NET_SNMP_CYCLE_COUNTER_NUM; i++ {
		if pnsm.cycleNum[i]++; pnsm.cycleNum[i] >= pnsm.fullMetricsFactor {
			pnsm.cycleNum[i] = 0
		}
	}

	// Toggle the buffers:
	pnsm.currIndex = 1 - pnsm.currIndex

	return actualMetricsCount, pnsm.totalMetricsCount
}

// Satisfy the TaskActivity interface:
func (pnsm *ProcNetSnmpMetrics) Execute() bool {
	timeNowFn := time.Now
	if pnsm.timeNowFn != nil {
		timeNowFn = pnsm.timeNowFn
	}

	metricsQueue := GlobalMetricsQueue
	if pnsm.metricsQueue != nil {
		metricsQueue = pnsm.metricsQueue
	}

	currProcNetSnmp := pnsm.procNetSnmp[pnsm.currIndex]
	if currProcNetSnmp == nil {
		prevProcNetSnmp := pnsm.procNetSnmp[1-pnsm.currIndex]
		if prevProcNetSnmp != nil {
			currProcNetSnmp = prevProcNetSnmp.Clone(false)
		} else {
			procfsRoot := GlobalProcfsRoot
			if pnsm.procfsRoot != "" {
				procfsRoot = pnsm.procfsRoot
			}
			currProcNetSnmp = procfs.NewNetSnmp(procfsRoot)
		}
		pnsm.procNetSnmp[pnsm.currIndex] = currProcNetSnmp
	}
	err := currProcNetSnmp.Parse()
	if err != nil {
		procNetSnmpMetricsLog.Warnf("%v: proc net snmp metrics will be disabled", err)
		return false
	}
	if currProcNetSnmp.InfoChanged != nil {
		procNetSnmpMetricsLog.Warn(string(currProcNetSnmp.InfoChanged))
		prevProcNetSnmp := pnsm.procNetSnmp[1-pnsm.currIndex]
		if prevProcNetSnmp != nil {
			prevProcNetSnmp.UpdateInfo(currProcNetSnmp)
		}
	}
	pnsm.procNetSnmpTs[pnsm.currIndex] = timeNowFn()

	buf := metricsQueue.GetBuf()
	actualMetricsCount, totalMetricsCount := pnsm.generateMetrics(buf)
	byteCount := buf.Len()
	metricsQueue.QueueBuf(buf)

	GlobalMetricsGeneratorStatsContainer.Update(
		pnsm.id, uint64(actualMetricsCount), uint64(totalMetricsCount), uint64(byteCount),
	)

	return true
}

// Define and register the task builder:
func ProcNetSnmpMetricsTaskBuilder(cfg *LsvmiConfig) ([]*Task, error) {
	pnsm, err := NewProcNetSnmpMetrics(cfg)
	if err != nil {
		return nil, err
	}
	if pnsm.interval <= 0 {
		procNetSnmpMetricsLog.Infof(
			"interval=%s, metrics disabled", pnsm.interval,
		)
		return nil, nil
	}
	tasks := []*Task{
		NewTask(pnsm.id, pnsm.interval, pnsm),
	}
	return tasks, nil
}

func init() {
	TaskBuilders.Register(ProcNetSnmpMetricsTaskBuilder)
}
