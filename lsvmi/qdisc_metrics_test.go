package lsvmi

import (
	"bytes"
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/bgp59/linux-stats-victoriametrics-importer/internal/testutils"
	"github.com/bgp59/linux-stats-victoriametrics-importer/qdisc"
)

type QdiscMetricsInfoTestData struct {
	QdiscInfoKey    qdisc.QdiscInfoKey
	Uint32ZeroDelta []bool
	Uint64ZeroDelta []bool
	CycleNum        int
}

type QdiscStatsInfoTestData struct {
	QdiscInfoKey qdisc.QdiscInfoKey
	QdiscInfo    *qdisc.QdiscInfo
}

type QdiscMetricsTestCase struct {
	Name                           string
	Description                    string
	Instance                       string
	Hostname                       string
	CurrQdiscStats, PrevQdiscStats []QdiscStatsInfoTestData
	CurrPromTs, PrevPromTs         int64
	QdiscMetricsInfo               []QdiscMetricsInfoTestData
	FullMetricsFactor              int
	WantMetricsCount               int
	WantMetrics                    []string
	ReportExtra                    bool
	WantQdiscMetricsInfo           []QdiscMetricsInfoTestData
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

	prevQdiscStats := qdisc.NewQdiscStats()
	outOfscopeQiKeys := make(map[qdisc.QdiscInfoKey]bool)
	for _, qiTD := range tc.PrevQdiscStats {
		qiKey := qiTD.QdiscInfoKey
		prevQdiscStats.Info[qiKey] = qiTD.QdiscInfo
		outOfscopeQiKeys[qiKey] = true
	}

	currQdiscStats := prevQdiscStats.Clone()
	for _, qiTD := range tc.CurrQdiscStats {
		qiKey := qiTD.QdiscInfoKey
		currQdiscStats.Info[qiKey] = qiTD.QdiscInfo
		delete(outOfscopeQiKeys, qiKey)
	}
	for qiKey := range outOfscopeQiKeys {
		delete(currQdiscStats.Info, qiKey)
	}

	qdiscMetrics.qdiscStats[currIndex] = currQdiscStats
	qdiscMetrics.qdiscStats[1-currIndex] = prevQdiscStats
	qdiscMetrics.qdiscStatsTs[currIndex] = time.UnixMilli(tc.CurrPromTs)
	qdiscMetrics.qdiscStatsTs[1-currIndex] = time.UnixMilli(tc.PrevPromTs)
	qdiscMetrics.fullMetricsFactor = tc.FullMetricsFactor

	if tc.QdiscMetricsInfo != nil {
		for _, qmidTD := range tc.QdiscMetricsInfo {
			qiKey := qmidTD.QdiscInfoKey
			if qi := qdiscMetrics.qdiscStats[1-currIndex].Info[qiKey]; qi != nil {
				qdiscMetrics.updateQdiscMetricsInfo(qiKey, qi)
				qim := qdiscMetrics.qdiscMetricsInfoMap[qiKey]
				for i := range qdiscUint32IndexToDeltaMetricNameMap {
					qim.uint32ZeroDelta[i] = qmidTD.Uint32ZeroDelta[i]
				}
				for i := range qdiscUint64IndexToDeltaMetricNameMap {
					qim.uint64ZeroDelta[i] = qmidTD.Uint64ZeroDelta[i]
				}
				qim.cycleNum = qmidTD.CycleNum
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
		for _, wantQimTD := range tc.WantQdiscMetricsInfo {
			qiKey := wantQimTD.QdiscInfoKey
			gotQim := qdiscMetrics.qdiscMetricsInfoMap[qiKey]
			if gotQim == nil {
				fmt.Fprintf(
					errBuf,
					"\n.qdiscMetricsInfoMap[%s]: missing",
					&qiKey,
				)
				continue
			}
			testutils.CompareSlices(
				wantQimTD.Uint32ZeroDelta,
				gotQim.uint32ZeroDelta,
				fmt.Sprintf(".qdiscMetricsInfoMap[%s].uint32ZeroDelta", &qiKey),
				errBuf,
			)
			testutils.CompareSlices(
				wantQimTD.Uint64ZeroDelta,
				gotQim.uint64ZeroDelta,
				fmt.Sprintf(".qdiscMetricsInfoMap[%s].uint64ZeroDelta", &qiKey),
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
