// Tests for proc_interrupts_metrics.go

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

type ProcInterruptsMetricsTestCase struct {
	Name                                  string
	Instance                              string
	Hostname                              string
	CrtProcInterrupts, PrevProcInterrupts *procfs.Interrupts
	CrtPromTs, PrevPromTs                 int64
	CycleNum, FullMetricsFactor           int
	ZeroDeltaMap                          map[string][]bool
	InfoMetricsCache                      map[string][]byte
	WantMetricsCount                      int
	WantMetrics                           []string
	ReportExtra                           bool
	WantZeroDeltaMap                      map[string][]bool
}

var procInterruptsMetricsTestcasesFile = path.Join(
	"..", testutils.LsvmiTestcasesSubdir,
	"proc_interrupts.json",
)

func testProcInterruptsMetrics(tc *ProcInterruptsMetricsTestCase, t *testing.T) {
	tlc := testutils.NewTestLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	procInterruptsMetrics, err := NewProcInterruptsMetrics(nil)
	if err != nil {
		t.Fatal(err)
	}
	procInterruptsMetrics.instance = tc.Instance
	procInterruptsMetrics.hostname = tc.Hostname
	crtIndex := procInterruptsMetrics.crtIndex
	procInterruptsMetrics.procInterrupts[crtIndex] = tc.CrtProcInterrupts
	procInterruptsMetrics.procInterruptsTs[crtIndex] = time.UnixMilli(tc.CrtPromTs)
	procInterruptsMetrics.procInterrupts[1-crtIndex] = tc.PrevProcInterrupts
	procInterruptsMetrics.procInterruptsTs[1-crtIndex] = time.UnixMilli(tc.PrevPromTs)
	procInterruptsMetrics.cycleNum = tc.CycleNum
	procInterruptsMetrics.fullMetricsFactor = tc.FullMetricsFactor
	for irq, zeroDeltaMap := range tc.ZeroDeltaMap {
		procInterruptsMetrics.zeroDeltaMap[irq] = make([]bool, procfs.NET_DEV_NUM_STATS)
		copy(procInterruptsMetrics.zeroDeltaMap[irq], zeroDeltaMap)
	}
	for irq, infoMetric := range tc.InfoMetricsCache {
		procInterruptsMetrics.infoMetricsCache[irq] = make([]byte, len(infoMetric))
		copy(procInterruptsMetrics.infoMetricsCache[irq], infoMetric)
	}

	wantCrtIndex := 1 - crtIndex
	testMetricsQueue := testutils.NewTestMetricsQueue(0)
	buf := testMetricsQueue.GetBuf()
	gotMetricsCount := procInterruptsMetrics.generateMetrics(buf)
	testMetricsQueue.QueueBuf(buf)

	errBuf := &bytes.Buffer{}

	gotCrtIndex := procInterruptsMetrics.crtIndex
	if wantCrtIndex != gotCrtIndex {
		fmt.Fprintf(
			errBuf,
			"\ncrtIndex: want: %d, got: %d",
			wantCrtIndex, gotCrtIndex,
		)
	}

	if tc.WantZeroDeltaMap != nil {
		for irq, wantZeroDeltaMap := range tc.WantZeroDeltaMap {
			gotZeroDeltaMap := procInterruptsMetrics.zeroDeltaMap[irq]
			if gotZeroDeltaMap == nil {
				fmt.Fprintf(errBuf, "\nZeroDeltaMap: missing IRQ %q", irq)
				continue
			}
			for index, wantVal := range wantZeroDeltaMap {
				gotVal := gotZeroDeltaMap[index]
				if wantVal != gotVal {
					fmt.Fprintf(
						errBuf,
						"\nZeroDeltaMap[%q][%d]: want: %v, got: %v",
						irq, index, wantVal, gotVal,
					)
				}
			}
		}
		for irq := range procInterruptsMetrics.zeroDeltaMap {
			if tc.WantZeroDeltaMap[irq] == nil {
				fmt.Fprintf(errBuf, "\nZeroDeltaMap: unexpected IRQ %q", irq)
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

func TestProcInterruptsMetrics(t *testing.T) {
	t.Logf("Loading testcases from %q ...", procInterruptsMetricsTestcasesFile)
	testcases := make([]*ProcInterruptsMetricsTestCase, 0)
	err := testutils.LoadJsonFile(procInterruptsMetricsTestcasesFile, &testcases)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range testcases {
		t.Run(
			tc.Name,
			func(t *testing.T) { testProcInterruptsMetrics(tc, t) },
		)
	}
}
