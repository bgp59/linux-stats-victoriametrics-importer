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

	WantMetricsCount int
	WantMetrics      []string
	ReportExtra      bool
	WantZeroDelta    *TestProcPidTidMetricsInfoData
}

type ProcPidMetricsExecuteTestCase struct {
	Name        string
	Description string

	PartNo            int
	FullMetricsFactor int
	UsePidStatus      bool
	CycleNum          [PROC_PID_METRICS_CYCLE_NUM_COUNTERS]int
	ScanNum           int

	Instance       string
	Hostname       string
	LinuxClktckSec float64
	BoottimeMsec   int64

	PidTidListResult       []procfs.PidTid
	PidTidMetricsInfoList  []*TestProcPidTidMetricsInfoData // != nil -> the metrics should be initialized
	PidParsersData         *TestPidParsersTestCaseData
	CurrPromTs, PrevPromTs int64 // Prometheus timestamps, i.e. milliseconds since the epoch, for the specific metrics

	WantMetricsCount  int
	WantMetrics       []string
	ReportExtra       bool
	WantZeroDeltaList []*TestProcPidTidMetricsInfoData
}

type TestPidTidListCache struct {
	pidTidList []procfs.PidTid
}

var procPidMetricsGenerateTestCaseFile = path.Join(
	"..", testutils.LsvmiTestCasesSubdir,
	"proc_pid_metrics_generate.json",
)

var procPidMetricsExecuteTestCaseFile = path.Join(
	"..", testutils.LsvmiTestCasesSubdir,
	"proc_pid_metrics_execute.json",
)

