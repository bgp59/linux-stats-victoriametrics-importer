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

type ProcInterruptsUpdateCpuListTestCase struct {
	currCpuList                   []int
	currNumCounters               int
	prevCpuList                   []int
	prevNumCounters               int
	wantCurrToPrevCounterIndexMap map[int]int
}

// Mirror ProcInterruptsMetricsIrqData for test purposes with a structure that
// can be JSON deserialized:
type ProcInterruptsMetricsIrqDataTest struct {
	CycleNum          int
	DeltaMetricPrefix string
	InfoMetric        string
	ZeroDelta         []bool
}

type ProcInterruptsMetricsTestCase struct {
	Name                                   string
	Description                            string
	Instance                               string
	Hostname                               string
	CurrProcInterrupts, PrevProcInterrupts *procfs.Interrupts
	CurrPromTs, PrevPromTs                 int64
	FullMetricsFactor                      int
	IrqDataCache                           map[string]*ProcInterruptsMetricsIrqDataTest
	WantMetricsCount                       int
	WantMetrics                            []string
	ReportExtra                            bool
	WantZeroDeltaMap                       map[string][]bool
}

var procInterruptsMetricsTestCasesFile = path.Join(
	"..", testutils.LsvmiTestCasesSubdir,
	"proc_interrupts.json",
)

func testProcInterruptsUpdateCpuList(tc *ProcInterruptsUpdateCpuListTestCase, t *testing.T) {
	tlc := testutils.NewTestLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	procInterruptsMetrics, err := NewProcInterruptsMetrics(nil)
	if err != nil {
		t.Fatal(err)
	}

	prevInterrupts := procfs.NewInterrupts("")
	currInterrupts := prevInterrupts.Clone(false)
	if tc.currCpuList != nil {
		currInterrupts.CpuList = make([]int, len(tc.currCpuList))
		copy(currInterrupts.CpuList, tc.currCpuList)
	}
	currInterrupts.NumCounters = tc.currNumCounters
	if tc.prevCpuList != nil {
		prevInterrupts.CpuList = make([]int, len(tc.prevCpuList))
		copy(prevInterrupts.CpuList, tc.prevCpuList)
	}
	prevInterrupts.NumCounters = tc.prevNumCounters

	procInterruptsMetrics.procInterrupts[procInterruptsMetrics.currIndex] = currInterrupts
	procInterruptsMetrics.procInterrupts[1-procInterruptsMetrics.currIndex] = prevInterrupts

	gotCurrToPrevCounterIndexMap := procInterruptsMetrics.updateCpuList()

	if len(tc.wantCurrToPrevCounterIndexMap) != len(gotCurrToPrevCounterIndexMap) {
		t.Fatalf(
			"len(currToPrevCounterIndexMap): want: %d, got: %d",
			len(tc.wantCurrToPrevCounterIndexMap), len(gotCurrToPrevCounterIndexMap),
		)
	}

	errBuf := &bytes.Buffer{}
	for i, wantI := range tc.wantCurrToPrevCounterIndexMap {
		gotI, ok := gotCurrToPrevCounterIndexMap[i]
		if !ok {
			fmt.Fprintf(errBuf, "currToPrevCounterIndexMap: missing %d", i)
			continue
		}
		if wantI != gotI {
			fmt.Fprintf(errBuf, "currToPrevCounterIndexMap[%d]: want: %d, got: %d", i, wantI, gotI)
		}
	}

	for i, gotI := range gotCurrToPrevCounterIndexMap {
		if _, ok := tc.wantCurrToPrevCounterIndexMap[i]; !ok {
			fmt.Fprintf(errBuf, "unexpected currToPrevCounterIndexMap[%d]=%d", i, gotI)
		}
	}

	if errBuf.Len() > 0 {
		t.Fatal(errBuf)
	}
}

