// /proc/net/snmp6 metrics

package lsvmi

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/emypar/linux-stats-victoriametrics-importer/procfs"
)

const (
	PROC_NET_SNMP6_METRICS_CONFIG_INTERVAL_DEFAULT            = "1s"
	PROC_NET_SNMP6_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT = 15

	// This generator id:
	PROC_NET_SNMP6_METRICS_ID = "proc_net_snmp6_metrics"
)

// Metrics definitions:
const (
	PROC_NET_SNMP6_IP6_IN_RECEIVES_DELTA_METRIC                   = "proc_net_snmp6_ip6_in_receives_delta"
	PROC_NET_SNMP6_IP6_IN_HDR_ERRORS_DELTA_METRIC                 = "proc_net_snmp6_ip6_in_hdr_errors_delta"
	PROC_NET_SNMP6_IP6_IN_TOO_BIG_ERRORS_DELTA_METRIC             = "proc_net_snmp6_ip6_in_too_big_errors_delta"
	PROC_NET_SNMP6_IP6_IN_NO_ROUTES_DELTA_METRIC                  = "proc_net_snmp6_ip6_in_no_routes_delta"
	PROC_NET_SNMP6_IP6_IN_ADDR_ERRORS_DELTA_METRIC                = "proc_net_snmp6_ip6_in_addr_errors_delta"
	PROC_NET_SNMP6_IP6_IN_UNKNOWN_PROTOS_DELTA_METRIC             = "proc_net_snmp6_ip6_in_unknown_protos_delta"
	PROC_NET_SNMP6_IP6_IN_TRUNCATED_PKTS_DELTA_METRIC             = "proc_net_snmp6_ip6_in_truncated_pkts_delta"
	PROC_NET_SNMP6_IP6_IN_DISCARDS_DELTA_METRIC                   = "proc_net_snmp6_ip6_in_discards_delta"
	PROC_NET_SNMP6_IP6_IN_DELIVERS_DELTA_METRIC                   = "proc_net_snmp6_ip6_in_delivers_delta"
	PROC_NET_SNMP6_IP6_OUT_FORW_DATAGRAMS_DELTA_METRIC            = "proc_net_snmp6_ip6_out_forw_datagrams_delta"
	PROC_NET_SNMP6_IP6_OUT_REQUESTS_DELTA_METRIC                  = "proc_net_snmp6_ip6_out_requests_delta"
	PROC_NET_SNMP6_IP6_OUT_DISCARDS_DELTA_METRIC                  = "proc_net_snmp6_ip6_out_discards_delta"
	PROC_NET_SNMP6_IP6_OUT_NO_ROUTES_DELTA_METRIC                 = "proc_net_snmp6_ip6_out_no_routes_delta"
	PROC_NET_SNMP6_IP6_REASM_TIMEOUT_DELTA_METRIC                 = "proc_net_snmp6_ip6_reasm_timeout_delta"
	PROC_NET_SNMP6_IP6_REASM_REQDS_DELTA_METRIC                   = "proc_net_snmp6_ip6_reasm_reqds_delta"
	PROC_NET_SNMP6_IP6_REASM_OKS_DELTA_METRIC                     = "proc_net_snmp6_ip6_reasm_oks_delta"
	PROC_NET_SNMP6_IP6_REASM_FAILS_DELTA_METRIC                   = "proc_net_snmp6_ip6_reasm_fails_delta"
	PROC_NET_SNMP6_IP6_FRAG_OKS_DELTA_METRIC                      = "proc_net_snmp6_ip6_frag_oks_delta"
	PROC_NET_SNMP6_IP6_FRAG_FAILS_DELTA_METRIC                    = "proc_net_snmp6_ip6_frag_fails_delta"
	PROC_NET_SNMP6_IP6_FRAG_CREATES_DELTA_METRIC                  = "proc_net_snmp6_ip6_frag_creates_delta"
	PROC_NET_SNMP6_IP6_IN_MCAST_PKTS_DELTA_METRIC                 = "proc_net_snmp6_ip6_in_mcast_pkts_delta"
	PROC_NET_SNMP6_IP6_OUT_MCAST_PKTS_DELTA_METRIC                = "proc_net_snmp6_ip6_out_mcast_pkts_delta"
	PROC_NET_SNMP6_IP6_IN_KBPS_METRIC                             = "proc_net_snmp6_ip6_in_kbps"
	PROC_NET_SNMP6_IP6_OUT_KBPS_METRIC                            = "proc_net_snmp6_ip6_out_kbps"
	PROC_NET_SNMP6_IP6_IN_MCAST_KBPS_METRIC                       = "proc_net_snmp6_ip6_in_mcast_kbps"
	PROC_NET_SNMP6_IP6_OUT_MCAST_KBPS_METRIC                      = "proc_net_snmp6_ip6_out_mcast_kbps"
	PROC_NET_SNMP6_IP6_IN_BCAST_KBPS_METRIC                       = "proc_net_snmp6_ip6_in_bcast_kbps"
	PROC_NET_SNMP6_IP6_OUT_BCAST_KBPS_METRIC                      = "proc_net_snmp6_ip6_out_bcast_kbps"
	PROC_NET_SNMP6_IP6_IN_NO_ECT_PKTS_DELTA_METRIC                = "proc_net_snmp6_ip6_in_no_ect_pkts_delta"
	PROC_NET_SNMP6_IP6_IN_ECT1_PKTS_DELTA_METRIC                  = "proc_net_snmp6_ip6_in_ect1_pkts_delta"
	PROC_NET_SNMP6_IP6_IN_ECT0_PKTS_DELTA_METRIC                  = "proc_net_snmp6_ip6_in_ect0_pkts_delta"
	PROC_NET_SNMP6_IP6_IN_CE_PKTS_DELTA_METRIC                    = "proc_net_snmp6_ip6_in_ce_pkts_delta"
	PROC_NET_SNMP6_ICMP6_IN_MSGS_DELTA_METRIC                     = "proc_net_snmp6_icmp6_in_msgs_delta"
	PROC_NET_SNMP6_ICMP6_IN_ERRORS_DELTA_METRIC                   = "proc_net_snmp6_icmp6_in_errors_delta"
	PROC_NET_SNMP6_ICMP6_OUT_MSGS_DELTA_METRIC                    = "proc_net_snmp6_icmp6_out_msgs_delta"
	PROC_NET_SNMP6_ICMP6_OUT_ERRORS_DELTA_METRIC                  = "proc_net_snmp6_icmp6_out_errors_delta"
	PROC_NET_SNMP6_ICMP6_IN_CSUM_ERRORS_DELTA_METRIC              = "proc_net_snmp6_icmp6_in_csum_errors_delta"
	PROC_NET_SNMP6_ICMP6_IN_DEST_UNREACHS_DELTA_METRIC            = "proc_net_snmp6_icmp6_in_dest_unreachs_delta"
	PROC_NET_SNMP6_ICMP6_IN_PKT_TOO_BIGS_DELTA_METRIC             = "proc_net_snmp6_icmp6_in_pkt_too_bigs_delta"
	PROC_NET_SNMP6_ICMP6_IN_TIME_EXCDS_DELTA_METRIC               = "proc_net_snmp6_icmp6_in_time_excds_delta"
	PROC_NET_SNMP6_ICMP6_IN_PARM_PROBLEMS_DELTA_METRIC            = "proc_net_snmp6_icmp6_in_parm_problems_delta"
	PROC_NET_SNMP6_ICMP6_IN_ECHOS_DELTA_METRIC                    = "proc_net_snmp6_icmp6_in_echos_delta"
	PROC_NET_SNMP6_ICMP6_IN_ECHO_REPLIES_DELTA_METRIC             = "proc_net_snmp6_icmp6_in_echo_replies_delta"
	PROC_NET_SNMP6_ICMP6_IN_GROUP_MEMB_QUERIES_DELTA_METRIC       = "proc_net_snmp6_icmp6_in_group_memb_queries_delta"
	PROC_NET_SNMP6_ICMP6_IN_GROUP_MEMB_RESPONSES_DELTA_METRIC     = "proc_net_snmp6_icmp6_in_group_memb_responses_delta"
	PROC_NET_SNMP6_ICMP6_IN_GROUP_MEMB_REDUCTIONS_DELTA_METRIC    = "proc_net_snmp6_icmp6_in_group_memb_reductions_delta"
	PROC_NET_SNMP6_ICMP6_IN_ROUTER_SOLICITS_DELTA_METRIC          = "proc_net_snmp6_icmp6_in_router_solicits_delta"
	PROC_NET_SNMP6_ICMP6_IN_ROUTER_ADVERTISEMENTS_DELTA_METRIC    = "proc_net_snmp6_icmp6_in_router_advertisements_delta"
	PROC_NET_SNMP6_ICMP6_IN_NEIGHBOR_SOLICITS_DELTA_METRIC        = "proc_net_snmp6_icmp6_in_neighbor_solicits_delta"
	PROC_NET_SNMP6_ICMP6_IN_NEIGHBOR_ADVERTISEMENTS_DELTA_METRIC  = "proc_net_snmp6_icmp6_in_neighbor_advertisements_delta"
	PROC_NET_SNMP6_ICMP6_IN_REDIRECTS_DELTA_METRIC                = "proc_net_snmp6_icmp6_in_redirects_delta"
	PROC_NET_SNMP6_ICMP6_IN_MLD_V2_REPORTS_DELTA_METRIC           = "proc_net_snmp6_icmp6_in_mld_v2_reports_delta"
	PROC_NET_SNMP6_ICMP6_OUT_DEST_UNREACHS_DELTA_METRIC           = "proc_net_snmp6_icmp6_out_dest_unreachs_delta"
	PROC_NET_SNMP6_ICMP6_OUT_PKT_TOO_BIGS_DELTA_METRIC            = "proc_net_snmp6_icmp6_out_pkt_too_bigs_delta"
	PROC_NET_SNMP6_ICMP6_OUT_TIME_EXCDS_DELTA_METRIC              = "proc_net_snmp6_icmp6_out_time_excds_delta"
	PROC_NET_SNMP6_ICMP6_OUT_PARM_PROBLEMS_DELTA_METRIC           = "proc_net_snmp6_icmp6_out_parm_problems_delta"
	PROC_NET_SNMP6_ICMP6_OUT_ECHOS_DELTA_METRIC                   = "proc_net_snmp6_icmp6_out_echos_delta"
	PROC_NET_SNMP6_ICMP6_OUT_ECHO_REPLIES_DELTA_METRIC            = "proc_net_snmp6_icmp6_out_echo_replies_delta"
	PROC_NET_SNMP6_ICMP6_OUT_GROUP_MEMB_QUERIES_DELTA_METRIC      = "proc_net_snmp6_icmp6_out_group_memb_queries_delta"
	PROC_NET_SNMP6_ICMP6_OUT_GROUP_MEMB_RESPONSES_DELTA_METRIC    = "proc_net_snmp6_icmp6_out_group_memb_responses_delta"
	PROC_NET_SNMP6_ICMP6_OUT_GROUP_MEMB_REDUCTIONS_DELTA_METRIC   = "proc_net_snmp6_icmp6_out_group_memb_reductions_delta"
	PROC_NET_SNMP6_ICMP6_OUT_ROUTER_SOLICITS_DELTA_METRIC         = "proc_net_snmp6_icmp6_out_router_solicits_delta"
	PROC_NET_SNMP6_ICMP6_OUT_ROUTER_ADVERTISEMENTS_DELTA_METRIC   = "proc_net_snmp6_icmp6_out_router_advertisements_delta"
	PROC_NET_SNMP6_ICMP6_OUT_NEIGHBOR_SOLICITS_DELTA_METRIC       = "proc_net_snmp6_icmp6_out_neighbor_solicits_delta"
	PROC_NET_SNMP6_ICMP6_OUT_NEIGHBOR_ADVERTISEMENTS_DELTA_METRIC = "proc_net_snmp6_icmp6_out_neighbor_advertisements_delta"
	PROC_NET_SNMP6_ICMP6_OUT_REDIRECTS_DELTA_METRIC               = "proc_net_snmp6_icmp6_out_redirects_delta"
	PROC_NET_SNMP6_ICMP6_OUT_MLD_V2_REPORTS_DELTA_METRIC          = "proc_net_snmp6_icmp6_out_mld_v2_reports_delta"
	PROC_NET_SNMP6_ICMP6_OUT_TYPE133_DELTA_METRIC                 = "proc_net_snmp6_icmp6_out_type133_delta"
	PROC_NET_SNMP6_ICMP6_OUT_TYPE135_DELTA_METRIC                 = "proc_net_snmp6_icmp6_out_type135_delta"
	PROC_NET_SNMP6_ICMP6_OUT_TYPE143_DELTA_METRIC                 = "proc_net_snmp6_icmp6_out_type143_delta"
	PROC_NET_SNMP6_UDP6_IN_DATAGRAMS_DELTA_METRIC                 = "proc_net_snmp6_udp6_in_datagrams_delta"
	PROC_NET_SNMP6_UDP6_NO_PORTS_DELTA_METRIC                     = "proc_net_snmp6_udp6_no_ports_delta"
	PROC_NET_SNMP6_UDP6_IN_ERRORS_DELTA_METRIC                    = "proc_net_snmp6_udp6_in_errors_delta"
	PROC_NET_SNMP6_UDP6_OUT_DATAGRAMS_DELTA_METRIC                = "proc_net_snmp6_udp6_out_datagrams_delta"
	PROC_NET_SNMP6_UDP6_RCVBUF_ERRORS_DELTA_METRIC                = "proc_net_snmp6_udp6_rcvbuf_errors_delta"
	PROC_NET_SNMP6_UDP6_SNDBUF_ERRORS_DELTA_METRIC                = "proc_net_snmp6_udp6_sndbuf_errors_delta"
	PROC_NET_SNMP6_UDP6_IN_CSUM_ERRORS_DELTA_METRIC               = "proc_net_snmp6_udp6_in_csum_errors_delta"
	PROC_NET_SNMP6_UDP6_IGNORED_MULTI_DELTA_METRIC                = "proc_net_snmp6_udp6_ignored_multi_delta"
	PROC_NET_SNMP6_UDP6_MEM_ERRORS_DELTA_METRIC                   = "proc_net_snmp6_udp6_mem_errors_delta"
	PROC_NET_SNMP6_UDPLITE6_IN_DATAGRAMS_DELTA_METRIC             = "proc_net_snmp6_udplite6_in_datagrams_delta"
	PROC_NET_SNMP6_UDPLITE6_NO_PORTS_DELTA_METRIC                 = "proc_net_snmp6_udplite6_no_ports_delta"
	PROC_NET_SNMP6_UDPLITE6_IN_ERRORS_DELTA_METRIC                = "proc_net_snmp6_udplite6_in_errors_delta"
	PROC_NET_SNMP6_UDPLITE6_OUT_DATAGRAMS_DELTA_METRIC            = "proc_net_snmp6_udplite6_out_datagrams_delta"
	PROC_NET_SNMP6_UDPLITE6_RCVBUF_ERRORS_DELTA_METRIC            = "proc_net_snmp6_udplite6_rcvbuf_errors_delta"
	PROC_NET_SNMP6_UDPLITE6_SNDBUF_ERRORS_DELTA_METRIC            = "proc_net_snmp6_udplite6_sndbuf_errors_delta"
	PROC_NET_SNMP6_UDPLITE6_IN_CSUM_ERRORS_DELTA_METRIC           = "proc_net_snmp6_udplite6_in_csum_errors_delta"
	PROC_NET_SNMP6_UDPLITE6_MEM_ERRORS_DELTA_METRIC               = "proc_net_snmp6_udplite6_mem_errors_delta"

	// Interval since last generation, i.e. the interval underlying the deltas.
	// Normally this should be close to scan interval, but this is the actual
	// value, rather than the desired one:
	PROC_NET_SNMP6_INTERVAL_METRIC = "proc_net_snmp6_metrics_delta_sec"
)

