// parser for /proc/pid/cmdline and /proc/pid/task/tid/cmdline

package procfs

import (
	"bytes"
	"path"
	"strconv"

	"github.com/eparparita/linux-stats-victoriametrics-importer/internal/utils"
)

// The parsed cmdline will be used as a Prometheus label value, which is a
// unicode string with certain characters being escaped; see `label_value' at
// https://github.com/prometheus/docs/blob/main/content/docs/instrumenting/exposition_formats.md
var cmdlineByteConvert = [256][]byte{
	0:    []byte(` `),
	'\n': []byte(`\n`),
	'\\': []byte(`\\`),
	'"':  []byte(`\"`),
}

type PidCmdline struct {
	// The buffer used to store the sanitized command line, if nil them it must
	// be allocated from the pool:
	Cmdline *bytes.Buffer
	// Proc file system root, needed if pid, tid are changed at parse time:
	procfsRoot string
	// Path to the file:
	path string
}

// The pool used for reading and sanitizing the command:
var pidCmdlineReadFileBufPool = ReadFileBufPool64k

func PidCmdlinePath(procfsRoot string, pid, tid int) string {
	if tid == PID_STAT_PID_ONLY_TID {
		return path.Join(procfsRoot, strconv.Itoa(pid), "cmdline")
	} else {
		return path.Join(procfsRoot, strconv.Itoa(pid), "task", strconv.Itoa(tid), "cmdline")
	}
}
func NewPidCmdline(procfsRoot string, pid, tid int) *PidCmdline {
	return &PidCmdline{
		path: PidCmdlinePath(procfsRoot, pid, tid),
	}
}

func (pidCmdline *PidCmdline) ReturnBuf() {
	if pidCmdline.Cmdline != nil {
		pidCmdlineReadFileBufPool.ReturnBuf(pidCmdline.Cmdline)
		pidCmdline.Cmdline = nil
	}
}

func (pidCmdline *PidCmdline) Parse(pid, tid int) error {
	if pid > 0 {
		if tid == PID_STAT_PID_ONLY_TID {
			pidCmdline.path = path.Join(pidCmdline.procfsRoot, strconv.Itoa(pid), "cmdline")
		} else {
			pidCmdline.path = path.Join(pidCmdline.procfsRoot, strconv.Itoa(pid), "task", strconv.Itoa(tid), "cmdline")
		}
	}
	fBuf, err := pidCmdlineReadFileBufPool.ReadFile(pidCmdline.path)
	defer pidCmdlineReadFileBufPool.ReturnBuf(fBuf)
	truncated := (err == utils.ErrReadFileBufPotentialTruncation)
	if err != nil && !truncated {
		return err
	}

	buf, l := fBuf.Bytes(), fBuf.Len()

	// If truncation occurred then the last 3 UTF-8 characters will be replaced
	// w/ `...':
	if truncated {
		// Locate the start of the rightmost UTF-8 char, at least 3 places away
		// from the end. The search will start at l-3 and it will end at l-6.
		// Note that any intermediate UTF-8 byte (bytes# 1.. 4) is 10xxxxxx.
		pos := l - 3
		for ; pos > 0 && pos > l-6 && buf[pos]&0b11000000 == 0b10000000; pos-- {
		}
		l = pos + 3
		for ; pos < l; pos++ {
			buf[pos] = '.'
		}
	}

	// The args in cmdline are '\0' separated; '\0' will be converted excepted
	// for the last one(s) which will be discarded.
	for ; l > 0 && buf[l-1] == 0; l-- {
	}

	// Build the parsed command line out of the read one, escaping single byte
	// chars as needed:
	cmdline := pidCmdline.Cmdline
	if cmdline == nil {
		cmdline = pidCmdlineReadFileBufPool.GetBuf()
		pidCmdline.Cmdline = cmdline
	} else {
		cmdline.Reset()
	}

	for pos := 0; pos < l; {
		startStretch, byteConvert := pos, []byte(nil)
		// Locate the next single byte character that needs escaping:
		for ; pos < l; pos++ {
			if byteConvert = cmdlineByteConvert[buf[pos]]; byteConvert != nil {
				break
			}
		}
		// Copy everything up to it as is:
		_, err = cmdline.Write(buf[startStretch:pos])
		// Copy the conversion:
		if err == nil && byteConvert != nil {
			_, err = cmdline.Write(byteConvert)
			pos++
		}
		if err != nil {
			pidCmdline.ReturnBuf()
			return err
		}
	}

	return nil
}
