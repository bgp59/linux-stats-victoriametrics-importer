// Parser for /proc/net/snmp6

package procfs

import (
	"fmt"
	"path"
)

// Ip6InMcastPkts                  	0
// Ip6OutMcastPkts                 	19
// Ip6InOctets                     	368
// Ip6OutOctets                    	1196
// Ip6InMcastOctets                	0
// Ip6OutMcastOctets               	1196
// Ip6InBcastOctets                	0

// References:
//   https://github.com/torvalds/linux/blob/master/net/ipv6/proc.c
//
// As per:
//   https://github.com/torvalds/linux/blob/6bc40e44f1ddef16a787f3501b97f1fff909177c/net/ipv6/proc.c#L221
// all values in this file are uint64.

// Begin of automatically generated content:
//  Script: tools/py/net_snmp6_parser_helper.py
//  Reference file: testdata/lsvmi/proc/net/snmp6

// Index definitions for parsed values:
const (
	NET_SNMP6_IP6_IN_RECEIVES = iota
	NET_SNMP6_IP6_IN_HDR_ERRORS
	NET_SNMP6_IP6_IN_TOO_BIG_ERRORS
	NET_SNMP6_IP6_IN_NO_ROUTES
	NET_SNMP6_IP6_IN_ADDR_ERRORS
	NET_SNMP6_IP6_IN_UNKNOWN_PROTOS
	NET_SNMP6_IP6_IN_TRUNCATED_PKTS
	NET_SNMP6_IP6_IN_DISCARDS
	NET_SNMP6_IP6_IN_DELIVERS
	NET_SNMP6_IP6_OUT_FORW_DATAGRAMS
	NET_SNMP6_IP6_OUT_REQUESTS
	NET_SNMP6_IP6_OUT_DISCARDS
	NET_SNMP6_IP6_OUT_NO_ROUTES
	NET_SNMP6_IP6_REASM_TIMEOUT
	NET_SNMP6_IP6_REASM_REQDS
	NET_SNMP6_IP6_REASM_OKS
	NET_SNMP6_IP6_REASM_FAILS
	NET_SNMP6_IP6_FRAG_OKS
	NET_SNMP6_IP6_FRAG_FAILS
	NET_SNMP6_IP6_FRAG_CREATES
	NET_SNMP6_IP6_IN_MCAST_PKTS
	NET_SNMP6_IP6_OUT_MCAST_PKTS
	NET_SNMP6_IP6_IN_OCTETS
	NET_SNMP6_IP6_OUT_OCTETS
	NET_SNMP6_IP6_IN_MCAST_OCTETS
	NET_SNMP6_IP6_OUT_MCAST_OCTETS
	NET_SNMP6_IP6_IN_BCAST_OCTETS
	NET_SNMP6_IP6_OUT_BCAST_OCTETS
	NET_SNMP6_IP6_IN_NO_ECT_PKTS
	NET_SNMP6_IP6_IN_ECT1_PKTS
	NET_SNMP6_IP6_IN_ECT0_PKTS
	NET_SNMP6_IP6_IN_CE_PKTS
	NET_SNMP6_ICMP6_IN_MSGS
	NET_SNMP6_ICMP6_IN_ERRORS
	NET_SNMP6_ICMP6_OUT_MSGS
	NET_SNMP6_ICMP6_OUT_ERRORS
	NET_SNMP6_ICMP6_IN_CSUM_ERRORS
	NET_SNMP6_ICMP6_IN_DEST_UNREACHS
	NET_SNMP6_ICMP6_IN_PKT_TOO_BIGS
	NET_SNMP6_ICMP6_IN_TIME_EXCDS
	NET_SNMP6_ICMP6_IN_PARM_PROBLEMS
	NET_SNMP6_ICMP6_IN_ECHOS
	NET_SNMP6_ICMP6_IN_ECHO_REPLIES
	NET_SNMP6_ICMP6_IN_GROUP_MEMB_QUERIES
	NET_SNMP6_ICMP6_IN_GROUP_MEMB_RESPONSES
	NET_SNMP6_ICMP6_IN_GROUP_MEMB_REDUCTIONS
	NET_SNMP6_ICMP6_IN_ROUTER_SOLICITS
	NET_SNMP6_ICMP6_IN_ROUTER_ADVERTISEMENTS
	NET_SNMP6_ICMP6_IN_NEIGHBOR_SOLICITS
	NET_SNMP6_ICMP6_IN_NEIGHBOR_ADVERTISEMENTS
	NET_SNMP6_ICMP6_IN_REDIRECTS
	NET_SNMP6_ICMP6_IN_MLD_V2_REPORTS
	NET_SNMP6_ICMP6_OUT_DEST_UNREACHS
	NET_SNMP6_ICMP6_OUT_PKT_TOO_BIGS
	NET_SNMP6_ICMP6_OUT_TIME_EXCDS
	NET_SNMP6_ICMP6_OUT_PARM_PROBLEMS
	NET_SNMP6_ICMP6_OUT_ECHOS
	NET_SNMP6_ICMP6_OUT_ECHO_REPLIES
	NET_SNMP6_ICMP6_OUT_GROUP_MEMB_QUERIES
	NET_SNMP6_ICMP6_OUT_GROUP_MEMB_RESPONSES
	NET_SNMP6_ICMP6_OUT_GROUP_MEMB_REDUCTIONS
	NET_SNMP6_ICMP6_OUT_ROUTER_SOLICITS
	NET_SNMP6_ICMP6_OUT_ROUTER_ADVERTISEMENTS
	NET_SNMP6_ICMP6_OUT_NEIGHBOR_SOLICITS
	NET_SNMP6_ICMP6_OUT_NEIGHBOR_ADVERTISEMENTS
	NET_SNMP6_ICMP6_OUT_REDIRECTS
	NET_SNMP6_ICMP6_OUT_MLD_V2_REPORTS
	NET_SNMP6_ICMP6_OUT_TYPE133
	NET_SNMP6_ICMP6_OUT_TYPE135
	NET_SNMP6_ICMP6_OUT_TYPE143
	NET_SNMP6_UDP6_IN_DATAGRAMS
	NET_SNMP6_UDP6_NO_PORTS
	NET_SNMP6_UDP6_IN_ERRORS
	NET_SNMP6_UDP6_OUT_DATAGRAMS
	NET_SNMP6_UDP6_RCVBUF_ERRORS
	NET_SNMP6_UDP6_SNDBUF_ERRORS
	NET_SNMP6_UDP6_IN_CSUM_ERRORS
	NET_SNMP6_UDP6_IGNORED_MULTI
	NET_SNMP6_UDP6_MEM_ERRORS
	NET_SNMP6_UDPLITE6_IN_DATAGRAMS
	NET_SNMP6_UDPLITE6_NO_PORTS
	NET_SNMP6_UDPLITE6_IN_ERRORS
	NET_SNMP6_UDPLITE6_OUT_DATAGRAMS
	NET_SNMP6_UDPLITE6_RCVBUF_ERRORS
	NET_SNMP6_UDPLITE6_SNDBUF_ERRORS
	NET_SNMP6_UDPLITE6_IN_CSUM_ERRORS
	NET_SNMP6_UDPLITE6_MEM_ERRORS

	// Must be last:
	NET_SNMP6_NUM_VALUES
)

