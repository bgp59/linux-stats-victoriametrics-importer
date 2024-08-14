// Tests for proc_pid_metrics

package lsvmi

import (
	"bytes"
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/emypar/linux-stats-victoriametrics-importer/internal/testutils"
	"github.com/emypar/linux-stats-victoriametrics-importer/procfs"
)

// See proc_pid_metrics_utils_test for structures supporting the test.

type ProcPidMetricsGenerateTestCase struct {
	Name        string
	Description string

	ProcfsRoot string

	Instance       string
	Hostname       string
	LinuxClktckSec float64
	BoottimeMsec   int64

	PidTidMetricsInfo *TestProcPidTidMetricsInfoData // != nil -> hasPrev is true
	ParserData        *TestPidParserData
	FullMetrics       bool

	CurrPromTs, PrevPromTs int64 // Prometheus timestamps, i.e. milliseconds since the epoch

	WantMetricsCount int
	WantMetrics      []string
	ReportExtra      bool
	WantZeroDelta    *TestProcPidTidMetricsInfoData
}

type ProcPidMetricsExecuteTestCase struct {
	Name        string
	Description string

	NPart             int
	FullMetricsFactor int
	UsePidStatus      bool
	CycleNum          [PROC_PID_METRICS_CYCLE_NUM_COUNTERS]int
	ScanNum           int

	Instance       string
	Hostname       string
	LinuxClktckSec float64
	BoottimeMsec   int64

	PidTidListResult       []procfs.PidTid
	PidTidMetricsInfo      []*TestProcPidTidMetricsInfoData // != nil -> the metrics should be initialized
	TestCaseData           *TestPidParsersTestCaseData
	CurrPromTs, PrevPromTs int64 // Prometheus timestamps, i.e. milliseconds since the epoch

	WantMetricsCount int
	WantMetrics      []string
	ReportExtra      bool
	WantZeroDelta    []*TestProcPidTidMetricsInfoData
}

type TestPidTidListCache struct {
	pidTidList []procfs.PidTid
}

var procPidMetricsGenerateTestCaseFile = path.Join(
	"..", testutils.LsvmiTestCasesSubdir,
	"proc_pid_metrics_generate.json",
)

func (testPidTidListCache *TestPidTidListCache) GetPidTidList(part int, into []procfs.PidTid) ([]procfs.PidTid, error) {
	pidListLen := len(testPidTidListCache.pidTidList)
	if into == nil || cap(into) < pidListLen {
		into = make([]procfs.PidTid, pidListLen)
	} else {
		into = into[:pidListLen]
	}
	copy(into, testPidTidListCache.pidTidList)
	return into, nil
}

func (testPidTidListCache *TestPidTidListCache) Invalidate() {}

func (testPidTidListCache *TestPidTidListCache) GetRefreshCount() uint64 { return 0 }

func buildTestProcPidMetricsForGenerate(tc *ProcPidMetricsGenerateTestCase) (*ProcPidMetrics, *ProcPidTidMetricsInfo, error) {
	var pidTidMetricsInfo *ProcPidTidMetricsInfo

	pm, err := NewProcProcPidMetrics(nil, 0, nil)
	if err != nil {
		return nil, nil, err
	}

	pm.procfsRoot = tc.ProcfsRoot
	pm.usePidStatus = tc.ParserData.PidStatus != nil

	pm.instance = tc.Instance
	pm.hostname = tc.Hostname
	timeNow := time.UnixMilli(tc.CurrPromTs)
	pm.timeNowFn = func() time.Time { return timeNow }
	pm.linuxClktckSec = tc.LinuxClktckSec
	pm.boottimeMsec = tc.BoottimeMsec

	tcd := TestPidParsersTestCaseData{}
	pm.newPidStatParser = tcd.NewPidStat
	if pm.usePidStatus {
		pm.newPidStatusParser = tcd.NewPidStatus
	}

	if tc.PidTidMetricsInfo != nil {
		pm.prevTs = time.UnixMilli(tc.PrevPromTs)
		pidTidMetricsInfo = buildTestPidTidMetricsInfo(pm, tc.PidTidMetricsInfo)
	} else {
		pidTidMetricsInfo = buildTestPidTidMetricsInfo(pm, tc.ParserData)
	}

	pm.pidStat = &TestPidStat{}
	setTestPidStatData(pm.pidStat, tc.ParserData.PidStat)
	if pm.usePidStatus {
		pm.pidStatus = &TestPidStatus{}
		setTestPidStatusData(pm.pidStatus, tc.ParserData.PidStatus)
	}
	pm.pidCmdline = &TestPidCmdline{}
	setTestPidCmdlineData(pm.pidCmdline, tc.ParserData.PidCmdline)

	pm.initMetricsCache()

	return pm, pidTidMetricsInfo, nil
}

