package procfs

import (
	"bytes"
	"fmt"
	"path"

	"testing"
)

var statTestdataDir = path.Join(TESTDATA_PROCFS_ROOT, "stat")

var statCpuStatsIndexNameMap = map[int]string{
	STAT_CPU_USER_TICKS:       "STAT_CPU_USER_TICKS",
	STAT_CPU_NICE_TICKS:       "STAT_CPU_NICE_TICKS",
	STAT_CPU_SYSTEM_TICKS:     "STAT_CPU_SYSTEM_TICKS",
	STAT_CPU_IDLE_TICKS:       "STAT_CPU_IDLE_TICKS",
	STAT_CPU_IOWAIT_TICKS:     "STAT_CPU_IOWAIT_TICKS",
	STAT_CPU_IRQ_TICKS:        "STAT_CPU_IRQ_TICKS",
	STAT_CPU_SOFTIRQ_TICKS:    "STAT_CPU_SOFTIRQ_TICKS",
	STAT_CPU_STEAL_TICKS:      "STAT_CPU_STEAL_TICKS",
	STAT_CPU_GUEST_TICKS:      "STAT_CPU_GUEST_TICKS",
	STAT_CPU_GUEST_NICE_TICKS: "STAT_CPU_GUEST_NICE_TICKS",
}

var statNumericFieldsIndexNameMap = map[int]string{
	STAT_PAGE_IN:       "STAT_PAGE_IN",
	STAT_PAGE_OUT:      "STAT_PAGE_OUT",
	STAT_SWAP_IN:       "STAT_SWAP_IN",
	STAT_SWAP_OUT:      "STAT_SWAP_OUT",
	STAT_CTXT:          "STAT_CTXT",
	STAT_BTIME:         "STAT_BTIME",
	STAT_PROCESSES:     "STAT_PROCESSES",
	STAT_PROCS_RUNNING: "STAT_PROCS_RUNNING",
	STAT_PROCS_BLOCKED: "STAT_PROCS_BLOCKED",
}

func testStatParser(wantStat *Stat, t *testing.T) {
	stat := NewStat(wantStat.path)

	err := stat.Parse()
	if err != nil {
		t.Fatal(err)
	}

	diffBuf := &bytes.Buffer{}

	for index, wantVal := range wantStat.CpuAll {
		gotVal := stat.CpuAll[index]
		if gotVal != wantVal {
			fmt.Fprintf(
				diffBuf,
				"\nCpuAll[%d (%s)]: want: %d, got: %d",
				index, statCpuStatsIndexNameMap[index],
				wantVal, gotVal,
			)
		}
	}

	cpuNumOk := true
	if stat.MaxCpuNum != wantStat.MaxCpuNum {
		cpuNumOk = false
		fmt.Fprintf(
			diffBuf,
			"\nMaxCpuNum: want: %d, got: %d",
			wantStat.MaxCpuNum, stat.MaxCpuNum,
		)
	}
	cpuPresentMaxChunkNum := wantStat.MaxCpuNum / 64 // use non binary optimized operation for clarity
	if cpuNumOk {
		if len(stat.CpuPresent) < cpuPresentMaxChunkNum+1 {
			fmt.Fprintf(
				diffBuf,
				"\nLen(CpuPresent): %d, < %d",
				len(stat.CpuPresent), cpuPresentMaxChunkNum+1,
			)
			cpuNumOk = false
		}
	}
	if cpuNumOk {
		for cpuNum := 0; cpuNum <= wantStat.MaxCpuNum; cpuNum++ {
			// use non binary optimized operation for clarity
			cpuPresentChunkNum, cpuMask := cpuNum/64, uint64(1<<(cpuNum%64))
			wantPresentBit := (wantStat.CpuPresent[cpuPresentChunkNum] & cpuMask) > 0
			gotPresentBit := (stat.CpuPresent[cpuPresentChunkNum] & cpuMask) > 0

			if wantPresentBit != gotPresentBit {
				fmt.Fprintf(
					diffBuf,
					"\nCpuPresent[%d]: want: %v, got: %v",
					cpuNum, wantPresentBit, gotPresentBit,
				)
				continue
			}

			wantCpu := wantStat.Cpu[cpuNum]
			gotCpu := stat.Cpu[cpuNum]
			for index, wantVal := range wantCpu {
				gotVal := gotCpu[index]
				if gotVal != wantVal {
					fmt.Fprintf(
						diffBuf,
						"\nCpu[%d][%d (%s)]: want: %d, got: %d",
						cpuNum, index, statCpuStatsIndexNameMap[index],
						wantVal, gotVal,
					)
				}
			}
		}
	}

	for index, wantVal := range wantStat.NumericFields {
		gotVal := stat.NumericFields[index]
		if gotVal != wantVal {
			fmt.Fprintf(
				diffBuf,
				"\nNumericFields[%s]: want: %d, got: %d",
				statNumericFieldsIndexNameMap[index],
				wantVal, gotVal,
			)
		}
	}
	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf.String())
	}
}

func TestStatParser(t *testing.T) {
	for _, wantStat := range []*Stat{
		{
			path:   path.Join(statTestdataDir, "field_mapping"),
			CpuAll: []uint64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			Cpu: [][]uint64{
				[]uint64{10, 11, 12, 13, 14, 15, 16, 17, 18, 19},
				[]uint64{20, 21, 22, 23, 24, 25, 26, 27, 28, 29},
			},
			MaxCpuNum: 1,
			CpuPresent: []uint64{
				0x00000003,
			},
			NumericFields: []uint64{30, 31, 32, 33, 34, 35, 36, 37, 38},
		},
	} {
		t.Run(
			fmt.Sprintf("procfsRoot=%s", wantStat.path),
			func(t *testing.T) { testStatParser(wantStat, t) },
		)
	}
}