// Certain values are used to generate rates:
type ProcNetSnmp6Rate struct {
	factor float64 // dVal/dTime * factor
	prec   int     // FormatFloat prec arg
}

var procNetSnmp6IndexRate = [procfs.NET_SNMP6_NUM_VALUES]*ProcNetSnmp6Rate{
	procfs.NET_SNMP6_IP6_IN_OCTETS:        {8. / 1000., 1},
	procfs.NET_SNMP6_IP6_OUT_OCTETS:       {8. / 1000., 1},
	procfs.NET_SNMP6_IP6_IN_MCAST_OCTETS:  {8. / 1000., 1},
	procfs.NET_SNMP6_IP6_OUT_MCAST_OCTETS: {8. / 1000., 1},
	procfs.NET_SNMP6_IP6_IN_BCAST_OCTETS:  {8. / 1000., 1},
	procfs.NET_SNMP6_IP6_OUT_BCAST_OCTETS: {8. / 1000., 1},
}

// Rather than having individual metric cycle counter, employ N < number of
// metrics whereby the metric generated from index i will use (i % N) counter.
// This grouping will slightly increase the efficiency, especially if N is a
// power of 2, for fast modulo (%) evaluation.
const (
	PROC_NET_SNMP6_CYCLE_COUNTER_EXP  = 4
	PROC_NET_SNMP6_CYCLE_COUNTER_NUM  = 1 << PROC_NET_SNMP6_CYCLE_COUNTER_EXP
	PROC_NET_SNMP6_CYCLE_COUNTER_MASK = PROC_NET_SNMP6_CYCLE_COUNTER_NUM - 1
)

