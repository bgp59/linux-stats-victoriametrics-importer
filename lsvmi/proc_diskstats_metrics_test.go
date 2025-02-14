package lsvmi

import (
	"bytes"
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/bgp59/linux-stats-victoriametrics-importer/internal/testutils"
	"github.com/bgp59/linux-stats-victoriametrics-importer/procfs"
)

type ProcMountinfoMetricsCacheUpdateTestCase struct {
	Name                           string
	Description                    string
	Instance                       string
	Hostname                       string
	Pid                            int
	ParsedLines                    []map[int]string
	DiskMajMinList                 []string
	PrimeMountinfoMetricsCache     []string
	WantMountinfoMetricsCache      []string
	WantMountinfoOutOfScopeMetrics []string
}

type ProcDiskstatsMetricsInfoTest struct {
	CycleNum     int
	ZeroDelta    []bool
	MetricsCache []string
	InfoMetric   string
}

type ProcDiskstatsMetricsTestCase struct {
	Name                                 string
	Description                          string
	Instance                             string
	Hostname                             string
	MountinfoPid                         int
	CurrProcDiskstats, PrevProcDiskstats *procfs.Diskstats
	CurrPromTs, PrevPromTs               int64
	PrimeDiskstatsMetricsInfo            map[string]*ProcDiskstatsMetricsInfoTest
	MountifoParsedLines                  []map[int]string
	MountinfoChanged                     bool
	PrimeMountinfoMetricsCache           []string
	MountinfoCycleNum                    int
	FullMetricsFactor                    int
	WantMetricsCount                     int
	WantMetrics                          []string
	ReportExtra                          bool
	WantZeroDeltaMap                     map[string][]bool
}

var procDiskstatsMetricsTestCasesFile = path.Join(
	"..", testutils.LsvmiTestCasesSubdir,
	"proc_diskstats.json",
)

func makeProcMountinfo(parsedLines []map[int]string) *procfs.Mountinfo {
	procMountinfo := procfs.NewMountinfo("", -1)
	for _, parsedLine := range parsedLines {
		mountinfoParsedLine := procfs.MountinfoParsedLine{}
		for i, info := range parsedLine {
			mountinfoParsedLine[i] = []byte(info)
		}
		procMountinfo.ParsedLines = append(procMountinfo.ParsedLines, &mountinfoParsedLine)
	}
	return procMountinfo
}

func makeProcDiskstatsMetrics(tc *ProcDiskstatsMetricsTestCase) (*ProcDiskstatsMetrics, error) {
	pdsm, err := NewProcDiskstatsMetrics(nil)
	if err != nil {
		return nil, err
	}

	pdsm.instance = tc.Instance
	pdsm.hostname = tc.Hostname
	pdsm.mountinfoPid = tc.MountinfoPid

	pdsm.procDiskstats[pdsm.currIndex] = tc.CurrProcDiskstats
	pdsm.procDiskstatsTs[pdsm.currIndex] = time.UnixMilli(tc.CurrPromTs)
	pdsm.procDiskstats[1-pdsm.currIndex] = tc.PrevProcDiskstats
	pdsm.procDiskstatsTs[1-pdsm.currIndex] = time.UnixMilli(tc.PrevPromTs)

	for majMin, infoTC := range tc.PrimeDiskstatsMetricsInfo {
		info := &ProcDiskstatsMetricsInfo{
			cycleNum: infoTC.CycleNum,
		}
		if infoTC.MetricsCache != nil {
			info.metricsCache = make([][]byte, len(infoTC.MetricsCache))
			for i, metric := range infoTC.MetricsCache {
				info.metricsCache[i] = []byte(metric)
			}
		}
		if infoTC.ZeroDelta != nil {
			info.zeroDelta = make([]bool, len(infoTC.ZeroDelta))
			copy(info.zeroDelta, infoTC.ZeroDelta)
		}
		if infoTC.InfoMetric != "" {
			info.infoMetric = []byte(infoTC.InfoMetric)
		}
		pdsm.diskstatsMetricsInfo[majMin] = info
	}

	if tc.MountifoParsedLines != nil {
		pdsm.procMountinfo = makeProcMountinfo(tc.MountifoParsedLines)
		pdsm.procMountinfo.Changed = tc.MountinfoChanged
		pdsm.mountinfoCycleNum = tc.MountinfoCycleNum
	} else {
		pdsm.mountifoDisabled = true
	}
	if tc.PrimeMountinfoMetricsCache != nil {
		pdsm.mountinfoMetricsCache = make([][]byte, len(tc.PrimeMountinfoMetricsCache))
		for i, metric := range tc.PrimeMountinfoMetricsCache {
			pdsm.mountinfoMetricsCache[i] = []byte(metric)
		}
	}

	pdsm.fullMetricsFactor = tc.FullMetricsFactor

	return pdsm, nil
}

