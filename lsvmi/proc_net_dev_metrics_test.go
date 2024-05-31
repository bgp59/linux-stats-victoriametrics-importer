// Tests for proc_net_dev_metrics.go

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

type ProcNetDevInfoTestData struct {
	CycleNum  int
	ZeroDelta []bool
}

type ProcNetDevMetricsTestCase struct {
	Name                           string
	Description                    string
	Instance                       string
	Hostname                       string
	CurrProcNetDev, PrevProcNetDev *procfs.NetDev
	CurrPromTs, PrevPromTs         int64
	FullMetricsFactor              int
	DevInfoMap                     map[string]*ProcNetDevInfoTestData
	WantMetricsCount               int
	WantMetrics                    []string
	ReportExtra                    bool
	WantZeroDeltaMap               map[string][]bool
}

var procNetDevMetricsTestCasesFile = path.Join(
	"..", testutils.LsvmiTestCasesSubdir,
	"proc_net_dev.json",
)

func testProcNetDevMetrics(tc *ProcNetDevMetricsTestCase, t *testing.T) {
	tlc := testutils.NewTestLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	t.Logf("Description: %s", tc.Description)

	procNetDevMetrics, err := NewProcNetDevMetrics(nil)
	if err != nil {
		t.Fatal(err)
	}
	procNetDevMetrics.instance = tc.Instance
	procNetDevMetrics.hostname = tc.Hostname
	currIndex := procNetDevMetrics.currIndex
	procNetDevMetrics.procNetDev[currIndex] = tc.CurrProcNetDev
	procNetDevMetrics.procNetDevTs[currIndex] = time.UnixMilli(tc.CurrPromTs)
	procNetDevMetrics.procNetDev[1-currIndex] = tc.PrevProcNetDev
	procNetDevMetrics.procNetDevTs[1-currIndex] = time.UnixMilli(tc.PrevPromTs)
	for dev, devInfo := range tc.DevInfoMap {
		procNetDevMetrics.updateDevInfo(dev)
		procNetDevMetrics.devInfoMap[dev].cycleNum = devInfo.CycleNum
		copy(procNetDevMetrics.devInfoMap[dev].zeroDelta, devInfo.ZeroDelta)
	}
	procNetDevMetrics.fullMetricsFactor = tc.FullMetricsFactor

	wantCurrIndex := 1 - currIndex
	testMetricsQueue := testutils.NewTestMetricsQueue(0)
	buf := testMetricsQueue.GetBuf()
	gotMetricsCount, _ := procNetDevMetrics.generateMetrics(buf)
	testMetricsQueue.QueueBuf(buf)

	errBuf := &bytes.Buffer{}

	gotCurrIndex := procNetDevMetrics.currIndex
	if wantCurrIndex != gotCurrIndex {
		fmt.Fprintf(
			errBuf,
			"\n.currIndex: want: %d, got: %d",
			wantCurrIndex, gotCurrIndex,
		)
	}

	if tc.WantZeroDeltaMap != nil {
		for dev, wantZeroDelta := range tc.WantZeroDeltaMap {
			devInfo := procNetDevMetrics.devInfoMap[dev]
			if devInfo == nil {
				fmt.Fprintf(errBuf, "\n.devInfo: missing dev %q", dev)
				continue
			}
			gotZeroDelta := devInfo.zeroDelta
			if len(wantZeroDelta) != len(gotZeroDelta) {
				fmt.Fprintf(
					errBuf,
					"\n.devInfo[%q].zeroDelta len: want: %d, got: %d",
					dev, len(wantZeroDelta), len(gotZeroDelta),
				)
				continue
			}
			for index, wantVal := range wantZeroDelta {
				gotVal := gotZeroDelta[index]
				if wantVal != gotVal {
					fmt.Fprintf(
						errBuf,
						"\n.devInfo[%q].zeroDelta[%d]: want: %v, got: %v",
						dev, index, wantVal, gotVal,
					)
				}
			}
		}
		for dev := range procNetDevMetrics.devInfoMap {
			if tc.WantZeroDeltaMap[dev] == nil {
				fmt.Fprintf(errBuf, "\n.devInfoMap: unexpected dev %q", dev)
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

func TestProcNetDevMetrics(t *testing.T) {
	t.Logf("Loading test cases from %q ...", procNetDevMetricsTestCasesFile)
	testCases := make([]*ProcNetDevMetricsTestCase, 0)
	err := testutils.LoadJsonFile(procNetDevMetricsTestCasesFile, &testCases)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range testCases {
		t.Run(
			tc.Name,
			func(t *testing.T) { testProcNetDevMetrics(tc, t) },
		)
	}
}
