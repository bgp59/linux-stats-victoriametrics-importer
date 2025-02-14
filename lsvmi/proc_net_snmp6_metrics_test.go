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

type ProcNetSnmp6MetricsTestCase struct {
	Name                               string
	Description                        string
	Instance                           string
	Hostname                           string
	CurrProcNetSnmp6, PrevProcNetSnmp6 *procfs.NetSnmp6
	CurrPromTs, PrevPromTs             int64
	CycleNum                           []int
	FullMetricsFactor                  int
	ZeroDelta                          []bool
	WantMetricsCount                   int
	WantMetrics                        []string
	ReportExtra                        bool
	WantZeroDelta                      []bool
}

var procNetSnmp6MetricsTestCasesFile = path.Join(
	"..", testutils.LsvmiTestCasesSubdir,
	"proc_net_snmp6.json",
)

func testProcNetSnmp6Metrics(tc *ProcNetSnmp6MetricsTestCase, t *testing.T) {
	tlc := testutils.NewTestLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	t.Logf("Description: %s", tc.Description)

	procNetSnmp6Metrics, err := NewProcNetSnmp6Metrics(nil)
	if err != nil {
		t.Fatal(err)
	}
	procNetSnmp6Metrics.instance = tc.Instance
	procNetSnmp6Metrics.hostname = tc.Hostname
	currIndex := procNetSnmp6Metrics.currIndex
	procNetSnmp6Metrics.procNetSnmp6[currIndex] = tc.CurrProcNetSnmp6
	procNetSnmp6Metrics.procNetSnmp6Ts[currIndex] = time.UnixMilli(tc.CurrPromTs)
	procNetSnmp6Metrics.procNetSnmp6[1-currIndex] = tc.PrevProcNetSnmp6
	procNetSnmp6Metrics.procNetSnmp6Ts[1-currIndex] = time.UnixMilli(tc.PrevPromTs)
	if tc.CycleNum != nil {
		procNetSnmp6Metrics.cycleNum = make([]int, len(tc.CycleNum))
		copy(procNetSnmp6Metrics.cycleNum, tc.CycleNum)
	}
	if tc.ZeroDelta != nil {
		copy(procNetSnmp6Metrics.zeroDelta, tc.ZeroDelta)
	}
	procNetSnmp6Metrics.fullMetricsFactor = tc.FullMetricsFactor

	wantCurrIndex := 1 - currIndex
	testMetricsQueue := testutils.NewTestMetricsQueue(0)
	buf := testMetricsQueue.GetBuf()
	gotMetricsCount, _ := procNetSnmp6Metrics.generateMetrics(buf)
	testMetricsQueue.QueueBuf(buf)

	errBuf := &bytes.Buffer{}

	gotCurrIndex := procNetSnmp6Metrics.currIndex
	if wantCurrIndex != gotCurrIndex {
		fmt.Fprintf(
			errBuf,
			"\ncurrIndex: want: %d, got: %d",
			wantCurrIndex, gotCurrIndex,
		)
	}

	if tc.WantZeroDelta != nil {
		if len(tc.WantZeroDelta) != len(procNetSnmp6Metrics.zeroDelta) {
			fmt.Fprintf(
				errBuf,
				"\nlen(zeroDelta): want: %d, got: %d",
				len(tc.WantZeroDelta), len(procNetSnmp6Metrics.zeroDelta),
			)
		} else {
			for i, want := range tc.WantZeroDelta {
				got := procNetSnmp6Metrics.zeroDelta[i]
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

func TestProcNetSnmp6Metrics(t *testing.T) {
	t.Logf("Loading test cases from %q ...", procNetSnmp6MetricsTestCasesFile)
	testCases := make([]*ProcNetSnmp6MetricsTestCase, 0)
	err := testutils.LoadJsonFile(procNetSnmp6MetricsTestCasesFile, &testCases)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range testCases {
		t.Run(
			tc.Name,
			func(t *testing.T) { testProcNetSnmp6Metrics(tc, t) },
		)
	}
}
