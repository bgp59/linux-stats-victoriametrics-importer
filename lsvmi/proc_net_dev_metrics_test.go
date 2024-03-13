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

type ProcNetDevMetricsTestCase struct {
	Name                          string
	Instance                      string
	Hostname                      string
	CrtProcNetDev, PrevProcNetDev *procfs.NetDev
	CrtPromTs, PrevPromTs         int64
	CycleNum, FullMetricsFactor   int
	ZeroDeltaMap                  map[string][]bool
	WantMetricsCount              int
	WantMetrics                   []string
	ReportExtra                   bool
	WantZeroDeltaMap              map[string][]bool
}

var procNetDevMetricsTestcasesFile = path.Join(
	"..", testutils.LsvmiTestcasesSubdir,
	"proc_net_dev.json",
)

func testProcNetDevMetrics(tc *ProcNetDevMetricsTestCase, t *testing.T) {
	tlc := testutils.NewTestLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	procNetDevMetrics, err := NewProcNetDevMetrics(nil)
	if err != nil {
		t.Fatal(err)
	}
	procNetDevMetrics.instance = tc.Instance
	procNetDevMetrics.hostname = tc.Hostname
	crtIndex := procNetDevMetrics.crtIndex
	procNetDevMetrics.procNetDev[crtIndex] = tc.CrtProcNetDev
	procNetDevMetrics.procNetDevTs[crtIndex] = time.UnixMilli(tc.CrtPromTs)
	procNetDevMetrics.procNetDev[1-crtIndex] = tc.PrevProcNetDev
	procNetDevMetrics.procNetDevTs[1-crtIndex] = time.UnixMilli(tc.PrevPromTs)
	procNetDevMetrics.cycleNum = tc.CycleNum
	procNetDevMetrics.fullMetricsFactor = tc.FullMetricsFactor
	for dev, zeroDeltaMap := range tc.ZeroDeltaMap {
		procNetDevMetrics.zeroDeltaMap[dev] = make([]bool, procfs.NET_DEV_NUM_STATS)
		copy(procNetDevMetrics.zeroDeltaMap[dev], zeroDeltaMap)
	}

	wantCrtIndex := 1 - crtIndex
	testMetricsQueue := testutils.NewTestMetricsQueue(0)
	buf := testMetricsQueue.GetBuf()
	gotMetricsCount := procNetDevMetrics.generateMetrics(buf)
	testMetricsQueue.QueueBuf(buf)

	errBuf := &bytes.Buffer{}

	gotCrtIndex := procNetDevMetrics.crtIndex
	if wantCrtIndex != gotCrtIndex {
		fmt.Fprintf(
			errBuf,
			"\ncrtIndex: want: %d, got: %d",
			wantCrtIndex, gotCrtIndex,
		)
	}

	if tc.WantZeroDeltaMap != nil {
		for dev, wantZeroDeltaMap := range tc.WantZeroDeltaMap {
			gotZeroDeltaMap := procNetDevMetrics.zeroDeltaMap[dev]
			if gotZeroDeltaMap == nil {
				fmt.Fprintf(errBuf, "\nZeroDeltaMap: missing dev %s", dev)
				continue
			}
			for index, wantVal := range wantZeroDeltaMap {
				gotVal := gotZeroDeltaMap[index]
				if wantVal != gotVal {
					fmt.Fprintf(
						errBuf,
						"\nZeroDeltaMap[%s][%d]: want: %v, got: %v",
						dev, index, wantVal, gotVal,
					)
				}
			}
		}
		for dev := range procNetDevMetrics.zeroDeltaMap {
			if tc.WantZeroDeltaMap[dev] == nil {
				fmt.Fprintf(errBuf, "\nZeroDeltaMap: unexpected dev %s", dev)
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
	t.Logf("Loading testcases from %q ...", procNetDevMetricsTestcasesFile)
	testcases := make([]*ProcNetDevMetricsTestCase, 0)
	err := testutils.LoadJsonFile(procNetDevMetricsTestcasesFile, &testcases)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range testcases {
		t.Run(
			tc.Name,
			func(t *testing.T) { testProcNetDevMetrics(tc, t) },
		)
	}
}
