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

	// Must be last! See NetDev struct for explanation about the scan#:
	NET_DEV_SCAN_NUMBER
)

const (
	NET_DEV_NUM_STATS = NET_DEV_SCAN_NUMBER + 1
)

type NetDev struct {
	// Stats indexed by device name:
	DevStats map[string][]uint64

	// The path file to  read:
	path string

	// Devices may appear/disappear dynamically. To detect and remove deleted
	// devices from DevStats, the scan# below is incremented at the beginning of
	// the scan and each device found at the current scan will have its
	// NET_DEV_SCAN_NUMBER value updated with it. At the end of the scan, all
	// devices found NET_DEV_SCAN_NUMBER not matching will be removed.
	scanNum uint64

	// The parser assumes certain fields, based on the first N lines. The file
	// will be validated only for the 1st pass, since the file syntax cannot
	// change without a kernel change, i.e. a reboot. The validated header is
	// remembered and it will be checked for changes at each pass as a sanity
	// check:
	validHeader    []byte
	numLinesHeader int
}

// Read the entire file in one go, using a ReadFileBufPool:
var netDevReadFileBufPool = ReadFileBufPool256k

// To protect against changes in kernel that may alter the exposed stats, the
// header of the file is checked once (1st time), against the known headers
// listed below:
var netDevValidHeaders = [][]byte{
	// Note: to make it easier to cut and paste actual lines, they are enclosed
	// between `` marks, the latter on *separate* lines for readability. This
	// introduces a `\n' as the first byte and it has to be removed from
	// comparison, hence the [1:] slice construct.
	[]byte(`
Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
`)[1:],
}

func NewNetDev(procfsRoot string) *NetDev {
	return &NetDev{
		DevStats: map[string][]uint64{},
		path:     path.Join(procfsRoot, "net", "dev"),
	}
}

func (netDev *NetDev) Clone(full bool) *NetDev {
	newNetDev := &NetDev{
		DevStats:       map[string][]uint64{},
		path:           netDev.path,
		validHeader:    make([]byte, len(netDev.validHeader)),
		numLinesHeader: netDev.numLinesHeader,
	}

	for dev, devStats := range netDev.DevStats {
		newNetDev.DevStats[dev] = make([]uint64, NET_DEV_NUM_STATS)
		if full {
			copy(newNetDev.DevStats[dev], devStats)
		}
	}

	if full {
		newNetDev.scanNum = netDev.scanNum
	}

	copy(newNetDev.validHeader, netDev.validHeader)

	return newNetDev
}

func (netDev *NetDev) ValidateHeader(buf []byte) {
	for _, header := range netDevValidHeaders {
		off := len(header)
		if off <= len(buf) && bytes.Equal(header, buf[:off]) {
			netDev.validHeader = make([]byte, off)
			copy(netDev.validHeader, header)
			netDev.numLinesHeader = 0
			for _, c := range header {
				if c == '\n' {
					netDev.numLinesHeader++
				}
			}
			break
		}
	}
}

func (netDev *NetDev) Parse() error {
	fBuf, err := netDevReadFileBufPool.ReadFile(netDev.path)
	if err != nil {
		return err
	}
	defer netDevReadFileBufPool.ReturnBuf(fBuf)

	buf, l := fBuf.Bytes(), fBuf.Len()

	validHeader := netDev.validHeader
	statsOff := len(validHeader)
	if validHeader == nil {
		netDev.ValidateHeader(buf)
		validHeader = netDev.validHeader
		if validHeader == nil {
			return fmt.Errorf("%s: unsupported file header", netDev.path)
		}
		statsOff = len(validHeader)
	} else if l < statsOff || !bytes.Equal(validHeader, buf[:statsOff]) {
		return fmt.Errorf("%s: invalid/changed file header", netDev.path)
	}

	scanNum := netDev.scanNum + 1
	for pos, lineNum := statsOff, netDev.numLinesHeader+1; pos < l; lineNum++ {
		// New line starts here:
		lineStart, eol := pos, false

		// Extract the device:
		for ; pos < l && isWhitespace[buf[pos]]; pos++ {
		}
		dev := ""
		for devStart, done := pos, false; !done && pos < l; pos++ {
			c := buf[pos]
			if c == ':' {
				if devStart < pos-1 {
					dev = string(buf[devStart:pos])
				}
				done = true
			} else if eol = (c == '\n'); eol || isWhitespace[c] {
				done = true
			}
		}
		if dev == "" {
			return fmt.Errorf(
				"%s#%d: %q: missing `DEV:'",
				netDev.path, lineNum, getCurrentLine(buf, lineStart),
			)
		}

		stats := netDev.DevStats[dev]
		if stats == nil {
			stats = make([]uint64, NET_DEV_NUM_STATS)
			netDev.DevStats[dev] = stats
		}

		// Extract stats values:
		statIndex := 0
		for !eol && pos < l && statIndex < NET_DEV_SCAN_NUMBER {
			for ; pos < l && isWhitespace[buf[pos]]; pos++ {
			}
			value, hasValue := uint64(0), false
			for done := false; !done && pos < l; pos++ {
				c := buf[pos]
				if digit := c - '0'; digit < 10 {
					value = (value << 3) + (value << 1) + uint64(digit)
					hasValue = true
				} else if eol = (c == '\n'); eol || isWhitespace[c] {
					done = true
				} else {
					return fmt.Errorf(
						"%s#%d: %q: invalid value",
						netDev.path, lineNum, getCurrentLine(buf, lineStart),
					)
				}
			}
			if hasValue {
				stats[statIndex] = value
				statIndex++
			}
		}

		// All values retrieved?
		if statIndex < NET_DEV_SCAN_NUMBER {
			return fmt.Errorf(
				"%s#%d: %q: not enough values: want: %d, got: %d",
				netDev.path, lineNum, getCurrentLine(buf, lineStart), NET_DEV_SCAN_NUMBER, statIndex,
			)
		}

		// Sync scan# for this device:
		stats[NET_DEV_SCAN_NUMBER] = scanNum

		// Advance to EOL:
		for ; !eol && pos < l; pos++ {
			c := buf[pos]
			if eol = (c == '\n'); !eol && !isWhitespace[c] {
				return fmt.Errorf(
					"%s#%d: %q: %q: unexpected content after dev counters",
					netDev.path, lineNum, getCurrentLine(buf, lineStart), getCurrentLine(buf, pos),
				)
			}
		}
	}

	// Prune dev stats index for devices no longer found:
	for dev, stats := range netDev.DevStats {
		if stats[NET_DEV_SCAN_NUMBER] != scanNum {
			delete(netDev.DevStats, dev)
		}
	}

	// Update scan#:
	netDev.scanNum = scanNum

	return nil
}
