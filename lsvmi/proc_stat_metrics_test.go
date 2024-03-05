// Tests for proc_stat_metrics.go

package lsvmi

import (
	"bytes"
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/eparparita/linux-stats-victoriametrics-importer/internal/testutils"
	"github.com/eparparita/linux-stats-victoriametrics-importer/procfs"
)

type ProcStatMetricsTestCase struct {
	Name                        string
	CrtProcStat, PrevProcStat   *procfs.Stat
	CrtPromTs, PrevPromTs       int64
	CycleNum, FullMetricsFactor int
	ZeroPcpu                    map[int][]bool
	WantMetricsCount            int
	WantMetrics                 []string
	ReportExtra                 bool
	WantZeroPcpu                map[int][]bool
	LinuxClktckSec              float64
}

var procStatMetricsTestcasesFile = path.Join(
	"..", testutils.LsvmiTestcasesSubdir,
	"proc_stat.json",
)

func testProcStatMetrics(tc *ProcStatMetricsTestCase, t *testing.T) {
	procStatMetrics, err := NewProcStatMetrics(nil)
	if err != nil {
		t.Fatal(err)
	}
	crtIndex := procStatMetrics.crtIndex
	procStatMetrics.procStat[crtIndex] = tc.CrtProcStat
	procStatMetrics.procStatTs[crtIndex] = time.UnixMilli(tc.CrtPromTs)
	procStatMetrics.procStat[1-crtIndex] = tc.PrevProcStat
	procStatMetrics.procStatTs[1-crtIndex] = time.UnixMilli(tc.PrevPromTs)
	procStatMetrics.cycleNum = tc.CycleNum
	procStatMetrics.fullMetricsFactor = tc.FullMetricsFactor
	for cpu, zeroPcpu := range tc.ZeroPcpu {
		procStatMetrics.zeroPcpu[cpu] = make([]bool, procfs.STAT_CPU_NUM_STATS)
		copy(procStatMetrics.zeroPcpu[cpu], zeroPcpu)
	}
	if tc.LinuxClktckSec > 0 {
		procStatMetrics.linuxClktckSec = tc.LinuxClktckSec
	}

	wantCrtIndex := 1 - crtIndex
	testMetricsQueue := testutils.NewTestMetricsQueue(0)
	buf := testMetricsQueue.GetBuf()
	gotMetricsCount := procStatMetrics.generateMetrics(buf)
	testMetricsQueue.QueueBuf(buf)

	errBuf := &bytes.Buffer{}

	gotCrtIndex := procStatMetrics.crtIndex
	if wantCrtIndex != gotCrtIndex {
		fmt.Fprintf(
			errBuf,
			"\ncrtIndex: want: %d, got: %d",
			wantCrtIndex, gotCrtIndex,
		)
	}

	if tc.WantZeroPcpu != nil {
		for cpu, wantZeroPcpu := range tc.WantZeroPcpu {
			gotZeroPcpu := procStatMetrics.zeroPcpu[cpu]
			if gotZeroPcpu == nil {
				fmt.Fprintf(errBuf, "\nzeroPcpu: missing cpu %d", cpu)
				continue
			}
			for index, wantVal := range wantZeroPcpu {
				gotVal := gotZeroPcpu[index]
				if wantVal != gotVal {
					fmt.Fprintf(
						errBuf,
						"\nzeroPcpu[%d][%d]: want: %v, got: %v",
						cpu, index, wantVal, gotVal,
					)
				}
			}
		}
		for cpu := range procStatMetrics.zeroPcpu {
			if tc.WantZeroPcpu[cpu] == nil {
				fmt.Fprintf(errBuf, "\nzeroPcpu: unexpected cpu %d", cpu)
			}
		}
	}

	if tc.WantMetricsCount != gotMetricsCount {
		fmt.Fprintf(
			errBuf,
			"\nmetrics count: want: %d, got: %d",
			tc.WantMetricsCount, gotMetricsCount,
		)
	}

	testMetricsQueue.GenerateReport(tc.WantMetrics, tc.ReportExtra, errBuf)

	if errBuf.Len() > 0 {
		t.Fatal(errBuf)
	}
}

func TestProcStatMetrics(t *testing.T) {
	t.Logf("Loading testcases from %q ...", procStatMetricsTestcasesFile)
	testcases := make([]*ProcStatMetricsTestCase, 0)
	err := testutils.LoadJsonFile(procStatMetricsTestcasesFile, &testcases)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range testcases {
		t.Run(
			tc.Name,
			func(t *testing.T) { testProcStatMetrics(tc, t) },
		)
	}

}
