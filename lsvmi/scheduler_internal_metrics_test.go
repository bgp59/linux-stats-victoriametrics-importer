// Tests for scheduler internal metrics

package lsvmi

import (
	"bytes"
	"fmt"
	"path"
	"testing"

	"github.com/eparparita/linux-stats-victoriametrics-importer/internal/testutils"
)

type SchedulerInternalMetricsTestCase struct {
	InternalMetricsTestCase
	CrtStats, PrevStats SchedulerStats
}

var schedulerInternalMetricsTestcasesFile = path.Join(
	"..", testutils.LsvmiTestcasesSubdir,
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
	tlc := testutils.NewTestingLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	internalMetrics, err := newTestSchedulerInternalMetrics(tc)
	if err != nil {
		tlc.Fatal(err)
	}
	schedulerInternalMetrics := internalMetrics.schedulerMetrics
	schedulerInternalMetrics.stats[schedulerInternalMetrics.crtIndex] = tc.CrtStats
	schedulerInternalMetrics.stats[1-schedulerInternalMetrics.crtIndex] = tc.PrevStats
	testMetricsQueue := testutils.NewTestMetricsQueue(0)

	wantCrtIndex := 1 - schedulerInternalMetrics.crtIndex

	buf := testMetricsQueue.GetBuf()
	gotMetricsCount := schedulerInternalMetrics.generateMetrics(buf, nil)
	testMetricsQueue.QueueBuf(buf)

	errBuf := &bytes.Buffer{}

	gotCrtIndex := schedulerInternalMetrics.crtIndex
	if wantCrtIndex != gotCrtIndex {
		fmt.Fprintf(
			errBuf,
			"\ncrtIndex: want: %d, got: %d",
			wantCrtIndex, gotCrtIndex,
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
		tlc.Fatal(errBuf)
	}
}

func TestSchedulerInternalMetrics(t *testing.T) {
	t.Logf("Loading testcases from %q ...", schedulerInternalMetricsTestcasesFile)
	testcases := make([]*SchedulerInternalMetricsTestCase, 0)
	err := testutils.LoadJsonFile(schedulerInternalMetricsTestcasesFile, &testcases)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range testcases {
		t.Run(
			tc.Name,
			func(t *testing.T) { testSchedulerInternalMetrics(tc, t) },
		)
	}
}
