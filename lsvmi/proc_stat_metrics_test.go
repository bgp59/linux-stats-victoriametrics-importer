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
	Instance                    string
	Hostname                    string
	CrtProcStat, PrevProcStat   *procfs.Stat
	CrtPromTs, PrevPromTs       int64
	CycleNum, FullMetricsFactor int
	ScaleCpuAll                 bool
	ZeroPcpuMap                 map[int][]bool
	WantMetricsCount            int
	WantMetrics                 []string
	ReportExtra                 bool
	WantZeroPcpuMap             map[int][]bool
	LinuxClktckSec              float64
	TimeSinceBtime              float64
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
	procStatMetrics.instance = tc.Instance
	procStatMetrics.hostname = tc.Hostname
	crtIndex := procStatMetrics.crtIndex
	procStatMetrics.procStat[crtIndex] = tc.CrtProcStat
	procStatMetrics.procStatTs[crtIndex] = time.UnixMilli(tc.CrtPromTs)
	procStatMetrics.procStat[1-crtIndex] = tc.PrevProcStat
	procStatMetrics.procStatTs[1-crtIndex] = time.UnixMilli(tc.PrevPromTs)
	procStatMetrics.cycleNum = tc.CycleNum
	procStatMetrics.fullMetricsFactor = tc.FullMetricsFactor
	procStatMetrics.scaleCpuAll = tc.ScaleCpuAll
	for cpu, ZeroPcpuMap := range tc.ZeroPcpuMap {
		procStatMetrics.zeroPcpuMap[cpu] = make([]bool, procfs.STAT_CPU_NUM_STATS)
		copy(procStatMetrics.zeroPcpuMap[cpu], ZeroPcpuMap)
	}
	if tc.LinuxClktckSec > 0 {
		procStatMetrics.linuxClktckSec = tc.LinuxClktckSec
	}
	procStatMetrics.timeSinceFn = func(t time.Time) time.Duration {
		return time.Duration(tc.TimeSinceBtime * float64(time.Second))
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

	if tc.WantZeroPcpuMap != nil {
		for cpu, wantZeroPcpuMap := range tc.WantZeroPcpuMap {
			gotZeroPcpuMap := procStatMetrics.zeroPcpuMap[cpu]
			if gotZeroPcpuMap == nil {
				fmt.Fprintf(errBuf, "\nZeroPcpuMap: missing cpu %d", cpu)
				continue
			}
			for index, wantVal := range wantZeroPcpuMap {
				gotVal := gotZeroPcpuMap[index]
				if wantVal != gotVal {
					fmt.Fprintf(
						errBuf,
						"\nZeroPcpuMap[%d][%d]: want: %v, got: %v",
						cpu, index, wantVal, gotVal,
					)
				}
			}
		}
		for cpu := range procStatMetrics.zeroPcpuMap {
			if tc.WantZeroPcpuMap[cpu] == nil {
				fmt.Fprintf(errBuf, "\nZeroPcpuMap: unexpected cpu %d", cpu)
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
