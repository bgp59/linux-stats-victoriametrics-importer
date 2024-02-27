// Tests for HTTP endpoint pool internal metrics
package lsvmi

import (
	"bytes"
	"fmt"
	"path"
	"testing"

	"github.com/eparparita/linux-stats-victoriametrics-importer/internal/testutils"
)

type HttpEndpointPoolInternalMetricsTestCase struct {
	InternalMetricsTestCase
	CrtStats, PrevStats *HttpEndpointPoolStats
}

var httpEndpointPoolInternalMetricsTestcasesFile = path.Join(
	"..", testutils.LsvmiTestcasesSubdir,
	"internal_metrics", "http_endpoint_pool.json",
)

func newTestHttpEndpointPoolInternalMetrics(tc *HttpEndpointPoolInternalMetricsTestCase) (*InternalMetrics, error) {
	internalMetrics, err := newTestInternalMetrics(&tc.InternalMetricsTestCase)
	if err != nil {
		return nil, err
	}
	httpEndpointPool, err := NewHttpEndpointPool(nil)
	if err != nil {
		return nil, err
	}
	internalMetrics.httpEndpointPool = httpEndpointPool
	return internalMetrics, nil
}

func testHttpEndpointPoolInternalMetrics(tc *HttpEndpointPoolInternalMetricsTestCase, t *testing.T) {
	tlc := testutils.NewTestingLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	internalMetrics, err := newTestHttpEndpointPoolInternalMetrics(tc)
	if err != nil {
		tlc.Fatal(err)
	}
	httpEndpointPoolInternalMetrics := internalMetrics.httpEndpointPoolMetrics
	httpEndpointPoolInternalMetrics.stats[httpEndpointPoolInternalMetrics.crtStatsIndx] = tc.CrtStats
	httpEndpointPoolInternalMetrics.stats[1-httpEndpointPoolInternalMetrics.crtStatsIndx] = tc.PrevStats
	testMetricsQueue := testutils.NewTestMetricsQueue(0)

	buf := testMetricsQueue.GetBuf()
	gotMetricsCount := httpEndpointPoolInternalMetrics.generateMetrics(buf, tc.FullCycle, nil)
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

func TestHttpEndpointPoolInternalMetrics(t *testing.T) {
	t.Logf("Loading testcases from %q ...", httpEndpointPoolInternalMetricsTestcasesFile)
	testcases := make([]*HttpEndpointPoolInternalMetricsTestCase, 0)
	err := testutils.LoadJsonFile(httpEndpointPoolInternalMetricsTestcasesFile, &testcases)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range testcases {
		t.Run(
			tc.Name,
			func(t *testing.T) { testHttpEndpointPoolInternalMetrics(tc, t) },
		)
	}
}
