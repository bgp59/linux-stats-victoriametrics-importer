// parser for /proc/stat

// Note: interrupts, hard and soft, are ignored from this source since more
// detailed information will be gleaned from interrupts and softirq files
// respectively.

package procfs

import (
	"fmt"
	"path"
)

type Stat struct {
	// The all CPU info:
	CpuAll []uint64
	// The per CPU info is stored in a list of lists, the 1st indexList being CPU#
	// (0..MAX_CPU-1) and the second the CPU statistic.
	Cpu [][]uint64
	// Some CPU's might be disabled via CPU Hot Plug and their stat may not be
	// available; maintain a bitmap for CPU# found in the current scan:
	CpuPresent []uint64
	// The max CPU# ever found:
	MaxCpuNum int
	// Any other info is a scalar in a list:
	NumericFields []uint64
	// The path file to read:
	path string
}

// Indexes for cpu[] stats:
const (
	STAT_CPU_USER_TICKS = iota
	STAT_CPU_NICE_TICKS
	STAT_CPU_SYSTEM_TICKS
	STAT_CPU_IDLE_TICKS
	STAT_CPU_IOWAIT_TICKS
	STAT_CPU_IRQ_TICKS
	STAT_CPU_SOFTIRQ_TICKS
	STAT_CPU_STEAL_TICKS
	STAT_CPU_GUEST_TICKS
	STAT_CPU_GUEST_NICE_TICKS

	STAT_CPU_STATS_COUNT // Must be last!
)

const (
	STAT_CPU_MIN_NUM_FIELDS = STAT_CPU_SOFTIRQ_TICKS + 1
)

// Indexes for NumericFields:
const (
	STAT_PAGE_IN = iota
	STAT_PAGE_OUT
	STAT_SWAP_IN
	STAT_SWAP_OUT
	STAT_CTXT
	STAT_BTIME
	STAT_PROCESSES
	STAT_PROCS_RUNNING
	STAT_PROCS_BLOCKED

	STAT_NUMERIC_FIELDS_COUNT // Must be last!
)

var statLinePrefixSeparator = [256]bool{
	' ':  true,
	'\t': true,
	// Include digits for cpuNN:
	'0': true,
	'1': true,
	'2': true,
	'3': true,
	'4': true,
	'5': true,
	'6': true,
	'7': true,
	'8': true,
	'9': true,
}

// Given a line prefix, map the associated data fields into indexes where the
// parsed value will be stored.
// e.g.:
//
//	"PREFIX": []int{INDEX0, INDEX1, ...}
//
// when the line:
//
//	PREFIX: VAL0 VAL1 ...
//
// is parsed, VAL0 -> DATA[INDEX0], VAL1 -> DATA[INDEX1], ...
// where DATA is ether a Cpu list or NumericFields:
var statPrefixToDstIndexList = map[string][]int{
	"cpu": []int{
		STAT_CPU_USER_TICKS,
		STAT_CPU_NICE_TICKS,
		STAT_CPU_SYSTEM_TICKS,
		STAT_CPU_IDLE_TICKS,
		STAT_CPU_IOWAIT_TICKS,
		STAT_CPU_IRQ_TICKS,
		STAT_CPU_SOFTIRQ_TICKS,
		STAT_CPU_STEAL_TICKS,
		STAT_CPU_GUEST_TICKS,
		STAT_CPU_GUEST_NICE_TICKS,
	},
	"page": []int{
		STAT_PAGE_IN,
		STAT_PAGE_OUT,
	},
	"swap": []int{
		STAT_SWAP_IN,
		STAT_SWAP_OUT,
	},
	"ctxt": []int{
		STAT_CTXT,
	},
	"btime": []int{
		STAT_BTIME,
	},
	"processes": []int{
		STAT_PROCESSES,
	},
	"procs_running": []int{
		STAT_PROCS_RUNNING,
	},
	"procs_blocked": []int{
		STAT_PROCS_BLOCKED,
	},
}

var statReadFileBufPool = ReadFileBufPool256k

func NewStat(procfsRoot string) *Stat {
	return &Stat{
		CpuAll:        make([]uint64, STAT_CPU_STATS_COUNT),
		Cpu:           make([][]uint64, 0),
		CpuPresent:    make([]uint64, 16),
		NumericFields: make([]uint64, STAT_NUMERIC_FIELDS_COUNT),
		path:          path.Join(procfsRoot, "stat"),
	}
}

func (stat *Stat) Clone() *Stat {
	newStat := &Stat{
		CpuAll:        make([]uint64, STAT_CPU_STATS_COUNT),
		Cpu:           make([][]uint64, len(stat.Cpu)),
		CpuPresent:    make([]uint64, len(stat.CpuPresent)),
		MaxCpuNum:     stat.MaxCpuNum,
		NumericFields: make([]uint64, STAT_NUMERIC_FIELDS_COUNT),
		path:          stat.path,
	}
	for i := 0; i < len(stat.Cpu); i++ {
		newStat.Cpu[i] = make([]uint64, STAT_CPU_STATS_COUNT)
	}
	return newStat
}

