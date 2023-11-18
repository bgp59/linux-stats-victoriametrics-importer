// Parser for /proc/net/snmp6

package procfs

import (
	"bytes"
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

const (
	NET_SNMP6_NAME_CHECK_SEP = ' '
)

type NetSnmp6 struct {
	// The data will be structured as 2 parallel lists:
	Names  []string
	Values []uint64
	// File path:
	path string
	// The names are evaluated only for the 1st pass; for sanity check in
	// subsequent passes, the names are verified against what was found at the
	// former. Store NAME<SEP>NAME<SEP>... as a signature for for the file. The
	// file being read as []byte, the sanity check info is stored in the same
	// data type.
	nameCheckRef []byte
}

// Word separators:
var netSnmp6IsSep = [255]bool{
	' ':  true,
	'\t': true,
	'\n': true,
}

func NewNetSnmp6(procfsRoot string) *NetSnmp6 {
	return &NetSnmp6{
		Names:  make([]string, 0),
		Values: make([]uint64, 0),
		path:   path.Join(procfsRoot, "net", "snmp6"),
	}
}

func (netSnmp6 *NetSnmp6) Clone(full bool) *NetSnmp6 {
	newNetSnmp6 := &NetSnmp6{
		Names:        make([]string, len(netSnmp6.Names)),
		Values:       make([]uint64, len(netSnmp6.Values)),
		path:         netSnmp6.path,
		nameCheckRef: make([]byte, len(netSnmp6.nameCheckRef)),
	}
	copy(newNetSnmp6.Names, netSnmp6.Names)
	copy(newNetSnmp6.nameCheckRef, netSnmp6.nameCheckRef)
	if full {
		copy(newNetSnmp6.Values, netSnmp6.Values)
	}
	return newNetSnmp6
}

func (netSnmp6 *NetSnmp6) makeErrorLine(buf []byte, nameStart int, reason any) error {
	if buf != nil {
		line := buf[nameStart:]
		lineEnd := bytes.IndexByte(line, '\n')
		if lineEnd > 0 {
			line = line[:lineEnd]
		}
		return fmt.Errorf("%s: %q: %v", netSnmp6.path, string(line), reason)
	} else {
		return fmt.Errorf("%s: %v", netSnmp6.path, reason)
	}
}

func (netSnmp6 *NetSnmp6) Parse() error {
	bBuf, err := ReadFileBufPool32k.ReadFile(netSnmp6.path)
	if err != nil {
		return err
	}
	defer ReadFileBufPool32k.ReturnBuf(bBuf)

	buf, l := bBuf.Bytes(), bBuf.Len()

	names, values, nameCheckRef := netSnmp6.Names, netSnmp6.Values, netSnmp6.nameCheckRef
	firstPass := nameCheckRef == nil
	if firstPass {
		nameCheckRef = make([]byte, 0)
	}

	nameCheckPos, nameCheckRefLen := 0, len(nameCheckRef)
	for pos, valueIndex := 0, 0; pos < l; {
		// Extract / verify name:
		for ; pos < l && netSnmp6IsSep[buf[pos]]; pos++ {
		}
		nameStart := pos
		if firstPass {
			for ; pos < l && !netSnmp6IsSep[buf[pos]]; pos++ {
			}
			name := buf[nameStart:pos]
			names = append(names, string(name))
			nameCheckRef = append(nameCheckRef, name...)
			nameCheckRef = append(nameCheckRef, NET_SNMP6_NAME_CHECK_SEP)
		} else {
			for isSep := false; !isSep && pos < l; pos++ {
				c := buf[pos]
				isSep = netSnmp6IsSep[c]
				if isSep {
					c = NET_SNMP6_NAME_CHECK_SEP
				}
				if nameCheckPos >= nameCheckRefLen || nameCheckRef[nameCheckPos] != c {
					return netSnmp6.makeErrorLine(buf, nameStart, "invalid name, not seen before at this line")
				}
				nameCheckPos++
			}
		}

		// Extract value:
		for ; pos < l && netSnmp6IsSep[buf[pos]]; pos++ {
		}
		value, hasValue := uint64(0), false
		for isSep := false; !isSep && pos < l; pos++ {
			c := buf[pos]
			if digit := c - '0'; digit < 10 {
				value = value<<3 + value<<1 + uint64(digit) // value*10 + ..., but faster
				hasValue = true
			} else if isSep = netSnmp6IsSep[c]; !isSep {
				return netSnmp6.makeErrorLine(buf, nameStart, "invalid value")
			}
		}
		if hasValue {
			if firstPass {
				values = append(values, value)
			} else {
				// Since the name was validated and since names and values are
				// parallel lists, it is safe to use the index below without
				// checking for overflow:
				values[valueIndex] = value
				valueIndex++
			}
		} else {
			return netSnmp6.makeErrorLine(buf, nameStart, "missing value")
		}
	}

	if firstPass {
		netSnmp6.Names = names
		netSnmp6.Values = values
		netSnmp6.nameCheckRef = nameCheckRef
	} else if nameCheckPos != nameCheckRefLen {
		return fmt.Errorf(
			"%s: missing names: %q",
			netSnmp6.path, string(nameCheckRef[nameCheckPos:]),
		)
	}

	return nil
}