func testProcMountinfoMetricsCacheUpdate(tc *ProcMountinfoMetricsCacheUpdateTestCase, t *testing.T) {
	tlc := testutils.NewTestLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	if tc.Description != "" {
		t.Logf("Description: %s", tc.Description)
	}

	pdsm, err := NewProcDiskstatsMetrics(nil)
	if err != nil {
		t.Fatal(err)
	}
	pdsm.instance = tc.Instance
	pdsm.hostname = tc.Hostname
	pdsm.mountinfoPid = tc.Pid

	procDiskstats := procfs.NewDiskstats("")
	procDiskstats.DevInfoMap = make(map[string]*procfs.DiskstatsDevInfo)
	if tc.DiskMajMinList != nil {
		for _, majMin := range tc.DiskMajMinList {
			procDiskstats.DevInfoMap[majMin] = &procfs.DiskstatsDevInfo{}
		}
	} else {
		for _, parsedLine := range tc.ParsedLines {
			procDiskstats.DevInfoMap[parsedLine[procfs.MOUNTINFO_MAJOR_MINOR]] = &procfs.DiskstatsDevInfo{}
		}
	}
	pdsm.procDiskstats[pdsm.currIndex] = procDiskstats

	pdsm.procMountinfo = makeProcMountinfo(tc.ParsedLines)
	if tc.PrimeMountinfoMetricsCache != nil {
		pdsm.mountinfoMetricsCache = make([][]byte, len(tc.PrimeMountinfoMetricsCache))
		for i, metric := range tc.PrimeMountinfoMetricsCache {
			pdsm.mountinfoMetricsCache[i] = []byte(metric)
		}
	}

	gotMountinfoOutOfScopeMetrics := pdsm.updateMountinfoMetricsCache()
	wantMountinfoOutOfScopeMetrics := make(map[string]bool)
	for _, metric := range tc.WantMountinfoOutOfScopeMetrics {
		wantMountinfoOutOfScopeMetrics[metric] = true
	}
	gotMountinfoMetricsCache := make(map[string]bool)
	for _, metric := range pdsm.mountinfoMetricsCache {
		gotMountinfoMetricsCache[string(metric)] = true
	}
	wantMountinfoMetricsCache := make(map[string]bool)
	for _, metric := range tc.WantMountinfoMetricsCache {
		wantMountinfoMetricsCache[metric] = true
	}

	errBuf := &bytes.Buffer{}

	for metric := range wantMountinfoMetricsCache {
		if _, ok := gotMountinfoMetricsCache[metric]; !ok {
			fmt.Fprintf(errBuf, "\n.mountinfoMetricsCache: missing: %q", metric)
		}
	}
	for metric := range gotMountinfoMetricsCache {
		if _, ok := wantMountinfoMetricsCache[metric]; !ok {
			fmt.Fprintf(errBuf, "\n.mountinfoMetricsCache: unexpected: %q", metric)
		}
	}

	for metric := range wantMountinfoOutOfScopeMetrics {
		if _, ok := gotMountinfoOutOfScopeMetrics[metric]; !ok {
			fmt.Fprintf(errBuf, "\nmountinfoOutOfScopeMetrics: missing: %q", metric)
		}
	}
	for metric := range gotMountinfoOutOfScopeMetrics {
		if _, ok := wantMountinfoOutOfScopeMetrics[metric]; !ok {
			fmt.Fprintf(errBuf, "\nmountinfoOutOfScopeMetrics: unexpected %q", metric)
		}
	}

	if errBuf.Len() > 0 {
		t.Fatal(errBuf)
	}
}

