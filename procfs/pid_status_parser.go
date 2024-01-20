// parser for /proc/pid/status and /proc/pid/task/tid/status

package procfs

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
)

// Name:	rs:main Q:Reg
// Umask:	0022
// State:	S (sleeping)
// Tgid:	468
// Ngid:	0
// Pid:	486
// PPid:	1
// TracerPid:	0
// Uid:	104	104	104	104
// Gid:	111	111	111	111
// FDSize:	128
// Groups:	4 111
// NStgid:	468
// NSpid:	486
// NSpgid:	468
// NSsid:	468
// VmPeak:	  222400 kB
// VmSize:	  222400 kB
// ...
// voluntary_ctxt_switches:	2588
// nonvoluntary_ctxt_switches:	12

// The data gleaned from this file has two use cases:
//  - as-is: the value from the file is the value associated w/ the metric, e.g. Vm... stats
//  - for numerical calculations, e.g. voluntary_ctxt_switches
// As-is data will be returned such that it can be easily converted to byte
// slices whereas numerical data will be returned as unsigned long.
// As-is data comes in 3 flavors:
//   - single value              // Umask:	0022
//   - single value + ByteSliceFieldUnit       // VmPeak:	  222400 kB
//   - list                      // Uid:	104	104	104	104

// Parsed data types:
const (
	PID_STATUS_SINGLE_VAL_DATA = iota
	PID_STATUS_SINGLE_VAL_UNIT_DATA
	PID_STATUS_LIST_DATA
	PID_STATUS_ULONG_DATA
)

// The parsed data will be stored into 2 array sets: one for byte slices, the
// other for numerical, using the following indexes:

// Indexes for byte slices data:
const (
	PID_STATUS_UID = iota
	PID_STATUS_GID
	PID_STATUS_GROUPS
	PID_STATUS_VM_PEAK
	PID_STATUS_VM_SIZE
	PID_STATUS_VM_LCK
	PID_STATUS_VM_PIN
	PID_STATUS_VM_HWM
	PID_STATUS_VM_RSS
	PID_STATUS_RSS_ANON
	PID_STATUS_RSS_FILE
	PID_STATUS_RSS_SHMEM
	PID_STATUS_VM_DATA
	PID_STATUS_VM_STK
	PID_STATUS_VM_EXE
	PID_STATUS_VM_LIB
	PID_STATUS_VM_PTE
	PID_STATUS_VM_PMD
	PID_STATUS_VM_SWAP
	PID_STATUS_HUGETLBPAGES
	PID_STATUS_CPUS_ALLOWED_LIST
	PID_STATUS_MEMS_ALLOWED_LIST
	// Must be last:
	PID_STATUS_BYTE_SLICE_NUM_FIELDS
)

// Indexes for numerical data:
const (
	PID_STATUS_VOLUNTARY_CTXT_SWITCHES = iota
	PID_STATUS_NONVOLUNTARY_CTXT_SWITCHES
	// Must be last:
	PID_STATUS_ULONG_NUM_FIELDS
)

const (
	PID_STATUS_LIST_DATA_SEP = ','
)

type PidStatus struct {
	// Backing buffer for byte slice fields:
	ByteSliceFieldsBuf *bytes.Buffer
	// Start/end offsets for byte slice fields;
	//  field# i = ByteSliceFieldsBuf[ByteSliceFieldOffsets[i].Start:ByteSliceFieldOffsets[i].End]
	ByteSliceFieldOffsets []SliceOffsets
	// ByteSliceFieldUnit, it will be replicated from pidStatusParserInfo:
	ByteSliceFieldUnit [][]byte
	// Unsigned log data:
	NumericFields []uint64
	// Path to the file:
	path string
	// Parsing info replicated as a reference; this is to avoid locking, once
	// the reference was pulled:
	lineHandling []*PidStatusLineHandling
}

