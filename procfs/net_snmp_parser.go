// Parser for /proc/net/snmp

package procfs

import (
	"bytes"
	"fmt"
	"path"
	"strings"
)

// References:
// 	https://datatracker.ietf.org/doc/html/rfc1213
// 	https://datatracker.ietf.org/doc/html/rfc2011
//  https://datatracker.ietf.org/doc/html/rfc5097
//  https://github.com/torvalds/linux/tree/master/include/uapi/linux/snmp.h
//  https://github.com/torvalds/linux/tree/master/Documentation/networking/snmp_counter.rst

// Tcp: RtoAlgorithm RtoMin RtoMax MaxConn ActiveOpens PassiveOpens AttemptFails EstabResets CurrEstab InSegs OutSegs RetransSegs InErrs OutRsts InCsumErrors
// Tcp: 1 200 120000 -1 98 63 4 1 1 5708 14228 35 0 15 0
// Udp: InDatagrams NoPorts InErrors OutDatagrams RcvbufErrors SndbufErrors InCsumErrors IgnoredMulti MemErrors
// Udp: 1006 16 0 1023 0 0 0 0 0

// Most values are of SNMP Counter32/Gauge32 type, some are INTEGER but limited
// to 0..N, N < max signed int32; only one value, tcpMaxConn, can be -1. To
// simplify the interface all values will be mapped into Golang uint32, w/ the
// negative values represented in two's complement. Before use, such values
// should be casted to int32(value).

// Begin of automatically generated content:
//  Script: tools/py/net_snmp_parser_helper.py
//  Reference file: testdata/lsvmi/proc/net/snmp

