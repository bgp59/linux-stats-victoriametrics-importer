// parser for /proc/pid/stat and /proc/pid/task/tid/stat

// "486 (rs:main Q:Reg) S 1 468 468 0 -1 1077936192 44 0 0 0 0 2 0 0 20 0 4 0 898 227737600 1340 18446744073709551615 94649719967744 94649720406605 140724805212720 0 0 0 2146171647 16781830 3227649 1 0 0 -1 0 0 0 0 0 0 94649720624720 94649720664912 94649728393216 140724805218000 140724805218029 140724805218029 140724805218277 0\n"

package procfs

import (
	"bytes"
	"fmt"
)

const (
	TASK_MAX_LEN                           = 16
	PID_STAT_MAX_SCANNED_FIELDS_AFTER_COMM = 41 - 3 + 1 // `(3) state' thru `(41) policy'
)

// Field separators, for all fields but (comm):
var pidStatSeparators = [256]byte{' ': 1, '\t': 1, '\n': 1}

// The parser will scan PID_STAT_MAX_SCANNED_FIELDS_AFTER_COMM, but not all fields will
// be used, ignore those marked below:
var pidStatIgnoreFields = [PID_STAT_MAX_SCANNED_FIELDS_AFTER_COMM]bool{
	// (3) state  %c
	// (4) ppid  %d
	// (5) pgrp  %d
	// (6) session  %d
	// (7) tty_nr  %d
	// (8) tpgid  %d
	// (9) flags  %u
	// (10) minflt  %lu
	8: true, // (11) cminflt  %lu
	// (12) majflt  %lu
	10: true, // (13) cmajflt  %lu
	// (14) utime  %lu
	// (15) stime  %lu
	13: true, // (16) cutime  %ld
	14: true, // (17) cstime  %ld
	// (18) priority  %ld
	// (19) nice  %ld
	// (20) num_threads  %ld
	18: true, // (21) itrealvalue  %ld
	// (22) starttime  %llu
	// (23) vsize  %lu
	// (24) rss  %ld
	// (25) rsslim  %lu
	23: true, // (26) startcode  %lu  [PT]
	24: true, // (27) endcode  %lu  [PT]
	25: true, // (28) startstack  %lu  [PT]
	26: true, // (29) kstkesp  %lu  [PT]
	27: true, // (30) kstkeip  %lu  [PT]
	28: true, // (31) signal  %lu
	29: true, // (32) blocked  %lu
	30: true, // (33) sigignore  %lu
	31: true, // (34) sigcatch  %lu
	32: true, // (35) wchan  %lu  [PT]
	33: true, // (36) nswap  %lu
	34: true, // (37) cnswap  %lu
	35: true, // (38) exit_signal  %d  (since Linux 2.1.22)
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
	PID_STAT_MINFLT
	PID_STAT_MAJLT
	PID_STAT_UTIME
	PID_STAT_STIME
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

	PID_STAT_NUM_FIELDS // Must be last!
)

type PidStatByteFields struct {
	// Read the raw content of the file here:
	Buf *bytes.Buffer
	// Start/stop for each field:
	FieldStart, FieldEnd [PID_STAT_NUM_FIELDS]int
}

func (bf *PidStatByteFields) Parse() error {
	b := bf.Buf.Bytes()
	l := bf.Buf.Len()

	// Locate '(' for comm start:
	commStart := bytes.IndexByte(b, '(') + 1
	if commStart <= 0 {
		return fmt.Errorf("cannot locate '('")
	}
	// Locate ')' for comm end, it should be at most TASK_MAX_LEN after
	// commStart:
	commEnd := commStart + TASK_MAX_LEN
	if commEnd >= l {
		commEnd = l - 1
	}
	for ; commEnd >= commStart && b[commEnd] != ')'; commEnd-- {
	}
	if commEnd < commStart {
		// This shouldn't happen but maybe the kernel was compiled w/ a bigger TASK_MAX_LEN?
		// Try again against the whole buffer:
		commEnd = bytes.LastIndexByte(b, ')')
		if commEnd < 0 {
			return fmt.Errorf("cannot locate ')'")
		}
	}

	fieldNum := 0

	bf.FieldStart[fieldNum] = commStart
	bf.FieldEnd[fieldNum] = commEnd
	fieldNum++

	for wasSep, scanComplete, i, scannedFieldNum := byte(1), false, commEnd+1, 0; !scanComplete && i < l; i++ {
		isSep := pidStatSeparators[b[i]]
		switch wasSep<<1 + isSep {
		case 0b10:
			if !pidStatIgnoreFields[scannedFieldNum] {
				bf.FieldStart[fieldNum] = i
			}
		case 0b01:
			if !pidStatIgnoreFields[scannedFieldNum] {
				bf.FieldEnd[fieldNum] = i
				fieldNum++
			}
			scannedFieldNum++
			scanComplete = scannedFieldNum == PID_STAT_MAX_SCANNED_FIELDS_AFTER_COMM || fieldNum == PID_STAT_NUM_FIELDS
		}
		wasSep = isSep
	}

	// Sanity check:
	if fieldNum != PID_STAT_NUM_FIELDS {
		return fmt.Errorf(
			"scan incomplete: field#: want: %d, got: %d", PID_STAT_NUM_FIELDS, fieldNum,
		)
	}

	return nil
}