func (stat *Stat) Parse() error {
	fBuf, err := statReadFileBufPool.ReadFile(stat.path)
	defer statReadFileBufPool.ReturnBuf(fBuf)
	if err != nil {
		return err
	}

	buf, l := fBuf.Bytes(), fBuf.Len()

	for i := 0; i < len(stat.CpuPresent) && i <= (stat.MaxCpuNum>>6); i++ {
		stat.CpuPresent[i] = 0
	}

	for pos, lineNum := 0, 1; pos < l; lineNum++ {
		var (
			// Where to store the data parsed from a line, it will be updated w/ the
			// proper reference depending upon the line prefix:
			numericFields []uint64

			// The minimum number of fields, prefix dependent:
			minNumFields int
		)

		// Line starts here:
		eol := false

		// Locate prefix start:
		for ; pos < l && isWhitespace[buf[pos]]; pos++ {
		}
		prefixStart := pos

		// Locate prefix end:
		for ; !eol && pos < l; pos++ {
			c := buf[pos]
			eol = (c == '\n')
			if statLinePrefixSeparator[c] {
				break
			}
		}
		if eol {
			// Ignore empty lines, there shouldn't be any, though:
			continue
		}

		// Extract the prefix and determine the destination for the parsed values:
		prefix := string(buf[prefixStart:pos])
		indexList := statPrefixToDstIndexList[prefix]
		if indexList != nil {
			if prefix == "cpu" {
				// Extract cpu#, if any:
				cpuNum := -1
				for done := false; !done && pos < l; pos++ {
					c := buf[pos]
					if eol = (c == '\n'); eol || isWhitespace[c] {
						done = true
					} else {
						if digit := c - '0'; digit < 10 {
							if cpuNum < 0 {
								cpuNum = 0
							}
							cpuNum = (cpuNum << 3) + (cpuNum << 1) + int(digit)
						} else {
							return fmt.Errorf(
								"%s#%d: %q: cpu# `%c': invalid digit",
								stat.path, lineNum, getCurrentLine(buf, pos), c,
							)
						}
					}
				}

				// Determine which cpuStats should hold the parsed data:
				if cpuNum >= 0 {
					if cpuNum >= stat.MaxCpuNum {
						stat.MaxCpuNum = cpuNum
					}
					cpuPresentChunkNum := cpuNum >> 6 // / 64
					if cpuPresentChunkNum >= len(stat.CpuPresent) {
						newCpuPresent := make([]uint64, cpuPresentChunkNum+1)
						copy(newCpuPresent, stat.CpuPresent)
						stat.CpuPresent = newCpuPresent
					}
					stat.CpuPresent[cpuPresentChunkNum] |= (1 << (cpuNum & ((1 << 6) - 1)))
					if cpuNum >= len(stat.Cpu) {
						newCpu := make([][]uint64, cpuNum+1)
						copy(newCpu, stat.Cpu)
						stat.Cpu = newCpu
					}
					numericFields = stat.Cpu[cpuNum]
					if numericFields == nil {
						numericFields = make([]uint64, STAT_CPU_STATS_COUNT)
						stat.Cpu[cpuNum] = numericFields
					}
				} else {
					numericFields = stat.CpuAll
				}
				minNumFields = STAT_CPU_MIN_NUM_FIELDS
			} else {
				numericFields = stat.NumericFields
				minNumFields = len(indexList)
			}

			// Parse the fields into numeric values, stored to the right index
			// of numericFields:
			fieldIndex, numFields := 0, len(indexList)
			for !eol && pos < l && fieldIndex < numFields {
				// Field start:
				for ; pos < l && isWhitespace[buf[pos]]; pos++ {
				}

				// Parse the numerical value:
				val, hasVal := uint64(0), false
				for done := false; !done && pos < l; pos++ {
					c := buf[pos]
					if digit := c - '0'; digit < 10 {
						hasVal = true
						val = (val << 3) + (val << 1) + uint64(digit)
					} else if eol = (c == '\n'); eol || isWhitespace[c] {
						done = true
					} else {
						return fmt.Errorf(
							"%s#%d: %q: `%c': invalid digit",
							stat.path, lineNum, getCurrentLine(buf, pos), c,
						)
					}
				}
				// If a value was found, assign it to the right index:
				if hasVal {
					numericFields[indexList[fieldIndex]] = val
					fieldIndex++
				}
			}

			// Verify that enough values were parsed:
			if fieldIndex < minNumFields {
				return fmt.Errorf(
					"%s#%d: %q: invalid value count: want (at least) %d, got: %d",
					stat.path, lineNum, getCurrentLine(buf, pos), minNumFields, fieldIndex,
				)
			}
		}

		// Locate EOL:
		for ; !eol && pos < l; pos++ {
			eol = (buf[pos] == '\n')
		}
	}

	return nil
}
