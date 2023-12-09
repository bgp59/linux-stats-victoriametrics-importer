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
//   - single value + unit       // VmPeak:	  222400 kB
//   - list                      // Uid:	104	104	104	104

// Parsed data types:
const (
	PID_STATUS_SINGLE_VAL_DATA = iota
	PID_STATUS_SINGLE_VAL_UNIT_DATA
	PID_STATUS_LIST_DATA
	PID_STATUS_ULONG_DATA
)

var PID_STATUS_LIST_DATA_JOIN_SEQ = []byte(",")

// The parsed data will be stored into 2 array sets: one for as-is, the other
// for numerical, using the following indexes:
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
	PID_STATUS_AS_IS_NUM_FIELDS
)

const (
	PID_STATUS_VOLUNTARY_CTXT_SWITCHES = iota
	PID_STATUS_NONVOLUNTARY_CTXT_SWITCHES
	// Must be last:
	PID_STATUS_ULONG_NUM_FIELDS
)

type PidStatusLineHandling struct {
	// How to parse the line:
	dataType byte
	// Array index where to store the result:
	index int
}

// Only the lines w/ the prefix in the map below will be processed. The map will
// be converted into an array, indexed by line# (starting from 0), at the first
// parse invocation (JIT).
var pidStatusLineHandlingMap = map[string]*PidStatusLineHandling{
	"Uid:":                        {PID_STATUS_LIST_DATA, PID_STATUS_UID},
	"Gid:":                        {PID_STATUS_LIST_DATA, PID_STATUS_GID},
	"Groups:":                     {PID_STATUS_LIST_DATA, PID_STATUS_GROUPS},
	"VmPeak:":                     {PID_STATUS_SINGLE_VAL_UNIT_DATA, PID_STATUS_VM_PEAK},
	"VmSize:":                     {PID_STATUS_SINGLE_VAL_UNIT_DATA, PID_STATUS_VM_SIZE},
	"VmLck:":                      {PID_STATUS_SINGLE_VAL_UNIT_DATA, PID_STATUS_VM_LCK},
	"VmPin:":                      {PID_STATUS_SINGLE_VAL_UNIT_DATA, PID_STATUS_VM_PIN},
	"VmHWM:":                      {PID_STATUS_SINGLE_VAL_UNIT_DATA, PID_STATUS_VM_HWM},
	"VmRSS:":                      {PID_STATUS_SINGLE_VAL_UNIT_DATA, PID_STATUS_VM_RSS},
	"RssAnon:":                    {PID_STATUS_SINGLE_VAL_UNIT_DATA, PID_STATUS_RSS_ANON},
	"RssFile:":                    {PID_STATUS_SINGLE_VAL_UNIT_DATA, PID_STATUS_RSS_FILE},
	"RssShmem:":                   {PID_STATUS_SINGLE_VAL_UNIT_DATA, PID_STATUS_RSS_SHMEM},
	"VmData:":                     {PID_STATUS_SINGLE_VAL_UNIT_DATA, PID_STATUS_VM_DATA},
	"VmStk:":                      {PID_STATUS_SINGLE_VAL_UNIT_DATA, PID_STATUS_VM_STK},
	"VmExe:":                      {PID_STATUS_SINGLE_VAL_UNIT_DATA, PID_STATUS_VM_EXE},
	"VmLib:":                      {PID_STATUS_SINGLE_VAL_UNIT_DATA, PID_STATUS_VM_LIB},
	"VmPTE:":                      {PID_STATUS_SINGLE_VAL_UNIT_DATA, PID_STATUS_VM_PTE},
	"VmPMD:":                      {PID_STATUS_SINGLE_VAL_UNIT_DATA, PID_STATUS_VM_PMD},
	"VmSwap:":                     {PID_STATUS_SINGLE_VAL_UNIT_DATA, PID_STATUS_VM_SWAP},
	"HugetlbPages:":               {PID_STATUS_SINGLE_VAL_UNIT_DATA, PID_STATUS_HUGETLBPAGES},
	"Cpus_allowed_list:":          {PID_STATUS_SINGLE_VAL_DATA, PID_STATUS_CPUS_ALLOWED_LIST},
	"Mems_allowed_list:":          {PID_STATUS_SINGLE_VAL_DATA, PID_STATUS_MEMS_ALLOWED_LIST},
	"voluntary_ctxt_switches:":    {PID_STATUS_ULONG_DATA, PID_STATUS_VOLUNTARY_CTXT_SWITCHES},
	"nonvoluntary_ctxt_switches:": {PID_STATUS_ULONG_DATA, PID_STATUS_NONVOLUNTARY_CTXT_SWITCHES},
}

