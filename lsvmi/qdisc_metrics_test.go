package lsvmi

import (
	"bytes"
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/eparparita/linux-stats-victoriametrics-importer/internal/testutils"
	"github.com/eparparita/linux-stats-victoriametrics-importer/qdisc"
)

type QdiscMetricsInfoTestData struct {
	Uint32ZeroDelta []bool
	Uint64ZeroDelta []bool
	CycleNum        int
}

type QdiscMetricsTestCase struct {
	Name                           string
	Description                    string
	Instance                       string
	Hostname                       string
	CurrQdiscStats, PrevQdiscStats *qdisc.QdiscStats
	CurrPromTs, PrevPromTs         int64
	QdiscMetricsInfo               map[qdisc.QdiscInfoKey]*QdiscMetricsInfoTestData
	FullMetricsFactor              int
	WantMetricsCount               int
	WantMetrics                    []string
	ReportExtra                    bool
	WantQdiscMetricsInfo           map[qdisc.QdiscInfoKey]*QdiscMetricsInfoTestData
}

var qdiscMetricsTestCasesFile = path.Join(
	"..", testutils.LsvmiTestCasesSubdir,
	"qdisc.json",
)

func testQdiscMetrics(tc *QdiscMetricsTestCase, t *testing.T) {
	tlc := testutils.NewTestLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	t.Logf("Description: %s", tc.Description)

	qdiscMetrics, err := NewQdiscMetrics(nil)
	if err != nil {
		t.Fatal(err)
	}
	qdiscMetrics.instance = tc.Instance
	qdiscMetrics.hostname = tc.Hostname
	currIndex := qdiscMetrics.currIndex
	qdiscMetrics.qdiscStats[currIndex] = tc.CurrQdiscStats
	qdiscMetrics.qdiscStatsTs[currIndex] = time.UnixMilli(tc.CurrPromTs)
	qdiscMetrics.qdiscStats[1-currIndex] = tc.PrevQdiscStats
	qdiscMetrics.qdiscStatsTs[1-currIndex] = time.UnixMilli(tc.PrevPromTs)
	qdiscMetrics.fullMetricsFactor = tc.FullMetricsFactor

	if tc.QdiscMetricsInfo != nil {
		for qiKey, qi := range qdiscMetrics.qdiscStats[currIndex].Info {
			qimTd := tc.QdiscMetricsInfo[qiKey]
			if qimTd != nil {
				qdiscMetrics.updateQdiscMetricsInfo(qiKey, qi)
				qim := qdiscMetrics.qdiscMetricsInfoMap[qiKey]
				copy(qim.uint32ZeroDelta, qimTd.Uint32ZeroDelta)
				copy(qim.uint64ZeroDelta, qimTd.Uint64ZeroDelta)
				qim.cycleNum = qimTd.CycleNum
			}
		}
	}

	wantCurrIndex := 1 - currIndex
	testMetricsQueue := testutils.NewTestMetricsQueue(0)
	buf := testMetricsQueue.GetBuf()
	gotMetricsCount, _ := qdiscMetrics.generateMetrics(buf)
	testMetricsQueue.QueueBuf(buf)

	errBuf := &bytes.Buffer{}

	gotCurrIndex := qdiscMetrics.currIndex
	if wantCurrIndex != gotCurrIndex {
		fmt.Fprintf(
			errBuf,
			"\ncurrIndex: want: %d, got: %d",
			wantCurrIndex, gotCurrIndex,
		)
	}

	if tc.WantQdiscMetricsInfo != nil {
		for qiKey, wantQimTD := range tc.WantQdiscMetricsInfo {
			gotQim := qdiscMetrics.qdiscMetricsInfoMap[qiKey]
			if gotQim == nil {
				fmt.Fprintf(
					errBuf,
					"\n.qdiscMetricsInfoMap[%v]: missing",
					qiKey,
				)
				continue
			}
			testutils.CompareSlices(
				wantQimTD.Uint32ZeroDelta,
				gotQim.uint32ZeroDelta,
				fmt.Sprintf(".qdiscMetricsInfoMap[%v].uint32ZeroDelta", qiKey),
				errBuf,
			)
			testutils.CompareSlices(
				wantQimTD.Uint64ZeroDelta,
				gotQim.uint64ZeroDelta,
				fmt.Sprintf(".qdiscMetricsInfoMap[%v].uint64ZeroDelta", qiKey),
				errBuf,
			)
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

func TestQdiscMetrics(t *testing.T) {
	t.Logf("Loading test cases from %q ...", qdiscMetricsTestCasesFile)
	testCases := make([]*QdiscMetricsTestCase, 0)
	err := testutils.LoadJsonFile(qdiscMetricsTestCasesFile, &testCases)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range testCases {
		t.Run(
			tc.Name,
			func(t *testing.T) { testQdiscMetrics(tc, t) },
		)
	}
}