// Stats index to metrics name map; indexes not in the map will be ignored:
var procNetSnmp6IndexToMetricNameMap = map[int]string{
	procfs.NET_SNMP6_IP6_IN_RECEIVES:                   PROC_NET_SNMP6_IP6_IN_RECEIVES_DELTA_METRIC,
	procfs.NET_SNMP6_IP6_IN_HDR_ERRORS:                 PROC_NET_SNMP6_IP6_IN_HDR_ERRORS_DELTA_METRIC,
	procfs.NET_SNMP6_IP6_IN_TOO_BIG_ERRORS:             PROC_NET_SNMP6_IP6_IN_TOO_BIG_ERRORS_DELTA_METRIC,
	procfs.NET_SNMP6_IP6_IN_NO_ROUTES:                  PROC_NET_SNMP6_IP6_IN_NO_ROUTES_DELTA_METRIC,
	procfs.NET_SNMP6_IP6_IN_ADDR_ERRORS:                PROC_NET_SNMP6_IP6_IN_ADDR_ERRORS_DELTA_METRIC,
	procfs.NET_SNMP6_IP6_IN_UNKNOWN_PROTOS:             PROC_NET_SNMP6_IP6_IN_UNKNOWN_PROTOS_DELTA_METRIC,
	procfs.NET_SNMP6_IP6_IN_TRUNCATED_PKTS:             PROC_NET_SNMP6_IP6_IN_TRUNCATED_PKTS_DELTA_METRIC,
	procfs.NET_SNMP6_IP6_IN_DISCARDS:                   PROC_NET_SNMP6_IP6_IN_DISCARDS_DELTA_METRIC,
	procfs.NET_SNMP6_IP6_IN_DELIVERS:                   PROC_NET_SNMP6_IP6_IN_DELIVERS_DELTA_METRIC,
	procfs.NET_SNMP6_IP6_OUT_FORW_DATAGRAMS:            PROC_NET_SNMP6_IP6_OUT_FORW_DATAGRAMS_DELTA_METRIC,
	procfs.NET_SNMP6_IP6_OUT_REQUESTS:                  PROC_NET_SNMP6_IP6_OUT_REQUESTS_DELTA_METRIC,
	procfs.NET_SNMP6_IP6_OUT_DISCARDS:                  PROC_NET_SNMP6_IP6_OUT_DISCARDS_DELTA_METRIC,
	procfs.NET_SNMP6_IP6_OUT_NO_ROUTES:                 PROC_NET_SNMP6_IP6_OUT_NO_ROUTES_DELTA_METRIC,
	procfs.NET_SNMP6_IP6_REASM_TIMEOUT:                 PROC_NET_SNMP6_IP6_REASM_TIMEOUT_DELTA_METRIC,
	procfs.NET_SNMP6_IP6_REASM_REQDS:                   PROC_NET_SNMP6_IP6_REASM_REQDS_DELTA_METRIC,
	procfs.NET_SNMP6_IP6_REASM_OKS:                     PROC_NET_SNMP6_IP6_REASM_OKS_DELTA_METRIC,
	procfs.NET_SNMP6_IP6_REASM_FAILS:                   PROC_NET_SNMP6_IP6_REASM_FAILS_DELTA_METRIC,
	procfs.NET_SNMP6_IP6_FRAG_OKS:                      PROC_NET_SNMP6_IP6_FRAG_OKS_DELTA_METRIC,
	procfs.NET_SNMP6_IP6_FRAG_FAILS:                    PROC_NET_SNMP6_IP6_FRAG_FAILS_DELTA_METRIC,
	procfs.NET_SNMP6_IP6_FRAG_CREATES:                  PROC_NET_SNMP6_IP6_FRAG_CREATES_DELTA_METRIC,
	procfs.NET_SNMP6_IP6_IN_MCAST_PKTS:                 PROC_NET_SNMP6_IP6_IN_MCAST_PKTS_DELTA_METRIC,
	procfs.NET_SNMP6_IP6_OUT_MCAST_PKTS:                PROC_NET_SNMP6_IP6_OUT_MCAST_PKTS_DELTA_METRIC,
	procfs.NET_SNMP6_IP6_IN_OCTETS:                     PROC_NET_SNMP6_IP6_IN_KBPS_METRIC,
	procfs.NET_SNMP6_IP6_OUT_OCTETS:                    PROC_NET_SNMP6_IP6_OUT_KBPS_METRIC,
	procfs.NET_SNMP6_IP6_IN_MCAST_OCTETS:               PROC_NET_SNMP6_IP6_IN_MCAST_KBPS_METRIC,
	procfs.NET_SNMP6_IP6_OUT_MCAST_OCTETS:              PROC_NET_SNMP6_IP6_OUT_MCAST_KBPS_METRIC,
	procfs.NET_SNMP6_IP6_IN_BCAST_OCTETS:               PROC_NET_SNMP6_IP6_IN_BCAST_KBPS_METRIC,
	procfs.NET_SNMP6_IP6_OUT_BCAST_OCTETS:              PROC_NET_SNMP6_IP6_OUT_BCAST_KBPS_METRIC,
	procfs.NET_SNMP6_IP6_IN_NO_ECT_PKTS:                PROC_NET_SNMP6_IP6_IN_NO_ECT_PKTS_DELTA_METRIC,
	procfs.NET_SNMP6_IP6_IN_ECT1_PKTS:                  PROC_NET_SNMP6_IP6_IN_ECT1_PKTS_DELTA_METRIC,
	procfs.NET_SNMP6_IP6_IN_ECT0_PKTS:                  PROC_NET_SNMP6_IP6_IN_ECT0_PKTS_DELTA_METRIC,
	procfs.NET_SNMP6_IP6_IN_CE_PKTS:                    PROC_NET_SNMP6_IP6_IN_CE_PKTS_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_IN_MSGS:                     PROC_NET_SNMP6_ICMP6_IN_MSGS_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_IN_ERRORS:                   PROC_NET_SNMP6_ICMP6_IN_ERRORS_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_OUT_MSGS:                    PROC_NET_SNMP6_ICMP6_OUT_MSGS_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_OUT_ERRORS:                  PROC_NET_SNMP6_ICMP6_OUT_ERRORS_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_IN_CSUM_ERRORS:              PROC_NET_SNMP6_ICMP6_IN_CSUM_ERRORS_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_IN_DEST_UNREACHS:            PROC_NET_SNMP6_ICMP6_IN_DEST_UNREACHS_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_IN_PKT_TOO_BIGS:             PROC_NET_SNMP6_ICMP6_IN_PKT_TOO_BIGS_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_IN_TIME_EXCDS:               PROC_NET_SNMP6_ICMP6_IN_TIME_EXCDS_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_IN_PARM_PROBLEMS:            PROC_NET_SNMP6_ICMP6_IN_PARM_PROBLEMS_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_IN_ECHOS:                    PROC_NET_SNMP6_ICMP6_IN_ECHOS_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_IN_ECHO_REPLIES:             PROC_NET_SNMP6_ICMP6_IN_ECHO_REPLIES_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_IN_GROUP_MEMB_QUERIES:       PROC_NET_SNMP6_ICMP6_IN_GROUP_MEMB_QUERIES_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_IN_GROUP_MEMB_RESPONSES:     PROC_NET_SNMP6_ICMP6_IN_GROUP_MEMB_RESPONSES_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_IN_GROUP_MEMB_REDUCTIONS:    PROC_NET_SNMP6_ICMP6_IN_GROUP_MEMB_REDUCTIONS_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_IN_ROUTER_SOLICITS:          PROC_NET_SNMP6_ICMP6_IN_ROUTER_SOLICITS_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_IN_ROUTER_ADVERTISEMENTS:    PROC_NET_SNMP6_ICMP6_IN_ROUTER_ADVERTISEMENTS_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_IN_NEIGHBOR_SOLICITS:        PROC_NET_SNMP6_ICMP6_IN_NEIGHBOR_SOLICITS_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_IN_NEIGHBOR_ADVERTISEMENTS:  PROC_NET_SNMP6_ICMP6_IN_NEIGHBOR_ADVERTISEMENTS_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_IN_REDIRECTS:                PROC_NET_SNMP6_ICMP6_IN_REDIRECTS_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_IN_MLD_V2_REPORTS:           PROC_NET_SNMP6_ICMP6_IN_MLD_V2_REPORTS_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_OUT_DEST_UNREACHS:           PROC_NET_SNMP6_ICMP6_OUT_DEST_UNREACHS_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_OUT_PKT_TOO_BIGS:            PROC_NET_SNMP6_ICMP6_OUT_PKT_TOO_BIGS_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_OUT_TIME_EXCDS:              PROC_NET_SNMP6_ICMP6_OUT_TIME_EXCDS_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_OUT_PARM_PROBLEMS:           PROC_NET_SNMP6_ICMP6_OUT_PARM_PROBLEMS_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_OUT_ECHOS:                   PROC_NET_SNMP6_ICMP6_OUT_ECHOS_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_OUT_ECHO_REPLIES:            PROC_NET_SNMP6_ICMP6_OUT_ECHO_REPLIES_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_OUT_GROUP_MEMB_QUERIES:      PROC_NET_SNMP6_ICMP6_OUT_GROUP_MEMB_QUERIES_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_OUT_GROUP_MEMB_RESPONSES:    PROC_NET_SNMP6_ICMP6_OUT_GROUP_MEMB_RESPONSES_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_OUT_GROUP_MEMB_REDUCTIONS:   PROC_NET_SNMP6_ICMP6_OUT_GROUP_MEMB_REDUCTIONS_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_OUT_ROUTER_SOLICITS:         PROC_NET_SNMP6_ICMP6_OUT_ROUTER_SOLICITS_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_OUT_ROUTER_ADVERTISEMENTS:   PROC_NET_SNMP6_ICMP6_OUT_ROUTER_ADVERTISEMENTS_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_OUT_NEIGHBOR_SOLICITS:       PROC_NET_SNMP6_ICMP6_OUT_NEIGHBOR_SOLICITS_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_OUT_NEIGHBOR_ADVERTISEMENTS: PROC_NET_SNMP6_ICMP6_OUT_NEIGHBOR_ADVERTISEMENTS_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_OUT_REDIRECTS:               PROC_NET_SNMP6_ICMP6_OUT_REDIRECTS_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_OUT_MLD_V2_REPORTS:          PROC_NET_SNMP6_ICMP6_OUT_MLD_V2_REPORTS_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_OUT_TYPE133:                 PROC_NET_SNMP6_ICMP6_OUT_TYPE133_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_OUT_TYPE135:                 PROC_NET_SNMP6_ICMP6_OUT_TYPE135_DELTA_METRIC,
	procfs.NET_SNMP6_ICMP6_OUT_TYPE143:                 PROC_NET_SNMP6_ICMP6_OUT_TYPE143_DELTA_METRIC,
	procfs.NET_SNMP6_UDP6_IN_DATAGRAMS:                 PROC_NET_SNMP6_UDP6_IN_DATAGRAMS_DELTA_METRIC,
	procfs.NET_SNMP6_UDP6_NO_PORTS:                     PROC_NET_SNMP6_UDP6_NO_PORTS_DELTA_METRIC,
	procfs.NET_SNMP6_UDP6_IN_ERRORS:                    PROC_NET_SNMP6_UDP6_IN_ERRORS_DELTA_METRIC,
	procfs.NET_SNMP6_UDP6_OUT_DATAGRAMS:                PROC_NET_SNMP6_UDP6_OUT_DATAGRAMS_DELTA_METRIC,
	procfs.NET_SNMP6_UDP6_RCVBUF_ERRORS:                PROC_NET_SNMP6_UDP6_RCVBUF_ERRORS_DELTA_METRIC,
	procfs.NET_SNMP6_UDP6_SNDBUF_ERRORS:                PROC_NET_SNMP6_UDP6_SNDBUF_ERRORS_DELTA_METRIC,
	procfs.NET_SNMP6_UDP6_IN_CSUM_ERRORS:               PROC_NET_SNMP6_UDP6_IN_CSUM_ERRORS_DELTA_METRIC,
	procfs.NET_SNMP6_UDP6_IGNORED_MULTI:                PROC_NET_SNMP6_UDP6_IGNORED_MULTI_DELTA_METRIC,
	procfs.NET_SNMP6_UDP6_MEM_ERRORS:                   PROC_NET_SNMP6_UDP6_MEM_ERRORS_DELTA_METRIC,
	procfs.NET_SNMP6_UDPLITE6_IN_DATAGRAMS:             PROC_NET_SNMP6_UDPLITE6_IN_DATAGRAMS_DELTA_METRIC,
	procfs.NET_SNMP6_UDPLITE6_NO_PORTS:                 PROC_NET_SNMP6_UDPLITE6_NO_PORTS_DELTA_METRIC,
	procfs.NET_SNMP6_UDPLITE6_IN_ERRORS:                PROC_NET_SNMP6_UDPLITE6_IN_ERRORS_DELTA_METRIC,
	procfs.NET_SNMP6_UDPLITE6_OUT_DATAGRAMS:            PROC_NET_SNMP6_UDPLITE6_OUT_DATAGRAMS_DELTA_METRIC,
	procfs.NET_SNMP6_UDPLITE6_RCVBUF_ERRORS:            PROC_NET_SNMP6_UDPLITE6_RCVBUF_ERRORS_DELTA_METRIC,
	procfs.NET_SNMP6_UDPLITE6_SNDBUF_ERRORS:            PROC_NET_SNMP6_UDPLITE6_SNDBUF_ERRORS_DELTA_METRIC,
	procfs.NET_SNMP6_UDPLITE6_IN_CSUM_ERRORS:           PROC_NET_SNMP6_UDPLITE6_IN_CSUM_ERRORS_DELTA_METRIC,
	procfs.NET_SNMP6_UDPLITE6_MEM_ERRORS:               PROC_NET_SNMP6_UDPLITE6_MEM_ERRORS_DELTA_METRIC,
}

