// Tests for compressor pool internal metrics

package lsvmi

import (
	"bytes"
	"fmt"
	"path"
	"testing"

	"github.com/eparparita/linux-stats-victoriametrics-importer/internal/testutils"
)

type CompressorPoolInternalMetricsTestCase struct {
	InternalMetricsTestCase
	CurrStats, PrevStats CompressorPoolStats
}

var compressorPoolInternalMetricsTestCasesFile = path.Join(
	"..", testutils.LsvmiTestCasesSubdir,
	"internal_metrics", "compressor_pool.json",
)

func newTestCompressorPoolInternalMetrics(tc *CompressorPoolInternalMetricsTestCase) (*InternalMetrics, error) {
	internalMetrics, err := newTestInternalMetrics(&tc.InternalMetricsTestCase)
	if err != nil {
		return nil, err
	}
	compressorPool, err := NewCompressorPool(nil)
	if err != nil {
		return nil, err
	}
	internalMetrics.compressorPool = compressorPool
	return internalMetrics, nil
}

func testCompressorPoolInternalMetrics(tc *CompressorPoolInternalMetricsTestCase, t *testing.T) {
	tlc := testutils.NewTestLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	internalMetrics, err := newTestCompressorPoolInternalMetrics(tc)
	if err != nil {
		t.Fatal(err)
	}
	compressorPoolInternalMetrics := internalMetrics.compressorPoolMetrics
	compressorPoolInternalMetrics.stats[compressorPoolInternalMetrics.currIndex] = tc.CurrStats
	compressorPoolInternalMetrics.stats[1-compressorPoolInternalMetrics.currIndex] = tc.PrevStats
	testMetricsQueue := testutils.NewTestMetricsQueue(0)

	wantCurrIndex := 1 - compressorPoolInternalMetrics.currIndex

	buf := testMetricsQueue.GetBuf()
	gotMetricsCount := compressorPoolInternalMetrics.generateMetrics(buf, nil)
	testMetricsQueue.QueueBuf(buf)

	errBuf := &bytes.Buffer{}

	gotCurrIndex := compressorPoolInternalMetrics.currIndex
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

func TestCompressorPoolInternalMetrics(t *testing.T) {
	t.Logf("Loading test cases from %q ...", compressorPoolInternalMetricsTestCasesFile)
	testCases := make([]*CompressorPoolInternalMetricsTestCase, 0)
	err := testutils.LoadJsonFile(compressorPoolInternalMetricsTestCasesFile, &testCases)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range testCases {
		t.Run(
			tc.Name,
			func(t *testing.T) { testCompressorPoolInternalMetrics(tc, t) },
		)
	}
}