type PidStatusLineHandling struct {
	// How to parse the line:
	dataType byte
	// Array index where to store the result; the actual array depends upon the
	// data type:
	index int
	// The file layout will be mapped once and it is assumed to be immutable
	// till next reboot. Store the prefix length detected at map time to make it
	// easier to extract the value, the latter will start in the line *after*
	// that length.
	prefixLen int
}

// Only the lines w/ the prefix in the map below will be processed. The map will
// be converted into an array, indexed by line# (starting from 0), at the first
// parse invocation (JIT).
var pidStatusLineHandlingMap = map[string]*PidStatusLineHandling{
	"Uid":                        {dataType: PID_STATUS_LIST_DATA, index: PID_STATUS_UID},
	"Gid":                        {dataType: PID_STATUS_LIST_DATA, index: PID_STATUS_GID},
	"Groups":                     {dataType: PID_STATUS_LIST_DATA, index: PID_STATUS_GROUPS},
	"VmPeak":                     {dataType: PID_STATUS_SINGLE_VAL_UNIT_DATA, index: PID_STATUS_VM_PEAK},
	"VmSize":                     {dataType: PID_STATUS_SINGLE_VAL_UNIT_DATA, index: PID_STATUS_VM_SIZE},
	"VmLck":                      {dataType: PID_STATUS_SINGLE_VAL_UNIT_DATA, index: PID_STATUS_VM_LCK},
	"VmPin":                      {dataType: PID_STATUS_SINGLE_VAL_UNIT_DATA, index: PID_STATUS_VM_PIN},
	"VmHWM":                      {dataType: PID_STATUS_SINGLE_VAL_UNIT_DATA, index: PID_STATUS_VM_HWM},
	"VmRSS":                      {dataType: PID_STATUS_SINGLE_VAL_UNIT_DATA, index: PID_STATUS_VM_RSS},
	"RssAnon":                    {dataType: PID_STATUS_SINGLE_VAL_UNIT_DATA, index: PID_STATUS_RSS_ANON},
	"RssFile":                    {dataType: PID_STATUS_SINGLE_VAL_UNIT_DATA, index: PID_STATUS_RSS_FILE},
	"RssShmem":                   {dataType: PID_STATUS_SINGLE_VAL_UNIT_DATA, index: PID_STATUS_RSS_SHMEM},
	"VmData":                     {dataType: PID_STATUS_SINGLE_VAL_UNIT_DATA, index: PID_STATUS_VM_DATA},
	"VmStk":                      {dataType: PID_STATUS_SINGLE_VAL_UNIT_DATA, index: PID_STATUS_VM_STK},
	"VmExe":                      {dataType: PID_STATUS_SINGLE_VAL_UNIT_DATA, index: PID_STATUS_VM_EXE},
	"VmLib":                      {dataType: PID_STATUS_SINGLE_VAL_UNIT_DATA, index: PID_STATUS_VM_LIB},
	"VmPTE":                      {dataType: PID_STATUS_SINGLE_VAL_UNIT_DATA, index: PID_STATUS_VM_PTE},
	"VmPMD":                      {dataType: PID_STATUS_SINGLE_VAL_UNIT_DATA, index: PID_STATUS_VM_PMD},
	"VmSwap":                     {dataType: PID_STATUS_SINGLE_VAL_UNIT_DATA, index: PID_STATUS_VM_SWAP},
	"HugetlbPages":               {dataType: PID_STATUS_SINGLE_VAL_UNIT_DATA, index: PID_STATUS_HUGETLBPAGES},
	"Cpus_allowed_list":          {dataType: PID_STATUS_SINGLE_VAL_DATA, index: PID_STATUS_CPUS_ALLOWED_LIST},
	"Mems_allowed_list":          {dataType: PID_STATUS_SINGLE_VAL_DATA, index: PID_STATUS_MEMS_ALLOWED_LIST},
	"voluntary_ctxt_switches":    {dataType: PID_STATUS_ULONG_DATA, index: PID_STATUS_VOLUNTARY_CTXT_SWITCHES},
	"nonvoluntary_ctxt_switches": {dataType: PID_STATUS_ULONG_DATA, index: PID_STATUS_NONVOLUNTARY_CTXT_SWITCHES},
}

