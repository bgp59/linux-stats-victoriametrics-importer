// Parser for /proc/net/snmp

package procfs

import (
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
// simplify the interface all values will be mapped into Golang int64.

// The file will be parsed as parallel lists of names and values: name[i] has
// value[i]. The names will formed as ProtoStat, e.g. UdpInDatagrams.

// Each data line (Proto: VAL ... VAL, that is) will be checked for sanity
// against the following info:
type NetSnmpLineInfo struct {
	prefix  []byte
	numVals int
}

type NetSnmp struct {
	// Parallel lists w/ the parsed info:
	Names  []string
	Values []int64
	// File path:
	path string
	// Line info for consistency check (index = line# / 2, since a data line has
	// an even line#):
	lineInfo []*NetSnmpLineInfo
}

var netSnmpReadFileBufPool = ReadFileBufPool32k

func NewNetSnmp(procfsRoot string) *NetSnmp {
	return &NetSnmp{
		Names:    make([]string, 0),
		Values:   make([]int64, 0),
		path:     path.Join(procfsRoot, "net", "snmp"),
		lineInfo: make([]*NetSnmpLineInfo, 0),
	}
}

func (netSnmp *NetSnmp) Clone(full bool) *NetSnmp {
	newNetSnmp := &NetSnmp{
		Names:    make([]string, len(netSnmp.Names)),
		Values:   make([]int64, len(netSnmp.Values)),
		lineInfo: make([]*NetSnmpLineInfo, len(netSnmp.lineInfo)),
	}
	copy(newNetSnmp.Names, netSnmp.Names)
	for i, lineInfo := range netSnmp.lineInfo {
		newLineInfo := &NetSnmpLineInfo{
			prefix:  make([]byte, len(lineInfo.prefix)),
			numVals: lineInfo.numVals,
		}
		copy(newLineInfo.prefix, lineInfo.prefix)
		newNetSnmp.lineInfo[i] = newLineInfo
	}
	if full {
		copy(newNetSnmp.Values, netSnmp.Values)
	}
	return newNetSnmp
}

func (netSnmp *NetSnmp) Parse() error {
	fBuf, err := netSnmpReadFileBufPool.ReadFile(netSnmp.path)
	defer netSnmpReadFileBufPool.ReturnBuf(fBuf)
	if err != nil {
		return err
	}

	buf, l := fBuf.Bytes(), fBuf.Len()

	parseNames := len(netSnmp.Names) == 0
	valueIndex := 0
	for pos, lineNum := 0, 1; pos < l; lineNum++ {
		// Line starts here:
		lineStart, eol := pos, false

		// For odd lines parse names as needed:
		if lineNum&1 == 1 {
			lineEnd := lineStart
			for ; lineEnd < l && buf[lineEnd] != '\n'; lineEnd++ {
			}
			if parseNames {
				line := string(buf[lineStart:lineEnd])
				fields := strings.Fields(line)
				if len(fields) < 2 || len(fields[0]) < 2 || fields[0][len(fields[0])-1] != ':' {
					return fmt.Errorf(
						"%s#%d: %q: invalid line, not Proto: Stat Stat... ",
						netSnmp.path, lineNum, line,
					)
				}
				proto, stats := fields[0][:len(fields[0])-1], fields[1:]
				numVals := len(stats)
				// Expand the parallel arrays:
				names := make([]string, numVals)
				for i, stat := range stats {
					names[i] = proto + stat
				}
				netSnmp.Names = append(netSnmp.Names, names...)
				netSnmp.Values = append(netSnmp.Values, make([]int64, numVals)...)
				lineInfo := &NetSnmpLineInfo{
					prefix:  []byte(fields[0]),
					numVals: numVals,
				}
				netSnmp.lineInfo = append(netSnmp.lineInfo, lineInfo)
			}
			pos = lineEnd + 1
			continue
		}

		// Even lines, parse data:

		// Validate prefix:
		lineInfoIndex := (lineNum - 1) >> 1
		if lineInfoIndex >= len(netSnmp.lineInfo) {
			return fmt.Errorf(
				"%s#%d: %q: unexpected line# (> %d)",
				netSnmp.path, lineNum, getCurrentLine(buf, lineStart), len(netSnmp.lineInfo)*2,
			)
		}
		lineInfo := netSnmp.lineInfo[lineInfoIndex]
		expectPrefix := lineInfo.prefix
		prefixPos, expectPrefixLen := 0, len(expectPrefix)
		for ; pos < l && prefixPos < expectPrefixLen && buf[pos] == expectPrefix[prefixPos]; pos++ {
			prefixPos++
		}
		if prefixPos != expectPrefixLen {
			return fmt.Errorf(
				"%s#%d: %q: unexpected prefix, want %q",
				netSnmp.path, lineNum, getCurrentLine(buf, lineStart), expectPrefix,
			)
		}

		lineValueIndex, lineExpectedNumVals := 0, lineInfo.numVals
		for !eol && pos < l && lineValueIndex < lineExpectedNumVals {
			for ; pos < l && isWhitespace[buf[pos]]; pos++ {
			}

			value, hasValue, isNegative := int64(0), false, false
			if buf[pos] == '-' {
				isNegative = true
				pos++
			}
			for done := false; !done && pos < l; pos++ {
				c := buf[pos]
				if digit := c - '0'; digit < 10 {
					value = (value << 3) + (value << 1) + int64(digit)
					hasValue = true
				} else if eol = (c == '\n'); eol || isWhitespace[c] {
					done = true
				} else {
					return fmt.Errorf(
						"%s#%d: %q: `%c': not a valid digit",
						netSnmp.path, lineNum, getCurrentLine(buf, lineStart), c,
					)
				}
			}
			if isNegative {
				value = -value
			}
			if hasValue {
				lineValueIndex++
				netSnmp.Values[valueIndex] = value
				valueIndex++
			}
		}

		// Enough values?
		if lineValueIndex < lineExpectedNumVals {
			return fmt.Errorf(
				"%s#%d: %q: missing values: want: %d, got: %d",
				netSnmp.path, lineNum, getCurrentLine(buf, lineStart), lineExpectedNumVals, lineValueIndex,
			)
		}

		// Locate EOL; only whitespaces are allowed at this point:
		for ; !eol && pos < l; pos++ {
			c := buf[pos]
			if eol = (c == '\n'); !eol && !isWhitespace[c] {
				return fmt.Errorf(
					"%s#%d: %q: %q unexpected content after IRQ counter(s)",
					netSnmp.path, lineNum, getCurrentLine(buf, lineStart), getCurrentLine(buf, pos),
				)
			}
		}
	}

	// Verify that expected number of values were parsed:
	if valueIndex != len(netSnmp.Values) {
		return fmt.Errorf(
			"%s: mismatched number of values: want: %d, got: %d",
			netSnmp.path, len(netSnmp.Values), valueIndex,
		)
	}

	return nil
}
