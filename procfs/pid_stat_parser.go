// parser for /proc/pid/stat and /proc/pid/task/tid/stat

// "486 (rs:main Q:Reg) S 1 468 468 0 -1 1077936192 44 0 0 0 0 2 0 0 20 0 4 0 898 227737600 1340 18446744073709551615 94649719967744 94649720406605 140724805212720 0 0 0 2146171647 16781830 3227649 1 0 0 -1 0 0 0 0 0 0 94649720624720 94649720664912 94649728393216 140724805218000 140724805218029 140724805218029 140724805218277 0\n"

package procfs

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"strconv"
)

const (
	// The max size of comm, defined in include/linux/sched.h:
	TASK_COMM_LEN = 16
	// The last field# scanned (note: man proc numbers fields from 1):
	PID_STAT_LAST_SCANNED_FIELD_NUM = 41 // `(41) policy'
	// Special handling for the fields 1..PID_STAT_LAST_SCANNED_FIELD_NUM: some
	// should be ignored while others should be parsed as numbers. The constants
	// below should be different than 0, which is the default action:
	PID_STAT_IGNORE_FIELD  = 1
	PID_STAT_NUMERIC_FIELD = 2
	// Special TID to indicate that the stats are for PID only:
	PID_STAT_PID_ONLY_TID = 0
)

// Field separators, for all fields but (comm):
var pidStatSeparators = [256]byte{' ': 1, '\t': 1, '\n': 1}

// Field handling special instructions. The default action is to mark their
// start:end boundaries and that's encoded by the default value, 0.
var pidStatFieldHandling = [PID_STAT_LAST_SCANNED_FIELD_NUM + 1]byte{
	// (3) state  %c
	// (4) ppid  %d
	// (5) pgrp  %d
	// (6) session  %d
	// (7) tty_nr  %d
	// (8) tpgid  %d
	// (9) flags  %u
	10: PID_STAT_NUMERIC_FIELD, // (10) minflt  %lu
	11: PID_STAT_IGNORE_FIELD,  // (11) cminflt  %lu
	12: PID_STAT_NUMERIC_FIELD, // (12) majflt  %lu
	13: PID_STAT_IGNORE_FIELD,  // (13) cmajflt  %lu
	14: PID_STAT_NUMERIC_FIELD, // (14) utime  %lu
	15: PID_STAT_NUMERIC_FIELD, // (15) stime  %lu
	16: PID_STAT_IGNORE_FIELD,  // (16) cutime  %ld
	17: PID_STAT_IGNORE_FIELD,  // (17) cstime  %ld
	// (18) priority  %ld
	// (19) nice  %ld
	// (20) num_threads  %ld
	21: PID_STAT_IGNORE_FIELD, // (21) itrealvalue  %ld
	// (22) starttime  %llu
	// (23) vsize  %lu
	// (24) rss  %ld
	// (25) rsslim  %lu
	26: PID_STAT_IGNORE_FIELD, // (26) startcode  %lu  [PT]
	27: PID_STAT_IGNORE_FIELD, // (27) endcode  %lu  [PT]
	28: PID_STAT_IGNORE_FIELD, // (28) startstack  %lu  [PT]
	29: PID_STAT_IGNORE_FIELD, // (29) kstkesp  %lu  [PT]
	30: PID_STAT_IGNORE_FIELD, // (30) kstkeip  %lu  [PT]
	31: PID_STAT_IGNORE_FIELD, // (31) signal  %lu
	32: PID_STAT_IGNORE_FIELD, // (32) blocked  %lu
	33: PID_STAT_IGNORE_FIELD, // (33) sigignore  %lu
	34: PID_STAT_IGNORE_FIELD, // (34) sigcatch  %lu
	35: PID_STAT_IGNORE_FIELD, // (35) wchan  %lu  [PT]
	36: PID_STAT_IGNORE_FIELD, // (36) nswap  %lu
	37: PID_STAT_IGNORE_FIELD, // (37) cnswap  %lu
	38: PID_STAT_IGNORE_FIELD, // (38) exit_signal  %d  (since Linux 2.1.22)
	// (39) processor  %d  (since Linux 2.2.8)
	// (40) rt_priority  %u  (since Linux 2.5.19)
	// (41) policy  %u  (since Linux 2.5.19)
}

// The following enumeration gives the indices for fields kept after the scan:
const (
	PID_STAT_COMM = iota
	PID_STAT_STATE
	PID_STAT_PPID
	PID_STAT_PGRP
	PID_STAT_SESSION
	PID_STAT_TTY_NR
	PID_STAT_TPGID
	PID_STAT_FLAGS
	PID_STAT_PRIORITY
	PID_STAT_NICE
	PID_STAT_NUM_THREADS
	PID_STAT_STARTTIME
	PID_STAT_VSIZE
	PID_STAT_RSS
	PID_STAT_RSSLIM
	PID_STAT_PROCESSOR
	PID_STAT_RT_PRIORITY
	PID_STAT_POLICY

	PID_STAT_BYTE_SLICE_FIELD_COUNT // Must be last!
)