// The following prefixes are optional:
var pidStatusOptionalPrefixes = []string{
	"VmPMD",
}

// The parser will use the following structure, built JIT at the 1st invocation:
var pidStatusParserInfo = struct {
	// Array indexed by line# (from 0), based on pidStatusLineHandlingMap:
	lineHandling []*PidStatusLineHandling
	// Units, where applicable. The units are kernel dependent so they are
	// discovered once and reused:
	byteSliceFieldUnit [][]byte
	// Lock protection for multi-threaded invocation, with multiple threads
	// happening to be at the 1st invocation:
	lock *sync.Mutex
}{
	lock: &sync.Mutex{},
}

var pidStatusReadFileBufPool = ReadFileBufPool16k

// Used for testing:
func resetPidStatusParserInfo() {
	pidStatusParserInfo.lock.Lock()
	pidStatusParserInfo.lineHandling = nil
	pidStatusParserInfo.byteSliceFieldUnit = nil
	pidStatusParserInfo.lock.Unlock()
}

func initPidStatusParserInfo(path string) error {
	var (
		err                error
		lineHandling       []*PidStatusLineHandling
		byteSliceFieldUnit [][]byte
	)

	defer func() {
		if err != nil {
			pidStatusParserInfo.lineHandling = nil
			pidStatusParserInfo.byteSliceFieldUnit = nil
		} else {
			pidStatusParserInfo.lineHandling = lineHandling
			pidStatusParserInfo.byteSliceFieldUnit = byteSliceFieldUnit
		}
	}()

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	optionalPrefixes := map[string]bool{}
	for _, prefix := range pidStatusOptionalPrefixes {
		optionalPrefixes[prefix] = true
	}

	mandatoryPrefixes := map[string]bool{}
	for prefix := range pidStatusLineHandlingMap {
		if !optionalPrefixes[prefix] {
			mandatoryPrefixes[prefix] = true
		}
	}

	lineHandling = make([]*PidStatusLineHandling, 0)
	byteSliceFieldUnit = make([][]byte, PID_STATUS_BYTE_SLICE_NUM_FIELDS)
	scanner := bufio.NewScanner(file)
	for lineIndex := 0; scanner.Scan(); lineIndex++ {
		line := scanner.Bytes()
		prefix := ""
		colIndex := bytes.IndexByte(line, ':')
		if colIndex > 0 {
			prefix = strings.TrimSpace(string(line[:colIndex]))
		}
		if prefix == "" {
			err = fmt.Errorf(
				"%s#%d: %q: missing `PREFIX:'",
				path, lineIndex+1, line,
			)
			return err
		}
		delete(mandatoryPrefixes, prefix)
		handling := pidStatusLineHandlingMap[prefix]
		if handling != nil {
			handling = &PidStatusLineHandling{
				dataType:  handling.dataType,
				index:     handling.index,
				prefixLen: colIndex + 1,
			}
		}
		lineHandling = append(lineHandling, handling)
		if handling == nil {
			continue
		}

		switch handling.dataType {
		case PID_STATUS_SINGLE_VAL_UNIT_DATA:
			fields := bytes.Fields(line[handling.prefixLen:])
			if len(fields) != 2 {
				err = fmt.Errorf(
					"%s#%d: %q: missing UNIT",
					path, lineIndex+1, line,
				)
				return err
			}
			byteSliceFieldUnit[handling.index] = make([]byte, len(fields[1]))
			copy(byteSliceFieldUnit[handling.index], fields[1])
		}
	}
	if scanner.Err() == nil {
		// Sanity check that all line handlers were resolved:
		if len(mandatoryPrefixes) > 0 {
			missingPrefixes := make([]string, 0)
			for prefix := range mandatoryPrefixes {
				missingPrefixes = append(missingPrefixes, prefix)
			}
			err = fmt.Errorf("%s: unresolved prefix(es): %q", path, missingPrefixes)
		}
	} else {
		err = fmt.Errorf("%s: %v", path, scanner.Err())
	}
	return err
}