var procNetSnmp6MetricsLog = NewCompLogger(PROC_NET_SNMP6_METRICS_ID)

type ProcNetSnmp6MetricsConfig struct {
	// How often to generate the metrics in time.ParseDuration() format:
	Interval string `yaml:"interval"`
	// Normally metrics are generated only if there is a change in value from
	// the previous scan. However every N cycles the full set is generated. Use
	// 0 to generate full metrics every cycle.
	FullMetricsFactor int `yaml:"full_metrics_factor"`
}

func DefaultProcNetSnmp6MetricsConfig() *ProcNetSnmp6MetricsConfig {
	return &ProcNetSnmp6MetricsConfig{
		Interval:          PROC_NET_SNMP6_METRICS_CONFIG_INTERVAL_DEFAULT,
		FullMetricsFactor: PROC_NET_SNMP6_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT,
	}
}

type ProcNetSnmp6Metrics struct {
	// id/task_id:
	id string
	// Scan interval:
	interval time.Duration
	// Dual storage for parsed stats used as previous, current:
	procNetSnmp6 [2]*procfs.NetSnmp6
	// Timestamp when the stats were collected:
	procNetSnmp6Ts [2]time.Time
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
	// counter index (see procfs.NetSnmp6.Values)
	zeroDelta []bool

	// The following are needed for testing only. Left to their default values,
	// the usual objects will be used.
	instance, hostname string
	timeNowFn          func() time.Time
	metricsQueue       MetricsQueue
	procfsRoot         string
}

