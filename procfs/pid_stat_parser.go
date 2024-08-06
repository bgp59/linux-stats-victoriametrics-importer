// parser for /proc/pid/stat and /proc/pid/task/tid/stat

package procfs

import (
	"bytes"
	"fmt"
	"os"
	"path"
)

// "486 (rs:main Q:Reg) S 1 468 468 0 -1 1077936192 44 0 0 0 0 2 0 0 20 0 4 0 898 227737600 1340 18446744073709551615 94649719967744 94649720406605 140724805212720 0 0 0 2146171647 16781830 3227649 1 0 0 -1 0 0 0 0 0 0 94649720624720 94649720664912 94649728393216 140724805218000 140724805218029 140724805218029 140724805218277 0\n"

const (
	// The max size of comm, defined in include/linux/sched.h:
	TASK_COMM_LEN = 16
)

// The data gleaned from this file is of two types, depending on its use case:
// - byte slice: used as-is, the value from the file is the (label) value
//   associated w/ the metric, e.g. priority
// - numerical: used for calculations, e.g. utime/stime

type PidStatParser interface {
	Parse(pidTidPath string) error
	GetByteSliceFields() [][]byte
	GetNumericFields() []uint64
}

type NewPidStatParser func(procfsRoot string, pid, tid int) PidStatParser

// Parsed data types:
const (
	PID_STAT_BYTES_DATA = iota
	PID_STAT_ULONG_DATA
)

type PidStat struct {
	// As-is fields:
	byteSliceFields [][]byte
	// Numeric fields:
	numericFields []uint64
	// Buffer to read the content of the file, also backing storage for the byteSliceFields:
	fBuf *bytes.Buffer
}

// The indices for byte slice fields;
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

	// Must be last!
	PID_STAT_BYTE_SLICE_NUM_FIELDS
)

// The indices for unsigned long fields:
const (
	PID_STAT_MINFLT = iota
	PID_STAT_MAJFLT
	PID_STAT_UTIME
	PID_STAT_STIME

	// Must be last!
	PID_STAT_ULONG_FIELD_NUM_FIELDS
)

// Field handling:
type PidStatFieldHandling struct {
	// How to parse the field:
	dataType byte
	// Array index where to store the result:
	index int
}

// Map field# into its handling; nil, the default, indicates that the field
// should be ignored:
const (
	PID_STAT_MAX_FIELD_NUM = 41
)

var pidStatFieldHandling = [PID_STAT_MAX_FIELD_NUM + 1]*PidStatFieldHandling{
	// (1) pid  %d
	// (2) comm  %s
	2: {PID_STAT_BYTES_DATA, PID_STAT_COMM},
	// (3) state  %c
	3: {PID_STAT_BYTES_DATA, PID_STAT_STATE},
	// (4) ppid  %d
	4: {PID_STAT_BYTES_DATA, PID_STAT_PPID},
	// (5) pgrp  %d
	5: {PID_STAT_BYTES_DATA, PID_STAT_PGRP},
	// (6) session  %d
	6: {PID_STAT_BYTES_DATA, PID_STAT_SESSION},
	// (7) tty_nr  %d
	7: {PID_STAT_BYTES_DATA, PID_STAT_TTY_NR},
	// (8) tpgid  %d
	8: {PID_STAT_BYTES_DATA, PID_STAT_TPGID},
	// (9) flags  %u
	9: {PID_STAT_BYTES_DATA, PID_STAT_FLAGS},
	// (10) minflt  %lu
	10: {PID_STAT_ULONG_DATA, PID_STAT_MINFLT},
	// (11) cminflt  %lu
	// (12) majflt  %lu
	12: {PID_STAT_ULONG_DATA, PID_STAT_MAJFLT},
	// (13) cmajflt  %lu
	// (14) utime  %lu
	14: {PID_STAT_ULONG_DATA, PID_STAT_UTIME},
	// (15) stime  %lu
	15: {PID_STAT_ULONG_DATA, PID_STAT_STIME},
	// (16) cutime  %ld
	// (17) cstime  %ld
	// (18) priority  %ld
	18: {PID_STAT_BYTES_DATA, PID_STAT_PRIORITY},
	// (19) nice  %ld
	19: {PID_STAT_BYTES_DATA, PID_STAT_NICE},
	// (20) num_threads  %ld
	20: {PID_STAT_BYTES_DATA, PID_STAT_NUM_THREADS},
	// (21) itrealvalue  %ld
	// (22) starttime  %llu
	22: {PID_STAT_BYTES_DATA, PID_STAT_STARTTIME},
	// (23) vsize  %lu
	23: {PID_STAT_BYTES_DATA, PID_STAT_VSIZE},
	// (24) rss  %ld
	24: {PID_STAT_BYTES_DATA, PID_STAT_RSS},
	// (25) rsslim  %lu
	25: {PID_STAT_BYTES_DATA, PID_STAT_RSSLIM},
	// (26) startcode  %lu  [PT]
	// (27) endcode  %lu  [PT]
	// (28) startstack  %lu  [PT]
	// (29) kstkesp  %lu  [PT]
	// (30) kstkeip  %lu  [PT]
	// (31) signal  %lu
	// (32) blocked  %lu
	// (33) sigignore  %lu
	// (34) sigcatch  %lu
	// (35) wchan  %lu  [PT]
	// (36) nswap  %lu
	// (37) cnswap  %lu
	// (38) exit_signal  %d  (since Linux 2.1.22)
	// (39) processor  %d  (since Linux 2.2.8)
	39: {PID_STAT_BYTES_DATA, PID_STAT_PROCESSOR},
	// (40) rt_priority  %u  (since Linux 2.5.19)
	40: {PID_STAT_BYTES_DATA, PID_STAT_RT_PRIORITY},
	// (41) policy  %u  (since Linux 2.5.19) == PID_STAT_MAX_FIELD_NUM
	41: {PID_STAT_BYTES_DATA, PID_STAT_POLICY},
	// (42) delayacct_blkio_ticks  %llu  (since Linux 2.6.18)
	// (43) guest_time  %lu  (since Linux 2.6.24)
	// (44) cguest_time  %ld  (since Linux 2.6.24)
	// (45) start_data  %lu  (since Linux 3.3)  [PT]
	// (46) end_data  %lu  (since Linux 3.3)  [PT]
	// (47) start_brk  %lu  (since Linux 3.3)  [PT]
	// (48) arg_start  %lu  (since Linux 3.5)  [PT]
	// (49) arg_end  %lu  (since Linux 3.5)  [PT]
	// (50) env_start  %lu  (since Linux 3.5)  [PT]
	// (51) env_end  %lu  (since Linux 3.5)  [PT]
	// (52) exit_code  %d  (since Linux 3.5)  [PT]
}