// Index definitions for parsed values:
const (
	NET_SNMP_IP_FORWARDING = iota
	NET_SNMP_IP_DEFAULT_TTL
	NET_SNMP_IP_IN_RECEIVES
	NET_SNMP_IP_IN_HDR_ERRORS
	NET_SNMP_IP_IN_ADDR_ERRORS
	NET_SNMP_IP_FORW_DATAGRAMS
	NET_SNMP_IP_IN_UNKNOWN_PROTOS
	NET_SNMP_IP_IN_DISCARDS
	NET_SNMP_IP_IN_DELIVERS
	NET_SNMP_IP_OUT_REQUESTS
	NET_SNMP_IP_OUT_DISCARDS
	NET_SNMP_IP_OUT_NO_ROUTES
	NET_SNMP_IP_REASM_TIMEOUT
	NET_SNMP_IP_REASM_REQDS
	NET_SNMP_IP_REASM_OKS
	NET_SNMP_IP_REASM_FAILS
	NET_SNMP_IP_FRAG_OKS
	NET_SNMP_IP_FRAG_FAILS
	NET_SNMP_IP_FRAG_CREATES
	NET_SNMP_ICMP_IN_MSGS
	NET_SNMP_ICMP_IN_ERRORS
	NET_SNMP_ICMP_IN_CSUM_ERRORS
	NET_SNMP_ICMP_IN_DEST_UNREACHS
	NET_SNMP_ICMP_IN_TIME_EXCDS
	NET_SNMP_ICMP_IN_PARM_PROBS
	NET_SNMP_ICMP_IN_SRC_QUENCHS
	NET_SNMP_ICMP_IN_REDIRECTS
	NET_SNMP_ICMP_IN_ECHOS
	NET_SNMP_ICMP_IN_ECHO_REPS
	NET_SNMP_ICMP_IN_TIMESTAMPS
	NET_SNMP_ICMP_IN_TIMESTAMP_REPS
	NET_SNMP_ICMP_IN_ADDR_MASKS
	NET_SNMP_ICMP_IN_ADDR_MASK_REPS
	NET_SNMP_ICMP_OUT_MSGS
	NET_SNMP_ICMP_OUT_ERRORS
	NET_SNMP_ICMP_OUT_DEST_UNREACHS
	NET_SNMP_ICMP_OUT_TIME_EXCDS
	NET_SNMP_ICMP_OUT_PARM_PROBS
	NET_SNMP_ICMP_OUT_SRC_QUENCHS
	NET_SNMP_ICMP_OUT_REDIRECTS
	NET_SNMP_ICMP_OUT_ECHOS
	NET_SNMP_ICMP_OUT_ECHO_REPS
	NET_SNMP_ICMP_OUT_TIMESTAMPS
	NET_SNMP_ICMP_OUT_TIMESTAMP_REPS
	NET_SNMP_ICMP_OUT_ADDR_MASKS
	NET_SNMP_ICMP_OUT_ADDR_MASK_REPS
	NET_SNMP_ICMPMSG_IN_TYPE3
	NET_SNMP_ICMPMSG_OUT_TYPE3
	NET_SNMP_TCP_RTO_ALGORITHM
	NET_SNMP_TCP_RTO_MIN
	NET_SNMP_TCP_RTO_MAX
	NET_SNMP_TCP_MAX_CONN
	NET_SNMP_TCP_ACTIVE_OPENS
	NET_SNMP_TCP_PASSIVE_OPENS
	NET_SNMP_TCP_ATTEMPT_FAILS
	NET_SNMP_TCP_ESTAB_RESETS
	NET_SNMP_TCP_CURR_ESTAB
	NET_SNMP_TCP_IN_SEGS
	NET_SNMP_TCP_OUT_SEGS
	NET_SNMP_TCP_RETRANS_SEGS
	NET_SNMP_TCP_IN_ERRS
	NET_SNMP_TCP_OUT_RSTS
	NET_SNMP_TCP_IN_CSUM_ERRORS
	NET_SNMP_UDP_IN_DATAGRAMS
	NET_SNMP_UDP_NO_PORTS
	NET_SNMP_UDP_IN_ERRORS
	NET_SNMP_UDP_OUT_DATAGRAMS
	NET_SNMP_UDP_RCVBUF_ERRORS
	NET_SNMP_UDP_SNDBUF_ERRORS
	NET_SNMP_UDP_IN_CSUM_ERRORS
	NET_SNMP_UDP_IGNORED_MULTI
	NET_SNMP_UDP_MEM_ERRORS
	NET_SNMP_UDPLITE_IN_DATAGRAMS
	NET_SNMP_UDPLITE_NO_PORTS
	NET_SNMP_UDPLITE_IN_ERRORS
	NET_SNMP_UDPLITE_OUT_DATAGRAMS
	NET_SNMP_UDPLITE_RCVBUF_ERRORS
	NET_SNMP_UDPLITE_SNDBUF_ERRORS
	NET_SNMP_UDPLITE_IN_CSUM_ERRORS
	NET_SNMP_UDPLITE_IGNORED_MULTI
	NET_SNMP_UDPLITE_MEM_ERRORS

	// Must be last:
	NET_SNMP_NUM_VALUES
)