func NewProcNetSnmp6Metrics(cfg any) (*ProcNetSnmp6Metrics, error) {
	var (
		err                    error
		procNetSnmp6MetricsCfg *ProcNetSnmp6MetricsConfig
	)

	switch cfg := cfg.(type) {
	case *LsvmiConfig:
		procNetSnmp6MetricsCfg = cfg.ProcNetSnmp6MetricsConfig
	case *ProcNetSnmp6MetricsConfig:
		procNetSnmp6MetricsCfg = cfg
	case nil:
		procNetSnmp6MetricsCfg = DefaultProcNetSnmp6MetricsConfig()
	default:
		return nil, fmt.Errorf("NewProcNetSnmp6Metrics: %T invalid config type", cfg)
	}

	interval, err := time.ParseDuration(procNetSnmp6MetricsCfg.Interval)
	if err != nil {
		return nil, err
	}
	procNetSnmp6Metrics := &ProcNetSnmp6Metrics{
		id:                PROC_NET_SNMP6_METRICS_ID,
		interval:          interval,
		fullMetricsFactor: procNetSnmp6MetricsCfg.FullMetricsFactor,
		cycleNum:          make([]int, PROC_NET_SNMP6_CYCLE_COUNTER_NUM),
		zeroDelta:         make([]bool, procfs.NET_SNMP6_NUM_VALUES),
		tsSuffixBuf:       &bytes.Buffer{},
	}

	for i := 0; i < len(procNetSnmp6Metrics.cycleNum); i++ {
		procNetSnmp6Metrics.cycleNum[i] = initialCycleNum.Get(procNetSnmp6Metrics.fullMetricsFactor)
	}

	procNetSnmp6MetricsLog.Infof("id=%s", procNetSnmp6Metrics.id)
	procNetSnmp6MetricsLog.Infof("interval=%s", procNetSnmp6Metrics.interval)
	procNetSnmp6MetricsLog.Infof("full_metrics_factor=%d", procNetSnmp6Metrics.fullMetricsFactor)
	return procNetSnmp6Metrics, nil
}

