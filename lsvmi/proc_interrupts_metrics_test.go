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

// Mirror ProcInterruptsMetricsIrqData for test purposes with a structure that
// can be JSON deserialized:
type ProcInterruptsMetricsIrqDataTest struct {
	CycleNum          int
	DeltaMetricPrefix string
	InfoMetric        string
	ZeroDelta         []bool
}

type ProcInterruptsMetricsTestCase struct {
	Name                                  string
	Instance                              string
	Hostname                              string
	CrtProcInterrupts, PrevProcInterrupts *procfs.Interrupts
	CrtPromTs, PrevPromTs                 int64
	FullMetricsFactor                     int
	IrqDataCache                          map[string]*ProcInterruptsMetricsIrqDataTest
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
	procInterruptsMetrics.fullMetricsFactor = tc.FullMetricsFactor
	procInterruptsMetrics.irqDataCache = make(map[string]*ProcInterruptsMetricsIrqData)
	for irq, irqDataTest := range tc.IrqDataCache {
		procInterruptsMetrics.irqDataCache[irq] = &ProcInterruptsMetricsIrqData{
			cycleNum:          irqDataTest.CycleNum,
			deltaMetricPrefix: []byte(irqDataTest.DeltaMetricPrefix),
			infoMetric:        []byte(irqDataTest.InfoMetric),
			zeroDelta:         make([]bool, len(irqDataTest.ZeroDelta)),
		}
		copy(procInterruptsMetrics.irqDataCache[irq].zeroDelta, irqDataTest.ZeroDelta)
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
		for irq, wantZeroDelta := range tc.WantZeroDeltaMap {
			irqData := procInterruptsMetrics.irqDataCache[irq]
			if irqData == nil {
				fmt.Fprintf(errBuf, "\nZeroDeltaMap: missing IRQ %q", irq)
				continue
			}
			gotZeroDelta := irqData.zeroDelta
			for index, wantVal := range wantZeroDelta {
				gotVal := gotZeroDelta[index]
				if wantVal != gotVal {
					fmt.Fprintf(
						errBuf,
						"\nZeroDeltaMap[%q][%d]: want: %v, got: %v",
						irq, index, wantVal, gotVal,
					)
				}
			}
		}
		for irq := range procInterruptsMetrics.irqDataCache {
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