// Map net/snmp6 VARIABLE into parsed value index:
var netSnmp6IndexMap = map[string]int{
	"Ip6InReceives":                  NET_SNMP6_IP6_IN_RECEIVES,
	"Ip6InHdrErrors":                 NET_SNMP6_IP6_IN_HDR_ERRORS,
	"Ip6InTooBigErrors":              NET_SNMP6_IP6_IN_TOO_BIG_ERRORS,
	"Ip6InNoRoutes":                  NET_SNMP6_IP6_IN_NO_ROUTES,
	"Ip6InAddrErrors":                NET_SNMP6_IP6_IN_ADDR_ERRORS,
	"Ip6InUnknownProtos":             NET_SNMP6_IP6_IN_UNKNOWN_PROTOS,
	"Ip6InTruncatedPkts":             NET_SNMP6_IP6_IN_TRUNCATED_PKTS,
	"Ip6InDiscards":                  NET_SNMP6_IP6_IN_DISCARDS,
	"Ip6InDelivers":                  NET_SNMP6_IP6_IN_DELIVERS,
	"Ip6OutForwDatagrams":            NET_SNMP6_IP6_OUT_FORW_DATAGRAMS,
	"Ip6OutRequests":                 NET_SNMP6_IP6_OUT_REQUESTS,
	"Ip6OutDiscards":                 NET_SNMP6_IP6_OUT_DISCARDS,
	"Ip6OutNoRoutes":                 NET_SNMP6_IP6_OUT_NO_ROUTES,
	"Ip6ReasmTimeout":                NET_SNMP6_IP6_REASM_TIMEOUT,
	"Ip6ReasmReqds":                  NET_SNMP6_IP6_REASM_REQDS,
	"Ip6ReasmOKs":                    NET_SNMP6_IP6_REASM_OKS,
	"Ip6ReasmFails":                  NET_SNMP6_IP6_REASM_FAILS,
	"Ip6FragOKs":                     NET_SNMP6_IP6_FRAG_OKS,
	"Ip6FragFails":                   NET_SNMP6_IP6_FRAG_FAILS,
	"Ip6FragCreates":                 NET_SNMP6_IP6_FRAG_CREATES,
	"Ip6InMcastPkts":                 NET_SNMP6_IP6_IN_MCAST_PKTS,
	"Ip6OutMcastPkts":                NET_SNMP6_IP6_OUT_MCAST_PKTS,
	"Ip6InOctets":                    NET_SNMP6_IP6_IN_OCTETS,
	"Ip6OutOctets":                   NET_SNMP6_IP6_OUT_OCTETS,
	"Ip6InMcastOctets":               NET_SNMP6_IP6_IN_MCAST_OCTETS,
	"Ip6OutMcastOctets":              NET_SNMP6_IP6_OUT_MCAST_OCTETS,
	"Ip6InBcastOctets":               NET_SNMP6_IP6_IN_BCAST_OCTETS,
	"Ip6OutBcastOctets":              NET_SNMP6_IP6_OUT_BCAST_OCTETS,
	"Ip6InNoECTPkts":                 NET_SNMP6_IP6_IN_NO_ECT_PKTS,
	"Ip6InECT1Pkts":                  NET_SNMP6_IP6_IN_ECT1_PKTS,
	"Ip6InECT0Pkts":                  NET_SNMP6_IP6_IN_ECT0_PKTS,
	"Ip6InCEPkts":                    NET_SNMP6_IP6_IN_CE_PKTS,
	"Icmp6InMsgs":                    NET_SNMP6_ICMP6_IN_MSGS,
	"Icmp6InErrors":                  NET_SNMP6_ICMP6_IN_ERRORS,
	"Icmp6OutMsgs":                   NET_SNMP6_ICMP6_OUT_MSGS,
	"Icmp6OutErrors":                 NET_SNMP6_ICMP6_OUT_ERRORS,
	"Icmp6InCsumErrors":              NET_SNMP6_ICMP6_IN_CSUM_ERRORS,
	"Icmp6InDestUnreachs":            NET_SNMP6_ICMP6_IN_DEST_UNREACHS,
	"Icmp6InPktTooBigs":              NET_SNMP6_ICMP6_IN_PKT_TOO_BIGS,
	"Icmp6InTimeExcds":               NET_SNMP6_ICMP6_IN_TIME_EXCDS,
	"Icmp6InParmProblems":            NET_SNMP6_ICMP6_IN_PARM_PROBLEMS,
	"Icmp6InEchos":                   NET_SNMP6_ICMP6_IN_ECHOS,
	"Icmp6InEchoReplies":             NET_SNMP6_ICMP6_IN_ECHO_REPLIES,
	"Icmp6InGroupMembQueries":        NET_SNMP6_ICMP6_IN_GROUP_MEMB_QUERIES,
	"Icmp6InGroupMembResponses":      NET_SNMP6_ICMP6_IN_GROUP_MEMB_RESPONSES,
	"Icmp6InGroupMembReductions":     NET_SNMP6_ICMP6_IN_GROUP_MEMB_REDUCTIONS,
	"Icmp6InRouterSolicits":          NET_SNMP6_ICMP6_IN_ROUTER_SOLICITS,
	"Icmp6InRouterAdvertisements":    NET_SNMP6_ICMP6_IN_ROUTER_ADVERTISEMENTS,
	"Icmp6InNeighborSolicits":        NET_SNMP6_ICMP6_IN_NEIGHBOR_SOLICITS,
	"Icmp6InNeighborAdvertisements":  NET_SNMP6_ICMP6_IN_NEIGHBOR_ADVERTISEMENTS,
	"Icmp6InRedirects":               NET_SNMP6_ICMP6_IN_REDIRECTS,
	"Icmp6InMLDv2Reports":            NET_SNMP6_ICMP6_IN_MLD_V2_REPORTS,
	"Icmp6OutDestUnreachs":           NET_SNMP6_ICMP6_OUT_DEST_UNREACHS,
	"Icmp6OutPktTooBigs":             NET_SNMP6_ICMP6_OUT_PKT_TOO_BIGS,
	"Icmp6OutTimeExcds":              NET_SNMP6_ICMP6_OUT_TIME_EXCDS,
	"Icmp6OutParmProblems":           NET_SNMP6_ICMP6_OUT_PARM_PROBLEMS,
	"Icmp6OutEchos":                  NET_SNMP6_ICMP6_OUT_ECHOS,
	"Icmp6OutEchoReplies":            NET_SNMP6_ICMP6_OUT_ECHO_REPLIES,
	"Icmp6OutGroupMembQueries":       NET_SNMP6_ICMP6_OUT_GROUP_MEMB_QUERIES,
	"Icmp6OutGroupMembResponses":     NET_SNMP6_ICMP6_OUT_GROUP_MEMB_RESPONSES,
	"Icmp6OutGroupMembReductions":    NET_SNMP6_ICMP6_OUT_GROUP_MEMB_REDUCTIONS,
	"Icmp6OutRouterSolicits":         NET_SNMP6_ICMP6_OUT_ROUTER_SOLICITS,
	"Icmp6OutRouterAdvertisements":   NET_SNMP6_ICMP6_OUT_ROUTER_ADVERTISEMENTS,
	"Icmp6OutNeighborSolicits":       NET_SNMP6_ICMP6_OUT_NEIGHBOR_SOLICITS,
	"Icmp6OutNeighborAdvertisements": NET_SNMP6_ICMP6_OUT_NEIGHBOR_ADVERTISEMENTS,
	"Icmp6OutRedirects":              NET_SNMP6_ICMP6_OUT_REDIRECTS,
	"Icmp6OutMLDv2Reports":           NET_SNMP6_ICMP6_OUT_MLD_V2_REPORTS,
	"Icmp6OutType133":                NET_SNMP6_ICMP6_OUT_TYPE133,
	"Icmp6OutType135":                NET_SNMP6_ICMP6_OUT_TYPE135,
	"Icmp6OutType143":                NET_SNMP6_ICMP6_OUT_TYPE143,
	"Udp6InDatagrams":                NET_SNMP6_UDP6_IN_DATAGRAMS,
	"Udp6NoPorts":                    NET_SNMP6_UDP6_NO_PORTS,
	"Udp6InErrors":                   NET_SNMP6_UDP6_IN_ERRORS,
	"Udp6OutDatagrams":               NET_SNMP6_UDP6_OUT_DATAGRAMS,
	"Udp6RcvbufErrors":               NET_SNMP6_UDP6_RCVBUF_ERRORS,
	"Udp6SndbufErrors":               NET_SNMP6_UDP6_SNDBUF_ERRORS,
	"Udp6InCsumErrors":               NET_SNMP6_UDP6_IN_CSUM_ERRORS,
	"Udp6IgnoredMulti":               NET_SNMP6_UDP6_IGNORED_MULTI,
	"Udp6MemErrors":                  NET_SNMP6_UDP6_MEM_ERRORS,
	"UdpLite6InDatagrams":            NET_SNMP6_UDPLITE6_IN_DATAGRAMS,
	"UdpLite6NoPorts":                NET_SNMP6_UDPLITE6_NO_PORTS,
	"UdpLite6InErrors":               NET_SNMP6_UDPLITE6_IN_ERRORS,
	"UdpLite6OutDatagrams":           NET_SNMP6_UDPLITE6_OUT_DATAGRAMS,
	"UdpLite6RcvbufErrors":           NET_SNMP6_UDPLITE6_RCVBUF_ERRORS,
	"UdpLite6SndbufErrors":           NET_SNMP6_UDPLITE6_SNDBUF_ERRORS,
	"UdpLite6InCsumErrors":           NET_SNMP6_UDPLITE6_IN_CSUM_ERRORS,
	"UdpLite6MemErrors":              NET_SNMP6_UDPLITE6_MEM_ERRORS,
}