func (pnsm6 *ProcNetSnmp6Metrics) updateMetricsCache() {
	instance, hostname := GlobalInstance, GlobalHostname
	if pnsm6.instance != "" {
		instance = pnsm6.instance
	}
	if pnsm6.hostname != "" {
		hostname = pnsm6.hostname
	}

	pnsm6.metricsCache = make([][]byte, procfs.NET_SNMP6_NUM_VALUES)
	pnsm6.totalMetricsCount = 1 // for interval metric
	for i := 0; i < len(pnsm6.metricsCache); i++ {
		name, ok := procNetSnmp6IndexToMetricNameMap[i]
		if ok {
			pnsm6.metricsCache[i] = []byte(fmt.Sprintf(
				`%s{%s="%s",%s="%s"} `, // N.B. include whitespace before value!
				name,
				INSTANCE_LABEL_NAME, instance,
				HOSTNAME_LABEL_NAME, hostname,
			))
			pnsm6.totalMetricsCount++
		}
	}
}

func (pnsm6 *ProcNetSnmp6Metrics) updateIntervalMetricsCache() {
	instance, hostname := GlobalInstance, GlobalHostname
	if pnsm6.instance != "" {
		instance = pnsm6.instance
	}
	if pnsm6.hostname != "" {
		hostname = pnsm6.hostname
	}
	pnsm6.intervalMetric = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s"} `, // N.B. include space before val
		PROC_NET_SNMP6_INTERVAL_METRIC,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
	))
}

