// parser for /proc/pid/mountinfo

package procfs

import (
	"bytes"
	"fmt"
	"path"
	"strconv"
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

type Mountinfo struct {
	// All information will be presented as byte slices, indexed by
	// "major:minor"; the backing for all the slices is `.content'.
	// e.g. MOUNTINFO_MOUNT_POINT for "major:minor"
	//   sliceOffset = .devMountInfo["major:minor"][MOUNTINFO_MOUNT_POINT]
	//   mountPoint = .content.Bytes()[sliceOffset.Start:sliceOffset.End]
	DevMountInfo map[string][]SliceOffsets

	// The file is not expected to change very often, so in order to avoid a
	// rather expensive parsing, its previous content is cached and the parsing
	// occurs only if there are changes.
	Changed                 bool
	ParseCount, ChangeCount uint64

	// Whether to force an update at every parse or not, regardless of content
	// change, in support of testing/benchmarking.
	ForceUpdate bool

	// File content, the backing []byte for the byte slices of the fields:
	content *bytes.Buffer

	// The path file to  read:
	path string
}

// Read the entire file in one go, using a ReadFileBufPool:
var mountinfoReadFileBufPool = ReadFileBufPool256k

func NewMountInfo(procfsRoot string, pid int) *Mountinfo {
	return &Mountinfo{
		DevMountInfo: map[string][]SliceOffsets{},
		content:      &bytes.Buffer{},
		path:         path.Join(procfsRoot, strconv.Itoa(pid), "mountinfo"),
	}
}

func (mountinfo *Mountinfo) Clone(full bool) *Mountinfo {
	newMountInfo := &Mountinfo{
		DevMountInfo: map[string][]SliceOffsets{},
		ParseCount:   mountinfo.ParseCount,
		ChangeCount:  mountinfo.ChangeCount,
		Changed:      mountinfo.Changed,
		path:         mountinfo.path,
	}

	if full {
		newMountInfo.content = bytes.NewBuffer(mountinfo.content.Bytes())
		newMountInfo.ForceUpdate = mountinfo.ForceUpdate
	} else {
		newMountInfo.content = &bytes.Buffer{}
	}

	for majorMinor, info := range mountinfo.DevMountInfo {
		newMountInfo.DevMountInfo[majorMinor] = make([]SliceOffsets, MOUNTINFO_NUM_FIELDS)
		if full {
			copy(newMountInfo.DevMountInfo[majorMinor], info)
		}
	}

	return newMountInfo
}

func (mountinfo *Mountinfo) update() error {
	buf, l := mountinfo.content.Bytes(), mountinfo.content.Len()
	info := make([]SliceOffsets, MOUNTINFO_NUM_FIELDS)

	devMountInfo := mountinfo.DevMountInfo
	for pos, lineNum := 0, 1; pos < l; lineNum++ {
		lineStart, fieldIndex, eol := pos, MOUNTINFO_MOUNT_ID, false
		optionalFieldsStart, optionalFieldsEnd := -1, -1
		for ; !eol && pos < l && fieldIndex < MOUNTINFO_NUM_FIELDS; pos++ {
			// Locate the next word start:
			for ; isWhitespace[buf[pos]]; pos++ {
			}
			wordStart := pos
			// Locate word end:
			for ; pos < l; pos++ {
				c := buf[pos]
				if eol = (c == '\n'); eol || isWhitespace[c] {
					break
				}
			}
			// Assign to parsed field:
			if fieldIndex == MOUNTINFO_OPTIONAL_FIELDS {
				if optionalFieldsStart < 0 {
					// First word of the optional fields:
					optionalFieldsStart = wordStart
					optionalFieldsEnd = wordStart
				}
				if pos == wordStart+1 && buf[wordStart] == '-' {
					// End of optional fields:
					info[fieldIndex].Start = optionalFieldsStart
					info[fieldIndex].End = optionalFieldsEnd
					fieldIndex++
				} else {
					// This word is part of the optional fields, advance the
					// latter's end position:
					optionalFieldsEnd = pos
					continue
				}
			}
			info[fieldIndex].Start = wordStart
			info[fieldIndex].End = pos
			fieldIndex++
		}
		if fieldIndex < MOUNTINFO_NUM_FIELDS {
			// Missing fields:
			return fmt.Errorf(
				"%s#%d: %q: missing fields: want: %d, got: %d",
				mountinfo.path, lineNum, getCurrentLine(buf, lineStart), MOUNTINFO_NUM_FIELDS, fieldIndex,
			)
		}
		// Advance to EOL:
		for ; !eol && pos < l; pos++ {
			c := buf[pos]
			if eol = (c == '\n'); !eol && !isWhitespace[c] {
				return fmt.Errorf(
					"%s#%d: %q: %q: unexpected content after the last field",
					mountinfo.path, lineNum, getCurrentLine(buf, lineStart), getCurrentLine(buf, pos),
				)
			}
		}

		// Update the "major:minor" index:
		startEnd := info[MOUNTINFO_MAJOR_MINOR]
		majorMinor := string(buf[startEnd.Start:startEnd.End])
		devInfo := devMountInfo[majorMinor]
		if devInfo == nil {
			devInfo = make([]SliceOffsets, MOUNTINFO_NUM_FIELDS)
			devMountInfo[majorMinor] = devInfo
		}
		copy(devInfo, info)
	}

	// Update change stats:
	mountinfo.Changed = true
	mountinfo.ChangeCount++

	return nil
}

func (mountinfo *Mountinfo) Parse() error {
	fBuf, err := mountinfoReadFileBufPool.ReadFile(mountinfo.path)
	if err != nil {
		return err
	}
	mountinfo.ParseCount++
	if !mountinfo.ForceUpdate && bytes.Equal(mountinfo.content.Bytes(), fBuf.Bytes()) {
		mountinfo.Changed = false
		mountinfoReadFileBufPool.ReturnBuf(fBuf)
		return nil
	}

	// Swap the buffers to reflect the file change: return the previous content
	// to the pool and keep the most recent buffer:
	mountinfoReadFileBufPool.ReturnBuf(mountinfo.content)
	mountinfo.content = fBuf

	err = mountinfo.update()
	return err
}
