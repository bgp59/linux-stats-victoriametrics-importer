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
	CurrStats, PrevStats *HttpEndpointPoolStats
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
	tlc := testutils.NewTestLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	internalMetrics, err := newTestHttpEndpointPoolInternalMetrics(tc)
	if err != nil {
		t.Fatal(err)
	}
	httpEndpointPoolInternalMetrics := internalMetrics.httpEndpointPoolMetrics
	httpEndpointPoolInternalMetrics.stats[httpEndpointPoolInternalMetrics.currIndex] = tc.CurrStats
	httpEndpointPoolInternalMetrics.stats[1-httpEndpointPoolInternalMetrics.currIndex] = tc.PrevStats
	testMetricsQueue := testutils.NewTestMetricsQueue(0)

	wantCurrIndex := 1 - httpEndpointPoolInternalMetrics.currIndex

	buf := testMetricsQueue.GetBuf()
	gotMetricsCount := httpEndpointPoolInternalMetrics.generateMetrics(buf, nil)
	testMetricsQueue.QueueBuf(buf)

	errBuf := &bytes.Buffer{}

	gotCurrIndex := httpEndpointPoolInternalMetrics.currIndex
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