func (pnsm6 *ProcNetSnmp6Metrics) generateMetrics(buf *bytes.Buffer) (int, int) {
	actualMetricsCount := 0
	currProcNetSnmp6, prevProcNetSnmp6 := pnsm6.procNetSnmp6[pnsm6.currIndex], pnsm6.procNetSnmp6[1-pnsm6.currIndex]
	// All values are deltas, prev is mandatory:
	if prevProcNetSnmp6 != nil {
		currValues, prevValues := currProcNetSnmp6.Values, prevProcNetSnmp6.Values
		currTs := pnsm6.procNetSnmp6Ts[pnsm6.currIndex]
		pnsm6.tsSuffixBuf.Reset()
		fmt.Fprintf(
			pnsm6.tsSuffixBuf, " %d\n", currTs.UnixMilli(),
		)
		promTs := pnsm6.tsSuffixBuf.Bytes()
		prevTs := pnsm6.procNetSnmp6Ts[1-pnsm6.currIndex]
		deltaSec := currTs.Sub(prevTs).Seconds()

		metricsCache := pnsm6.metricsCache
		if metricsCache == nil {
			pnsm6.updateMetricsCache()
			metricsCache = pnsm6.metricsCache
		}

		zeroDelta := pnsm6.zeroDelta
		for index, currValue := range currValues {
			metric := metricsCache[index]
			if metric == nil {
				// This value is ignored
				continue
			}
			prevValue := prevValues[index]

			if currProcNetSnmp6.IsUint32[index] && currValue < prevValue {
				// Compensate for rollover explicitly:
				currValue += (1 << 32)
			}

			delta := currValue - prevValue
			if delta != 0 ||
				pnsm6.cycleNum[index&PROC_NET_SNMP6_CYCLE_COUNTER_MASK] == 0 || // i.e. full cycle
				!zeroDelta[index] { // i.e. after non-zero
				buf.Write(metricsCache[index])
				if rate := procNetSnmp6IndexRate[index]; rate != nil {
					buf.WriteString(strconv.FormatFloat(
						float64(delta)*rate.factor/deltaSec, 'f', rate.prec, 64,
					))
				} else {
					buf.WriteString(strconv.FormatUint(uint64(delta), 10))
				}
				buf.Write(promTs)
				actualMetricsCount++
			}
			zeroDelta[index] = delta == 0
		}

		if pnsm6.intervalMetric == nil {
			pnsm6.updateIntervalMetricsCache()
		}
		buf.Write(pnsm6.intervalMetric)
		buf.WriteString(strconv.FormatFloat(deltaSec, 'f', 6, 64))
		buf.Write(promTs)
		actualMetricsCount++
	}

	// Update cycle counters:
	for i := 0; i < PROC_NET_SNMP6_CYCLE_COUNTER_NUM; i++ {
		if pnsm6.cycleNum[i]++; pnsm6.cycleNum[i] >= pnsm6.fullMetricsFactor {
			pnsm6.cycleNum[i] = 0
		}
	}

	// Toggle the buffers:
	pnsm6.currIndex = 1 - pnsm6.currIndex

	return actualMetricsCount, pnsm6.totalMetricsCount
}