const (
	PID_STAT_MINFLT = iota
	PID_STAT_MAJLT
	PID_STAT_UTIME
	PID_STAT_STIME

	PID_STAT_NUMERIC_FIELD_COUNT // Must be last!
)

type PidStatByteFields struct {
	// Read the raw content of the file here:
	Buf *bytes.Buffer
	// Start/stop for each byte slice field:
	FieldStart, FieldEnd [PID_STAT_BYTE_SLICE_FIELD_COUNT]int
	// Numeric fields:
	NumericFields [PID_STAT_NUMERIC_FIELD_COUNT]uint64
	// The path file to read; if empty then the raw content is assumed loaded
	// already:
	filePath string
}

func (bf *PidStatByteFields) SetPath(procfsRoot string, pid, tid int) {
	if tid == PID_STAT_PID_ONLY_TID {
		bf.filePath = path.Join(procfsRoot, strconv.Itoa(pid), "stat")
	} else {
		bf.filePath = path.Join(procfsRoot, strconv.Itoa(pid), "task", strconv.Itoa(tid), "stat")
	}
}

func (bf *PidStatByteFields) ClearPath() {
	bf.filePath = ""
}

// Read file if .filePath is set and parse content.
func (bf *PidStatByteFields) Parse() error {
	var err error

	if bf.filePath != "" {
		file, err := os.Open(bf.filePath)
		if err != nil {
			return err
		}
		_, err = bf.Buf.ReadFrom(file)
		if err != nil {
			return err
		}
	}
	b := bf.Buf.Bytes()
	l := bf.Buf.Len()

	// Locate '(' for comm start:
	commStart := bytes.IndexByte(b, '(') + 1
	if commStart <= 0 {
		return fmt.Errorf("%s: cannot locate '('", bf.filePath)
	}
	// Locate ')' for comm end, it should be at most TASK_COMM_LEN after
	// commStart:
	commEnd := commStart + TASK_COMM_LEN
	if commEnd >= l {
		commEnd = l - 1
	}
	for ; commEnd >= commStart && b[commEnd] != ')'; commEnd-- {
	}
	if commEnd < commStart {
		// This shouldn't happen but maybe the kernel was compiled w/ a bigger
		// TASK_COMM_LEN? Try again against the whole buffer:
		commEnd = bytes.LastIndexByte(b, ')')
		if commEnd < 0 {

			return fmt.Errorf("%s: cannot locate ')'", bf.filePath)
		}
	}

	byteSliceFieldNum := 0

	bf.FieldStart[byteSliceFieldNum] = commStart
	bf.FieldEnd[byteSliceFieldNum] = commEnd
	byteSliceFieldNum++

	numericFieldNum := 0
	fieldStart := 0
	for wasSep, scanComplete, i, scannedFieldNum := byte(1), false, commEnd+1, 2; !scanComplete && i < l; i++ {
		isSep := pidStatSeparators[b[i]]
		switch wasSep<<1 + isSep {
		case 0b10:
			scannedFieldNum++
			fieldStart = i
		case 0b01:
			switch pidStatFieldHandling[scannedFieldNum] {
			case PID_STAT_IGNORE_FIELD:
				break
			case PID_STAT_NUMERIC_FIELD:
				bf.NumericFields[numericFieldNum], err = strconv.ParseUint(string(b[fieldStart:i]), 10, 64)
				if err != nil {
					return fmt.Errorf("%s: field# %d: %v", bf.filePath, scannedFieldNum, err)
				}
				numericFieldNum++
			default:
				bf.FieldStart[byteSliceFieldNum] = fieldStart
				bf.FieldEnd[byteSliceFieldNum] = i
				byteSliceFieldNum++
			}
			scanComplete = scannedFieldNum == PID_STAT_LAST_SCANNED_FIELD_NUM
		}
		wasSep = isSep
	}

	// Sanity check:
	if byteSliceFieldNum != PID_STAT_BYTE_SLICE_FIELD_COUNT || numericFieldNum != PID_STAT_NUMERIC_FIELD_COUNT {
		return fmt.Errorf(
			"%s: scan incomplete: byte slice got/want field#: %d/%d, numeric got/want field#: %d/%d",
			bf.filePath,
			byteSliceFieldNum, PID_STAT_BYTE_SLICE_FIELD_COUNT,
			numericFieldNum, PID_STAT_NUMERIC_FIELD_COUNT,
		)
	}

	return nil
}
