// parser for /proc/net/dev

package procfs

import (
	"bytes"
	"fmt"
	"path"
)

// Inter-|   Receive                                                |  Transmit
//  face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
//     lo:    6740      68    0    0    0     0          0         0     6740      68    0    0    0     0       0          0
//   eth0: 1936365    7267    0    0    0     0          0         0 14322183    7122    0    0    0     0       0          0

// References:
//  https://github.com/torvalds/linux/blob/b8f1fa2419c19c81bc386a6b350879ba54a573e1/net/core/net-procfs.c#L77
//  https://github.com/torvalds/linux/blob/791c8ab095f71327899023223940dd52257a4173/tools/include/uapi/linux/if_link.h#L43
//  https://github.com/torvalds/linux/blob/791c8ab095f71327899023223940dd52257a4173/Documentation/networking/statistics.rst#procfs

// Each interface will present data as []uint64, indexed as follows:
const (
	NET_DEV_RX_BYTES = iota
	NET_DEV_RX_PACKETS
	NET_DEV_RX_ERRS
	NET_DEV_RX_DROP
	NET_DEV_RX_FIFO
	NET_DEV_RX_FRAME
	NET_DEV_RX_COMPRESSED
	NET_DEV_RX_MULTICAST
	NET_DEV_TX_BYTES
	NET_DEV_TX_PACKETS
	NET_DEV_TX_ERRS
	NET_DEV_TX_DROP
	NET_DEV_TX_FIFO
	NET_DEV_TX_COLLS
	NET_DEV_TX_CARRIER
	NET_DEV_TX_COMPRESSED

	// Must be last:
	NET_DEV_NUM_STATS
)

type NetDev struct {
	// Stats indexed by interface name:
	DevStats map[string][]uint64
	// Interfaces may be created/deleted dynamically. To keep track of deletion,
	// each parse invocation is associated with a different from before scan#
	// and each found interface name will be updated below for it. At the end of
	// the pass, the interfaces that have a different scan# are leftover from a
	// previous scan and they are deleted from the stats.
	devScanNum map[string]int
	scanNum    int
	// The path file to  read:
	path string
	// The parser assumes certain fields, based on the first N lines. The file
	// will be validated only for the 1st pass, since the file syntax cannot
	// change without a kernel change, i.e. a reboot. The validated header is
	// remembered and it will be checked for changes at each pass as a sanity
	// check:
	validHeader []byte
}

// To protect against changes in kernel that may alter the exposed stats, the
// header of the file is checked, once/1st time, against the known headers
// listed below; the comparison will be performed after some normalization:
//   - trimmed
//   - all lowercase
//   - all spaces that do not separate a word are removed,
//     e.g. `Inter-|   Receive  |' -> `Inter-|Receive|'
//   - all multiple spaces separating words are replaced w/ a single one
//   - multiple newlines are replaced w/ a single one

var netDevValidNormHeaders = [][]byte{
	normalizeNetDevHeader(`
Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
`),
}

// Word separators:
var netDevIsSep = [256]bool{
	' ':  true,
	'\t': true,
}

var netDevIsSepNl = [256]bool{
	' ':  true,
	'\t': true,
	'\n': true,
}

func normalizeNetDevHeader(header any) []byte {
	var hBytes, normHeader []byte
	switch header := header.(type) {
	case string:
		hBytes = []byte(header)
	case []byte:
		hBytes = header
	default:
		return nil
	}
	hBytes = bytes.ToLower(bytes.TrimSpace(hBytes))
	normHeader = make([]byte, len(hBytes))

	pos, wasC, wasSpace, lastNonSpaceWasWordC := 0, byte(0), false, false
	for _, c := range hBytes {
		isSpace := c == ' ' || c == '\t'
		isWordC := 'a' <= c && c <= 'z' || '0' <= c && c <= '9'
		if isWordC && wasSpace && lastNonSpaceWasWordC {
			normHeader[pos] = ' '
			pos++
		}
		if !isSpace {
			if wasC != '\n' || c != '\n' {
				normHeader[pos] = c
				pos++
			}
			lastNonSpaceWasWordC = isWordC
		}
		wasC, wasSpace = c, isSpace
	}
	return normHeader[:pos]
}

func NewNetDev(procfsRoot string) *NetDev {
	return &NetDev{
		DevStats:   map[string][]uint64{},
		devScanNum: map[string]int{},
		scanNum:    -1,
		path:       path.Join(procfsRoot, "net", "dev"),
	}
}