// Map net/snmp [PROTO][VARIABLE] pairs into parsed value indexes:
var netSnmpIndexMap = map[string]map[string]int{
	"Ip": {
		"Forwarding":      NET_SNMP_IP_FORWARDING,
		"DefaultTTL":      NET_SNMP_IP_DEFAULT_TTL,
		"InReceives":      NET_SNMP_IP_IN_RECEIVES,
		"InHdrErrors":     NET_SNMP_IP_IN_HDR_ERRORS,
		"InAddrErrors":    NET_SNMP_IP_IN_ADDR_ERRORS,
		"ForwDatagrams":   NET_SNMP_IP_FORW_DATAGRAMS,
		"InUnknownProtos": NET_SNMP_IP_IN_UNKNOWN_PROTOS,
		"InDiscards":      NET_SNMP_IP_IN_DISCARDS,
		"InDelivers":      NET_SNMP_IP_IN_DELIVERS,
		"OutRequests":     NET_SNMP_IP_OUT_REQUESTS,
		"OutDiscards":     NET_SNMP_IP_OUT_DISCARDS,
		"OutNoRoutes":     NET_SNMP_IP_OUT_NO_ROUTES,
		"ReasmTimeout":    NET_SNMP_IP_REASM_TIMEOUT,
		"ReasmReqds":      NET_SNMP_IP_REASM_REQDS,
		"ReasmOKs":        NET_SNMP_IP_REASM_OKS,
		"ReasmFails":      NET_SNMP_IP_REASM_FAILS,
		"FragOKs":         NET_SNMP_IP_FRAG_OKS,
		"FragFails":       NET_SNMP_IP_FRAG_FAILS,
		"FragCreates":     NET_SNMP_IP_FRAG_CREATES,
	},
	"Icmp": {
		"InMsgs":           NET_SNMP_ICMP_IN_MSGS,
		"InErrors":         NET_SNMP_ICMP_IN_ERRORS,
		"InCsumErrors":     NET_SNMP_ICMP_IN_CSUM_ERRORS,
		"InDestUnreachs":   NET_SNMP_ICMP_IN_DEST_UNREACHS,
		"InTimeExcds":      NET_SNMP_ICMP_IN_TIME_EXCDS,
		"InParmProbs":      NET_SNMP_ICMP_IN_PARM_PROBS,
		"InSrcQuenchs":     NET_SNMP_ICMP_IN_SRC_QUENCHS,
		"InRedirects":      NET_SNMP_ICMP_IN_REDIRECTS,
		"InEchos":          NET_SNMP_ICMP_IN_ECHOS,
		"InEchoReps":       NET_SNMP_ICMP_IN_ECHO_REPS,
		"InTimestamps":     NET_SNMP_ICMP_IN_TIMESTAMPS,
		"InTimestampReps":  NET_SNMP_ICMP_IN_TIMESTAMP_REPS,
		"InAddrMasks":      NET_SNMP_ICMP_IN_ADDR_MASKS,
		"InAddrMaskReps":   NET_SNMP_ICMP_IN_ADDR_MASK_REPS,
		"OutMsgs":          NET_SNMP_ICMP_OUT_MSGS,
		"OutErrors":        NET_SNMP_ICMP_OUT_ERRORS,
		"OutDestUnreachs":  NET_SNMP_ICMP_OUT_DEST_UNREACHS,
		"OutTimeExcds":     NET_SNMP_ICMP_OUT_TIME_EXCDS,
		"OutParmProbs":     NET_SNMP_ICMP_OUT_PARM_PROBS,
		"OutSrcQuenchs":    NET_SNMP_ICMP_OUT_SRC_QUENCHS,
		"OutRedirects":     NET_SNMP_ICMP_OUT_REDIRECTS,
		"OutEchos":         NET_SNMP_ICMP_OUT_ECHOS,
		"OutEchoReps":      NET_SNMP_ICMP_OUT_ECHO_REPS,
		"OutTimestamps":    NET_SNMP_ICMP_OUT_TIMESTAMPS,
		"OutTimestampReps": NET_SNMP_ICMP_OUT_TIMESTAMP_REPS,
		"OutAddrMasks":     NET_SNMP_ICMP_OUT_ADDR_MASKS,
		"OutAddrMaskReps":  NET_SNMP_ICMP_OUT_ADDR_MASK_REPS,
	},
	"IcmpMsg": {
		"InType3":  NET_SNMP_ICMPMSG_IN_TYPE3,
		"OutType3": NET_SNMP_ICMPMSG_OUT_TYPE3,
	},
	"Tcp": {
		"RtoAlgorithm": NET_SNMP_TCP_RTO_ALGORITHM,
		"RtoMin":       NET_SNMP_TCP_RTO_MIN,
		"RtoMax":       NET_SNMP_TCP_RTO_MAX,
		"MaxConn":      NET_SNMP_TCP_MAX_CONN,
		"ActiveOpens":  NET_SNMP_TCP_ACTIVE_OPENS,
		"PassiveOpens": NET_SNMP_TCP_PASSIVE_OPENS,
		"AttemptFails": NET_SNMP_TCP_ATTEMPT_FAILS,
		"EstabResets":  NET_SNMP_TCP_ESTAB_RESETS,
		"CurrEstab":    NET_SNMP_TCP_CURR_ESTAB,
		"InSegs":       NET_SNMP_TCP_IN_SEGS,
		"OutSegs":      NET_SNMP_TCP_OUT_SEGS,
		"RetransSegs":  NET_SNMP_TCP_RETRANS_SEGS,
		"InErrs":       NET_SNMP_TCP_IN_ERRS,
		"OutRsts":      NET_SNMP_TCP_OUT_RSTS,
		"InCsumErrors": NET_SNMP_TCP_IN_CSUM_ERRORS,
	},
	"Udp": {
		"InDatagrams":  NET_SNMP_UDP_IN_DATAGRAMS,
		"NoPorts":      NET_SNMP_UDP_NO_PORTS,
		"InErrors":     NET_SNMP_UDP_IN_ERRORS,
		"OutDatagrams": NET_SNMP_UDP_OUT_DATAGRAMS,
		"RcvbufErrors": NET_SNMP_UDP_RCVBUF_ERRORS,
		"SndbufErrors": NET_SNMP_UDP_SNDBUF_ERRORS,
		"InCsumErrors": NET_SNMP_UDP_IN_CSUM_ERRORS,
		"IgnoredMulti": NET_SNMP_UDP_IGNORED_MULTI,
		"MemErrors":    NET_SNMP_UDP_MEM_ERRORS,
	},
	"UdpLite": {
		"InDatagrams":  NET_SNMP_UDPLITE_IN_DATAGRAMS,
		"NoPorts":      NET_SNMP_UDPLITE_NO_PORTS,
		"InErrors":     NET_SNMP_UDPLITE_IN_ERRORS,
		"OutDatagrams": NET_SNMP_UDPLITE_OUT_DATAGRAMS,
		"RcvbufErrors": NET_SNMP_UDPLITE_RCVBUF_ERRORS,
		"SndbufErrors": NET_SNMP_UDPLITE_SNDBUF_ERRORS,
		"InCsumErrors": NET_SNMP_UDPLITE_IN_CSUM_ERRORS,
		"IgnoredMulti": NET_SNMP_UDPLITE_IGNORED_MULTI,
		"MemErrors":    NET_SNMP_UDPLITE_MEM_ERRORS,
	},
}