// Satisfy the TaskActivity interface:
func (pnsm6 *ProcNetSnmp6Metrics) Execute() bool {
	timeNowFn := time.Now
	if pnsm6.timeNowFn != nil {
		timeNowFn = pnsm6.timeNowFn
	}

	metricsQueue := GlobalMetricsQueue
	if pnsm6.metricsQueue != nil {
		metricsQueue = pnsm6.metricsQueue
	}

	currProcNetSnmp6 := pnsm6.procNetSnmp6[pnsm6.currIndex]
	if currProcNetSnmp6 == nil {
		prevProcNetSnmp6 := pnsm6.procNetSnmp6[1-pnsm6.currIndex]
		if prevProcNetSnmp6 != nil {
			currProcNetSnmp6 = prevProcNetSnmp6.Clone(false)
		} else {
			procfsRoot := GlobalProcfsRoot
			if pnsm6.procfsRoot != "" {
				procfsRoot = pnsm6.procfsRoot
			}
			currProcNetSnmp6 = procfs.NewNetSnmp6(procfsRoot)
		}
		pnsm6.procNetSnmp6[pnsm6.currIndex] = currProcNetSnmp6
	}
	err := currProcNetSnmp6.Parse()
	if err != nil {
		procNetSnmp6MetricsLog.Warnf("%v: proc net snmp6 metrics will be disabled", err)
		return false
	}
	pnsm6.procNetSnmp6Ts[pnsm6.currIndex] = timeNowFn()

	buf := metricsQueue.GetBuf()
	actualMetricsCount, totalMetricsCount := pnsm6.generateMetrics(buf)
	byteCount := buf.Len()
	metricsQueue.QueueBuf(buf)

	GlobalMetricsGeneratorStatsContainer.Update(
		pnsm6.id, uint64(actualMetricsCount), uint64(totalMetricsCount), uint64(byteCount),
	)

	return true
}

// Define and register the task builder:
func ProcNetSnmp6MetricsTaskBuilder(cfg *LsvmiConfig) ([]*Task, error) {
	pnsm6, err := NewProcNetSnmp6Metrics(cfg)
	if err != nil {
		return nil, err
	}
	if pnsm6.interval <= 0 {
		procNetSnmp6MetricsLog.Infof(
			"interval=%s, metrics disabled", pnsm6.interval,
		)
		return nil, nil
	}
	tasks := []*Task{
		NewTask(pnsm6.id, pnsm6.interval, pnsm6),
	}
	return tasks, nil
}

func init() {
	TaskBuilders.Register(ProcNetSnmp6MetricsTaskBuilder)
}