// End of automatically generated content.

type NetSnmp6LineInfo struct {
	// Variable name, discovered during the 1st pass, it will be used for sanity
	// checks in all subsequent passes:
	name []byte
	// NET_SNMP6_... index where to store the parsed value:
	index int
}

type NetSnmp6 struct {
	// Values, indexed by NET_SNMP6_...:
	Values []uint64
	// File path:
	path string
	// Line info, used for parsing; the index below is by occurence#, starting from 0:
	lineInfo []*NetSnmp6LineInfo
}

// Pool for reading the file in one go:
var netSnmp6ReadFileBufPool = ReadFileBufPool32k

func NewNetSnmp6(procfsRoot string) *NetSnmp6 {
	return &NetSnmp6{
		Values:   make([]uint64, NET_SNMP6_NUM_VALUES),
		path:     path.Join(procfsRoot, "net", "snmp6"),
		lineInfo: make([]*NetSnmp6LineInfo, 0),
	}
}

func (netSnmp6 *NetSnmp6) Clone(full bool) *NetSnmp6 {
	newNetSnmp6 := &NetSnmp6{
		Values:   make([]uint64, len(netSnmp6.Values)),
		path:     netSnmp6.path,
		lineInfo: make([]*NetSnmp6LineInfo, len(netSnmp6.lineInfo)),
	}
	copy(newNetSnmp6.lineInfo, netSnmp6.lineInfo)
	if full {
		copy(newNetSnmp6.Values, netSnmp6.Values)
	}
	return newNetSnmp6
}