// End of automatically generated content.

// The file consists of pairs of lines:
//   PROTO: VAR VAR ... VAR
//   PROTO: VAL VAL ... VAL
// the index map above will be used to construct structures that map a value
// position number in the VAL line into an index in the parsed value list.
// Values that are to be ignored will be mapped into a negative inddex.

type NetSnmpLineInfo struct {
	// Raw line, it will be used to determine changes:
	line []byte
	// Prefix end, inclusive of `:', used for sanity check for data line:
	prefixLen int
	// Mapping the variable position within the line 0..N-1 -> index0, index1,
	// ..., index(N-1), where N = number of variables
	indexMap []int
}

type NetSnmp struct {
	// Parsed values:
	Values []uint32

	// Whether the info changed during the parse or not; a nil value indicates
	// no change, != nil the reason for change:
	InfoChanged []byte

	// File path:
	path string

	// Line info, used for parsing; the index below is (line# - 1) / 2:
	lineInfo []*NetSnmpLineInfo
}

// The following fields may hold negative values:
var NetSnmpValueMayBeNegative = [NET_SNMP_NUM_VALUES]bool{
	NET_SNMP_TCP_MAX_CONN: true,
}

// Pool for reading the file in one go:
var netSnmpReadFileBufPool = ReadFileBufPool32k

func NetSnmpPath(procfsRoot string) string {
	return path.Join(procfsRoot, "net", "snmp")
}

