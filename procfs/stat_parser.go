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

var statLinePrefixSeparator = [256]byte{
	' ':  1,
	'\t': 1,
	// Include digits for cpuNN:
	'0': 1,
	'1': 1,
	'2': 1,
	'3': 1,
	'4': 1,
	'5': 1,
	'6': 1,
	'7': 1,
	'8': 1,
	'9': 1,
}

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
	file, err := os.Open(stat.path)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(file)

	for i := 0; i < len(stat.CpuPresent) && i <= (stat.MaxCpuNum>>6); i++ {
		stat.CpuPresent[i] = 0
	}

	for scanner.Scan() {
		lineBytes := scanner.Bytes()
		sepIndex := -1
		for i, b := range lineBytes {
			if statLinePrefixSeparator[b] > 0 {
				sepIndex = i
				break
			}
		}
		if sepIndex <= 0 {
			continue
		}
		prefix := string(lineBytes[:sepIndex])
		indexList := statPrefixToDstIndexList[prefix]
		if indexList == nil {
			continue
		}
		line := string(lineBytes)
		fields := strings.Fields(line)
		if prefix == "cpu" {
			var cpuStats []uint64
			if len(fields[0]) > 3 {
				cpuNum, err := strconv.Atoi(fields[0][3:])
				if err != nil {
					return fmt.Errorf("%s: %s: %v for cpu#", stat.path, line, err)
				}
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
			for i, numericFieldsIndex := range indexList {
				stat.NumericFields[numericFieldsIndex], err = strconv.ParseUint(fields[i+1], 10, 64)
				if err != nil {
					return fmt.Errorf("%s: %s: %v", stat.path, line, err)
				}
			}
		}
	}

	err = scanner.Err()
	if err != nil {
		return fmt.Errorf("%s: %v", stat.path, err)
	}
	return nil
}
