// Tests for scheduler internal metrics

package lsvmi

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/eparparita/linux-stats-victoriametrics-importer/internal/testutils"
)

type InternalMetricsTestCase struct {
	Instance         string
	Hostname         string
	PromTs           int64
	FullCycle        bool
	WantMetricsCount int
	WantMetrics      []string
	ReportExtra      bool
}

type SchedulerInternalMetricsTestCase struct {
	InternalMetricsTestCase
	CrtStats, PrevStats SchedulerStats
}

func newTestSchedulerInternalMetrics(tc *InternalMetricsTestCase) (*InternalMetrics, error) {
	internalMetrics, err := NewInternalMetrics(nil)
	if err != nil {
		return nil, err
	}
	scheduler, err := NewScheduler(nil)
	if err != nil {
		return nil, err
	}

	internalMetrics.instance = tc.Instance
	internalMetrics.hostname = tc.Hostname
	timeNowRetVal := time.UnixMilli(tc.PromTs)
	internalMetrics.timeNowFn = func() time.Time { return timeNowRetVal }
	internalMetrics.scheduler = scheduler

	return internalMetrics, nil
}

func testSchedulerInternalMetrics(tc *SchedulerInternalMetricsTestCase, t *testing.T) {
	tlc := testutils.NewTestingLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	internalMetrics, err := newTestSchedulerInternalMetrics(&tc.InternalMetricsTestCase)
	if err != nil {
		tlc.Fatal(err)
	}
	schedulerInternalMetrics := internalMetrics.schedulerMetrics
	schedulerInternalMetrics.stats[schedulerInternalMetrics.crtStatsIndx] = tc.CrtStats
	schedulerInternalMetrics.stats[1-schedulerInternalMetrics.crtStatsIndx] = tc.PrevStats
	testMetricsQueue := testutils.NewTestMetricsQueue(0)

	buf := testMetricsQueue.GetBuf()
	gotMetricsCount := schedulerInternalMetrics.generateMetrics(buf, tc.FullCycle, nil)
	testMetricsQueue.QueueBuf(buf)

	errBuf := &bytes.Buffer{}

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
	for _, tc := range []*SchedulerInternalMetricsTestCase{
		{},
	} {
		t.Run(
			"",
			func(t *testing.T) { testSchedulerInternalMetrics(tc, t) },
		)
	}
}
