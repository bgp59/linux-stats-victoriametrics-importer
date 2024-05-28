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

type ProcStatMetricsCpuInfoTestData struct {
	CycleNum int
	ZeroPcpu []bool
}

type ProcStatMetricsTestCase struct {
	Name                             string
	Description                      string
	Instance                         string
	Hostname                         string
	CurrProcStat, PrevProcStat       *procfs.Stat
	CurrPromTs, PrevPromTs           int64
	CpuInfo                          map[int]*ProcStatMetricsCpuInfoTestData
	OtherCycleNum, FullMetricsFactor int
	OtherZeroDelta                   []bool
	WantMetricsCount                 int
	WantMetrics                      []string
	ReportExtra                      bool
	WantZeroPcpuMap                  map[int][]bool
	WantOtherZeroDelta               []bool
	LinuxClktckSec                   float64
	TimeSinceBtime                   float64
}

var procStatMetricsTestcasesFile = path.Join(
	"..", testutils.LsvmiTestcasesSubdir,
	"proc_stat.json",
)

func testProcStatMetrics(tc *ProcStatMetricsTestCase, t *testing.T) {
	tlc := testutils.NewTestLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	t.Logf("Description: %s", tc.Description)

	procStatMetrics, err := NewProcStatMetrics(nil)
	if err != nil {
		t.Fatal(err)
	}
	procStatMetrics.instance = tc.Instance
	procStatMetrics.hostname = tc.Hostname
	currIndex := procStatMetrics.currIndex
	procStatMetrics.procStat[currIndex] = tc.CurrProcStat
	procStatMetrics.procStatTs[currIndex] = time.UnixMilli(tc.CurrPromTs)
	procStatMetrics.procStat[1-currIndex] = tc.PrevProcStat
	procStatMetrics.procStatTs[1-currIndex] = time.UnixMilli(tc.PrevPromTs)
	procStatMetrics.fullMetricsFactor = tc.FullMetricsFactor

	if tc.CpuInfo != nil {
		for cpu, cpuInfo := range tc.CpuInfo {
			procStatMetrics.updateCpuInfo(cpu)
			procStatMetrics.cpuInfo[cpu].cycleNum = cpuInfo.CycleNum
			if cpuInfo.ZeroPcpu != nil {
				copy(procStatMetrics.cpuInfo[cpu].zeroPcpu, cpuInfo.ZeroPcpu)
			}
		}
	}

	procStatMetrics.otherCycleNum = tc.OtherCycleNum
	if tc.OtherZeroDelta != nil {
		copy(procStatMetrics.otherZeroDelta, tc.OtherZeroDelta)
	}

	if tc.LinuxClktckSec > 0 {
		procStatMetrics.linuxClktckSec = tc.LinuxClktckSec
	}
	procStatMetrics.timeSinceFn = func(t time.Time) time.Duration {
		return time.Duration(tc.TimeSinceBtime * float64(time.Second))
	}

	wantCurrIndex := 1 - currIndex
	testMetricsQueue := testutils.NewTestMetricsQueue(0)
	buf := testMetricsQueue.GetBuf()
	gotMetricsCount, _ := procStatMetrics.generateMetrics(buf)
	testMetricsQueue.QueueBuf(buf)

	errBuf := &bytes.Buffer{}

	gotCurrIndex := procStatMetrics.currIndex
	if wantCurrIndex != gotCurrIndex {
		fmt.Fprintf(
			errBuf,
			"\ncurrIndex: want: %d, got: %d",
			wantCurrIndex, gotCurrIndex,
		)
	}

	if tc.WantZeroPcpuMap != nil {
		for cpu, wantZeroPcpu := range tc.WantZeroPcpuMap {
			gotZeroPcpu := procStatMetrics.cpuInfo[cpu].zeroPcpu
			if gotZeroPcpu == nil {
				fmt.Fprintf(errBuf, "\n.cpuInfo: missing cpu %d", cpu)
				continue
			}
			for index, wantVal := range wantZeroPcpu {
				gotVal := gotZeroPcpu[index]
				if wantVal != gotVal {
					fmt.Fprintf(
						errBuf,
						"\n.cpuInfo[%d].zeroPcpu[%d]: want: %v, got: %v",
						cpu, index, wantVal, gotVal,
					)
				}
			}
		}
		for cpu := range procStatMetrics.cpuInfo {
			if tc.WantZeroPcpuMap[cpu] == nil {
				fmt.Fprintf(errBuf, "\n.cpuInfo: unexpected cpu %d", cpu)
			}
		}
	}

	if tc.WantOtherZeroDelta != nil {
		for index, wantVal := range tc.WantOtherZeroDelta {
			gotVal := procStatMetrics.otherZeroDelta[index]
			if wantVal != gotVal {
				fmt.Fprintf(
					errBuf,
					"\n.otherZeroDelta[%d]: want: %v, got: %v",
					index, wantVal, gotVal,
				)
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
