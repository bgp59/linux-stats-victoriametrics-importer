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
	//   mountPoint = .devMountInfo["major:minor"][MOUNTINFO_MOUNT_POINT]
	DevMountInfo map[string][][]byte
	// The file is not expected to change very often, so in order to avoid a
	// rather expensive parsing, its previous content is cached and the parsing
	// occurs only if there are changes.
	Changed                 bool
	ParseCount, ChangeCount uint64

	// Whether to force an update at every parse or not, regardless of content
	// change, in support of testing/benchmarking.
	ForceUpdate bool

	// File content, used to determine changes:
	content *bytes.Buffer

	// The path file to  read:
	path string
}

// Read the entire file in one go, using a ReadFileBufPool:
var mountinfoReadFileBufPool = ReadFileBufPool256k

func MountinfoPath(procfsRoot string, pid int) string {
	return path.Join(procfsRoot, strconv.Itoa(pid), "mountinfo")
}

func NewMountinfo(procfsRoot string, pid int) *Mountinfo {
	return &Mountinfo{
		DevMountInfo: make(map[string][][]byte),
		content:      &bytes.Buffer{},
		path:         MountinfoPath(procfsRoot, pid),
	}
}

func (mountinfo *Mountinfo) Clone(full bool) *Mountinfo {
	NewMountinfo := &Mountinfo{
		DevMountInfo: make(map[string][][]byte),
		ParseCount:   mountinfo.ParseCount,
		ChangeCount:  mountinfo.ChangeCount,
		Changed:      mountinfo.Changed,
		path:         mountinfo.path,
	}

	if full {
		NewMountinfo.content = bytes.NewBuffer(mountinfo.content.Bytes())
		NewMountinfo.ForceUpdate = mountinfo.ForceUpdate
		NewMountinfo.update()
	} else {
		NewMountinfo.content = &bytes.Buffer{}
	}

	return NewMountinfo
}

func (mountinfo *Mountinfo) update() error {
	buf, l := mountinfo.content.Bytes(), mountinfo.content.Len()
	info := make([][]byte, MOUNTINFO_NUM_FIELDS)

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
					info[fieldIndex] = buf[optionalFieldsStart:optionalFieldsEnd]
					fieldIndex++
				} else {
					// This word is part of the optional fields, advance the
					// latter's end position:
					optionalFieldsEnd = pos
					continue
				}
			}
			info[fieldIndex] = buf[wordStart:pos]
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
		majorMinor := string(info[MOUNTINFO_MAJOR_MINOR])
		devInfo := devMountInfo[majorMinor]
		if devInfo == nil {
			devInfo = make([][]byte, MOUNTINFO_NUM_FIELDS)
			devMountInfo[majorMinor] = devInfo
		}
		copy(devInfo, info)
	}

	return nil
}

func (mountinfo *Mountinfo) Parse() error {
	fBuf, err := mountinfoReadFileBufPool.ReadFile(mountinfo.path)
	defer mountinfoReadFileBufPool.ReturnBuf(fBuf)
	if err == nil {
		mountinfo.ParseCount++
		mountinfo.Changed = mountinfo.ForceUpdate || !bytes.Equal(mountinfo.content.Bytes(), fBuf.Bytes())
		if mountinfo.Changed {
			mountinfo.ChangeCount++
			mountinfo.content = bytes.NewBuffer(fBuf.Bytes())
			err = mountinfo.update()
		}
	}
	return err
}