func NewNetSnmp(procfsRoot string) *NetSnmp {
	return &NetSnmp{
		Values:   make([]uint32, NET_SNMP_NUM_VALUES),
		path:     NetSnmpPath(procfsRoot),
		lineInfo: make([]*NetSnmpLineInfo, 0),
	}
}

func (netSnmp *NetSnmp) UpdateInfo(from *NetSnmp) {
	netSnmp.lineInfo = make([]*NetSnmpLineInfo, len(from.lineInfo))
	for i, lineInfo := range from.lineInfo {
		newLineInfo := &NetSnmpLineInfo{
			line:      bytes.Clone(lineInfo.line),
			prefixLen: lineInfo.prefixLen,
			indexMap:  make([]int, len(lineInfo.indexMap)),
		}
		copy(newLineInfo.indexMap, lineInfo.indexMap)
		netSnmp.lineInfo[i] = newLineInfo
	}
}

func (netSnmp *NetSnmp) Clone(full bool) *NetSnmp {
	newNetSnmp := &NetSnmp{
		Values: make([]uint32, len(netSnmp.Values)),
		path:   netSnmp.path,
	}
	newNetSnmp.UpdateInfo(netSnmp)
	if full {
		copy(newNetSnmp.Values, netSnmp.Values)
	}
	return newNetSnmp
}

func buildSnmpLineInfo(line []byte) (*NetSnmpLineInfo, error) {
	var variables []string
	prefixLen := bytes.IndexByte(line, ':') + 1
	if prefixLen > 1 && prefixLen < len(line) {
		variables = strings.Fields(string(line[prefixLen+1:]))
	}
	if len(variables) == 0 {
		return nil, fmt.Errorf("invalid line, no PROTO: VAR VAR ... VAR")
	}
	proto := strings.TrimSpace(string(line[:prefixLen-1]))
	lineInfo := &NetSnmpLineInfo{
		line:      bytes.Clone(line),
		prefixLen: prefixLen,
		indexMap:  make([]int, len(variables)),
	}
	for i := 0; i < len(lineInfo.indexMap); i++ {
		lineInfo.indexMap[i] = -1 // i.e. assume un-mapped
	}
	protoIndexMap := netSnmpIndexMap[proto]
	if protoIndexMap != nil {
		for i, variable := range variables {
			valueIndex, ok := protoIndexMap[variable]
			if ok {
				lineInfo.indexMap[i] = valueIndex
			}
		}
	}
	return lineInfo, nil
}

