// Parser for /proc/net/snmp

package procfs

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strings"
)

// References:
// 	https://datatracker.ietf.org/doc/html/rfc1213
// 	https://datatracker.ietf.org/doc/html/rfc2011
//  https://datatracker.ietf.org/doc/html/rfc5097
//  https://github.com/torvalds/linux/blob/master/include/uapi/linux/snmp.h

// Tcp: RtoAlgorithm RtoMin RtoMax MaxConn ActiveOpens PassiveOpens AttemptFails EstabResets CurrEstab InSegs OutSegs RetransSegs InErrs OutRsts InCsumErrors
// Tcp: 1 200 120000 -1 98 63 4 1 1 5708 14228 35 0 15 0
// Udp: InDatagrams NoPorts InErrors OutDatagrams RcvbufErrors SndbufErrors InCsumErrors IgnoredMulti MemErrors
// Udp: 1006 16 0 1023 0 0 0 0 0

// Most values are of SNMP Counter32/Gauge32 type, some are INTEGER but limited
// to 0..N, N < max signed int32; only one value, tcpMaxConn, can be -1. To
// simplify the interface all values will be mapped into Golang int64.

// The file will be parsed as parallel lists of names and values: name[i] has
// value[i]. The names will formed as protoStat, e.g. udpInDatagrams to match
// the RFC's.

// Each data line (PROTO: VAL ... VAL, that is) will be checked for sanity
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

// Word separators:
var netSnmpIsSep = [255]bool{
	' ':  true,
	'\t': true,
}

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
	file, err := os.Open(netSnmp.path)
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(file)
	parseNames := len(netSnmp.Names) == 0
	for lineNum, valueIndex, valuesLen := 1, 0, len(netSnmp.Values); scanner.Scan(); lineNum++ {
		// For odd lines parse names as needed:
		if lineNum&1 == 1 {
			if parseNames {
				line := scanner.Text()
				fields := strings.Fields(line)
				if len(fields) < 2 || len(fields[0]) < 2 || fields[0][len(fields[0])-1] != ':' {
					return fmt.Errorf(
						"%s#%d: %q: invalid line, not PROTO: STAT STAT... ",
						netSnmp.path, lineNum, line,
					)
				}
				proto, stats := strings.ToLower(fields[0][:len(fields[0])-1]), fields[1:]
				numVals := len(stats)
				// Expand the parallel arrays:
				names := make([]string, numVals)
				for i, stat := range stats {
					names[i] = proto + stat
				}
				netSnmp.Names = append(netSnmp.Names, names...)
				netSnmp.Values = append(netSnmp.Values, make([]int64, numVals)...)
				valuesLen = len(netSnmp.Values)
				lineInfo := &NetSnmpLineInfo{
					prefix:  []byte(fields[0]),
					numVals: numVals,
				}
				netSnmp.lineInfo = append(netSnmp.lineInfo, lineInfo)
			}
			continue
		}
		// Even lines, parse data:
		line := scanner.Bytes()
		pos, l := 0, len(line)

		lineInfoIndex := (lineNum - 1) >> 1
		if lineInfoIndex >= len(netSnmp.lineInfo) {
			return fmt.Errorf(
				"%s#%d: %q: unexpected line# (> %d)",
				netSnmp.path, lineNum, string(line), len(netSnmp.lineInfo)*2,
			)
		}
		lineInfo := netSnmp.lineInfo[lineInfoIndex]
		expectPrefix, expectNumVals := lineInfo.prefix, lineInfo.numVals

		for ; pos < l && pos < len(expectPrefix) && line[pos] == expectPrefix[pos]; pos++ {
		}
		if pos != len(expectPrefix) {
			return fmt.Errorf(
				"%s#%d: %q: unexpected prefix, want %q",
				netSnmp.path, lineNum, line, expectPrefix,
			)
		}

		numVals := 0
		for pos < l {
			for ; pos < l && netSnmpIsSep[line[pos]]; pos++ {
			}
			value, hasValue, isNegative := int64(0), false, false
			if line[pos] == '-' {
				isNegative = true
				pos++
			}
			for ; pos < l; pos++ {
				c := line[pos]
				if '0' <= c && c <= '9' {
					value = (value<<3 + value<<1) + int64(c-'0') // value*10+... that is, hopefully faster
					hasValue = true
				} else if netSnmpIsSep[c] {
					pos++
					break
				} else {
					return fmt.Errorf(
						"%s#%d: %q: invalid value ",
						netSnmp.path, lineNum, line,
					)
				}
			}
			if hasValue {
				numVals++
				if numVals > expectNumVals || valueIndex >= valuesLen {
					return fmt.Errorf(
						"%s#%d: %q: too many values (> %d)",
						netSnmp.path, lineNum, line, expectNumVals,
					)
				}
				if isNegative {
					value = -value
				}
				netSnmp.Values[valueIndex] = value
				valueIndex++
			}
		}
		if numVals < expectNumVals {
			return fmt.Errorf(
				"%s#%d: %q: not enough values (< %d)",
				netSnmp.path, lineNum, line, expectNumVals,
			)
		}
	}
	err = scanner.Err()
	if err != nil {
		return fmt.Errorf("%s: %v", netSnmp.path, err)
	}
	return nil
}
