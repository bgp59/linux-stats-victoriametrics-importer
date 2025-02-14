// Tests for scheduler internal metrics

package lsvmi

import (
	"bytes"
	"fmt"
	"path"
	"testing"

	"github.com/bgp59/linux-stats-victoriametrics-importer/internal/testutils"
)

type SchedulerInternalMetricsTestCase struct {
	InternalMetricsTestCase
	CurrStats, PrevStats SchedulerStats
}

var schedulerInternalMetricsTestCasesFile = path.Join(
	"..", testutils.LsvmiTestCasesSubdir,
	"internal_metrics", "scheduler.json",
)

func newTestSchedulerInternalMetrics(tc *SchedulerInternalMetricsTestCase) (*InternalMetrics, error) {
	internalMetrics, err := newTestInternalMetrics(&tc.InternalMetricsTestCase)
	if err != nil {
		return nil, err
	}
	scheduler, err := NewScheduler(nil)
	if err != nil {
		return nil, err
	}
	internalMetrics.scheduler = scheduler
	return internalMetrics, nil
}

func testSchedulerInternalMetrics(tc *SchedulerInternalMetricsTestCase, t *testing.T) {
	tlc := testutils.NewTestLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	t.Logf("Description: %s", tc.Description)

	internalMetrics, err := newTestSchedulerInternalMetrics(tc)
	if err != nil {
		t.Fatal(err)
	}
	schedulerInternalMetrics := internalMetrics.schedulerMetrics
	schedulerInternalMetrics.stats[schedulerInternalMetrics.currIndex] = tc.CurrStats
	schedulerInternalMetrics.stats[1-schedulerInternalMetrics.currIndex] = tc.PrevStats
	testMetricsQueue := testutils.NewTestMetricsQueue(0)

	wantCurrIndex := 1 - schedulerInternalMetrics.currIndex

	buf := testMetricsQueue.GetBuf()
	gotMetricsCount := schedulerInternalMetrics.generateMetrics(buf, nil)
	testMetricsQueue.QueueBuf(buf)

	errBuf := &bytes.Buffer{}

	gotCurrIndex := schedulerInternalMetrics.currIndex
	if wantCurrIndex != gotCurrIndex {
		fmt.Fprintf(
			errBuf,
			"\ncurrIndex: want: %d, got: %d",
			wantCurrIndex, gotCurrIndex,
		)
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

func TestSchedulerInternalMetrics(t *testing.T) {
	t.Logf("Loading test cases from %q ...", schedulerInternalMetricsTestCasesFile)
	testCases := make([]*SchedulerInternalMetricsTestCase, 0)
	err := testutils.LoadJsonFile(schedulerInternalMetricsTestCasesFile, &testCases)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range testCases {
		t.Run(
			tc.Name,
			func(t *testing.T) { testSchedulerInternalMetrics(tc, t) },
		)
	}
}
