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
	crtCpuList                   []int
	crtNumCounters               int
	prevCpuList                  []int
	prevNumCounters              int
	wantCrtToPrevCounterIndexMap map[int]int
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
	Name                                  string
	Description                           string
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

func testProcInterruptsUpdateCpuList(tc *ProcInterruptsUpdateCpuListTestCase, t *testing.T) {
	tlc := testutils.NewTestLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	procInterruptsMetrics, err := NewProcInterruptsMetrics(nil)
	if err != nil {
		t.Fatal(err)
	}

	prevInterrupts := procfs.NewInterrupts("")
	crtInterrupts := prevInterrupts.Clone(false)
	if tc.crtCpuList != nil {
		crtInterrupts.CpuList = make([]int, len(tc.crtCpuList))
		copy(crtInterrupts.CpuList, tc.crtCpuList)
	}
	crtInterrupts.NumCounters = tc.crtNumCounters
	if tc.prevCpuList != nil {
		prevInterrupts.CpuList = make([]int, len(tc.prevCpuList))
		copy(prevInterrupts.CpuList, tc.prevCpuList)
	}
	prevInterrupts.NumCounters = tc.prevNumCounters

	procInterruptsMetrics.procInterrupts[procInterruptsMetrics.crtIndex] = crtInterrupts
	procInterruptsMetrics.procInterrupts[1-procInterruptsMetrics.crtIndex] = prevInterrupts

	gotCrtToPrevCounterIndexMap := procInterruptsMetrics.updateCpuList()

	if len(tc.wantCrtToPrevCounterIndexMap) != len(gotCrtToPrevCounterIndexMap) {
		t.Fatalf(
			"len(crtToPrevCounterIndexMap): want: %d, got: %d",
			len(tc.wantCrtToPrevCounterIndexMap), len(gotCrtToPrevCounterIndexMap),
		)
	}

	errBuf := &bytes.Buffer{}
	for i, wantI := range tc.wantCrtToPrevCounterIndexMap {
		gotI, ok := gotCrtToPrevCounterIndexMap[i]
		if !ok {
			fmt.Fprintf(errBuf, "crtToPrevCounterIndexMap: missing %d", i)
			continue
		}
		if wantI != gotI {
			fmt.Fprintf(errBuf, "crtToPrevCounterIndexMap[%d]: want: %d, got: %d", i, wantI, gotI)
		}
	}

	for i, gotI := range gotCrtToPrevCounterIndexMap {
		if _, ok := tc.wantCrtToPrevCounterIndexMap[i]; !ok {
			fmt.Fprintf(errBuf, "unexpected crtToPrevCounterIndexMap[%d]=%d", i, gotI)
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
	gotMetricsCount, _ := procInterruptsMetrics.generateMetrics(buf)
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
