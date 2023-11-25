// Parser for /proc/diskstats

package procfs

import (
	"bytes"
	"fmt"
	"path"
)

// Reference:
//  https://github.com/torvalds/linux/blob/master/Documentation/admin-guide/iostats.rst

// The information will be presented as a map indexed bt the dev name with
// the values presented as a []uint32 slice since most values being unsigned
// long and the few rest can be represented as such.

// Indexes for values:
const (
	DISKSTATS_MAJOR_NUM = iota
	DISKSTATS_MINOR_NUM
	// The following index is not associated w/ a numerical value, but it is
	// included here to ensure a 1:1 mapping between file field# and stats index:
	DISKSTATS_DEVICE
	DISKSTATS_NUM_READS_COMPLETED
	DISKSTATS_NUM_READS_MERGED
	DISKSTATS_NUM_READ_SECTORS
	DISKSTATS_READ_MILLISEC
	DISKSTATS_NUM_WRITES_COMPLETED
	DISKSTATS_NUM_WRITES_MERGED
	DISKSTATS_NUM_WRITE_SECTORS
	DISKSTATS_WRITE_MILLISEC
	DISKSTATS_NUM_IO_IN_PROGRESS
	DISKSTATS_IO_MILLISEC_OR_JIFFIES
	DISKSTATS_IO_WEIGTHED_MILLISEC
	DISKSTATS_NUM_DISCARDS_COMPLETED
	DISKSTATS_NUM_DISCARDS_MERGED
	DISKSTATS_NUM_DISCARD_SECTORS
	DISKSTATS_DISCARD_MILLISEC
	DISKSTATS_FLUSH_MILLISEC

	// Must be last:
	NUM_DISKSTATS_VALUES
)

const DISKSTATS_DEVICE_FIELD_NUM = DISKSTATS_DEVICE

// The actual number of fields, which could be < NUM_DISKSTATS_VALUES for older
// versions of the kernel; define the minimum number expected:
var minNumDiskstatsValues = 10

type Diskstats struct {
	DevStats map[string][]uint32
	// Devices may be appear/disappear dynamically. To keep track of deletion,
	// each parse invocation is associated with a different from before scan#
	// and each found dev will be updated below for it. At the end of the
	// pass, the devices that have a different scan# are leftover from a
	// previous scan and they are deleted from the stats.
	devScanNum map[string]int
	scanNum    int
	// The path file to  read:
	path string
}

// Read the entire file in one go, using a ReadFileBufPool:
var diskstatsReadFileBufPool = ReadFileBufPool256k

// Word separators:
var diskstatsIsSep = [256]bool{
	' ':  true,
	'\t': true,
}

var diskstatsIsSepNl = [256]bool{
	' ':  true,
	'\t': true,
	'\n': true,
}

func NewDiskstats(procfsRoot string) *Diskstats {
	return &Diskstats{
		DevStats:   make(map[string][]uint32),
		devScanNum: make(map[string]int),
		scanNum:    -1,
		path:       path.Join(procfsRoot, "diskstats"),
	}
}

func (diskstats *Diskstats) Clone(full bool) *Diskstats {
	newDiskstats := &Diskstats{
		DevStats:   make(map[string][]uint32),
		devScanNum: make(map[string]int),
		scanNum:    diskstats.scanNum,
		path:       diskstats.path,
	}

	for dev := range diskstats.DevStats {
		newDiskstats.DevStats[dev] = make([]uint32, NUM_DISKSTATS_VALUES)
		if full {
			copy(newDiskstats.DevStats[dev], diskstats.DevStats[dev])
		}
	}

	for dev, scanNum := range diskstats.devScanNum {
		newDiskstats.devScanNum[dev] = scanNum
	}

	return newDiskstats
}

func (diskstats *Diskstats) makeErrorLine(buf []byte, lineStart int, reason any) error {
	if buf != nil {
		line := buf[lineStart:]
		lineEnd := bytes.IndexByte(line, '\n')
		if lineEnd > 0 {
			line = line[:lineEnd]
		}
		return fmt.Errorf("%s: %q: %v", diskstats.path, string(line), reason)
	} else {
		return fmt.Errorf("%s: %v", diskstats.path, reason)
	}
}

func (diskstats *Diskstats) Parse() error {
	bBuf, err := diskstatsReadFileBufPool.ReadFile(diskstats.path)
	if err != nil {
		return err
	}
	defer diskstatsReadFileBufPool.ReturnBuf(bBuf)

	buf, l := bBuf.Bytes(), bBuf.Len()

	devStats, stats := diskstats.DevStats, []uint32(nil)
	scanNum := diskstats.scanNum + 1
	for pos := 0; pos < l; pos++ {
		lineStart, eol, fieldNum := pos, false, 0
		major, minor, dev := uint32(0), uint32(0), ""

		for ; !eol && pos < l && fieldNum < NUM_DISKSTATS_VALUES; pos++ {
			for ; pos < l && diskstatsIsSep[buf[pos]]; pos++ {
			}
			fieldStart, value := pos, uint32(0)
			for ; pos < l; pos++ {
				c := buf[pos]
				if fieldNum != DISKSTATS_DEVICE {
					if digit := c - '0'; digit < 10 {
						value += (value << 3) + (value << 1) + uint32(digit)
					} else if diskstatsIsSepNl[c] {
						eol = (c == '\n')
						break
					} else {
						return diskstats.makeErrorLine(buf, lineStart, "invalid value")
					}
				} else if diskstatsIsSepNl[c] {
					dev = string(buf[fieldStart:pos])
					eol = (c == '\n')
					break
				}
			}
			if fieldStart > pos {
				switch fieldNum {
				case DISKSTATS_MAJOR_NUM:
					major = value
				case DISKSTATS_MINOR_NUM:
					minor = value
				case DISKSTATS_DEVICE:
					stats = devStats[dev]
					if stats == nil {
						stats = make([]uint32, NUM_DISKSTATS_VALUES)
						devStats[dev] = stats
					}
					stats[DISKSTATS_MAJOR_NUM] = major
					stats[DISKSTATS_MINOR_NUM] = minor
				default:
					stats[fieldNum] = value
				}
				fieldNum++
			}
		}

		if fieldNum < minNumDiskstatsValues {
			return diskstats.makeErrorLine(
				buf, lineStart,
				fmt.Errorf("missing fields (< %d)", minNumDiskstatsValues),
			)
		}

		// Advance to EOL:
		for ; !eol && pos < l; pos++ {
			if c := buf[pos]; c == '\n' {
				eol = true
			} else if !diskstatsIsSep[buf[pos]] {
				return diskstats.makeErrorLine(buf, lineStart, "invalid value")
			}
		}

		// Update scan# for device:
		diskstats.devScanNum[dev] = scanNum
	}

	// Remove devices not found at this scan:
	for dev, devScanNum := range diskstats.devScanNum {
		if scanNum != devScanNum {
			delete(diskstats.DevStats, dev)
		}
	}
	diskstats.scanNum = scanNum

	return nil
}