func (netDev *NetDev) Clone(full bool) *NetDev {
	newNetDev := &NetDev{
		DevStats:    map[string][]uint64{},
		devScanNum:  map[string]int{},
		scanNum:     netDev.scanNum,
		path:        netDev.path,
		validHeader: make([]byte, len(netDev.validHeader)),
	}

	for dev, devStats := range netDev.DevStats {
		newNetDev.DevStats[dev] = make([]uint64, len(devStats))
		if full {
			copy(newNetDev.DevStats[dev], devStats)
		}
	}

	for dev, scanNum := range netDev.devScanNum {
		newNetDev.devScanNum[dev] = scanNum
	}

	copy(newNetDev.validHeader, netDev.validHeader)

	return newNetDev
}

func (netDev *NetDev) ValidateHeader(buf []byte, numLines int) {
	off, lineCnt := 0, 0
	for ; off < len(buf) && lineCnt < numLines; off++ {
		if buf[off] == '\n' {
			lineCnt++
		}
	}
	if lineCnt < numLines {
		return
	}
	header := buf[:off]
	normHeader := normalizeNetDevHeader(header)
	for _, validNormHeader := range netDevValidNormHeaders {
		if bytes.Equal(normHeader, validNormHeader) {
			netDev.validHeader = make([]byte, off)
			copy(netDev.validHeader, header)
			break
		}
	}
}

func (netDev *NetDev) makeErrorLine(buf []byte, devStart int, reason any) error {
	if buf != nil {
		line := buf[devStart:]
		lineEnd := bytes.IndexByte(line, '\n')
		if lineEnd > 0 {
			line = line[:lineEnd]
		}
		return fmt.Errorf("%s: %q: %v", netDev.path, string(line), reason)
	} else {
		return fmt.Errorf("%s: %v", netDev.path, reason)
	}
}

func (netDev *NetDev) Parse() error {
	bBuf, err := ReadFileBufPool256k.ReadFile(netDev.path)
	if err != nil {
		return err
	}
	defer ReadFileBufPool32k.ReturnBuf(bBuf)

	buf, l := bBuf.Bytes(), bBuf.Len()

	validHeader := netDev.validHeader
	statsOff := len(validHeader)
	if validHeader == nil {
		netDev.ValidateHeader(buf, 2)
		validHeader = netDev.validHeader
		if validHeader == nil {
			return fmt.Errorf("%s: unsupported file header", netDev.path)
		}
		statsOff = len(validHeader)
	} else if l < statsOff || !bytes.Equal(validHeader, buf[:statsOff]) {
		return fmt.Errorf("%s: invalid/changed file header", netDev.path)
	}

	scanNum := netDev.scanNum + 1

	for pos := statsOff; pos < l; pos++ {
		// Skip over spaces/empty lines:
		for ; pos < l && netDevIsSepNl[buf[pos]]; pos++ {
		}
		if pos == l {
			break
		}

		// Extract the device spec:
		devStart, dev := pos, ""
		for ; pos < l; pos++ {
			if c := buf[pos]; c == ':' {
				dev = string(buf[devStart:pos])
				pos++
				break
			} else if netDevIsSepNl[c] {
				break
			}
		}
		if dev == "" {
			return netDev.makeErrorLine(buf, devStart, "missing `DEV:'")
		}

		// Extract stats values:
		devStats := netDev.DevStats[dev]
		if devStats == nil {
			devStats = make([]uint64, NET_DEV_NUM_STATS)
			netDev.DevStats[dev] = devStats
		}

		eol, statIndex := false, 0
		for !eol && pos < l && statIndex < NET_DEV_NUM_STATS {
			for ; pos < l && netDevIsSep[buf[pos]]; pos++ {
			}
			value, hasValue := uint64(0), false
			for ; pos < l; pos++ {
				c := buf[pos]
				digit := c - '0'
				if digit < 10 {
					value = (value << 3) + (value << 1) + uint64(digit)
					hasValue = true
				} else if netDevIsSepNl[c] {
					eol = (c == '\n')
					pos++
					break
				} else {
					return netDev.makeErrorLine(buf, devStart, "invalid value")
				}
			}
			if hasValue {
				devStats[statIndex] = value
				statIndex++
			}
		}

		// All values retrieved?
		if statIndex < NET_DEV_NUM_STATS {
			return netDev.makeErrorLine(buf, devStart, "not enough values")
		}

		// Advance to EOL:
		for ; !eol && pos < l; pos++ {
			if c := buf[pos]; c == '\n' {
				eol = true
			} else if !netDevIsSep[buf[pos]] {
				return netDev.makeErrorLine(buf, devStart, "invalid value")
			}
		}

		// Update scan# for device:
		netDev.devScanNum[dev] = scanNum
	}

	// Remove devices not found at this scan:
	for dev, devScanNum := range netDev.devScanNum {
		if scanNum != devScanNum {
			delete(netDev.DevStats, dev)
		}
	}
	netDev.scanNum = scanNum

	return nil
}
