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
	CrtStats, PrevStats CompressorPoolStats
}

var compressorPoolInternalMetricsTestcasesFile = path.Join(
	"..", testutils.LsvmiTestcasesSubdir,
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
	tlc := testutils.NewTestingLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	internalMetrics, err := newTestCompressorPoolInternalMetrics(tc)
	if err != nil {
		tlc.Fatal(err)
	}
	compressorPoolInternalMetrics := internalMetrics.compressorPoolMetrics
	compressorPoolInternalMetrics.stats[compressorPoolInternalMetrics.crtStatsIndx] = tc.CrtStats
	compressorPoolInternalMetrics.stats[1-compressorPoolInternalMetrics.crtStatsIndx] = tc.PrevStats
	testMetricsQueue := testutils.NewTestMetricsQueue(0)

	buf := testMetricsQueue.GetBuf()
	gotMetricsCount := compressorPoolInternalMetrics.generateMetrics(buf, tc.FullCycle, nil)
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

func TestCompressorPoolInternalMetrics(t *testing.T) {
	t.Logf("Loading testcases from %q ...", compressorPoolInternalMetricsTestcasesFile)
	testcases := make([]*CompressorPoolInternalMetricsTestCase, 0)
	err := testutils.LoadJsonFile(compressorPoolInternalMetricsTestcasesFile, &testcases)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range testcases {
		t.Run(
			tc.Name,
			func(t *testing.T) { testCompressorPoolInternalMetrics(tc, t) },
		)
	}
}
