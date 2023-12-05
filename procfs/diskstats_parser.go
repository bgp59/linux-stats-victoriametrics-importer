// Parser for /proc/diskstats

package procfs

import (
	"bytes"
	"fmt"
	"path"
)

// Reference:
//  https://github.com/torvalds/linux/blob/master/Documentation/admin-guide/iostats.rst

// The information will be presented as a map indexed by the device major:minor
// with the values presented as a []uint32 slice since most values being
// unsigned long and the few rest can be represented as such.

// Indexes for major, minor and device name:
const (
	DISKSTATS_MAJOR_NUM = iota
	DISKSTATS_MINOR_NUM
	DISKSTATS_DEVICE_NAME

	// Must by last:
	DISKSTATS_INFO_FIELDS_NUM
)

// Indexes for values mirror the field#-1 from the documentation file:
const (
	DISKSTATS_NUM_READS_COMPLETED = iota
	DISKSTATS_NUM_READS_MERGED
	DISKSTATS_NUM_READ_SECTORS
	DISKSTATS_READ_MILLISEC
	DISKSTATS_NUM_WRITES_COMPLETED
	DISKSTATS_NUM_WRITES_MERGED
	DISKSTATS_NUM_WRITE_SECTORS
	DISKSTATS_WRITE_MILLISEC
	DISKSTATS_NUM_IO_IN_PROGRESS
	DISKSTATS_IO_MILLISEC
	DISKSTATS_IO_WEIGTHED_MILLISEC
	DISKSTATS_NUM_DISCARDS_COMPLETED
	DISKSTATS_NUM_DISCARDS_MERGED
	DISKSTATS_NUM_DISCARD_SECTORS
	DISKSTATS_DISCARD_MILLISEC
	DISKSTATS_NUM_FLUSH_REQUESTS
	DISKSTATS_FLUSH_MILLISEC

	// Must be last:
	DISKSTATS_VALUE_FIELDS_NUM
)

// The actual number of fields, which could be < DISKSTATS_VALUE_FIELDS_NUM for older
// versions of the kernel; define the minimum number expected:
var minNumDiskstatsValues = 10

// Some fields may need conversion from jiffies to millisec:
var diskstatsFieldsInJiffies = [DISKSTATS_VALUE_FIELDS_NUM]bool{
	DISKSTATS_IO_MILLISEC: true,
}

type DiskstatsDevInfo struct {
	Name  string
	Stats []uint32
	// Devices may be appear/disappear dynamically. To keep track of deletion,
	// each parse invocation is associated with a different from before scan#
	// and each found device will be updated below for it. At the end of the
	// pass, the devices that have a different scan# are leftover from a
	// previous scan and they are deleted from the stats.
	scanNum int
}

type Diskstats struct {
	// All device stats and info is indexed by "major:minor":
	DevInfoMap map[string]*DiskstatsDevInfo
	// Devices may be appear/disappear dynamically. To keep track of deletion,
	// each parse invocation is associated with a different from before scan#
	// and each found devMajMin will be updated below for it. At the end of the
	// pass, the devices that have a different scan# are leftover from a
	// previous scan and they are deleted from the stats.
	scanNum int
	// The path file to  read:
	path string
	// Jiffies -> millisec conversion info; keep it per-instance to allow
	// per-instance overriding:
	// Conversion factor, use 0 to disable:
	jiffiesToMillisec uint32
	// Fields that need conversion:
	fieldsInJiffies [DISKSTATS_VALUE_FIELDS_NUM]bool
}

// Read the entire file in one go, using a ReadFileBufPool:
var diskstatsReadFileBufPool = ReadFileBufPool256k

func NewDiskstats(procfsRoot string) *Diskstats {
	newDiskstats := &Diskstats{
		DevInfoMap:      make(map[string]*DiskstatsDevInfo),
		scanNum:         0,
		path:            path.Join(procfsRoot, "diskstats"),
		fieldsInJiffies: diskstatsFieldsInJiffies,
	}

	if OSName == "linux" && len(OSReleaseVer) > 0 && OSReleaseVer[0] >= 5 {
		newDiskstats.jiffiesToMillisec = uint32(LinuxClktckSec * 1000.)
	}

	return newDiskstats
}

