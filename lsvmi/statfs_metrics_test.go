package lsvmi

import (
	"bytes"
	"fmt"
	"path"
	"testing"

	"golang.org/x/sys/unix"

	"github.com/emypar/linux-stats-victoriametrics-importer/internal/testutils"
	"github.com/emypar/linux-stats-victoriametrics-importer/procfs"
)

type StatfsMetricsKeepFsTypeTestCase struct {
	name        string
	includeList []string
	excludeList []string
	wantKeep    []string
	wantNotKeep []string
}

type StatfsInfoTestData struct {
	Fs, MountPoint, FsType string
	Statfs                 *unix.Statfs_t
	CycleNum               int
	ScanNum                int
}

type StatfsMetricsTestCase struct {
	Name                           string
	Description                    string
	Instance                       string
	Hostname                       string
	CurrStatfsInfo, PrevStatfsInfo []*StatfsInfoTestData
	WantMetricsCount               int
	WantMetrics                    []string
	ReportExtra                    bool
}

var statfsMetricsTestCasesFile = path.Join(
	"..", testutils.LsvmiTestCasesSubdir,
	"statfs.json",
)

func testStatfsKeepFsType(tc *StatfsMetricsKeepFsTypeTestCase, t *testing.T) {
	tlc := testutils.NewTestLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	cfg := DefaultStatfsMetricsConfig()
	if tc.includeList != nil {
		cfg.IncludeFilesystemTypes = make([]string, len(tc.includeList))
		copy(cfg.IncludeFilesystemTypes, tc.includeList)
	}
	if tc.excludeList != nil {
		cfg.ExcludeFilesystemTypes = make([]string, len(tc.excludeList))
		copy(cfg.ExcludeFilesystemTypes, tc.excludeList)
	}

	statfsMetrics, err := NewStatfsMetrics(cfg)
	if err != nil {
		t.Fatal(err)
	}

	errBuf := &bytes.Buffer{}
	errFsType := make([]string, 0)

	errFsType = errFsType[:0]
	for _, fsType := range tc.wantKeep {
		if !statfsMetrics.keepFsType(fsType) {
			errFsType = append(errFsType, fsType)
		}
	}
	if len(errFsType) > 0 {
		fmt.Fprintf(errBuf, "\nmissing keep: %q", errFsType)
	}

	errFsType = errFsType[:0]
	for _, fsType := range tc.wantNotKeep {
		if statfsMetrics.keepFsType(fsType) {
			errFsType = append(errFsType, fsType)
		}
	}
	if len(errFsType) > 0 {
		fmt.Fprintf(errBuf, "\nunexpected keep: %q", errFsType)
	}

	if errBuf.Len() > 0 {
		t.Fatal(errBuf)
	}
}

func initStatfsMetricsStatfsInfo(sfsm *StatfsMetrics, sfsiList []*StatfsInfoTestData) {
	sfsm.statfsInfo = make(map[StatfsMountinfo]*StatfsInfo)
	for _, sfsi := range sfsiList {
		mountinfo := StatfsMountinfo{
			fs:         sfsi.Fs,
			mountPoint: sfsi.MountPoint,
			fsType:     sfsi.FsType,
		}
		sfsm.statfsInfo[mountinfo] = &StatfsInfo{
			cycleNum: sfsi.CycleNum,
			scanNum:  sfsi.ScanNum,
		}
		if sfsi.Statfs != nil {
			statfsBuf := &unix.Statfs_t{}
			*statfsBuf = *sfsi.Statfs
			sfsm.statfsInfo[mountinfo].statfsBuf[sfsm.currIndex] = statfsBuf
		}
	}
}

func testStatfsMetrics(tc *StatfsMetricsTestCase, t *testing.T) {
	tlc := testutils.NewTestLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	t.Logf("Description: %s", tc.Description)

	statfsMetrics, err := NewStatfsMetrics(nil)
	if err != nil {
		t.Fatal(err)
	}
	statfsMetrics.procMountinfo = procfs.NewMountinfo("", 0)
	statfsMetrics.instance = tc.Instance
	statfsMetrics.hostname = tc.Hostname

	if tc.PrevStatfsInfo != nil {
		statfsMetrics.currIndex = 1 - statfsMetrics.currIndex
		initStatfsMetricsStatfsInfo(statfsMetrics, tc.PrevStatfsInfo)
		statfsMetrics.currIndex = 1 - statfsMetrics.currIndex
	}

	wantCurrIndex := 1 - statfsMetrics.currIndex
	testMetricsQueue := testutils.NewTestMetricsQueue(0)
	buf := testMetricsQueue.GetBuf()
	gotMetricsCount, _ := statfsMetrics.generateMetrics(buf)
	testMetricsQueue.QueueBuf(buf)

	errBuf := &bytes.Buffer{}

	gotCurrIndex := statfsMetrics.currIndex
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

func TestStatfsKeepFsType(t *testing.T) {
	for _, tc := range []*StatfsMetricsKeepFsTypeTestCase{
		{
			name: "include_all,exclude_none",
			wantKeep: []string{
				"incFsType",
				"otherFsType",
			},
		},
		{
			name: "include_some,exclude_none",
			includeList: []string{
				"incFsType",
			},
			wantKeep: []string{
				"incFsType",
			},
			wantNotKeep: []string{
				"excFsType",
				"otherFsType",
			},
		},
		{
			name: "include_all,exclude_some",
			excludeList: []string{
				"excFsType",
			},
			wantKeep: []string{
				"incFsType",
				"otherFsType",
			},
			wantNotKeep: []string{
				"excFsType",
			},
		},
		{
			name: "include_some,exclude_some",
			includeList: []string{
				"incFsType",
			},
			excludeList: []string{
				"excFsType",
			},
			wantKeep: []string{
				"incFsType",
			},
			wantNotKeep: []string{
				"excFsType",
				"otherFsType",
			},
		},
	} {
		t.Run(
			tc.name,
			func(t *testing.T) { testStatfsKeepFsType(tc, t) },
		)
	}
}

func TestStatfsMetrics(t *testing.T) {
	t.Logf("Loading test cases from %q ...", statfsMetricsTestCasesFile)
	testCases := make([]*StatfsMetricsTestCase, 0)
	err := testutils.LoadJsonFile(statfsMetricsTestCasesFile, &testCases)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range testCases {
		t.Run(
			tc.Name,
			func(t *testing.T) { testStatfsMetrics(tc, t) },
		)
	}
}