// The parser will use the following structure, built JIT at the 1st invocation:
var pidStatusParserInfo = struct {
	// Array indexed by line# (from 0), based on pidStatusLineHandlingMap:
	lineHandling []*PidStatusLineHandling
	// Units, where applicable. The units are kernel dependent so they are
	// discovered once and reused:
	unit []string
	// Lock protection for multi-threaded invocation, with multiple threads
	// happening to be at the 1st invocation:
	lock *sync.Mutex
}{
	lock: &sync.Mutex{},
}

// Used for testing:
func resetPidStatusParserInfo() {
	pidStatusParserInfo.lock.Lock()
	pidStatusParserInfo.lineHandling = nil
	pidStatusParserInfo.unit = nil
	pidStatusParserInfo.lock.Unlock()
}

func initPidStatusParserInfo(path string) error {
	var (
		err          error
		lineHandling []*PidStatusLineHandling
		unit         []string
	)

	defer func() {
		if err != nil {
			pidStatusParserInfo.lineHandling = nil
			pidStatusParserInfo.unit = nil
		} else {
			pidStatusParserInfo.lineHandling = lineHandling
			pidStatusParserInfo.unit = unit
		}
	}()

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	lineHandling = make([]*PidStatusLineHandling, 0)
	unit = make([]string, PID_STATUS_AS_IS_NUM_FIELDS)
	scanner := bufio.NewScanner(file)
	for lineIndex := 0; scanner.Scan(); lineIndex++ {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			err = fmt.Errorf(
				"%s: line#: %d: %q: insufficient fields (< 2)",
				path, lineIndex+1,
				scanner.Text(),
			)
			return err
		}
		handling := pidStatusLineHandlingMap[fields[0]]
		lineHandling = append(lineHandling, handling)
		if handling == nil {
			continue
		}
		switch handling.dataType {
		case PID_STATUS_SINGLE_VAL_UNIT_DATA:
			if len(fields) != 3 {
				err = fmt.Errorf(
					"%s: line#: %d: %q: invalid field# (!= 3)",
					path, lineIndex+1,
					scanner.Text(),
				)
				return err
			}
			unit[handling.index] = string(fields[2])
		}
	}
	if scanner.Err() != nil {
		err = fmt.Errorf("%s: %v", path, scanner.Err())
	}

	return err
}

type PidStatus struct {
	// For as-is data there is a backing buffer + start,end indexes to build the
	// slice representing the field:
	Buf *bytes.Buffer
	// Start/end index for each as-is field (byte slice):
	ByteFields []SliceOffsets
	// Unsigned log data:
	NumericFields []uint64
	// Unit, it will be replicated from pidStatusParserInfo:
	Unit []string
	// Path to the file:
	path string
	// Parsing info replicated as a reference; this is to avoid locking, once
	// the reference was pulled:
	lineHandling []*PidStatusLineHandling
}

func NewPidStatus(procfsRoot string, pid, tid int) *PidStatus {
	pidStatus := &PidStatus{
		Buf:           &bytes.Buffer{},
		ByteFields:    make([]SliceOffsets, PID_STATUS_AS_IS_NUM_FIELDS),
		NumericFields: make([]uint64, PID_STATUS_ULONG_NUM_FIELDS),
	}
	if tid == PID_STAT_PID_ONLY_TID {
		pidStatus.path = path.Join(procfsRoot, strconv.Itoa(pid), "status")
	} else {
		pidStatus.path = path.Join(procfsRoot, strconv.Itoa(pid), "task", strconv.Itoa(tid), "status")
	}
	return pidStatus
}

// Clone is used by the double storage approach for deltas: previous + current.
func (pidStatus *PidStatus) Clone() *PidStatus {
	newPidStatus := &PidStatus{
		ByteFields:    make([]SliceOffsets, PID_STATUS_AS_IS_NUM_FIELDS),
		NumericFields: make([]uint64, PID_STATUS_ULONG_NUM_FIELDS),
		path:          pidStatus.path,
		lineHandling:  pidStatus.lineHandling,
		Unit:          pidStatus.Unit,
	}
	// If there is any backing storage then create a new one of the same
	// capacity since most likely it has the right capacity:
	if pidStatus.Buf == nil {
		newPidStatus.Buf = &bytes.Buffer{}
	} else {
		newPidStatus.Buf = bytes.NewBuffer(make([]byte, pidStatus.Buf.Cap()))
	}
	return newPidStatus
}

