package lsvmi

import (
	"bytes"
	"fmt"
	"testing"

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

type StatfsMetricsMountifoTestData struct {
	MountPoint  string
	MountSource string
	FsType      string
}

type UpdateStatfsInfoTestCase struct {
	name                 string
	instance             string
	hostname             string
	primeStatfsMountinfo []*StatfsMetricsMountifoTestData
	statfsMountinfo      []*StatfsMetricsMountifoTestData
	wantStatfsInfo       map[string]*StatfsInfo
}

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

	sfsm, err := NewStatfsMetrics(cfg)
	if err != nil {
		t.Fatal(err)
	}

	errBuf := &bytes.Buffer{}
	errFsType := make([]string, 0)

	errFsType = errFsType[:0]
	for _, fsType := range tc.wantKeep {
		if !sfsm.keepFsType(fsType) {
			errFsType = append(errFsType, fsType)
		}
	}
	if len(errFsType) > 0 {
		fmt.Fprintf(errBuf, "\nmissing keep: %q", errFsType)
	}

	errFsType = errFsType[:0]
	for _, fsType := range tc.wantNotKeep {
		if sfsm.keepFsType(fsType) {
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

func primeMountinfoParsedLinesForStatfsTest(
	sfsm *StatfsMetrics, mitdList []*StatfsMetricsMountifoTestData,
) {
	sfsm.procMountinfo.ParsedLines = make([]*procfs.MountinfoParsedLine, len(mitdList))
	for i, mitd := range mitdList {
		mountinfoParsedLine := procfs.MountinfoParsedLine{}
		mountinfoParsedLine[procfs.MOUNTINFO_MOUNT_POINT] = []byte(mitd.MountPoint)
		mountinfoParsedLine[procfs.MOUNTINFO_MOUNT_SOURCE] = []byte(mitd.MountSource)
		mountinfoParsedLine[procfs.MOUNTINFO_FS_TYPE] = []byte(mitd.FsType)
		sfsm.procMountinfo.ParsedLines[i] = &mountinfoParsedLine
	}
}

func testUpdateStatfsInfo(tc *UpdateStatfsInfoTestCase, t *testing.T) {
	tlc := testutils.NewTestLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	sfsm, err := NewStatfsMetrics(nil)
	if err != nil {
		t.Fatal(err)
	}
	sfsm.procMountinfo = procfs.NewMountinfo("", 0)
	sfsm.instance = tc.instance
	sfsm.hostname = tc.hostname

	wantOutOfScopePresentMetrics := make(map[string]bool)
	if tc.primeStatfsMountinfo != nil {
		primeMountinfoParsedLinesForStatfsTest(sfsm, tc.primeStatfsMountinfo)
		sfsm.updateStatfsInfo()
	}
	if tc.statfsMountinfo != nil {
		for _, statfsInfo := range sfsm.statfsInfo {
			wantOutOfScopePresentMetrics[string(statfsInfo.enabledMetric)] = true
		}
		primeMountinfoParsedLinesForStatfsTest(sfsm, tc.statfsMountinfo)
		sfsm.updateStatfsInfo()
		for _, statfsInfo := range sfsm.statfsInfo {
			delete(wantOutOfScopePresentMetrics, string(statfsInfo.enabledMetric))
		}
	}

	errBuf := &bytes.Buffer{}

	for mountSource, wantStatfsInfo := range tc.wantStatfsInfo {
		gotStatfsInfo := sfsm.statfsInfo[mountSource]
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

	for mountSource := range sfsm.statfsInfo {
		if tc.wantStatfsInfo[mountSource] == nil {
			fmt.Fprintf(errBuf, "\n.statfsInfo[%q]: unexpected", mountSource)
		}
	}
	for i, metric := range sfsm.outOfScopePresentMetrics {
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
			statfsMountinfo: []*StatfsMetricsMountifoTestData{
				{"/m1", "/dev/1", "fs1"},
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
			statfsMountinfo: []*StatfsMetricsMountifoTestData{
				{"/m1", "/dev/1", "fs1"},
				{"/m1-1", "/dev/1", "fs1"},
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
			primeStatfsMountinfo: []*StatfsMetricsMountifoTestData{
				{"/m1-before", "/dev/1", "fs1-before"},
			},
			statfsMountinfo: []*StatfsMetricsMountifoTestData{
				{"/m1", "/dev/1", "fs1"},
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
			primeStatfsMountinfo: []*StatfsMetricsMountifoTestData{
				{"/m2", "/dev/2", "fs2"},
			},
			statfsMountinfo: []*StatfsMetricsMountifoTestData{
				{"/m1", "/dev/1", "fs1"},
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
