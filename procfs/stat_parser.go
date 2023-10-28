// parser for /proc/stat

// Note: interrupts, hard and soft, are ignored from this source since more
// detailed information will be gleaned from interrupts and softirq files
// respectively.

package procfs

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
)

type Stat struct {
	// The all CPU info:
	CpuAll []uint64
	// The per CPU info is stored in a list of lists, the 1st indexList being CPU#
	// (0..MAX_CPU-1) and the second the CPU statistic.
	Cpu [][]uint64
	// Some CPU's might be disabled and their stat may not be available;
	// maintain a bitmap for CPU# found in the current scan:
	CpuPresent []uint64
	// The max CPU# ever found:
	MaxCpuNum int
	// Any other info is a scalar in a list:
	NumericFields []uint64
	// The path file to read:
	path string
}

// Indexes for cpu[] lists:
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

type StatLinePrefixHandling struct {
	prefix    string
	prefixLen int
	// Index list mapping data field# into either Cpu* or NumericFields:
	indexList []int
}

var statPrefixHandling = []*StatLinePrefixHandling{
	{
		prefix: "cpu",
		indexList: []int{
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
	},
	{
		prefix: "page",
		indexList: []int{
			STAT_PAGE_IN,
			STAT_PAGE_OUT,
		},
	},
	{
		prefix: "swap",
		indexList: []int{
			STAT_SWAP_IN,
			STAT_SWAP_OUT,
		},
	},
	{
		prefix: "ctxt",
		indexList: []int{
			STAT_CTXT,
		},
	},
	{
		prefix: "btime",
		indexList: []int{
			STAT_BTIME,
		},
	},
	{
		prefix: "processes",
		indexList: []int{
			STAT_PROCESSES,
		},
	},
	{
		prefix: "procs_running",
		indexList: []int{
			STAT_PROCS_RUNNING,
		},
	},
	{
		prefix: "procs_blocked",
		indexList: []int{
			STAT_PROCS_BLOCKED,
		},
	},
}

func init() {
	for _, prefixHandling := range statPrefixHandling {
		prefixHandling.prefixLen = len(prefixHandling.prefix)
	}
}

func NewStat(procfsRoot string) *Stat {
	return &Stat{
		CpuAll:        make([]uint64, STAT_CPU_STATS_COUNT),
		Cpu:           make([][]uint64, 0),
		CpuPresent:    make([]uint64, 16),
		NumericFields: make([]uint64, STAT_NUMERIC_FIELDS_COUNT),
		path:          path.Join(procfsRoot, "stat"),
	}
}

func (stat *Stat) Parse() error {
	file, err := os.Open(stat.path)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(file)

	for i := 0; i <= stat.MaxCpuNum; i++ {
		stat.CpuPresent[i] = 0
	}

	for scanner.Scan() {
		line := scanner.Text()
		lineLen := len(line)

		for _, prefixHandling := range statPrefixHandling {
			prefix, prefixLen := prefixHandling.prefix, prefixHandling.prefixLen
			if lineLen >= prefixLen && line[:prefixLen] == prefix {
				fields := strings.Fields(line)
				indexList := prefixHandling.indexList
				if prefix == "cpu" {
					var cpuStats []uint64
					if len(prefix) > 3 {
						cpuNum, err := strconv.Atoi(prefix[3:])
						if err != nil {
							return fmt.Errorf("%s: %s: %v for cpu#", stat.path, line, err)
						}
						if cpuNum >= stat.MaxCpuNum {
							stat.MaxCpuNum = cpuNum
						}
						cpuPresentChunk := cpuNum >> 6 // / 64
						if cpuPresentChunk >= len(stat.CpuPresent) {
							newCpuPresent := make([]uint64, cpuPresentChunk+1)
							copy(newCpuPresent, stat.CpuPresent)
							stat.CpuPresent = newCpuPresent
						}
						stat.CpuPresent[cpuPresentChunk] |= (1 << (cpuNum & ((1 << 6) - 1)))
						if cpuNum >= len(stat.Cpu) {
							newCpu := make([][]uint64, cpuNum+1)
							copy(newCpu, stat.Cpu)
						}
						cpuStats = stat.Cpu[cpuNum]
						if cpuStats == nil {
							cpuStats = make([]uint64, STAT_CPU_STATS_COUNT)
							stat.Cpu[cpuNum] = cpuStats
						}
					} else {
						cpuStats = stat.CpuAll
					}
					index := 0
					for ; index < len(fields)-1 && index < len(indexList); index++ {
						cpuStats[index], err = strconv.ParseUint(fields[index+1], 10, 64)
						if err != nil {
							return fmt.Errorf("%s: %s: %v", stat.path, line, err)
						}
					}
					if index <= STAT_CPU_SOFTIRQ_TICKS {
						return fmt.Errorf("%s: %s: invalid value count (< %d)", stat.path, line, STAT_CPU_SOFTIRQ_TICKS)
					}
				} else {
					if len(fields)-1 != len(indexList) {
						return fmt.Errorf("%s: %s: invalid value count (!= %d)", stat.path, line, len(indexList))
					}
					for _, index := range indexList {
						stat.NumericFields[index], err = strconv.ParseUint(fields[index+1], 10, 64)
						if err != nil {
							return fmt.Errorf("%s: %s: %v", stat.path, line, err)
						}
					}
				}
				continue
			}
		}
	}

	err = scanner.Err()
	if err != nil {
		return fmt.Errorf("%s: %v", stat.path, err)
	}
	return nil
}