func buildTestProcPidMetricsForExecute(tc *ProcPidMetricsExecuteTestCase) (*ProcPidMetrics, error) {
	pm, err := NewProcProcPidMetrics(nil, tc.NPart, &TestPidTidListCache{tc.PidTidListResult})
	if err != nil {
		return nil, err
	}
	pm.fullMetricsFactor = tc.FullMetricsFactor
	pm.usePidStatus = tc.UsePidStatus
	pm.scanNum = tc.ScanNum
	pm.cycleNum = tc.CycleNum
	pm.scanNum = tc.ScanNum

	pm.instance = tc.Instance
	pm.hostname = tc.Hostname
	timeNow := time.UnixMilli(tc.CurrPromTs)
	pm.timeNowFn = func() time.Time { return timeNow }
	pm.procfsRoot = tc.TestCaseData.ProcfsRoot
	pm.linuxClktckSec = tc.LinuxClktckSec
	pm.boottimeMsec = tc.BoottimeMsec
	pm.newPidStatParser = tc.TestCaseData.NewPidStat
	pm.newPidStatusParser = tc.TestCaseData.NewPidStatus
	pm.newPidCmdlineParser = tc.TestCaseData.NewPidCmdline

	if tc.PidTidMetricsInfo != nil {
		pm.initMetricsCache()
		pm.prevTs = time.UnixMilli(tc.PrevPromTs)
		for _, primeInfo := range tc.PidTidMetricsInfo {
			pidTid := primeInfo.PidTid
			pm.pidTidMetricsInfo[pidTid] = buildTestPidTidMetricsInfo(pm, primeInfo)
		}
	}

	return pm, nil
}

func testProcPidMetricsGenerate(tc *ProcPidMetricsGenerateTestCase, t *testing.T) {
	tlc := testutils.NewTestLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	t.Logf("Description: %s", tc.Description)

	pm, pidTidMetricsInfo, err := buildTestProcPidMetricsForGenerate(tc)
	if err != nil {
		t.Fatal(err)
	}

	testMetricsQueue := testutils.NewTestMetricsQueue(0)
	buf := testMetricsQueue.GetBuf()

	hasPrev := tc.PidTidMetricsInfo != nil
	isPid := tc.ParserData.PidTid.Tid == procfs.PID_ONLY_TID
	fullMetrics := !hasPrev || tc.FullMetrics
	gotMetricsCount := pm.generateMetrics(
		pidTidMetricsInfo, hasPrev, isPid, fullMetrics, time.UnixMilli(tc.CurrPromTs), buf,
	)

	errBuf := &bytes.Buffer{}

	fmt.Fprintf(
		errBuf,
		"\nmetrics count: want: %d, got: %d",
		tc.WantMetricsCount, gotMetricsCount,
	)

	testMetricsQueue.GenerateReport(tc.WantMetrics, tc.ReportExtra, errBuf)

	if tc.WantZeroDelta != nil {
		cmpPidTidMetricsZeroDelta(tc.ParserData.PidTid, pidTidMetricsInfo, tc.WantZeroDelta, errBuf)
	}

	if errBuf.Len() > 0 {
		t.Fatal(errBuf)
	}
}

func TestProcPidMetricsGenerate(t *testing.T) {
	t.Logf("Loading test cases from %q ...", procPidMetricsGenerateTestCaseFile)
	testCases := make([]*ProcPidMetricsGenerateTestCase, 0)
	err := testutils.LoadJsonFile(procPidMetricsGenerateTestCaseFile, &testCases)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range testCases {
		t.Run(
			tc.Name,
			func(t *testing.T) { testProcPidMetricsGenerate(tc, t) },
		)
	}
}