func NewPidStatus(procfsRoot string, pid, tid int) *PidStatus {
	pidStatus := &PidStatus{
		ByteSliceFieldsBuf:    &bytes.Buffer{},
		ByteSliceFieldOffsets: make([]SliceOffsets, PID_STATUS_BYTE_SLICE_NUM_FIELDS),
		NumericFields:         make([]uint64, PID_STATUS_ULONG_NUM_FIELDS),
	}
	if tid == PID_STAT_PID_ONLY_TID {
		pidStatus.path = path.Join(procfsRoot, strconv.Itoa(pid), "status")
	} else {
		pidStatus.path = path.Join(procfsRoot, strconv.Itoa(pid), "task", strconv.Itoa(tid), "status")
	}
	return pidStatus
}

func (pidStatus *PidStatus) Clone(full bool) *PidStatus {
	newPidStatus := &PidStatus{
		ByteSliceFieldOffsets: make([]SliceOffsets, PID_STATUS_BYTE_SLICE_NUM_FIELDS),
		NumericFields:         make([]uint64, PID_STATUS_ULONG_NUM_FIELDS),
		path:                  pidStatus.path,
		lineHandling:          pidStatus.lineHandling,
		ByteSliceFieldUnit:    pidStatus.ByteSliceFieldUnit,
	}

	if pidStatus.ByteSliceFieldsBuf == nil {
		newPidStatus.ByteSliceFieldsBuf = &bytes.Buffer{}
	} else {
		newPidStatus.ByteSliceFieldsBuf = bytes.NewBuffer(make([]byte, pidStatus.ByteSliceFieldsBuf.Cap()))
	}
	if full {
		newPidStatus.ByteSliceFieldsBuf.Write(pidStatus.ByteSliceFieldsBuf.Bytes())
		copy(newPidStatus.ByteSliceFieldOffsets, pidStatus.ByteSliceFieldOffsets)
		copy(newPidStatus.NumericFields, pidStatus.NumericFields)
	}
	return newPidStatus
}

func (pidStatus *PidStatus) setLineHandling() error {
	pidStatusParserInfo.lock.Lock()
	defer pidStatusParserInfo.lock.Unlock()
	if pidStatusParserInfo.lineHandling == nil {
		err := initPidStatusParserInfo(pidStatus.path)
		if err != nil {
			return err
		}
	}
	pidStatus.lineHandling = pidStatusParserInfo.lineHandling
	pidStatus.ByteSliceFieldUnit = pidStatusParserInfo.byteSliceFieldUnit
	return nil
}