func NewPidStat() PidStatParser {
	return &PidStat{
		byteSliceFields: make([][]byte, PID_STAT_BYTE_SLICE_NUM_FIELDS),
		numericFields:   make([]uint64, PID_STAT_ULONG_FIELD_NUM_FIELDS),
		fBuf:            &bytes.Buffer{},
	}
}

// Parse file and update the fields.
func (pidStat *PidStat) Parse(pidTidPath string) error {
	pidStatPath := path.Join(pidTidPath, "stat")
	file, err := os.Open(pidStatPath)
	if err != nil {
		return err
	}
	defer file.Close()
	pidStat.fBuf.Reset()
	_, err = pidStat.fBuf.ReadFrom(file)
	if err != nil {
		return err
	}

	buf, l := pidStat.fBuf.Bytes(), pidStat.fBuf.Len()

	// Locate '(' for comm start:
	commStart := -1
	for pos := 0; pos < l; pos++ {
		if buf[pos] == '(' {
			pos++
			commStart = pos
			break
		}
	}
	if commStart < 0 {
		return fmt.Errorf("%s: cannot locate '('", pidStatPath)
	}
	// Locate ')' for comm end, it should be at most TASK_COMM_LEN after
	// commStart:
	commEnd := commStart + TASK_COMM_LEN
	if commEnd >= l {
		commEnd = l - 1
	}
	for ; commEnd >= commStart && buf[commEnd] != ')'; commEnd-- {
	}
	if commEnd < commStart {
		// This shouldn't happen but maybe the kernel was compiled w/ a bigger
		// TASK_COMM_LEN? Try again against the whole buffer:
		commEnd = l - 1
		for ; commEnd >= commStart && buf[commEnd] != ')'; commEnd-- {
		}
		if commEnd < commStart {
			return fmt.Errorf("%s: cannot locate ')'", pidStatPath)
		}
	}

	fieldNum := 2
	pidStat.byteSliceFields[PID_STAT_COMM] = buf[commStart:commEnd]

	for pos := commEnd + 1; pos < l && fieldNum < PID_STAT_MAX_FIELD_NUM; pos++ {
		for ; pos < l && isWhitespaceNl[buf[pos]]; pos++ {
		}
		fieldStart := pos
		for ; pos < l && !isWhitespaceNl[buf[pos]]; pos++ {
		}
		// fieldEnd := pos

		fieldNum++
		fieldHandling := pidStatFieldHandling[fieldNum]
		if fieldHandling == nil {
			continue
		}

		index := fieldHandling.index
		switch fieldHandling.dataType {
		case PID_STAT_BYTES_DATA:
			pidStat.byteSliceFields[index] = buf[fieldStart:pos]
		case PID_STAT_ULONG_DATA:
			val := uint64(0)
			for i := fieldStart; i < pos; i++ {
				if digit := buf[i] - '0'; digit < 10 {
					val = (val << 3) + (val << 1) + uint64(digit)
				} else {
					return fmt.Errorf(
						"%s: field# %d: %q: invalid numerical value",
						pidStatPath, fieldNum, string(buf[fieldStart:pos]),
					)
				}
			}
			pidStat.numericFields[index] = val
		}
	}
	// Sanity check:
	if fieldNum != PID_STAT_MAX_FIELD_NUM {
		return fmt.Errorf(
			"%s: not enough fields: want: %d, got: %d",
			pidStatPath, PID_STAT_MAX_FIELD_NUM, fieldNum,
		)
	}
	return nil
}

func (pidStat *PidStat) GetByteSliceFields() [][]byte {
	return pidStat.byteSliceFields
}

func (pidStat *PidStat) GetNumericFields() []uint64 {
	return pidStat.numericFields
}
