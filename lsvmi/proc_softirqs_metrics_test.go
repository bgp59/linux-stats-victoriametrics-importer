// Tests for proc_softirqs_metrics.go

package lsvmi

import (
	"bytes"
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/bgp59/linux-stats-victoriametrics-importer/internal/testutils"
	"github.com/bgp59/linux-stats-victoriametrics-importer/procfs"
)

type ProcSoftirqsUpdateCpuListTestCase struct {
	currCpuList                   []int
	currNumCounters               int
	prevCpuList                   []int
	prevNumCounters               int
	wantCurrToPrevCounterIndexMap map[int]int
}

// Mirror ProcSoftirqsMetricsIrqData for test purposes with a structure that
// can be JSON deserialized:
type ProcSoftirqsMetricsIrqDataTest struct {
	CycleNum          int
	DeltaMetricPrefix string
	InfoMetric        string
	ZeroDelta         []bool
}

type ProcSoftirqsMetricsTestCase struct {
	Name                               string
	Description                        string
	Instance                           string
	Hostname                           string
	CurrProcSoftirqs, PrevProcSoftirqs *procfs.Softirqs
	CurrPromTs, PrevPromTs             int64
	FullMetricsFactor                  int
	IrqDataCache                       map[string]*ProcSoftirqsMetricsIrqDataTest
	WantMetricsCount                   int
	WantMetrics                        []string
	ReportExtra                        bool
	WantZeroDeltaMap                   map[string][]bool
}

var procSoftirqsMetricsTestCasesFile = path.Join(
	"..", testutils.LsvmiTestCasesSubdir,
	"proc_softirqs.json",
)

func testProcSoftirqsUpdateCpuList(tc *ProcSoftirqsUpdateCpuListTestCase, t *testing.T) {
	tlc := testutils.NewTestLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	procSoftirqsMetrics, err := NewProcSoftirqsMetrics(nil)
	if err != nil {
		t.Fatal(err)
	}

	prevSoftirqs := procfs.NewSoftirqs("")
	currSoftirqs := prevSoftirqs.Clone(false)
	if tc.currCpuList != nil {
		currSoftirqs.CpuList = make([]int, len(tc.currCpuList))
		copy(currSoftirqs.CpuList, tc.currCpuList)
	}
	currSoftirqs.NumCounters = tc.currNumCounters
	if tc.prevCpuList != nil {
		prevSoftirqs.CpuList = make([]int, len(tc.prevCpuList))
		copy(prevSoftirqs.CpuList, tc.prevCpuList)
	}
	prevSoftirqs.NumCounters = tc.prevNumCounters

	procSoftirqsMetrics.procSoftirqs[procSoftirqsMetrics.currIndex] = currSoftirqs
	procSoftirqsMetrics.procSoftirqs[1-procSoftirqsMetrics.currIndex] = prevSoftirqs

	gotCurrToPrevCounterIndexMap := procSoftirqsMetrics.updateCpuList()

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

func TestProcSoftirqsUpdateCpuList(t *testing.T) {
	for _, tc := range []*ProcSoftirqsUpdateCpuListTestCase{
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
			func(t *testing.T) { testProcSoftirqsUpdateCpuList(tc, t) },
		)
	}
}

func testProcSoftirqsMetrics(tc *ProcSoftirqsMetricsTestCase, t *testing.T) {
	tlc := testutils.NewTestLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	t.Logf("Description: %s", tc.Description)

	procSoftirqsMetrics, err := NewProcSoftirqsMetrics(nil)
	if err != nil {
		t.Fatal(err)
	}
	procSoftirqsMetrics.instance = tc.Instance
	procSoftirqsMetrics.hostname = tc.Hostname
	currIndex := procSoftirqsMetrics.currIndex
	procSoftirqsMetrics.procSoftirqs[currIndex] = tc.CurrProcSoftirqs
	procSoftirqsMetrics.procSoftirqsTs[currIndex] = time.UnixMilli(tc.CurrPromTs)
	procSoftirqsMetrics.procSoftirqs[1-currIndex] = tc.PrevProcSoftirqs
	procSoftirqsMetrics.procSoftirqsTs[1-currIndex] = time.UnixMilli(tc.PrevPromTs)
	procSoftirqsMetrics.fullMetricsFactor = tc.FullMetricsFactor
	procSoftirqsMetrics.irqDataCache = make(map[string]*ProcSoftirqsMetricsIrqData)
	for irq, irqDataTest := range tc.IrqDataCache {
		procSoftirqsMetrics.irqDataCache[irq] = &ProcSoftirqsMetricsIrqData{
			cycleNum:          irqDataTest.CycleNum,
			deltaMetricPrefix: []byte(irqDataTest.DeltaMetricPrefix),
			infoMetric:        []byte(irqDataTest.InfoMetric),
			zeroDelta:         make([]bool, len(irqDataTest.ZeroDelta)),
		}
		copy(procSoftirqsMetrics.irqDataCache[irq].zeroDelta, irqDataTest.ZeroDelta)
	}

	wantCurrIndex := 1 - currIndex
	testMetricsQueue := testutils.NewTestMetricsQueue(0)
	buf := testMetricsQueue.GetBuf()
	gotMetricsCount, _ := procSoftirqsMetrics.generateMetrics(buf)
	testMetricsQueue.QueueBuf(buf)

	errBuf := &bytes.Buffer{}

	gotCurrIndex := procSoftirqsMetrics.currIndex
	if wantCurrIndex != gotCurrIndex {
		fmt.Fprintf(
			errBuf,
			"\ncurrIndex: want: %d, got: %d",
			wantCurrIndex, gotCurrIndex,
		)
	}

	if tc.WantZeroDeltaMap != nil {
		for irq, wantZeroDelta := range tc.WantZeroDeltaMap {
			irqData := procSoftirqsMetrics.irqDataCache[irq]
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
		for irq := range procSoftirqsMetrics.irqDataCache {
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

func TestProcSoftirqsMetrics(t *testing.T) {
	t.Logf("Loading test cases from %q ...", procSoftirqsMetricsTestCasesFile)
	testCases := make([]*ProcSoftirqsMetricsTestCase, 0)
	err := testutils.LoadJsonFile(procSoftirqsMetricsTestCasesFile, &testCases)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range testCases {
		t.Run(
			tc.Name,
			func(t *testing.T) { testProcSoftirqsMetrics(tc, t) },
		)
	}
}