// The parser will populate PidStatus with the latest information. If the
// returned error is not nil, then the data part of PidStatus is undefined, it
// may contain leftovers + partially new data, up to the point of error.
func (pidStatus *PidStatus) Parse() error {
	// Copy reference of parser info, as needed. Build the latter JIT as needed.
	if pidStatus.lineHandling == nil {
		var err error
		pidStatusParserInfo.lock.Lock()
		if pidStatusParserInfo.lineHandling == nil {
			err = initPidStatusParserInfo(pidStatus.path)
		}
		if err == nil {
			pidStatus.lineHandling = pidStatusParserInfo.lineHandling
			pidStatus.Unit = pidStatusParserInfo.unit
		}
		pidStatusParserInfo.lock.Unlock()
		if err != nil {
			return err
		}
	}

	file, err := os.Open(pidStatus.path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineHandling := pidStatus.lineHandling
	maxLineIndex := len(lineHandling)
	fieldBuf := pidStatus.Buf
	if fieldBuf == nil {
		fieldBuf := &bytes.Buffer{}
		pidStatus.Buf = fieldBuf
	}
	fieldBuf.Reset()

	byteFields := pidStatus.ByteFields
	for fieldBufPos, lineIndex := 0, 0; scanner.Scan(); lineIndex++ {
		if lineIndex >= maxLineIndex {
			return fmt.Errorf(
				"%s: too many lines (> %d)",
				pidStatus.path,
				maxLineIndex,
			)
		}
		handling := lineHandling[lineIndex]
		if handling == nil {
			continue
		}
		dataType, index := handling.dataType, handling.index
		fields := bytes.Fields(scanner.Bytes())
		switch dataType {
		case PID_STATUS_SINGLE_VAL_DATA, PID_STATUS_SINGLE_VAL_UNIT_DATA:
			expectedNumFields := 2
			if dataType == PID_STATUS_SINGLE_VAL_UNIT_DATA {
				expectedNumFields = 3
			}
			if len(fields) != expectedNumFields {
				return fmt.Errorf(
					"%s: line# %d: %q: invalid number of fields (!= %d)",
					pidStatus.path,
					lineIndex+1,
					scanner.Text(),
					expectedNumFields,
				)
			}
			byteFields[index].Start = fieldBufPos
			n, err := fieldBuf.Write(fields[1])
			if err != nil {
				return fmt.Errorf(
					"%s: line# %d: %q: %v",
					pidStatus.path,
					lineIndex+1,
					scanner.Text(),
					err,
				)
			}
			fieldBufPos += n
			byteFields[index].End = fieldBufPos
		case PID_STATUS_LIST_DATA:
			if len(fields) < 2 {
				return fmt.Errorf(
					"%s: line# %d: %q: invalid number of fields (< 2)",
					pidStatus.path,
					lineIndex+1,
					scanner.Text(),
				)
			}
			byteFields[index].Start = fieldBufPos
			n, err := fieldBuf.Write(bytes.Join(fields[1:], PID_STATUS_LIST_DATA_JOIN_SEQ))
			if err != nil {
				return fmt.Errorf(
					"%s: line# %d: %q: %v",
					pidStatus.path,
					lineIndex+1,
					scanner.Text(),
					err,
				)
			}
			fieldBufPos += n
			byteFields[index].End = fieldBufPos
		case PID_STATUS_ULONG_DATA:
			if len(fields) != 2 {
				return fmt.Errorf(
					"%s: line# %d: %q: invalid number of fields (!= 2)",
					pidStatus.path,
					lineIndex+1,
					scanner.Text(),
				)
			}
			value, buf := uint64(0), fields[1]
			for pos := 0; pos < len(buf); pos++ {
				if digit := buf[pos] - '0'; digit < 10 {
					value = (value << 3) + (value << 1) + uint64(digit)
				} else {
					return fmt.Errorf(
						"%s: line# %d: %q: `%c' invalid value for a digit",
						pidStatus.path,
						lineIndex+1,
						scanner.Text(),
						buf[pos],
					)
				}
			}
			pidStatus.NumericFields[index] = value
		}
	}

	err = scanner.Err()
	if err != nil {
		return fmt.Errorf("%s: %v", pidStatus.path, err)
	}
	fmt.Println(pidStatus.Buf.Len())
	return nil
}
