// parser for /proc/stat

// Note: interrupts, hard and soft, are ignored from this source since more
// detailed information will be gleaned from interrupts and softirq files
// respectively.

package procfs

import (
	"fmt"
	"path"
)

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

	// Must be last!
	STAT_CPU_NUM_STATS
)

const (
	// The pseudo cpu# used for all:
	STAT_CPU_ALL = -1
	// The pseudo cpu# used to indicate that the prefix is not a CPU:
	STAT_NO_CPU_PREFIX = -2

	// The minimum number of columns expected for cpu stats:
	STAT_CPU_MIN_NUM_FIELDS = STAT_CPU_SOFTIRQ_TICKS + 1

	// Piggy back scan# (see Stat struct for explanation) into the regular stats array:
	STAT_CPU_SCAN_NUMBER = STAT_CPU_NUM_STATS

	// The size of the per cpu []int list:
	STAT_CPU_STATS_SZ = STAT_CPU_SCAN_NUMBER + 1
)

// Indexes for Values:
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

	// Must be last!
	STAT_NUMERIC_NUM_STATS
)

type Stat struct {
	// CPU stats indexed by CPU#; STAT_CPU_ALL is the index for all CPU:
	Cpu map[int][]uint64
	// Any other info is a scalar in a list:
	NumericFields []uint64
	// The path file to read:
	path string
	// CPUs may appear/disappear dynamically via the Hot Plug. The scan# below
	// is incremented at the beginning of the scan and each CPU found at the
	// current scan will have its STAT_CPU_SCAN_NUMBER value updated with it. At
	// the end of the scan, all CPUs with STAT_CPU_SCAN_NUMBER not matching will
	// be removed.
	scanNum uint64
}

// Given a line prefix, map the associated data fields into indexes where the
// parsed value will be stored.
var statPrefixToDstIndexList = map[string][]int{
	"cpu": {
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
	"page": {
		STAT_PAGE_IN,
		STAT_PAGE_OUT,
	},
	"swap": {
		STAT_SWAP_IN,
		STAT_SWAP_OUT,
	},
	"ctxt": {
		STAT_CTXT,
	},
	"btime": {
		STAT_BTIME,
	},
	"processes": {
		STAT_PROCESSES,
	},
	"procs_running": {
		STAT_PROCS_RUNNING,
	},
	"procs_blocked": {
		STAT_PROCS_BLOCKED,
	},
}

var statCpuPrefix = []byte("cpu")
var statCpuPrefixLen = len(statCpuPrefix)

var statReadFileBufPool = ReadFileBufPool256k

func StatPath(procfsRoot string) string {
	return path.Join(procfsRoot, "stat")
}

func NewStat(procfsRoot string) *Stat {
	return &Stat{
		Cpu:           make(map[int][]uint64),
		NumericFields: make([]uint64, STAT_NUMERIC_NUM_STATS),
		path:          StatPath(procfsRoot),
	}
}

func (stat *Stat) Clone(full bool) *Stat {
	newStat := &Stat{
		Cpu:           make(map[int][]uint64),
		NumericFields: make([]uint64, STAT_NUMERIC_NUM_STATS),
		scanNum:       stat.scanNum,
		path:          stat.path,
	}
	for cpu, cpuStats := range stat.Cpu {
		newStat.Cpu[cpu] = make([]uint64, STAT_CPU_STATS_SZ)
		if full {
			copy(newStat.Cpu[cpu], cpuStats)
		} else {
			newStat.Cpu[cpu][STAT_CPU_SCAN_NUMBER] = cpuStats[STAT_CPU_SCAN_NUMBER]
		}
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

	var (
		// Where to store the data parsed from a line, it will be updated w/ the
		// proper reference depending upon the line prefix:
		Values []uint64

		// The minimum number of fields, prefix dependent:
		minNumValues int

		// The col# -> index list:
		indexList []int
	)

	scanNum := stat.scanNum + 1

	pos, lineNum, eol := 0, 0, true
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

		// Parse prefix; during the scan assess whether it is "cpu", "cpuNN" or
		// something else:
		for ; pos < l && isWhitespace[buf[pos]]; pos++ {
		}
		prefixStartPos, prefixIndex, cpuNum := pos, 0, STAT_CPU_ALL

		for done := false; !eol && pos < l && !done; pos++ {
			c := buf[pos]
			if eol = (c == '\n'); eol || isWhitespace[c] {
				done = true
				continue
			}
			if prefixIndex < statCpuPrefixLen {
				if c != statCpuPrefix[prefixIndex] {
					cpuNum = STAT_NO_CPU_PREFIX
				}
			} else if cpuNum != STAT_NO_CPU_PREFIX {
				if cpuNum == STAT_CPU_ALL {
					cpuNum = 0
				}
				if digit := c - '0'; digit < 10 {
					cpuNum = (cpuNum << 3) + (cpuNum << 1) + int(digit)
				} else {
					return fmt.Errorf(
						"%s:%d: %q: cpu# `%c': invalid digit",
						stat.path, lineNum, getCurrentLine(buf, lineStartPos), c,
					)

				}
			}
			prefixIndex++
		}
		if prefixIndex == 0 {
			// Ignore empty lines, there shouldn't be any, though:
			continue
		}

		if cpuNum != STAT_NO_CPU_PREFIX {
			// It is a cpu prefix:
			indexList = statPrefixToDstIndexList["cpu"]
			Values = stat.Cpu[cpuNum]
			if Values == nil {
				Values = make([]uint64, STAT_CPU_STATS_SZ)
				stat.Cpu[cpuNum] = Values
			}
			minNumValues = STAT_CPU_MIN_NUM_FIELDS
			Values[STAT_CPU_SCAN_NUMBER] = scanNum
		} else {
			indexList = statPrefixToDstIndexList[string(buf[prefixStartPos:prefixStartPos+prefixIndex])]
			minNumValues = len(indexList)
			Values = stat.NumericFields
		}

		// Parse the fields into numeric values, stored to the right index
		// of Values:
		fieldIndex, numValues := 0, len(indexList)
		for !eol && pos < l && fieldIndex < numValues {
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
						"%s:%d: %q: `%c': invalid digit",
						stat.path, lineNum, getCurrentLine(buf, pos), c,
					)
				}
			}
			// If a value was found, assign it to the right index:
			if hasVal {
				Values[indexList[fieldIndex]] = val
				fieldIndex++
			}
		}

		// Verify that enough values were parsed:
		if fieldIndex < minNumValues {
			return fmt.Errorf(
				"%s:%d: %q: invalid value count: want (at least) %d, got: %d",
				stat.path, lineNum, getCurrentLine(buf, lineStartPos), minNumValues, fieldIndex,
			)
		}
	}

	// Remove stats for CPUs no longer found at this scan; update the scan#:
	for cpu, cpuStats := range stat.Cpu {
		if cpuStats[STAT_CPU_SCAN_NUMBER] != scanNum {
			delete(stat.Cpu, cpu)
		}
	}
	stat.scanNum = scanNum

	return nil
}
