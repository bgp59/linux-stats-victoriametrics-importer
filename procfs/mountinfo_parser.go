// parser for /proc/pid/mountinfo

package procfs

import (
	"bytes"
	"fmt"
	"path"
	"strconv"
)

// Reference:
// https://man7.org/linux/man-pages/man5/proc_pid_mountinfo.5.html

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

type MountinfoParsedLine [MOUNTINFO_NUM_FIELDS][]byte

type Mountinfo struct {
	ParsedLines []*MountinfoParsedLine
	// The file is not expected to change very often, so in order to avoid a
	// rather expensive parsing, its previous content is cached and the parsing
	// occurs only if there are changes.
	Changed bool

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
	pidPart := "self"
	if pid > 0 {
		pidPart = strconv.Itoa(pid)
	}
	return path.Join(procfsRoot, pidPart, "mountinfo")
}

func NewMountinfo(procfsRoot string, pid int) *Mountinfo {
	return &Mountinfo{
		ParsedLines: make([]*MountinfoParsedLine, 0),
		content:     &bytes.Buffer{},
		path:        MountinfoPath(procfsRoot, pid),
	}
}

func (mountinfo *Mountinfo) Clone(full bool) *Mountinfo {
	NewMountinfo := &Mountinfo{
		ParsedLines: make([]*MountinfoParsedLine, 0),
		Changed:     mountinfo.Changed,
		ForceUpdate: mountinfo.ForceUpdate,
		path:        mountinfo.path,
	}
	NewMountinfo.content = &bytes.Buffer{}
	if full {
		NewMountinfo.content.Write(mountinfo.content.Bytes())
	}

	return NewMountinfo
}

func (mountinfo *Mountinfo) update() error {
	buf, l := mountinfo.content.Bytes(), mountinfo.content.Len()

	if mountinfo.ParsedLines == nil {
		mountinfo.ParsedLines = make([]*MountinfoParsedLine, 0)
	} else {
		mountinfo.ParsedLines = mountinfo.ParsedLines[:0]
	}
	for pos, lineNum := 0, 1; pos < l; lineNum++ {
		info := new(MountinfoParsedLine)
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
				"%s:%d: %q: missing fields: want: %d, got: %d",
				mountinfo.path, lineNum, getCurrentLine(buf, lineStart), MOUNTINFO_NUM_FIELDS, fieldIndex,
			)
		}
		// Advance to EOL:
		for ; !eol && pos < l; pos++ {
			c := buf[pos]
			if eol = (c == '\n'); !eol && !isWhitespace[c] {
				return fmt.Errorf(
					"%s:%d: %q: %q: unexpected content after the last field",
					mountinfo.path, lineNum, getCurrentLine(buf, lineStart), getCurrentLine(buf, pos),
				)
			}
		}

		mountinfo.ParsedLines = append(mountinfo.ParsedLines, info)
	}

	return nil
}

func (mountinfo *Mountinfo) Parse() error {
	fBuf, err := mountinfoReadFileBufPool.ReadFile(mountinfo.path)
	if err == nil {
		mountinfo.Changed = mountinfo.ForceUpdate || !bytes.Equal(mountinfo.content.Bytes(), fBuf.Bytes())
		if mountinfo.Changed {
			fBuf, mountinfo.content = mountinfo.content, fBuf
			err = mountinfo.update()
		}
	}
	mountinfoReadFileBufPool.ReturnBuf(fBuf)
	return err
}
