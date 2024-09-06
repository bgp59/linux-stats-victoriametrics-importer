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

	PageSize uint64

	Instance       string
	Hostname       string
	LinuxClktckSec float64
	BoottimeMsec   int64

	PidTidMetricsInfo *TestPidParserStateData
	ParserData        *TestPidParserStateData
	FullMetrics       bool

	WantMetricsCount int
	WantMetrics      []string
	ReportExtra      bool
	WantZeroDelta    *TestPidParserStateData
}

type ProcPidMetricsExecuteTestCase struct {
	Name        string
	Description string

	ProcfsRoot        string
	PartNo            int
	FullMetricsFactor int
	UsePidStatus      bool
	CycleNum          [PROC_PID_METRICS_CYCLE_NUM_COUNTERS]int
	ScanNum           int

	PageSize uint64

	Instance       string
	Hostname       string
	LinuxClktckSec float64
	BoottimeMsec   int64

	PidTidListResult             []procfs.PidTid
	PidTidMetricsInfoList        []*TestPidParserStateData // != nil -> the metrics should be initialized
	PidParsersDataList           []*TestPidParserStateData
	CurrUnixMilli, PrevUnixMilli int64 // Timestamps for specific metrics

	WantMetricsCount  int
	WantMetrics       []string
	ReportExtra       bool
	WantZeroDeltaList []*TestPidParserStateData
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

func testProcPidMetricsGenerate(tc *ProcPidMetricsGenerateTestCase, t *testing.T) {
	tlc := testutils.NewTestLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	t.Logf("Description: %s", tc.Description)

	pm, err := NewProcProcPidMetrics(nil, 0, nil)
	if err != nil {
		t.Fatal(err)
	}

	pm.procfsRoot = tc.ProcfsRoot
	pm.usePidStatus = tc.ParserData.PidStatus != nil

	if tc.PageSize != 0 {
		pm.pageSize = tc.PageSize
	}

	pm.instance = tc.Instance
	pm.hostname = tc.Hostname
	pm.linuxClktckSec = tc.LinuxClktckSec
	pm.boottimeMsec = tc.BoottimeMsec

	tpp := TestPidParsers{}
	pm.newPidStatParser = tpp.NewPidStat
	if pm.usePidStatus {
		pm.newPidStatusParser = tpp.NewPidStatus
	}

	var pidTidMetricsInfo *ProcPidTidMetricsInfo
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

	testMetricsQueue := testutils.NewTestMetricsQueue(0)
	buf := testMetricsQueue.GetBuf()

	hasPrev := tc.PidTidMetricsInfo != nil
	isPid := tc.ParserData.PidTid.Tid == procfs.PID_ONLY_TID
	fullMetrics := !hasPrev || tc.FullMetrics
	gotMetricsCount := pm.generateMetrics(
		pidTidMetricsInfo, hasPrev, isPid, fullMetrics, time.UnixMilli(tc.ParserData.UnixMilli), buf,
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
		cmpPidTidMetricsZeroDelta(pidTidMetricsInfo, tc.WantZeroDelta, errBuf)
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

	pm, err := NewProcProcPidMetrics(nil, tc.PartNo, &TestPidTidListCache{tc.PidTidListResult})
	if err != nil {
		t.Fatal(err)
	}

	if tc.PageSize != 0 {
		pm.pageSize = tc.PageSize
	}

	pm.instance = tc.Instance
	pm.hostname = tc.Hostname
	pm.linuxClktckSec = tc.LinuxClktckSec
	pm.boottimeMsec = tc.BoottimeMsec
	pm.fullMetricsFactor = tc.FullMetricsFactor
	pm.usePidStatus = tc.UsePidStatus
	pm.cycleNum = tc.CycleNum
	pm.scanNum = tc.ScanNum

	tpp := NewTestPidParsers(tc.PidParsersDataList, tc.ProcfsRoot, tc.CurrUnixMilli)
	pm.newPidStatParser = tpp.NewPidStat
	pm.newPidStatusParser = tpp.NewPidStatus
	pm.newPidCmdlineParser = tpp.NewPidCmdline
	pm.timeNowFn = tpp.timeNow

	if len(tc.PidTidMetricsInfoList) > 0 {
		for _, pidParserState := range tc.PidTidMetricsInfoList {
			pidTidMetricsInfo := buildTestPidTidMetricsInfo(pm, pidParserState)
			pm.pidTidMetricsInfo[pidTidMetricsInfo.pidTid] = pidTidMetricsInfo
			pidTidMetricsInfo.next = pm.pidTidMetricsInfoHead
			pm.pidTidMetricsInfoHead = pidTidMetricsInfo
			if pidTidMetricsInfo.next == nil {
				pm.pidTidMetricsInfoTail = pidTidMetricsInfo
			} else {
				pidTidMetricsInfo.next.prev = pidTidMetricsInfo
			}
		}
		pm.initialize()
		pm.prevTs = time.UnixMilli(tc.PrevUnixMilli)
	}

	pm.metricsQueue = testutils.NewTestMetricsQueue(0)
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
		pidTid := *wantZeroDelta.PidTid
		pidTidMetricsInfo := pm.pidTidMetricsInfo[pidTid]
		if pidTidMetricsInfo != nil {
			cmpPidTidMetricsZeroDelta(pidTidMetricsInfo, wantZeroDelta, errBuf)
		} else {
			fmt.Fprintf(errBuf, "\npidTidMetricsInfo[%#v]: missing PidTid key for zero delta check", pidTid)
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

	// Verify metrics info cache consistency; only the PID,TID's in good
	// standing (i.e. no simulated parse errors) in the test data should be keys
	// in the cache:
	expectedPidTid := make(map[procfs.PidTid]bool)
	for _, testPidParserData := range tc.PidParsersDataList {
		pidTid := *testPidParserData.PidTid
		if !tpp.failedPidTid[pidTid] {
			expectedPidTid[pidTid] = true
		}
	}
	for pidTid := range pm.pidTidMetricsInfo {
		if !expectedPidTid[pidTid] {
			fmt.Fprintf(errBuf, "\npidTidMetricsInfo[%#v]: unexpected PidTid key at consistency check", pidTid)
		} else {
			delete(expectedPidTid, pidTid)
		}
	}
	for pidTid := range expectedPidTid {
		fmt.Fprintf(errBuf, "\npidTidMetricsInfo[%#v]: missing PidTid key at consistency check", pidTid)
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
