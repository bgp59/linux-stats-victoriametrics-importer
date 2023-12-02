// parser for /proc/pid/mountinfo

package procfs

import (
	"bufio"
	"bytes"
	"fmt"
	"path"
	"strconv"
	"strings"
)

// Reference:
// https://man7.org/linux/man-pages/man5/proc.5.html

const (
	// 0 based indices for mountinfo file fields as well as for the parsed
	// information:
	MOUNTINFO_MOUNT_ID = iota
	MOUNTINFO_PARENT_ID
	MOUNTINFO_MAJOR_MINOR
	MOUNTINFO_ROOT
	MOUNTINFO_MOUNT_POINT
	MOUNTINFO_MOUNT_OPTIONS
	MOUNTINFO_OPTIONAL_FIELDS
	MOUNTINFO_OPTIONAL_FIELDS_SEPARATOR
	MOUNTINFO_FS_TYPE
	MOUNTINFO_MOUNT_SOURCE
	MOUNTINFO_SUPER_OPTIONS

	// Must be last:
	MOUNTINFO_NUM_FIELDS
)

const (
	// The optional fields will be terminated by this string:
	MOUNTINFO_OPTIONAL_FIELDS_TERMINATOR = "-"

	// The optional fields will be presented as a single string joined by the
	// following separator:
	MOUNTINFO_OPTIONAL_FIELDS_JOIN_SEP = ","
)

type Mountinfo struct {
	// The info is indexed by the device "major:minor":
	DevMountInfo map[string][]string

	// The file is not expected to change very often, so in order to avoid a
	// rather expensive parsing, its previous content is cached and the parsing
	// occurs only if there are changes.
	Changed                 bool
	ParseCount, ChangeCount uint64

	// Cache for content:
	content *bytes.Buffer

	// The path file to  read:
	path string
}

// Read the entire file in one go, using a ReadFileBufPool:
var mountinfoReadFileBufPool = ReadFileBufPool256k

// Word separators:
var mountinfoIsSep = [256]bool{
	' ':  true,
	'\t': true,
}

func NewMountInfo(procfsRoot string, pid int) *Mountinfo {
	return &Mountinfo{
		DevMountInfo: map[string][]string{},
		content:      &bytes.Buffer{},
		path:         path.Join(procfsRoot, strconv.Itoa(pid), "mountinfo"),
	}
}

func (mountinfo *Mountinfo) Clone() *Mountinfo {
	newMountInfo := &Mountinfo{
		DevMountInfo: map[string][]string{},
		ParseCount:   mountinfo.ParseCount,
		ChangeCount:  mountinfo.ChangeCount,
		Changed:      mountinfo.Changed,
		content:      bytes.NewBuffer(mountinfo.content.Bytes()),
		path:         mountinfo.path,
	}
	for dev, info := range mountinfo.DevMountInfo {
		newMountInfo.DevMountInfo[dev] = make([]string, MOUNTINFO_NUM_FIELDS)
		copy(newMountInfo.DevMountInfo[dev], info)
	}
	return newMountInfo
}

func (mountinfo *Mountinfo) update(fBuf *bytes.Buffer) error {
	devMountInfo := mountinfo.DevMountInfo
	foundDev := map[string]bool{}
	scanner := bufio.NewScanner(fBuf)
	for scanner.Scan() {
		line := scanner.Text()
		words := strings.Fields(line)
		if len(words) <= MOUNTINFO_MAJOR_MINOR {
			return fmt.Errorf("%q: missing fields", line)
		}
		dev := string(words[MOUNTINFO_MAJOR_MINOR])
		info := devMountInfo[dev]
		if info == nil {
			info = make([]string, MOUNTINFO_NUM_FIELDS)
			devMountInfo[dev] = info
		}

		fieldIndex, optionalsStart := 0, -1
		for i, word := range words {
			if fieldIndex >= MOUNTINFO_NUM_FIELDS {
				return fmt.Errorf("%q:  too many fields", line)
			} else if fieldIndex != MOUNTINFO_OPTIONAL_FIELDS {
				if info[fieldIndex] != word {
					info[fieldIndex] = word
				}
				fieldIndex++
			} else {
				if optionalsStart < 0 {
					optionalsStart = i
				}
				if word == MOUNTINFO_OPTIONAL_FIELDS_TERMINATOR {
					optionals := strings.Join(
						words[optionalsStart:i],
						MOUNTINFO_OPTIONAL_FIELDS_JOIN_SEP,
					)
					if info[fieldIndex] != optionals {
						info[fieldIndex] = optionals
					}
					fieldIndex++
					if info[fieldIndex] != word {
						info[fieldIndex] = word
					}
					fieldIndex++
				}
			}
		}
		if fieldIndex < MOUNTINFO_NUM_FIELDS {
			return fmt.Errorf("%q: missing fields", line)
		}
		foundDev[dev] = true
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	// Remove leftover devices (from prev scans):
	for dev := range devMountInfo {
		if !foundDev[dev] {
			delete(devMountInfo, dev)
		}
	}

	// Update cached content:
	mountinfo.content.Reset()
	if _, err := mountinfo.content.ReadFrom(fBuf); err != nil {
		return err
	}

	mountinfo.Changed = true
	mountinfo.ChangeCount++

	return nil
}

func (mountinfo *Mountinfo) Parse() error {
	fBuf, err := mountinfoReadFileBufPool.ReadFile(mountinfo.path)
	if err != nil {
		return err
	}
	defer mountinfoReadFileBufPool.ReturnBuf(fBuf)

	mountinfo.ParseCount++
	if bytes.Equal(mountinfo.content.Bytes(), fBuf.Bytes()) {
		mountinfo.Changed = false
		return nil
	}
	err = mountinfo.update(fBuf)
	if err != nil {
		return fmt.Errorf("%s: %v", mountinfo.path, err)
	}
	return nil
}
