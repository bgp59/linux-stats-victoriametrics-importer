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

// The data gleaned from this file has two use cases:
//  - as-is: the value from the file is the (label) value associated w/ the metric, e.g. priority
//  - for numerical calculations, e.g. utime/stime
// As-is data will be returned such that it can be easily converted to byte
// slices whereas numerical data will be returned as unsigned long.

// Parsed data types:
const (
	PID_STAT_BYTES_DATA = iota
	PID_STAT_ULONG_DATA
)

const (
	// The max size of comm, defined in include/linux/sched.h:
	TASK_COMM_LEN = 16
	// Special TID to indicate that the stats are for PID only:
	PID_STAT_PID_ONLY_TID = 0
)

type PidStat struct {
	// Read the raw content of the file here:
	Buf *bytes.Buffer
	// Start/stop index for each as-is field (byte slice):
	FieldStart, FieldEnd []int
	// Numeric fields:
	NumericFields []uint64
	// The path file to read:
	path string
}

// Field separators, for all fields but (comm):
var pidStatSeparators = [256]byte{' ': 1, '\t': 1, '\n': 1}

// The following enumeration gives the indices for byte slice fields;
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

	// Must by last!
	PID_STAT_BYTE_SLICE_FIELD_COUNT
)

// The field# to use for byte slice fields (as per man proc field# starts from 1):
var pidStatByteSliceFieldNum = [PID_STAT_BYTE_SLICE_FIELD_COUNT]int{
	PID_STAT_COMM:        2,
	PID_STAT_STATE:       3,
	PID_STAT_PPID:        4,
	PID_STAT_PGRP:        5,
	PID_STAT_SESSION:     6,
	PID_STAT_TTY_NR:      7,
	PID_STAT_TPGID:       8,
	PID_STAT_FLAGS:       9,
	PID_STAT_PRIORITY:    18,
	PID_STAT_NICE:        19,
	PID_STAT_NUM_THREADS: 20,
	PID_STAT_STARTTIME:   22,
	PID_STAT_VSIZE:       23,
	PID_STAT_RSS:         24,
	PID_STAT_RSSLIM:      25,
	PID_STAT_PROCESSOR:   39,
	PID_STAT_RT_PRIORITY: 40,
	PID_STAT_POLICY:      41,
}

// The following enumeration gives the indices for unsigned long fields:
const (
	PID_STAT_MINFLT = iota
	PID_STAT_MAJLT
	PID_STAT_UTIME
	PID_STAT_STIME

	// Must by last!
	PID_STAT_ULONG_DATA_COUNT
)

// The field# to use for unsigned long fields (as per man proc field# starts from 1):
var pidStatUlongFieldNum = [PID_STAT_ULONG_DATA_COUNT]int{
	PID_STAT_MINFLT: 10,
	PID_STAT_MAJLT:  12,
	PID_STAT_UTIME:  14,
	PID_STAT_STIME:  15,
}

// Field handling:
type PidStatFieldHandling struct {
	// How to parse the field:
	dataType byte
	// Array index where to store the result:
	index int
}

// The following list maps field# into its handling; it will be built during
// init based on the ...FieldNum above; nil, the default, indicates that the
// field should be ignored:
var pidStatFieldNumHandling []*PidStatFieldHandling
var pidStatMaxFieldNum = 0

func init() {
	for _, fieldNum := range pidStatByteSliceFieldNum {
		if fieldNum > pidStatMaxFieldNum {
			pidStatMaxFieldNum = fieldNum
		}
	}
	for _, fieldNum := range pidStatUlongFieldNum {
		if fieldNum > pidStatMaxFieldNum {
			pidStatMaxFieldNum = fieldNum
		}
	}

	pidStatFieldNumHandling = make([]*PidStatFieldHandling, pidStatMaxFieldNum+1)
	for index, fieldNum := range pidStatByteSliceFieldNum {
		pidStatFieldNumHandling[fieldNum] = &PidStatFieldHandling{PID_STAT_BYTES_DATA, index}
	}
	for index, fieldNum := range pidStatUlongFieldNum {
		pidStatFieldNumHandling[fieldNum] = &PidStatFieldHandling{PID_STAT_ULONG_DATA, index}
	}
}

func NewPidStat(procfsRoot string, pid, tid int) *PidStat {
	pidStat := &PidStat{
		Buf:           &bytes.Buffer{},
		FieldStart:    make([]int, PID_STAT_BYTE_SLICE_FIELD_COUNT),
		FieldEnd:      make([]int, PID_STAT_BYTE_SLICE_FIELD_COUNT),
		NumericFields: make([]uint64, PID_STAT_ULONG_DATA_COUNT),
	}
	if tid == PID_STAT_PID_ONLY_TID {
		pidStat.path = path.Join(procfsRoot, strconv.Itoa(pid), "stat")
	} else {
		pidStat.path = path.Join(procfsRoot, strconv.Itoa(pid), "task", strconv.Itoa(tid), "stat")
	}
	return pidStat
}

func (pidStat *PidStat) Parse() error {
	file, err := os.Open(pidStat.path)
	if err != nil {
		return err
	}
	_, err = pidStat.Buf.ReadFrom(file)
	if err != nil {
		return err
	}
	b := pidStat.Buf.Bytes()
	l := pidStat.Buf.Len()

	// Locate '(' for comm start:
	commStart := bytes.IndexByte(b, '(') + 1
	if commStart <= 0 {
		return fmt.Errorf("%s: cannot locate '('", pidStat.path)
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
			return fmt.Errorf("%s: cannot locate ')'", pidStat.path)
		}
	}

	fieldNum := pidStatByteSliceFieldNum[PID_STAT_COMM]
	pidStat.FieldStart[PID_STAT_COMM] = commStart
	pidStat.FieldEnd[PID_STAT_COMM] = commEnd

	for wasSep, i, fieldStart := byte(1), commEnd+1, commEnd+1; i < l && fieldNum < pidStatMaxFieldNum; i++ {
		isSep := pidStatSeparators[b[i]]
		switch wasSep<<1 + isSep {
		case 0b10:
			fieldStart = i
		case 0b01:
			fieldNum++
			fieldHandling := pidStatFieldNumHandling[fieldNum]
			if fieldHandling != nil {
				index := fieldHandling.index
				switch fieldHandling.dataType {
				case PID_STAT_BYTES_DATA:
					pidStat.FieldStart[index] = fieldStart
					pidStat.FieldEnd[index] = i
				case PID_STAT_ULONG_DATA:
					pidStat.NumericFields[index], err = strconv.ParseUint(string(b[fieldStart:i]), 10, 64)
					if err != nil {
						return fmt.Errorf("%s: field# %d: %v", pidStat.path, fieldNum, err)
					}
				}
			}
		}
		wasSep = isSep
	}
	// Sanity check:
	if fieldNum != pidStatMaxFieldNum {
		return fmt.Errorf("%s: not enough fields: want: %d, got: %d", pidStat.path, pidStatMaxFieldNum, fieldNum)
	}
	return nil
}
