// Parser for /proc/diskstats

package procfs

import (
	"fmt"
	"path"

	"github.com/bgp59/linux-stats-victoriametrics-importer/internal/utils"
)

// Reference:
//  https://github.com/torvalds/linux/blob/master/Documentation/admin-guide/iostats.rst

// Each line of the file will be parsed into 2 parts:
//  - info: major:minor and device name
//  - stats: []uint32 indexed by field#-1
// major:minor will be used to index the name and stats

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
	// Device stats and info, indexed by "major:minor":
	DevInfoMap map[string]*DiskstatsDevInfo
	// Whether there was any change in disk info (major:minor or name) from the
	// previous scan or not; a change here may be used to force an early parse
	// (i.e. not waiting for a full cycle) of related info, such as mount info:
	Changed bool
	// Devices may be appear/disappear dynamically. To keep track of removals,
	// each parse invocation is associated with a different from before scan#
	// and each found majorMinor will be updated below for it. At the end of the
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

func DiskstatsPath(procfsRoot string) string {
	return path.Join(procfsRoot, "diskstats")
}

func NewDiskstats(procfsRoot string) *Diskstats {
	newDiskstats := &Diskstats{
		DevInfoMap:      make(map[string]*DiskstatsDevInfo),
		scanNum:         0,
		path:            DiskstatsPath(procfsRoot),
		fieldsInJiffies: diskstatsFieldsInJiffies,
	}

	if utils.OSNameNorm == "linux" && len(utils.OSReleaseVer) > 0 && utils.OSReleaseVer[0] >= 5 {
		newDiskstats.jiffiesToMillisec = uint32(utils.LinuxClktckSec * 1000.)
	}

	return newDiskstats
}

func (diskstats *Diskstats) Clone(full bool) *Diskstats {
	newDiskstats := &Diskstats{
		DevInfoMap:        make(map[string]*DiskstatsDevInfo),
		Changed:           diskstats.Changed,
		scanNum:           diskstats.scanNum,
		path:              diskstats.path,
		jiffiesToMillisec: diskstats.jiffiesToMillisec,
		fieldsInJiffies:   diskstats.fieldsInJiffies,
	}

	for majorMinor, devInfo := range diskstats.DevInfoMap {
		newDiskstats.DevInfoMap[majorMinor] = &DiskstatsDevInfo{
			Name:    devInfo.Name,
			Stats:   make([]uint32, DISKSTATS_VALUE_FIELDS_NUM),
			scanNum: devInfo.scanNum,
		}
		if full {
			copy(newDiskstats.DevInfoMap[majorMinor].Stats, devInfo.Stats)
		}
	}

	return newDiskstats
}

func (diskstats *Diskstats) Parse() error {
	fBuf, err := diskstatsReadFileBufPool.ReadFile(diskstats.path)
	defer diskstatsReadFileBufPool.ReturnBuf(fBuf)
	if err != nil {
		return err
	}

	buf, l := fBuf.Bytes(), fBuf.Len()

	diskstats.Changed = false
	devInfoMap := diskstats.DevInfoMap
	jiffiesToMillisec, fieldsInJiffies := diskstats.jiffiesToMillisec, diskstats.fieldsInJiffies
	scanNum := diskstats.scanNum + 1
	var (
		fieldNum                int
		major, majorMinor, name string
	)
	for pos, lineNum := 0, 1; pos < l; lineNum++ {
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
					majorMinor = major + ":" + string(buf[fieldStart:pos])
				case DISKSTATS_DEVICE_NAME:
					name = string(buf[fieldStart:pos])
				}
				fieldNum++
			}
		}
		if fieldNum < DISKSTATS_INFO_FIELDS_NUM {
			return fmt.Errorf(
				"%s:%d: %q: missing info fields: want: %d, got: %d",
				diskstats.path, lineNum, getCurrentLine(buf, lineStart), DISKSTATS_INFO_FIELDS_NUM, fieldNum,
			)
		}

		devInfo := devInfoMap[majorMinor]
		if devInfo == nil {
			devInfo = &DiskstatsDevInfo{
				Name:  name,
				Stats: make([]uint32, DISKSTATS_VALUE_FIELDS_NUM),
			}
			devInfoMap[majorMinor] = devInfo
			diskstats.Changed = true
		} else if devInfo.Name != name {
			devInfo.Name = name
			diskstats.Changed = true
		}
		stats := devInfo.Stats
		for fieldNum = 0; !eol && pos < l && fieldNum < DISKSTATS_VALUE_FIELDS_NUM; pos++ {
			for ; pos < l && isWhitespace[buf[pos]]; pos++ {
			}
			value, valueFound := uint32(0), false
			for ; pos < l; pos++ {
				c := buf[pos]
				if digit := c - '0'; digit < 10 {
					value = (value << 3) + (value << 1) + uint32(digit)
					valueFound = true
				} else if eol = (c == '\n'); eol || isWhitespace[c] {
					break
				} else {
					return fmt.Errorf(
						"%s:%d: %q: `%c' not a valid digit",
						diskstats.path, lineNum, getCurrentLine(buf, lineStart), c,
					)
				}
			}
			if valueFound {
				if jiffiesToMillisec > 0 && fieldsInJiffies[fieldNum] {
					value *= jiffiesToMillisec
				}
				stats[fieldNum] = value
				fieldNum++
			}
		}
		if fieldNum < minNumDiskstatsValues {
			return fmt.Errorf(
				"%s:%d: %q: missing stats fields: want (at least): %d, got: %d",
				diskstats.path, lineNum, getCurrentLine(buf, lineStart), minNumDiskstatsValues, fieldNum,
			)
		}

		// Advance to EOL:
		for ; !eol && pos < l; pos++ {
			c := buf[pos]
			if eol = (c == '\n'); !eol && !isWhitespace[c] {
				return fmt.Errorf(
					"%s:%d: %q: %q: unexpected content after the last field",
					diskstats.path, lineNum, getCurrentLine(buf, lineStart), getCurrentLine(buf, pos),
				)
			}
		}

		// Update scan# for device:
		devInfo.scanNum = scanNum
	}

	// Remove devices not found at this scan:
	for majorMinor, devInfo := range diskstats.DevInfoMap {
		if scanNum != devInfo.scanNum {
			delete(diskstats.DevInfoMap, majorMinor)
			diskstats.Changed = true
		}
	}
	diskstats.scanNum = scanNum

	return nil
}
