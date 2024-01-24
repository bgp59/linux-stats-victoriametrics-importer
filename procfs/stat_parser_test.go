package procfs

import (
	"bytes"
	"fmt"
	"path"
	"strconv"

	"testing"
)

type StatTestCase struct {
	procfsRoot string
	primeStat  *Stat
	wantStat   *Stat
	wantError  error
}

var statTestdataDir = path.Join(PROCFS_TESTDATA_ROOT, "stat")

var statCpuStatsIndexNameMap = []string{
	"STAT_CPU_USER_TICKS",
	"STAT_CPU_NICE_TICKS",
	"STAT_CPU_SYSTEM_TICKS",
	"STAT_CPU_IDLE_TICKS",
	"STAT_CPU_IOWAIT_TICKS",
	"STAT_CPU_IRQ_TICKS",
	"STAT_CPU_SOFTIRQ_TICKS",
	"STAT_CPU_STEAL_TICKS",
	"STAT_CPU_GUEST_TICKS",
	"STAT_CPU_GUEST_NICE_TICKS",
}

var statNumericFieldsIndexNameMap = []string{
	"STAT_PAGE_IN",
	"STAT_PAGE_OUT",
	"STAT_SWAP_IN",
	"STAT_SWAP_OUT",
	"STAT_CTXT",
	"STAT_BTIME",
	"STAT_PROCESSES",
	"STAT_PROCS_RUNNING",
	"STAT_PROCS_BLOCKED",
}

func testStatParser(tc *StatTestCase, t *testing.T) {
	getCpuName := func(cpu int) string {
		if cpu == STAT_CPU_ALL {
			return "all"
		}
		return strconv.Itoa(cpu)
	}

	var stat *Stat
	if tc.primeStat != nil {
		stat = tc.primeStat.Clone(true)
		if stat.path == "" {
			stat.path = NewStat(tc.procfsRoot).path
		}
	} else {
		stat = NewStat(tc.procfsRoot)
	}

	err := stat.Parse()
	if tc.wantError != nil {
		if err == nil || tc.wantError.Error() != err.Error() {
			t.Fatalf("want: %v error, got: %v", tc.wantError, err)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}

	wantStat := tc.wantStat
	diffBuf := &bytes.Buffer{}
	for cpu, wantCpuStats := range wantStat.Cpu {
		gotCpuStats := stat.Cpu[cpu]
		if gotCpuStats == nil {
			fmt.Fprintf(
				diffBuf, "\nCpu: %s: missing cpu", getCpuName(cpu),
			)
			continue
		}
		for index, wantVal := range wantCpuStats {
			if index == STAT_CPU_SCAN_NUMBER {
				continue
			}
			gotVal := gotCpuStats[index]
			if gotVal != wantVal {
				fmt.Fprintf(
					diffBuf,
					"\nCpu[%s][%s]: want: %d, got: %d",
					getCpuName(cpu), statCpuStatsIndexNameMap[index],
					wantVal, gotVal,
				)
			}
		}
	}

	for cpu := range stat.Cpu {
		if wantStat.Cpu[cpu] == nil {
			fmt.Fprintf(
				diffBuf, "\nCpu: %s: unexpected cpu", getCpuName(cpu),
			)
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
	for i, tc := range []*StatTestCase{
		{
			procfsRoot: path.Join(statTestdataDir, "field_mapping"),
			wantStat: &Stat{
				Cpu: map[int][]uint64{
					STAT_CPU_ALL: {0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
					0:            {10, 11, 12, 13, 14, 15, 16, 17, 18, 19},
					1:            {20, 21, 22, 23, 24, 25, 26, 27, 28, 29},
				},
				NumericFields: []uint64{30, 31, 32, 33, 34, 35, 36, 37, 38},
			},
		},
		{
			procfsRoot: path.Join(statTestdataDir, "missing_cpu"),
			wantStat: &Stat{
				Cpu: map[int][]uint64{
					STAT_CPU_ALL: {0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
					0:            {10, 11, 12, 13, 14, 15, 16, 17, 18, 19},
					2:            {20, 21, 22, 23, 24, 25, 26, 27, 28, 29},
				},
				NumericFields: []uint64{30, 31, 32, 33, 34, 35, 36, 37, 38},
			},
		},
		{
			procfsRoot: path.Join(statTestdataDir, "missing_cpu"),
			primeStat: &Stat{
				Cpu: map[int][]uint64{
					STAT_CPU_ALL: {10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 11},
					0:            {101, 111, 112, 113, 114, 115, 116, 117, 118, 119, 11},
					1:            {20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 11},
				},
				NumericFields: []uint64{130, 131, 132, 133, 134, 135, 136, 137, 138},
				scanNum:       11,
			},
			wantStat: &Stat{
				Cpu: map[int][]uint64{
					STAT_CPU_ALL: {0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
					0:            {10, 11, 12, 13, 14, 15, 16, 17, 18, 19},
					2:            {20, 21, 22, 23, 24, 25, 26, 27, 28, 29},
				},
				NumericFields: []uint64{30, 31, 32, 33, 34, 35, 36, 37, 38},
			},
		},
	} {
		t.Run(
			fmt.Sprintf("tc=%d,procfsRoot=%s,primeStat=%v", i, tc.procfsRoot, tc.primeStat != nil),
			func(t *testing.T) { testStatParser(tc, t) },
		)
	}
}