func (testPidTidListCache *TestPidTidListCache) GetPidTidList(partNo int, into []procfs.PidTid) ([]procfs.PidTid, error) {
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
	pm.linuxClktckSec = tc.LinuxClktckSec
	pm.boottimeMsec = tc.BoottimeMsec

	tcd := TestPidParsersTestCaseData{}
	pm.newPidStatParser = tcd.NewPidStat
	if pm.usePidStatus {
		pm.newPidStatusParser = tcd.NewPidStatus
	}

	if tc.PidTidMetricsInfo != nil {
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
	pm, err := NewProcProcPidMetrics(nil, tc.PartNo, &TestPidTidListCache{tc.PidTidListResult})
	if err != nil {
		return nil, err
	}
	pm.fullMetricsFactor = tc.FullMetricsFactor
	pm.usePidStatus = tc.UsePidStatus
	pm.cycleNum = tc.CycleNum
	pm.scanNum = tc.ScanNum

	pm.instance = tc.Instance
	pm.hostname = tc.Hostname

	// The time.Now() should return the timestamp from the most recent
	// successfully parsed data exactly once; otherwise it should return the
	// test case's timestamp.
	pm.timeNowFn = func() time.Time {
		promTs := tc.CurrPromTs
		if tc.PidParsersData.validPromTs {
			promTs = tc.PidParsersData.promTs
			tc.PidParsersData.validPromTs = false // to be used only once
		}
		return time.UnixMilli(promTs)
	}

	pm.metricsQueue = testutils.NewTestMetricsQueue(0)
	pm.procfsRoot = tc.PidParsersData.ProcfsRoot
	pm.linuxClktckSec = tc.LinuxClktckSec
	pm.boottimeMsec = tc.BoottimeMsec
	pm.newPidStatParser = tc.PidParsersData.NewPidStat
	pm.newPidStatusParser = tc.PidParsersData.NewPidStatus
	pm.newPidCmdlineParser = tc.PidParsersData.NewPidCmdline

	if tc.PidTidMetricsInfoList != nil {
		pm.initialize()
		pm.prevTs = time.UnixMilli(tc.PrevPromTs)
		for _, primeInfo := range tc.PidTidMetricsInfoList {
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
		pidTidMetricsInfo, hasPrev, isPid, fullMetrics, time.UnixMilli(tc.ParserData.CurrPromTs), buf,
	)
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

func testProcPidMetricsExecute(tc *ProcPidMetricsExecuteTestCase, t *testing.T) {
	tlc := testutils.NewTestLogCollect(t, Log, nil)
	defer tlc.RestoreLog()
	savedGlobalMetricsGeneratorStatsContainer := GlobalMetricsGeneratorStatsContainer
	defer func() { GlobalMetricsGeneratorStatsContainer = savedGlobalMetricsGeneratorStatsContainer }()
	GlobalMetricsGeneratorStatsContainer = NewMetricsGeneratorStatsContainer()

	t.Logf("Description: %s", tc.Description)

	pm, err := buildTestProcPidMetricsForExecute(tc)
	if err != nil {
		t.Fatal(err)
	}

	pm.Execute()

	errBuf := &bytes.Buffer{}

	// Verify metrics:
	gotMetricsCount := int(
		GlobalMetricsGeneratorStatsContainer.stats[pm.id][METRICS_GENERATOR_ACTUAL_METRICS_COUNT])
	if tc.WantMetricsCount != gotMetricsCount {
		fmt.Fprintf(
			errBuf,
			"\nmetrics count: want: %d, got: %d",
			tc.WantMetricsCount, gotMetricsCount,
		)
	}
	testMetricsQueue := pm.metricsQueue.(*testutils.TestMetricsQueue)
	testMetricsQueue.GenerateReport(tc.WantMetrics, tc.ReportExtra, errBuf)

	// Verify zero delta state:
	for _, wantZeroDelta := range tc.WantZeroDeltaList {
		pidTid := wantZeroDelta.PidTid
		pidTidMetricsInfo := pm.pidTidMetricsInfo[pidTid]
		if pidTidMetricsInfo != nil {
			cmpPidTidMetricsZeroDelta(&pidTid, pidTidMetricsInfo, wantZeroDelta, errBuf)
		} else {
			fmt.Fprintf(errBuf, "\npidTidMetricsInfo[%v]: missing", pidTid)
		}
	}

	// Verify cycle info:
	for i, want := range tc.CycleNum {
		if want++; want >= tc.FullMetricsFactor {
			want = 0
		}
		got := pm.cycleNum[i]
		if want != got {
			fmt.Fprintf(errBuf, "\ncycleNum[%d]: want: %d, got: %d", i, want, got)
		}
	}

	want := tc.ScanNum + 1
	if want != pm.scanNum {
		fmt.Fprintf(errBuf, "\nscanNum,: want: %d, got: %d", want, pm.scanNum)
	}

	// Verify metrics info cache consistency; only the PID,TID's in the test
	// data should be keys in the cache:
	expectedPidTid := make(map[procfs.PidTid]bool)
	for _, testPidParserData := range tc.PidParsersData.ParserDataList {
		expectedPidTid[*testPidParserData.PidTid] = true
	}
	for pidTid := range pm.pidTidMetricsInfo {
		if !expectedPidTid[pidTid] {
			fmt.Fprintf(errBuf, "\npidTidMetricsInfo[%#v]: unexpected PidTid key", pidTid)
		} else {
			delete(expectedPidTid, pidTid)
		}
	}
	for pidTid := range expectedPidTid {
		fmt.Fprintf(errBuf, "\npidTidMetricsInfo[%#v]: missing PidTid key", pidTid)
	}

	if errBuf.Len() > 0 {
		t.Fatal(errBuf)
	}
}

func TestProcPidMetricsExecute(t *testing.T) {
	t.Logf("Loading test cases from %q ...", procPidMetricsExecuteTestCaseFile)
	testCases := make([]*ProcPidMetricsExecuteTestCase, 0)
	err := testutils.LoadJsonFile(procPidMetricsExecuteTestCaseFile, &testCases)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range testCases {
		t.Run(
			tc.Name,
			func(t *testing.T) { testProcPidMetricsExecute(tc, t) },
		)
	}
}
