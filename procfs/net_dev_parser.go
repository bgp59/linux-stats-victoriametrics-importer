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
	// Stats per line:
	DevStats [][]uint64

	// Indexed by device name; DevStats[DevStatsIndex[dev]]
	DevStatsIndex map[string]int
	// The path file to  read:
	path string
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
// header of the file is checked, once/1st time, against the known headers
// listed below:
var netDevValidHeaders = [][]byte{
	// Note: to make it easier to cut and paste actual lines, they are enclosed
	// between `` marks, the latter on *separate* lines for readability. This
	// introduces a `\n` as the first byte and it has to be removed from
	// comparison, hence the [1:] slice construct.
	[]byte(`
Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
`)[1:],
}

func NewNetDev(procfsRoot string) *NetDev {
	return &NetDev{
		DevStats:      make([][]uint64, 0),
		DevStatsIndex: map[string]int{},
		path:          path.Join(procfsRoot, "net", "dev"),
	}
}

func (netDev *NetDev) Clone(full bool) *NetDev {
	newNetDev := &NetDev{
		DevStats:       make([][]uint64, len(netDev.DevStats)),
		DevStatsIndex:  map[string]int{},
		path:           netDev.path,
		validHeader:    make([]byte, len(netDev.validHeader)),
		numLinesHeader: netDev.numLinesHeader,
	}

	for i, devStats := range netDev.DevStats {
		newNetDev.DevStats[i] = make([]uint64, NET_DEV_NUM_STATS)
		if full {
			copy(newNetDev.DevStats[i], devStats)
		}
	}

	if full {
		for dev, index := range netDev.DevStatsIndex {
			newNetDev.DevStatsIndex[dev] = index
		}
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

	// Max out the capacity already available in dev stats. The actual length
	// will be adjusted at the end:
	devStats := netDev.DevStats[:cap(netDev.DevStats)]

	// Initialize dev index w/ -1; the actual index will be updated as devices
	// are being parsed and entries for devices no longer available will be
	// pruned at the end:
	devStatsIndex := netDev.DevStatsIndex
	for dev := range devStatsIndex {
		devStatsIndex[dev] = -1
	}

	devIndex := 0
	for pos := statsOff; pos < l; devIndex++ {
		// Start parsing a new line:
		if len(devStats) <= devIndex {
			devStats = append(devStats, make([]uint64, NET_DEV_NUM_STATS))
		}
		stats := devStats[devIndex]

		lineStart, eol := pos, false
		for ; pos < l && isWhitespace[buf[pos]]; pos++ {
		}

		// Extract the device:
		hasDev := false
		for devStart, done := pos, false; !done && pos < l; pos++ {
			c := buf[pos]
			if c == ':' {
				if devStart < pos-1 {
					devStatsIndex[string(buf[devStart:pos])] = devIndex
					hasDev = true
				}
				done = true
			} else if eol = (c == '\n'); eol || isWhitespace[c] {
				done = true
			}
		}
		if !hasDev {
			return fmt.Errorf(
				"%s#%d: %q: missing `DEV:'",
				netDev.path, netDev.numLinesHeader+devIndex+1, getCurrentLine(buf, lineStart),
			)
		}

		// Extract stats values:
		statIndex := 0
		for !eol && pos < l && statIndex < NET_DEV_NUM_STATS {
			for ; pos < l && isWhitespace[buf[pos]]; pos++ {
			}
			value, hasValue := uint64(0), false
			for done := false; !done && pos < l; pos++ {
				c := buf[pos]
				digit := c - '0'
				if digit < 10 {
					value = (value << 3) + (value << 1) + uint64(digit)
					hasValue = true
				} else if eol = (c == '\n'); eol || isWhitespace[c] {
					done = true
				} else {
					return fmt.Errorf(
						"%s#%d: %q: invalid value",
						netDev.path, netDev.numLinesHeader+devIndex+1, getCurrentLine(buf, lineStart),
					)
				}
			}
			if hasValue {
				stats[statIndex] = value
				statIndex++
			}
		}

		// All values retrieved?
		if statIndex < NET_DEV_NUM_STATS {
			return fmt.Errorf(
				"%s#%d: %q: not enough values (< %d)",
				netDev.path, netDev.numLinesHeader+devIndex+1, getCurrentLine(buf, lineStart), NET_DEV_NUM_STATS,
			)
		}

		// Advance to EOL:
		for ; !eol && pos < l; pos++ {
			c := buf[pos]
			if eol = (c == '\n'); !eol && !isWhitespace[c] {
				return fmt.Errorf(
					"%s#%d: %q: invalid value",
					netDev.path, netDev.numLinesHeader+devIndex+1, getCurrentLine(buf, lineStart),
				)
			}
		}
	}

	// Trim back dev stats to match the actual number of devices:
	netDev.DevStats = devStats[:devIndex]

	// Prune dev stats index for devices no longer found:
	for dev, index := range devStatsIndex {
		if index == -1 {
			delete(devStatsIndex, dev)
		}
	}

	return nil
}