func TestProcInterruptsUpdateCpuList(t *testing.T) {
	for _, tc := range []*ProcInterruptsUpdateCpuListTestCase{
		{
			[]int{1, 2, 3}, 3,
			nil, 4,
			map[int]int{0: 1, 1: 2, 2: 3},
		},
		{
			[]int{0, 2, 3}, 3,
			nil, 4,
			map[int]int{0: 0, 1: 2, 2: 3},
		},
		{
			[]int{2}, 1,
			nil, 4,
			map[int]int{0: 2},
		},
		{
			nil, 4,
			[]int{1, 2, 3}, 3,
			map[int]int{1: 0, 2: 1, 3: 2},
		},
		{
			nil, 4,
			nil, 4,
			map[int]int{0: 0, 1: 1, 2: 2, 3: 3},
		},
		{
			[]int{0, 1, 2}, 3,
			[]int{1, 2, 3}, 3,
			map[int]int{1: 0, 2: 1},
		},
	} {
		t.Run(
			"",
			func(t *testing.T) { testProcInterruptsUpdateCpuList(tc, t) },
		)
	}
}

func testProcInterruptsMetrics(tc *ProcInterruptsMetricsTestCase, t *testing.T) {
	tlc := testutils.NewTestLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	t.Logf("Description: %s", tc.Description)

	procInterruptsMetrics, err := NewProcInterruptsMetrics(nil)
	if err != nil {
		t.Fatal(err)
	}
	procInterruptsMetrics.instance = tc.Instance
	procInterruptsMetrics.hostname = tc.Hostname
	currIndex := procInterruptsMetrics.currIndex
	procInterruptsMetrics.procInterrupts[currIndex] = tc.CurrProcInterrupts
	procInterruptsMetrics.procInterruptsTs[currIndex] = time.UnixMilli(tc.CurrPromTs)
	procInterruptsMetrics.procInterrupts[1-currIndex] = tc.PrevProcInterrupts
	procInterruptsMetrics.procInterruptsTs[1-currIndex] = time.UnixMilli(tc.PrevPromTs)
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

	wantCurrIndex := 1 - currIndex
	testMetricsQueue := testutils.NewTestMetricsQueue(0)
	buf := testMetricsQueue.GetBuf()
	gotMetricsCount, _ := procInterruptsMetrics.generateMetrics(buf)
	testMetricsQueue.QueueBuf(buf)

	errBuf := &bytes.Buffer{}

	gotCurrIndex := procInterruptsMetrics.currIndex
	if wantCurrIndex != gotCurrIndex {
		fmt.Fprintf(
			errBuf,
			"\ncurrIndex: want: %d, got: %d",
			wantCurrIndex, gotCurrIndex,
		)
	}

	if tc.WantZeroDeltaMap != nil {
		for irq, wantZeroDelta := range tc.WantZeroDeltaMap {
			irqData := procInterruptsMetrics.irqDataCache[irq]
			if irqData == nil {
				fmt.Fprintf(errBuf, "\nZeroDelta: missing IRQ %q", irq)
				continue
			}
			gotZeroDelta := irqData.zeroDelta
			for index, wantVal := range wantZeroDelta {
				gotVal := gotZeroDelta[index]
				if wantVal != gotVal {
					fmt.Fprintf(
						errBuf,
						"\nZeroDelta[%q][%d]: want: %v, got: %v",
						irq, index, wantVal, gotVal,
					)
				}
			}
		}
		for irq := range procInterruptsMetrics.irqDataCache {
			if tc.WantZeroDeltaMap[irq] == nil {
				fmt.Fprintf(errBuf, "\nZeroDelta: unexpected IRQ %q", irq)
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
	t.Logf("Loading test cases from %q ...", procInterruptsMetricsTestCasesFile)
	testCases := make([]*ProcInterruptsMetricsTestCase, 0)
	err := testutils.LoadJsonFile(procInterruptsMetricsTestCasesFile, &testCases)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range testCases {
		t.Run(
			tc.Name,
			func(t *testing.T) { testProcInterruptsMetrics(tc, t) },
		)
	}
}