func testProcDiskstatsMetrics(tc *ProcDiskstatsMetricsTestCase, t *testing.T) {
	tlc := testutils.NewTestLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	if tc.Description != "" {
		t.Logf("Description: %s", tc.Description)
	}
	pdsm, err := makeProcDiskstatsMetrics(tc)
	if err != nil {
		t.Fatal(err)
	}

	wantCurrIndex := 1 - pdsm.currIndex
	testMetricsQueue := testutils.NewTestMetricsQueue(0)
	buf := testMetricsQueue.GetBuf()
	gotMetricsCount, _ := pdsm.generateMetrics(buf)
	testMetricsQueue.QueueBuf(buf)

	errBuf := &bytes.Buffer{}

	gotCurrIndex := pdsm.currIndex
	if wantCurrIndex != gotCurrIndex {
		fmt.Fprintf(
			errBuf,
			"\ncurrIndex: want: %d, got: %d",
			wantCurrIndex, gotCurrIndex,
		)
	}

	if tc.WantZeroDeltaMap != nil {
		for majMin, wantZeroDelta := range tc.WantZeroDeltaMap {
			info := pdsm.diskstatsMetricsInfo[majMin]
			if info == nil {
				fmt.Fprintf(errBuf, "\nzeroDelta: missing %q", majMin)
				continue
			}
			gotZeroDelta := info.zeroDelta
			if len(wantZeroDelta) != len(gotZeroDelta) {
				fmt.Fprintf(
					errBuf,
					"\nzeroDelta[%q] length: want: %d, got: %d",
					majMin, len(wantZeroDelta), len(gotZeroDelta),
				)
				continue
			}
			for i, wantVal := range wantZeroDelta {
				gotVal := gotZeroDelta[i]
				if wantVal != gotVal {
					fmt.Fprintf(
						errBuf,
						"\nzeroDelta[%q][%d]: want: %v, got: %v",
						majMin, i, wantVal, gotVal,
					)
				}
			}
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

func TestProcMountinfoMetricsCacheUpdate(t *testing.T) {
	for _, tc := range []*ProcMountinfoMetricsCacheUpdateTestCase{
		{
			Name:     "initial",
			Instance: "test_lsvmi",
			Hostname: "test-lsvmi",
			Pid:      -1,
			ParsedLines: []map[int]string{
				{
					procfs.MOUNTINFO_MAJOR_MINOR:  "100:0",
					procfs.MOUNTINFO_ROOT:         "/100:0",
					procfs.MOUNTINFO_MOUNT_POINT:  "/mout/100/0",
					procfs.MOUNTINFO_FS_TYPE:      "fs100_0",
					procfs.MOUNTINFO_MOUNT_SOURCE: "/dev/disk/100_0",
				},
				{
					procfs.MOUNTINFO_MAJOR_MINOR:  "100:1",
					procfs.MOUNTINFO_ROOT:         "/100:1",
					procfs.MOUNTINFO_MOUNT_POINT:  "/mout/100/1",
					procfs.MOUNTINFO_FS_TYPE:      "fs100_1",
					procfs.MOUNTINFO_MOUNT_SOURCE: "/dev/disk/100_1",
				},
			},
			WantMountinfoMetricsCache: []string{
				`proc_mountinfo{instance="test_lsvmi",hostname="test-lsvmi",pid="-1",maj_min="100:0",root="/100:0",mount_point="/mout/100/0",fs_type="fs100_0",source="/dev/disk/100_0"} `,
				`proc_mountinfo{instance="test_lsvmi",hostname="test-lsvmi",pid="-1",maj_min="100:1",root="/100:1",mount_point="/mout/100/1",fs_type="fs100_1",source="/dev/disk/100_1"} `,
			},
		},
		{
			Name:     "add",
			Instance: "test_lsvmi",
			Hostname: "test-lsvmi",
			Pid:      -1,
			ParsedLines: []map[int]string{
				{
					procfs.MOUNTINFO_MAJOR_MINOR:  "100:0",
					procfs.MOUNTINFO_ROOT:         "/100:0",
					procfs.MOUNTINFO_MOUNT_POINT:  "/mout/100/0",
					procfs.MOUNTINFO_FS_TYPE:      "fs100_0",
					procfs.MOUNTINFO_MOUNT_SOURCE: "/dev/disk/100_0",
				},
				{
					procfs.MOUNTINFO_MAJOR_MINOR:  "100:1",
					procfs.MOUNTINFO_ROOT:         "/100:1",
					procfs.MOUNTINFO_MOUNT_POINT:  "/mout/100/1",
					procfs.MOUNTINFO_FS_TYPE:      "fs100_1",
					procfs.MOUNTINFO_MOUNT_SOURCE: "/dev/disk/100_1",
				},
			},
			PrimeMountinfoMetricsCache: []string{
				`proc_mountinfo{instance="test_lsvmi",hostname="test-lsvmi",pid="-1",maj_min="100:0",root="/100:0",mount_point="/mout/100/0",fs_type="fs100_0",source="/dev/disk/100_0"} `,
			},
			WantMountinfoMetricsCache: []string{
				`proc_mountinfo{instance="test_lsvmi",hostname="test-lsvmi",pid="-1",maj_min="100:0",root="/100:0",mount_point="/mout/100/0",fs_type="fs100_0",source="/dev/disk/100_0"} `,
				`proc_mountinfo{instance="test_lsvmi",hostname="test-lsvmi",pid="-1",maj_min="100:1",root="/100:1",mount_point="/mout/100/1",fs_type="fs100_1",source="/dev/disk/100_1"} `,
			},
		},
		{
			Name:     "remove",
			Instance: "test_lsvmi",
			Hostname: "test-lsvmi",
			Pid:      -1,
			ParsedLines: []map[int]string{
				{
					procfs.MOUNTINFO_MAJOR_MINOR:  "100:0",
					procfs.MOUNTINFO_ROOT:         "/100:0",
					procfs.MOUNTINFO_MOUNT_POINT:  "/mout/100/0",
					procfs.MOUNTINFO_FS_TYPE:      "fs100_0",
					procfs.MOUNTINFO_MOUNT_SOURCE: "/dev/disk/100_0",
				},
				{
					procfs.MOUNTINFO_MAJOR_MINOR:  "100:1",
					procfs.MOUNTINFO_ROOT:         "/100:1",
					procfs.MOUNTINFO_MOUNT_POINT:  "/mout/100/1",
					procfs.MOUNTINFO_FS_TYPE:      "fs100_1",
					procfs.MOUNTINFO_MOUNT_SOURCE: "/dev/disk/100_1",
				},
			},
			PrimeMountinfoMetricsCache: []string{
				`proc_mountinfo{instance="test_lsvmi",hostname="test-lsvmi",pid="-1",maj_min="100:0",root="/100:0",mount_point="/mout/100/0",fs_type="fs100_0",source="/dev/disk/100_0"} `,
				`proc_mountinfo{instance="test_lsvmi",hostname="test-lsvmi",pid="-1",maj_min="100:1",root="/100:1",mount_point="/mout/100/1",fs_type="fs100_1",source="/dev/disk/100_1"} `,
				`proc_mountinfo{instance="test_lsvmi",hostname="test-lsvmi",pid="-1",maj_min="100:2",root="/100:2",mount_point="/mout/100/2",fs_type="fs100_2",source="/dev/disk/100_2"} `,
			},
			WantMountinfoMetricsCache: []string{
				`proc_mountinfo{instance="test_lsvmi",hostname="test-lsvmi",pid="-1",maj_min="100:0",root="/100:0",mount_point="/mout/100/0",fs_type="fs100_0",source="/dev/disk/100_0"} `,
				`proc_mountinfo{instance="test_lsvmi",hostname="test-lsvmi",pid="-1",maj_min="100:1",root="/100:1",mount_point="/mout/100/1",fs_type="fs100_1",source="/dev/disk/100_1"} `,
			},
			WantMountinfoOutOfScopeMetrics: []string{
				`proc_mountinfo{instance="test_lsvmi",hostname="test-lsvmi",pid="-1",maj_min="100:2",root="/100:2",mount_point="/mout/100/2",fs_type="fs100_2",source="/dev/disk/100_2"} `,
			},
		},
	} {
		t.Run(
			tc.Name,
			func(t *testing.T) { testProcMountinfoMetricsCacheUpdate(tc, t) },
		)
	}
}

func TestProcDiskstatsMetrics(t *testing.T) {
	t.Logf("Loading test cases from %q ...", procDiskstatsMetricsTestCasesFile)
	testCases := make([]*ProcDiskstatsMetricsTestCase, 0)
	err := testutils.LoadJsonFile(procDiskstatsMetricsTestCasesFile, &testCases)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range testCases {
		t.Run(
			tc.Name,
			func(t *testing.T) { testProcDiskstatsMetrics(tc, t) },
		)
	}
}
