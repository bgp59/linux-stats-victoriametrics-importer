// parser for /proc/pid/status and /proc/pid/task/tid/status

package procfs

import (
	"fmt"
	"path"
	"strconv"
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
//   - single value + unit (a parallel [][]byte)	// VmPeak:	  222400 kB <- unit
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

type PidStatusParser interface {
	Parse(pidTidPath string) error
	GetByteSliceFieldsAndUnits() ([][]byte, [][]byte)
	GetNumericFields() []uint64
}

type NewPidStatusParser func() PidStatusParser

type PidStatus struct {
	// As-is fields:
	byteSliceFields [][]byte
	// byteSliceFieldUnit:
	byteSliceFieldUnit [][]byte
	// Numeric fields:
	numericFields []uint64
}

type PidStatusLineHandling struct {
	// How to parse the line:
	dataType byte
	// Array index where to store the result; the actual array depends upon the
	// data type:
	index int
	// Whether an empty value is accepted or not:
	emptyValueOK bool
}

// Only the lines w/ the prefix in the map below will be processed. The map will
// be converted into an array, indexed by line# (starting from 0), at the first
// parse invocation (JIT).
var pidStatusLineHandlingMap = map[string]*PidStatusLineHandling{
	"Uid":                        {dataType: PID_STATUS_LIST_DATA, index: PID_STATUS_UID},
	"Gid":                        {dataType: PID_STATUS_LIST_DATA, index: PID_STATUS_GID},
	"Groups":                     {dataType: PID_STATUS_LIST_DATA, index: PID_STATUS_GROUPS, emptyValueOK: true},
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

var pidStatusReadFileBufPool = ReadFileBufPool16k

func PidStatusNameToIndex(name string) int {
	lh, ok := pidStatusLineHandlingMap[name]
	if ok {
		return lh.index
	}
	return -1
}

func PidStatusPath(procfsRoot string, pid, tid int) string {
	if tid == PID_ONLY_TID {
		return path.Join(procfsRoot, strconv.Itoa(pid), "status")
	} else {
		return path.Join(procfsRoot, strconv.Itoa(pid), "task", strconv.Itoa(tid), "status")
	}
}

func NewPidStatus() PidStatusParser {
	return &PidStatus{
		byteSliceFields:    make([][]byte, PID_STATUS_BYTE_SLICE_NUM_FIELDS),
		byteSliceFieldUnit: make([][]byte, PID_STATUS_BYTE_SLICE_NUM_FIELDS),
		numericFields:      make([]uint64, PID_STATUS_ULONG_NUM_FIELDS),
	}
}

func (pidStatus *PidStatus) Parse(pidTidPath string) error {
	pidStatusPath := path.Join(pidTidPath, "status")
	fBuf, err := pidStatusReadFileBufPool.ReadFile(pidStatusPath)
	defer pidStatusReadFileBufPool.ReturnBuf(fBuf)
	if err != nil {
		return err
	}
	buf, l := fBuf.Bytes(), fBuf.Len()

	byteSliceFields, byteSliceFieldUnit := pidStatus.byteSliceFields, pidStatus.byteSliceFieldUnit
	numericFields := pidStatus.numericFields

	pos, lineNum, eol := 0, 0, true
	// Keep track of found fields; those not found should be cleared at the end:
	foundByteSliceFields, missingByteSliceFieldsCnt := [PID_STATUS_BYTE_SLICE_NUM_FIELDS]bool{}, PID_STATUS_BYTE_SLICE_NUM_FIELDS
	for {
		// It not at EOL, locate the end of the current line and move past it;
		// this may happen if the previous line wasn't fully used:
		for ; !eol && pos < l; pos++ {
			eol = (buf[pos] == '\n')
		}

		// Loop end condition, the entire buffer was used:
		if pos >= l {
			break
		}

		// Line starts here:
		lineStartPos := pos
		eol = false
		lineNum++

		// Identify prefix and based on it, how to handle the line:
		for ; pos < l && isWhitespace[buf[pos]]; pos++ {
		}
		prefixStartPos, prefixEndPos := pos, -1
		for ; !eol && pos < l && prefixEndPos < 0; pos++ {
			c := buf[pos]
			if c == ':' {
				prefixEndPos = pos
			} else {
				eol = (c == '\n')
			}
		}
		if prefixEndPos <= prefixStartPos {
			return fmt.Errorf(
				"%s:%d: %q: `PREFIX:' not found",
				pidStatusPath, lineNum, getCurrentLine(buf, lineStartPos),
			)
		}
		handling := pidStatusLineHandlingMap[string(buf[prefixStartPos:prefixEndPos])]
		if handling == nil {
			continue
		}

		// Locate value start:
		for ; pos < l && isWhitespace[buf[pos]]; pos++ {
		}

		// Handle the field according to its type:
		dataType, fieldIndex, emptyValueOK := handling.dataType, handling.index, handling.emptyValueOK

		if dataType == PID_STATUS_ULONG_DATA {
			value := uint64(0)
			for done := false; !eol && pos < l && !done; pos++ {
				c := buf[pos]
				if digit := c - '0'; digit < 10 {
					value = (value << 3) + (value << 1) + uint64(digit)
				} else if eol = (c == '\n'); eol || isWhitespace[c] {
					done = true
				} else {
					return fmt.Errorf(
						"%s:%d: %q: `%c' invalid value for a digit",
						pidStatusPath, lineNum, getCurrentLine(buf, lineStartPos), c,
					)
				}
			}
			numericFields[fieldIndex] = value
			continue
		}

		// Byte slice field:
		field := byteSliceFields[fieldIndex]
		field = field[:cap(field)] // max out usable storage
		valueLen := 0

		if dataType == PID_STATUS_LIST_DATA {
			for wasSep := true; !eol && pos < l; pos++ {
				c := buf[pos]
				eol = (c == '\n')
				isSep := eol || isWhitespace[c]
				if !isSep {
					if wasSep {
						// A new word in the list, if this not the 1st one then
						// add the separator:
						if valueLen > 0 {
							if len(field) <= valueLen {
								field = append(field, PID_STATUS_LIST_DATA_SEP)
							} else {
								field[valueLen] = PID_STATUS_LIST_DATA_SEP
							}
							valueLen++
						}
					}
					if len(field) <= valueLen {
						field = append(field, c)
					} else {
						field[valueLen] = c
					}
					valueLen++
				}
				wasSep = isSep
			}
		} else { // if dataType == PID_STATUS_SINGLE_VAL_DATA || dataType == PID_STATUS_SINGLE_VAL_UNIT_DATA
			valueStartPos := pos
			for done := false; !eol && pos < l && !done; pos++ {
				c := buf[pos]
				if eol = (c == '\n'); eol || isWhitespace[c] {
					valueLen = pos - valueStartPos
					done = true
				}
			}
			if len(field) < valueLen {
				field = make([]byte, valueLen)
			}
			copy(field, buf[valueStartPos:valueStartPos+valueLen])

			// If the field has a unit then parse it unless already determined
			// from a previous scan; the unit is supposed to be coded into the
			// kernel procfs so it cannot change during runtime:
			if dataType == PID_STATUS_SINGLE_VAL_UNIT_DATA && len(byteSliceFieldUnit[fieldIndex]) == 0 {
				for ; pos < l && isWhitespace[buf[pos]]; pos++ {
				}
				unitStartPos, unitLen := pos, 0
				for done := false; !eol && pos < l && !done; pos++ {
					c := buf[pos]
					if eol = (c == '\n'); eol || isWhitespace[c] {
						unitLen = pos - unitStartPos
						done = true
					}
				}
				if unitLen == 0 {
					return fmt.Errorf(
						"%s:%d: %q: missing unit",
						pidStatusPath, lineNum, getCurrentLine(buf, lineStartPos),
					)
				}
				byteSliceFieldUnit[fieldIndex] = make([]byte, unitLen)
				copy(byteSliceFieldUnit[fieldIndex], buf[unitStartPos:unitStartPos+unitLen])
			}
		}

		// Check for empty field:
		if valueLen == 0 && !emptyValueOK {
			return fmt.Errorf(
				"%s:%d: %q: empty value(s)",
				pidStatusPath, lineNum, getCurrentLine(buf, lineStartPos),
			)

		}

		// Store the value:
		byteSliceFields[fieldIndex] = field[:valueLen]

		// Mark it as found:
		foundByteSliceFields[fieldIndex] = true
		missingByteSliceFieldsCnt--
	}

	if missingByteSliceFieldsCnt > 0 {
		// Clear not found fields:
		for fieldIndex, found := range foundByteSliceFields {
			if !found && byteSliceFields[fieldIndex] != nil {
				byteSliceFields[fieldIndex] = nil
			}
		}
	}

	return nil
}

func (pidStatus *PidStatus) GetByteSliceFieldsAndUnits() ([][]byte, [][]byte) {
	return pidStatus.byteSliceFields, pidStatus.byteSliceFieldUnit
}

func (pidStatus *PidStatus) GetNumericFields() []uint64 {
	return pidStatus.numericFields
}