func (diskstats *Diskstats) Clone(full bool) *Diskstats {
	newDiskstats := &Diskstats{
		DevInfoMap:        make(map[string]*DiskstatsDevInfo),
		scanNum:           diskstats.scanNum,
		path:              diskstats.path,
		jiffiesToMillisec: diskstats.jiffiesToMillisec,
		fieldsInJiffies:   diskstats.fieldsInJiffies,
	}

	for devMajMin, devInfo := range diskstats.DevInfoMap {
		newDiskstats.DevInfoMap[devMajMin] = &DiskstatsDevInfo{
			Name:    devInfo.Name,
			Stats:   make([]uint32, len(devInfo.Stats)),
			scanNum: devInfo.scanNum,
		}
		if full {
			copy(newDiskstats.DevInfoMap[devMajMin].Stats, devInfo.Stats)

		}
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
	fBuf, err := diskstatsReadFileBufPool.ReadFile(diskstats.path)
	if err != nil {
		return err
	}
	defer diskstatsReadFileBufPool.ReturnBuf(fBuf)

	buf, l := fBuf.Bytes(), fBuf.Len()

	devInfoMap := diskstats.DevInfoMap
	jiffiesToMillisec, fieldsInJiffies := diskstats.jiffiesToMillisec, diskstats.fieldsInJiffies
	scanNum := diskstats.scanNum + 1
	for pos, lineNum := 0, 1; pos < l; lineNum++ {
		var (
			fieldNum               int
			major, devMajMin, name string
		)
		lineStart, eol := pos, false

		for fieldNum = 0; !eol && pos < l && fieldNum < DISKSTATS_INFO_FIELDS_NUM; pos++ {
			for ; pos < l && isWhitespace[buf[pos]]; pos++ {
			}
			fieldStart := pos
			for ; pos < l; pos++ {
				c := buf[pos]
				if eol = (c == '\n'); eol || isWhitespace[c] {
					break
				}
			}
			if fieldStart < pos {
				switch fieldNum {
				case DISKSTATS_MAJOR_NUM:
					major = string(buf[fieldStart:pos])
				case DISKSTATS_MINOR_NUM:
					devMajMin = major + ":" + string(buf[fieldStart:pos])
				case DISKSTATS_DEVICE_NAME:
					name = string(buf[fieldStart:pos])
				}
				fieldNum++
			}
		}
		if fieldNum < DISKSTATS_INFO_FIELDS_NUM {
			return fmt.Errorf(
				"%s#%d: %q: missing info fields (< %d)",
				diskstats.path, lineNum, getCurrentLine(buf, lineStart), DISKSTATS_INFO_FIELDS_NUM,
			)
		}

		devInfo := devInfoMap[devMajMin]
		if devInfo == nil {
			devInfo = &DiskstatsDevInfo{
				Name:  name,
				Stats: make([]uint32, DISKSTATS_VALUE_FIELDS_NUM),
			}
			devInfoMap[devMajMin] = devInfo
		} else if devInfo.Name != name {
			devInfo.Name = name
		}
		stats := devInfo.Stats
		for fieldNum = 0; !eol && pos < l && fieldNum < DISKSTATS_VALUE_FIELDS_NUM; pos++ {
			for ; pos < l && isWhitespace[buf[pos]]; pos++ {
			}
			fieldStart, value := pos, uint32(0)
			for ; pos < l; pos++ {
				c := buf[pos]
				if digit := c - '0'; digit < 10 {
					value = (value << 3) + (value << 1) + uint32(digit)
				} else if eol = (c == '\n'); eol || isWhitespace[c] {
					break
				} else {
					return diskstats.makeErrorLine(buf, lineStart, "invalid value")
				}
			}
			if fieldStart < pos {
				if jiffiesToMillisec > 0 && fieldsInJiffies[fieldNum] {
					value *= jiffiesToMillisec
				}
				stats[fieldNum] = value
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
			c := buf[pos]
			if eol = (c == '\n'); !eol && !isWhitespace[c] {
				return diskstats.makeErrorLine(buf, lineStart, "invalid value")
			}
		}

		// Update scan# for device:
		devInfo.scanNum = scanNum
	}

	// Remove devices not found at this scan:
	for devMajMin, devInfo := range diskstats.DevInfoMap {
		if scanNum != devInfo.scanNum {
			delete(diskstats.DevInfoMap, devMajMin)
		}
	}
	diskstats.scanNum = scanNum

	return nil
}
