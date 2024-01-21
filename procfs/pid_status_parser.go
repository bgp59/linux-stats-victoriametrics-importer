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

// The data gleaned from this file is of two types, depending on its use case:
// - byte slice: used as-is, the value from the file is the (label) value
//   associated w/ the metric, e.g. Vm... stats
// - numerical: used for calculations, e.g. voluntary_ctxt_switches
// As-is data comes in 3 flavors:
//   - single value              					// Umask:	0022
//   - single value + unit (a parallel [][]byte)	// VmPeak:	  222400 kB <- umnit
//   - list											// Uid:	104	104	104	104

// Parsed data types:
const (
	PID_STATUS_SINGLE_VAL_DATA = iota
	PID_STATUS_SINGLE_VAL_UNIT_DATA
	PID_STATUS_LIST_DATA
	PID_STATUS_ULONG_DATA
)

// The parsed data will be stored into 2 lists: one for byte slices, the
// other for numerical, using the following indexes:

// Indexes for byte slice data:
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
	// The character used to concatenate list values:
	PID_STATUS_LIST_DATA_SEP = ','
)

type PidStatus struct {
	// As-is fields:
	ByteSliceFields [][]byte
	// ByteSliceFieldUnit, it will be replicated from pidStatusParserInfo:
	ByteSliceFieldUnit [][]byte
	// Numeric fierlds:
	NumericFields []uint64
	// The path file to read as a pointer (see "Note about PID stats parsers" in
	// pid_stat_parser.go):
	path *string
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
	// Array by line index (from 0), based on pidStatusLineHandlingMap:
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
		ByteSliceFields: make([][]byte, PID_STATUS_BYTE_SLICE_NUM_FIELDS),
		NumericFields:   make([]uint64, PID_STATUS_ULONG_NUM_FIELDS),
	}
	var fPath string
	if tid == PID_STAT_PID_ONLY_TID {
		fPath = path.Join(procfsRoot, strconv.Itoa(pid), "status")
	} else {
		fPath = path.Join(procfsRoot, strconv.Itoa(pid), "task", strconv.Itoa(tid), "status")
	}
	pidStatus.path = &fPath
	return pidStatus
}

func (pidStatus *PidStatus) setLineHandling() error {
	pidStatusParserInfo.lock.Lock()
	defer pidStatusParserInfo.lock.Unlock()
	if pidStatusParserInfo.lineHandling == nil {
		err := initPidStatusParserInfo(*pidStatus.path)
		if err != nil {
			return err
		}
	}
	pidStatus.lineHandling = pidStatusParserInfo.lineHandling
	pidStatus.ByteSliceFieldUnit = pidStatusParserInfo.byteSliceFieldUnit
	return nil
}

func (pidStatus *PidStatus) Parse(pathFrom *PidStatus) error {
	if pathFrom != nil {
		pidStatus.path = pathFrom.path
	}

	// Copy reference of parser info, as needed. Build the latter JIT as needed.
	if pidStatus.lineHandling == nil {
		err := pidStatus.setLineHandling()
		if err != nil {
			return err
		}
	}

	fBuf, err := pidStatusReadFileBufPool.ReadFile(*pidStatus.path)
	defer pidStatusReadFileBufPool.ReturnBuf(fBuf)
	if err != nil {
		return err
	}
	buf, l := fBuf.Bytes(), fBuf.Len()

	byteSliceFields, numericFields := pidStatus.ByteSliceFields, pidStatus.NumericFields
	lineHandling := pidStatus.lineHandling
	lineIndex, maxLineIndex := 0, len(lineHandling)
	for lineStartPos := 0; lineStartPos < l; lineIndex++ {
		// Sanity check: too many lines?
		if lineIndex >= maxLineIndex {
			return fmt.Errorf(
				"%s: too many lines (> %d)",
				*pidStatus.path, maxLineIndex,
			)
		}

		// TODO: find a strict one pass implementation. For now the compromise
		// is to perform quick scan to detect the line end:
		lineEndPos := lineStartPos
		for ; lineEndPos < l && buf[lineEndPos] != '\n'; lineEndPos++ {
		}

		// Handle this line or ignore?
		handling := lineHandling[lineIndex]
		if handling != nil {
			// Locate value start:
			pos := lineStartPos + handling.prefixLen
			for ; pos < lineEndPos && isWhitespace[buf[pos]]; pos++ {
			}
			if pos >= lineEndPos {
				return fmt.Errorf(
					"%s#%d: %q: truncated line",
					*pidStatus.path, lineIndex+1, string(buf[lineStartPos:lineEndPos]),
				)
			}
			// Handle the field according to its type:
			dataType, fieldIndex := handling.dataType, handling.index
			switch dataType {
			case PID_STATUS_SINGLE_VAL_DATA, PID_STATUS_SINGLE_VAL_UNIT_DATA:
				field := byteSliceFields[fieldIndex]
				valueStart := pos
				for ; pos < lineEndPos && !isWhitespace[buf[pos]]; pos++ {
				}
				valueLen := pos - valueStart
				if cap(field) < valueLen {
					field = make([]byte, valueLen)
				} else if len(field) != valueLen {
					field = field[:valueLen]
				}
				copy(field, buf[valueStart:pos])
				byteSliceFields[fieldIndex] = field

			case PID_STATUS_LIST_DATA:
				// Join the words separated by the single standard separator:
				field := byteSliceFields[fieldIndex]
				// For capacity purposes assume the worst case scenario:
				maxFieldLen := lineEndPos - pos
				if cap(field) < maxFieldLen {
					field = make([]byte, maxFieldLen)
				} else if len(field) < maxFieldLen {
					field = field[:maxFieldLen]
				}
				fieldPos := 0
				for firstWord := true; pos < lineEndPos; {
					valueStart := pos
					for ; pos < lineEndPos && !isWhitespace[buf[pos]]; pos++ {
					}
					if !firstWord {
						field[fieldPos] = PID_STATUS_LIST_DATA_SEP
						fieldPos++
					} else {
						firstWord = false
					}
					copy(field[fieldPos:], buf[valueStart:pos])
					fieldPos += pos - valueStart
					// Skip over trailing whitespaces:
					for ; pos < lineEndPos && isWhitespace[buf[pos]]; pos++ {
					}
				}
				byteSliceFields[fieldIndex] = field[:fieldPos]

			case PID_STATUS_ULONG_DATA:
				value := uint64(0)
				for ; pos < lineEndPos; pos++ {
					c := buf[pos]
					if digit := c - '0'; digit < 10 {
						value = (value << 3) + (value << 1) + uint64(digit)
					} else if isWhitespace[c] {
						break
					} else {
						return fmt.Errorf(
							"%s#%d: %q: `%c' invalid value for a digit",
							*pidStatus.path, lineIndex+1, string(buf[lineStartPos:lineEndPos]), c,
						)
					}
				}
				numericFields[fieldIndex] = value
			}
		}

		lineStartPos = lineEndPos + 1
	}

	// Sanity check: got all the expected lines?
	if lineIndex < maxLineIndex {
		return fmt.Errorf(
			"%s: missing lines: want: %d, got: %d",
			*pidStatus.path, maxLineIndex, lineIndex,
		)
	}

	return nil
}
