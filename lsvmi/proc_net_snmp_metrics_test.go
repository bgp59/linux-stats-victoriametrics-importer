package lsvmi

import (
	"bytes"
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/emypar/linux-stats-victoriametrics-importer/internal/testutils"
	"github.com/emypar/linux-stats-victoriametrics-importer/procfs"
)

type ProcNetSnmpMetricsTestCase struct {
	Name                             string
	Description                      string
	Instance                         string
	Hostname                         string
	CurrProcNetSnmp, PrevProcNetSnmp *procfs.NetSnmp
	CurrPromTs, PrevPromTs           int64
	CycleNum                         []int
	FullMetricsFactor                int
	ZeroDelta                        []bool
	WantMetricsCount                 int
	WantMetrics                      []string
	ReportExtra                      bool
	WantZeroDelta                    []bool
}

var procNetSnmpMetricsTestCasesFile = path.Join(
	"..", testutils.LsvmiTestCasesSubdir,
	"proc_net_snmp.json",
)

func testProcNetSnmpMetrics(tc *ProcNetSnmpMetricsTestCase, t *testing.T) {
	tlc := testutils.NewTestLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	t.Logf("Description: %s", tc.Description)

	procNetSnmpMetrics, err := NewProcNetSnmpMetrics(nil)
	if err != nil {
		t.Fatal(err)
	}
	procNetSnmpMetrics.instance = tc.Instance
	procNetSnmpMetrics.hostname = tc.Hostname
	currIndex := procNetSnmpMetrics.currIndex
	procNetSnmpMetrics.procNetSnmp[currIndex] = tc.CurrProcNetSnmp
	procNetSnmpMetrics.procNetSnmpTs[currIndex] = time.UnixMilli(tc.CurrPromTs)
	procNetSnmpMetrics.procNetSnmp[1-currIndex] = tc.PrevProcNetSnmp
	procNetSnmpMetrics.procNetSnmpTs[1-currIndex] = time.UnixMilli(tc.PrevPromTs)
	if tc.CycleNum != nil {
		procNetSnmpMetrics.cycleNum = make([]int, len(tc.CycleNum))
		copy(procNetSnmpMetrics.cycleNum, tc.CycleNum)
	}
	if tc.ZeroDelta != nil {
		copy(procNetSnmpMetrics.zeroDelta, tc.ZeroDelta)
	}
	procNetSnmpMetrics.fullMetricsFactor = tc.FullMetricsFactor

	wantCurrIndex := 1 - currIndex
	testMetricsQueue := testutils.NewTestMetricsQueue(0)
	buf := testMetricsQueue.GetBuf()
	gotMetricsCount, _ := procNetSnmpMetrics.generateMetrics(buf)
	testMetricsQueue.QueueBuf(buf)

	errBuf := &bytes.Buffer{}

	gotCurrIndex := procNetSnmpMetrics.currIndex
	if wantCurrIndex != gotCurrIndex {
		fmt.Fprintf(
			errBuf,
			"\ncurrIndex: want: %d, got: %d",
			wantCurrIndex, gotCurrIndex,
		)
	}

	if tc.WantZeroDelta != nil {
		if len(tc.WantZeroDelta) != len(procNetSnmpMetrics.zeroDelta) {
			fmt.Fprintf(
				errBuf,
				"\nlen(zeroDelta): want: %d, got: %d",
				len(tc.WantZeroDelta), len(procNetSnmpMetrics.zeroDelta),
			)
		} else {
			for i, want := range tc.WantZeroDelta {
				got := procNetSnmpMetrics.zeroDelta[i]
				if want != got {
					fmt.Fprintf(
						errBuf,
						"\nzeroDelta[%d]: want: %v, got: %v",
						i, want, got,
					)
				}
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

func TestProcNetSnmpMetrics(t *testing.T) {
	t.Logf("Loading test cases from %q ...", procNetSnmpMetricsTestCasesFile)
	testCases := make([]*ProcNetSnmpMetricsTestCase, 0)
	err := testutils.LoadJsonFile(procNetSnmpMetricsTestCasesFile, &testCases)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range testCases {
		t.Run(
			tc.Name,
			func(t *testing.T) { testProcNetSnmpMetrics(tc, t) },
		)
	}
}