// The parser will populate PidStatus with the latest information. If the
// returned error is not nil, then the data part of PidStatus is undefined, it
// may contain leftovers + partially new data, up to the point of error.
func (pidStatus *PidStatus) Parse() error {
	// Copy reference of parser info, as needed. Build the latter JIT as needed.
	if pidStatus.lineHandling == nil {
		err := pidStatus.setLineHandling()
		if err != nil {
			return err
		}
	}

	fBuf, err := pidStatusReadFileBufPool.ReadFile(pidStatus.path)
	defer pidStatusReadFileBufPool.ReturnBuf(fBuf)
	if err != nil {
		return err
	}
	buf, l := fBuf.Bytes(), fBuf.Len()

	byteSliceFieldsBuf := pidStatus.ByteSliceFieldsBuf
	if byteSliceFieldsBuf == nil {
		byteSliceFieldsBuf := &bytes.Buffer{}
		pidStatus.ByteSliceFieldsBuf = byteSliceFieldsBuf
	} else {
		byteSliceFieldsBuf.Reset()
	}
	byteSliceFieldOffsets := pidStatus.ByteSliceFieldOffsets
	lineHandling := pidStatus.lineHandling
	lineIndex, maxLineIndex := 0, len(lineHandling)
	for lineStart, byteSliceFieldsBufOff := 0, 0; lineStart < l; lineIndex++ {
		// Sanity check: too many lines?
		if lineIndex >= maxLineIndex {
			return fmt.Errorf(
				"%s: too many lines (> %d)",
				pidStatus.path, maxLineIndex,
			)
		}

		lineEnd := lineStart
		for ; lineEnd < l && buf[lineEnd] != '\n'; lineEnd++ {
		}

		// Handle this line or ignore?
		handling := lineHandling[lineIndex]
		if handling != nil {
			// Locate value start:
			pos := lineStart + handling.prefixLen
			for ; pos < lineEnd && isWhitespace[buf[pos]]; pos++ {
			}
			if pos == lineEnd {
				return fmt.Errorf(
					"%s#%d: %q: truncated line",
					pidStatus.path, lineIndex+1, string(buf[lineStart:lineEnd]),
				)
			}
			// Handle the field according to its type:
			dataType, fieldIndex := handling.dataType, handling.index
			switch dataType {
			case PID_STATUS_SINGLE_VAL_DATA, PID_STATUS_SINGLE_VAL_UNIT_DATA:
				valueStart := pos
				for ; pos < lineEnd && !isWhitespace[buf[pos]]; pos++ {
				}
				n, err := byteSliceFieldsBuf.Write(buf[valueStart:pos])
				if err != nil {
					return fmt.Errorf(
						"%s#%d: %q: %v",
						pidStatus.path, lineIndex+1, string(buf[lineStart:lineEnd]), err,
					)
				}
				byteSliceFieldOffsets[fieldIndex].Start = byteSliceFieldsBufOff
				byteSliceFieldsBufOff += n
				byteSliceFieldOffsets[fieldIndex].End = byteSliceFieldsBufOff

			case PID_STATUS_LIST_DATA:
				// Join the words separated by a single, standard, sep:
				for firstWord := true; err == nil && pos < lineEnd; {
					valueStart, n := pos, 0
					for ; pos < lineEnd && !isWhitespace[buf[pos]]; pos++ {
					}
					if firstWord {
						byteSliceFieldOffsets[fieldIndex].Start = byteSliceFieldsBufOff
						firstWord = false
					} else {
						err = byteSliceFieldsBuf.WriteByte(PID_STATUS_LIST_DATA_SEP)
						byteSliceFieldsBufOff++
					}
					if err == nil {
						n, err = byteSliceFieldsBuf.Write(buf[valueStart:pos])
					}
					if err == nil {
						byteSliceFieldsBufOff += n
						for ; pos < lineEnd && isWhitespace[buf[pos]]; pos++ {
						}
					}
				}
				if err != nil {
					return fmt.Errorf(
						"%s#%d: %q: %v",
						pidStatus.path, lineIndex+1, string(buf[lineStart:lineEnd]), err,
					)
				}
				byteSliceFieldOffsets[fieldIndex].End = byteSliceFieldsBufOff

			case PID_STATUS_ULONG_DATA:
				value := uint64(0)
				for ; pos < lineEnd; pos++ {
					c := buf[pos]
					if digit := c - '0'; digit < 10 {
						value = (value << 3) + (value << 1) + uint64(digit)
					} else if isWhitespace[c] {
						break
					} else {
						return fmt.Errorf(
							"%s#%d: %q: `%c' invalid value for a digit",
							pidStatus.path, lineIndex+1, string(buf[lineStart:lineEnd]), c,
						)
					}
				}
				pidStatus.NumericFields[fieldIndex] = value
			}
		}

		lineStart = lineEnd + 1
	}

	// Sanity check: got all the expected lines?
	if lineIndex < maxLineIndex {
		return fmt.Errorf(
			"%s: missing lines: want: %d, got %d",
			pidStatus.path, maxLineIndex, lineIndex,
		)
	}

	return nil
}
