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
	MountPoint            string
	MountSource           string
	FsType                string
	Statfs                *unix.Statfs_t
	CycleNum              int
	Disabled, WasDisabled bool
}

type UpdateStatfsInfoTestCase struct {
	name                 string
	instance             string
	hostname             string
	primeStatfsMountinfo []*StatfsInfoTestData
	statfsMountinfo      []*StatfsInfoTestData
	wantStatfsInfo       map[string]*StatfsInfo
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

func initStatfsMetricsStatfsInfo(sfsm *StatfsMetrics, sfsiTd []*StatfsInfoTestData, update bool) {
	sfsm.procMountinfo.ParsedLines = make([]*procfs.MountinfoParsedLine, len(sfsiTd))
	for i, sfsi := range sfsiTd {
		mountinfoParsedLine := procfs.MountinfoParsedLine{}
		mountinfoParsedLine[procfs.MOUNTINFO_MOUNT_POINT] = []byte(sfsi.MountPoint)
		mountinfoParsedLine[procfs.MOUNTINFO_MOUNT_SOURCE] = []byte(sfsi.MountSource)
		mountinfoParsedLine[procfs.MOUNTINFO_FS_TYPE] = []byte(sfsi.FsType)
		sfsm.procMountinfo.ParsedLines[i] = &mountinfoParsedLine
	}
	if update {
		sfsm.updateStatfsInfo()
	}

	for _, primeSfsi := range sfsiTd {
		sfsi := sfsm.statfsInfo[primeSfsi.MountSource]
		if primeSfsi.Statfs != nil {
			sfsi.statfsBuf[sfsm.currIndex] = new(unix.Statfs_t)
			*sfsi.statfsBuf[sfsm.currIndex] = *primeSfsi.Statfs
		} else {
			sfsi.statfsBuf[sfsm.currIndex] = nil
		}
		sfsi.cycleNum = primeSfsi.CycleNum
		sfsi.disabled = primeSfsi.Disabled
		sfsi.wasDisabled = primeSfsi.WasDisabled
	}
}

func testUpdateStatfsInfo(tc *UpdateStatfsInfoTestCase, t *testing.T) {
	tlc := testutils.NewTestLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	statfsMetrics, err := NewStatfsMetrics(nil)
	if err != nil {
		t.Fatal(err)
	}
	statfsMetrics.procMountinfo = procfs.NewMountinfo("", 0)
	statfsMetrics.instance = tc.instance
	statfsMetrics.hostname = tc.hostname

	wantOutOfScopePresentMetrics := make(map[string]bool)
	if tc.primeStatfsMountinfo != nil {
		initStatfsMetricsStatfsInfo(statfsMetrics, tc.primeStatfsMountinfo, true)
	}
	if tc.statfsMountinfo != nil {
		for _, statfsInfo := range statfsMetrics.statfsInfo {
			wantOutOfScopePresentMetrics[string(statfsInfo.enabledMetric)] = true
		}
		initStatfsMetricsStatfsInfo(statfsMetrics, tc.statfsMountinfo, true)
		for _, statfsInfo := range statfsMetrics.statfsInfo {
			delete(wantOutOfScopePresentMetrics, string(statfsInfo.enabledMetric))
		}
	}

	errBuf := &bytes.Buffer{}

	for mountSource, wantStatfsInfo := range tc.wantStatfsInfo {
		gotStatfsInfo := statfsMetrics.statfsInfo[mountSource]
		if gotStatfsInfo == nil {
			fmt.Fprintf(errBuf, "\n.statfsInfo[%q]: missing", mountSource)
			continue
		}
		if wantStatfsInfo.mountSource != gotStatfsInfo.mountSource {
			fmt.Fprintf(
				errBuf,
				"\n.statfsInfo[%q]: mountSource: want: %q, got %q",
				mountSource, wantStatfsInfo.mountSource, gotStatfsInfo.mountSource,
			)
		}
		if wantStatfsInfo.mountPoint != gotStatfsInfo.mountPoint {
			fmt.Fprintf(
				errBuf,
				"\n.statfsInfo[%q]: mountPoint: want: %q, got %q",
				mountSource, wantStatfsInfo.mountPoint, gotStatfsInfo.mountPoint,
			)
		}
		if wantStatfsInfo.fsType != gotStatfsInfo.fsType {
			fmt.Fprintf(
				errBuf,
				"\n.statfsInfo[%q]: fsType: want: %q, got %q",
				mountSource, wantStatfsInfo.fsType, gotStatfsInfo.fsType,
			)
		}
	}

	for mountSource := range statfsMetrics.statfsInfo {
		if tc.wantStatfsInfo[mountSource] == nil {
			fmt.Fprintf(errBuf, "\n.statfsInfo[%q]: unexpected", mountSource)
		}
	}
	for i, metric := range statfsMetrics.outOfScopePresentMetrics {
		gotMetric := string(metric)
		if !wantOutOfScopePresentMetrics[gotMetric] {
			fmt.Fprintf(
				errBuf,
				"\n.outOfScopePresentMetrics[%d]=%q: unexpected",
				i, gotMetric,
			)
		} else {
			delete(wantOutOfScopePresentMetrics, gotMetric)
		}
	}
	for metric := range wantOutOfScopePresentMetrics {
		fmt.Fprintf(errBuf, "\n.outOfScopePresentMetrics: %q missing", metric)
	}

	if errBuf.Len() > 0 {
		t.Fatal(errBuf)
	}
}

func testStatfsMountinfoChanged(currSfsiTd, prevSfsiTd []*StatfsInfoTestData) bool {
	if len(currSfsiTd) != len(prevSfsiTd) {
		return true
	}

	byMountSource := map[string]*StatfsInfoTestData{}
	for _, sfsi := range currSfsiTd {
		byMountSource[sfsi.MountSource] = sfsi
	}

	// Look for new/changes:
	for _, prevSfsi := range prevSfsiTd {
		currSfsi := byMountSource[prevSfsi.MountSource]
		if currSfsi == nil || currSfsi.MountPoint != prevSfsi.MountPoint || currSfsi.FsType != prevSfsi.FsType {
			return true
		}
		delete(byMountSource, prevSfsi.MountSource)
	}

	// Look for disappearances:
	return len(byMountSource) > 0
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
		initStatfsMetricsStatfsInfo(statfsMetrics, tc.PrevStatfsInfo, true)
		statfsMetrics.currIndex = 1 - statfsMetrics.currIndex
	}

	initStatfsMetricsStatfsInfo(
		statfsMetrics,
		tc.CurrStatfsInfo,
		testStatfsMountinfoChanged(tc.CurrStatfsInfo, tc.PrevStatfsInfo),
	)

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

func TestUpdateStatfsInfo(t *testing.T) {
	instance, hostname := "test_lsvmi", "test-lsvmi"
	for _, tc := range []*UpdateStatfsInfoTestCase{
		{
			name:     "new",
			instance: instance,
			hostname: hostname,
			statfsMountinfo: []*StatfsInfoTestData{
				{MountPoint: "/m1", MountSource: "/dev/1", FsType: "fs1"},
			},
			wantStatfsInfo: map[string]*StatfsInfo{
				"/dev/1": {
					mountPoint:  "/m1",
					mountSource: "/dev/1",
					fsType:      "fs1",
				},
			},
		},
		{
			name:     "duplicate_mount_source",
			instance: instance,
			hostname: hostname,
			statfsMountinfo: []*StatfsInfoTestData{
				{MountPoint: "/m1", MountSource: "/dev/1", FsType: "fs1"},
				{MountPoint: "/m1-1", MountSource: "/dev/1", FsType: "fs1"},
			},
			wantStatfsInfo: map[string]*StatfsInfo{
				"/dev/1": {
					mountPoint:  "/m1",
					mountSource: "/dev/1",
					fsType:      "fs1",
				},
			},
		},
		{
			name:     "update_mount_source",
			instance: instance,
			hostname: hostname,
			primeStatfsMountinfo: []*StatfsInfoTestData{
				{MountPoint: "/m1-before", MountSource: "/dev/1", FsType: "fs1-before"},
			},
			statfsMountinfo: []*StatfsInfoTestData{
				{MountPoint: "/m1", MountSource: "/dev/1", FsType: "fs1"},
			},
			wantStatfsInfo: map[string]*StatfsInfo{
				"/dev/1": {
					mountPoint:  "/m1",
					mountSource: "/dev/1",
					fsType:      "fs1",
				},
			},
		},
		{
			name:     "remove_mount_source",
			instance: instance,
			hostname: hostname,
			primeStatfsMountinfo: []*StatfsInfoTestData{
				{MountPoint: "/m2", MountSource: "/dev/2", FsType: "fs2"},
			},
			statfsMountinfo: []*StatfsInfoTestData{
				{MountPoint: "/m1", MountSource: "/dev/1", FsType: "fs1"},
			},
			wantStatfsInfo: map[string]*StatfsInfo{
				"/dev/1": {
					mountPoint:  "/m1",
					mountSource: "/dev/1",
					fsType:      "fs1",
				},
			},
		},
	} {
		t.Run(
			tc.name,
			func(t *testing.T) { testUpdateStatfsInfo(tc, t) },
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