func (netSnmp6 *NetSnmp6) Parse() error {
	fBuf, err := netSnmp6ReadFileBufPool.ReadFile(netSnmp6.path)
	defer netSnmp6ReadFileBufPool.ReturnBuf(fBuf)
	if err != nil {
		return err
	}

	buf, l := fBuf.Bytes(), fBuf.Len()

	buildLineInfo := len(netSnmp6.lineInfo) == 0
	values, variableIndex := netSnmp6.Values, 0
	for pos := 0; pos < l; {
		// Extract / verify name:
		for ; pos < l && isWhitespaceNl[buf[pos]]; pos++ {
		}
		nameStart, index, ok := pos, -1, false
		if buildLineInfo {
			for ; pos < l && !isWhitespaceNl[buf[pos]]; pos++ {
			}
			name := string(buf[nameStart:pos])
			index, ok = netSnmp6IndexMap[name]
			if !ok {
				index = -1
			}
			lineInfo := &NetSnmp6LineInfo{
				name:  []byte(name),
				index: index,
			}
			netSnmp6.lineInfo = append(netSnmp6.lineInfo, lineInfo)
		} else {
			if variableIndex >= len(netSnmp6.lineInfo) {
				return fmt.Errorf(
					"%s: %q: unexpected number of variables (> %d)",
					netSnmp6.path, getCurrentLine(buf, nameStart), len(netSnmp6.lineInfo),
				)
			}
			lineInfo := netSnmp6.lineInfo[variableIndex]
			name := lineInfo.name
			prefixPos, prefixLen := 0, len(name)
			for pos < l && prefixPos < prefixLen && buf[pos] == name[prefixPos] {
				pos++
				prefixPos++
			}
			if prefixPos != prefixLen || (pos < l && !isWhitespaceNl[buf[pos]]) {
				return fmt.Errorf(
					"%s: %q: %q: invalid name, not seen before",
					netSnmp6.path, getCurrentLine(buf, nameStart), string(name),
				)
			}
			index = lineInfo.index
		}

		// Extract value:
		for ; pos < l && isWhitespaceNl[buf[pos]]; pos++ {
		}
		value, hasValue := uint64(0), false
		for ; !hasValue && pos < l; pos++ {
			c := buf[pos]
			if digit := c - '0'; digit < 10 {
				value = (value << 3) + (value << 1) + uint64(digit)
			} else if isWhitespaceNl[c] {
				hasValue = true
			} else {
				return fmt.Errorf(
					"%s: %q: `%c' not a valid digit",
					netSnmp6.path, getCurrentLine(buf, nameStart), c,
				)
			}
		}
		if hasValue {
			if index >= 0 {
				values[index] = value
			}
			variableIndex++
		} else {
			return fmt.Errorf(
				"%s: %q: missing value",
				netSnmp6.path, getCurrentLine(buf, nameStart),
			)
		}
	}

	if variableIndex != len(netSnmp6.lineInfo) {
		return fmt.Errorf(
			"%s: unexpected number of variables: want: %d, got: %d",
			netSnmp6.path, len(netSnmp6.lineInfo), variableIndex,
		)
	}

	return nil
}