func (netSnmp *NetSnmp) Parse() error {
	fBuf, err := netSnmpReadFileBufPool.ReadFile(netSnmp.path)
	defer netSnmpReadFileBufPool.ReturnBuf(fBuf)
	if err != nil {
		return err
	}

	buf, l := fBuf.Bytes(), fBuf.Len()
	if netSnmp.InfoChanged != nil {
		netSnmp.InfoChanged = nil
	}

	infoIndex := 0
	var lineInfo *NetSnmpLineInfo
	for pos, lineNum := 0, 1; pos < l; lineNum++ {
		lineStart, eol := pos, false

		// Odd line# are for parsing info; if the latter was already determined
		// in a previous pass, run a sanity check on it to make sure it hasn't
		// changed (IcmpMsg is dynamic, it may appear later). If it has changed
		// or if it wasn't parsed before, parse it now.
		if lineNum&1 == 1 {
			infoIndex = lineNum >> 1

			var expectedLine []byte = nil
			expectedInfoLineLen := 0
			if infoIndex < len(netSnmp.lineInfo) {
				lineInfo = netSnmp.lineInfo[infoIndex]
				expectedLine = lineInfo.line
				expectedInfoLineLen = len(expectedLine)
			} else {
				lineInfo = nil
			}

			lineEnd := pos
			for i := 0; lineEnd < l; lineEnd++ {
				if c := buf[lineEnd]; c == '\n' {
					break
				} else if i < expectedInfoLineLen && c != expectedLine[i] {
					expectedInfoLineLen = 0
				}
				i++
			}
			if lineEnd-lineStart != expectedInfoLineLen {
				expectedInfoLineLen = 0
			}

			if expectedInfoLineLen == 0 {
				if lineInfo != nil {
					netSnmp.InfoChanged = []byte(fmt.Sprintf(
						"%s:%d: %q: unexpected line, want %q, will rebuild info",
						netSnmp.path, lineNum, string(buf[lineStart:lineEnd]), expectedLine,
					))
					netSnmp.lineInfo = netSnmp.lineInfo[:infoIndex]
				}
				lineInfo, err = buildSnmpLineInfo(buf[lineStart:lineEnd])
				if err != nil {
					return fmt.Errorf(
						"%s:%d: %q: %v",
						netSnmp.path, lineNum-1, string(buf[lineStart:lineEnd]), err,
					)
				}
				netSnmp.lineInfo = append(netSnmp.lineInfo, lineInfo)
			}

			pos = lineEnd + 1
			continue
		}

		// Even lines, parse data:

		// Validate prefix:
		expectPrefix := lineInfo.line[:lineInfo.prefixLen]
		prefixPos, expectPrefixLen := 0, len(expectPrefix)
		for ; pos < l && prefixPos < expectPrefixLen && buf[pos] == expectPrefix[prefixPos]; pos++ {
			prefixPos++
		}
		if prefixPos != expectPrefixLen {
			return fmt.Errorf(
				"%s:%d: %q: unexpected prefix, want %q, will rebuild info",
				netSnmp.path, lineNum, getCurrentLine(buf, lineStart), expectPrefix,
			)
		}
		valueIndexMap := lineInfo.indexMap
		lineValueIndex, lineExpectedNumVals := 0, len(valueIndexMap)
		for !eol && pos < l && lineValueIndex < lineExpectedNumVals {
			// Locate the start of the value:
			for ; pos < l && isWhitespace[buf[pos]]; pos++ {
			}

			// Parse the value:
			value, isNegative, hasValue := uint32(0), false, false
			if pos < l && buf[pos] == '-' {
				isNegative = true
				pos++
			}
			for done := false; !done && pos < l; pos++ {
				c := buf[pos]
				if digit := c - '0'; digit < 10 {
					value = (value << 3) + (value << 1) + uint32(digit)
					hasValue = true
				} else if eol = (c == '\n'); eol || isWhitespace[c] {
					done = true
				} else {
					return fmt.Errorf(
						"%s:%d: %q: `%c': not a valid digit",
						netSnmp.path, lineNum, getCurrentLine(buf, lineStart), c,
					)
				}
			}
			if hasValue {
				valueIndex := valueIndexMap[lineValueIndex]
				if valueIndex >= 0 {
					if isNegative {
						if NetSnmpValueMayBeNegative[valueIndex] {
							value = (value ^ ((1 << 32) - 1)) + 1
						} else {
							return fmt.Errorf(
								"%s:%d: %q: -%d: unexpected negative value# %d",
								netSnmp.path, lineNum, getCurrentLine(buf, lineStart), value, lineValueIndex+1,
							)
						}
					}
					netSnmp.Values[valueIndex] = value
				}
				lineValueIndex++
			}
		}

		// Enough values?
		if lineValueIndex < lineExpectedNumVals {
			return fmt.Errorf(
				"%s:%d: %q: missing values: want: %d, got: %d",
				netSnmp.path, lineNum, getCurrentLine(buf, lineStart), lineExpectedNumVals, lineValueIndex,
			)
		}

		// Locate EOL; only whitespaces are allowed at this point:
		for ; !eol && pos < l; pos++ {
			c := buf[pos]
			if eol = (c == '\n'); !eol && !isWhitespace[c] {
				return fmt.Errorf(
					"%s:%d: %q: %q unexpected content after value(s)",
					netSnmp.path, lineNum, getCurrentLine(buf, lineStart), getCurrentLine(buf, pos),
				)
			}
		}
	}

	return nil
}
