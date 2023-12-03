package procfs

import (
	"bytes"
	"fmt"
	"path"
	"testing"
)

type MountinfoTestCase struct {
	name                  string
	procfsRoot            string
	wantDevMountInfo      [][]string
	wantDevMountInfoIndex map[string]int
	wantError             error
}

var mountinfoIndexName = []string{
	"MOUNTINFO_MOUNT_ID",
	"MOUNTINFO_PARENT_ID",
	"MOUNTINFO_MAJOR_MINOR",
	"MOUNTINFO_ROOT",
	"MOUNTINFO_MOUNT_POINT",
	"MOUNTINFO_MOUNT_OPTIONS",
	"MOUNTINFO_OPTIONAL_FIELDS",
	"MOUNTINFO_OPTIONAL_FIELDS_SEPARATOR",
	"MOUNTINFO_FS_TYPE",
	"MOUNTINFO_MOUNT_SOURCE",
	"MOUNTINFO_SUPER_OPTIONS",
}

var mountinfoTestdataDir = path.Join(PROCFS_TESTDATA_ROOT, "mountinfo")

func testMountinfoParser(tc *MountinfoTestCase, t *testing.T) {
	mountinfo := NewMountInfo(tc.procfsRoot, 1)
	err := mountinfo.Parse()
	if tc.wantError != nil {
		if err == nil || tc.wantError.Error() != err.Error() {
			t.Fatalf("want: %v error, got: %v", tc.wantError, err)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}

	wantDevMountinfo := tc.wantDevMountInfo
	if wantDevMountinfo == nil {
		return
	}

	if len(wantDevMountinfo) != len(mountinfo.DevMountInfo) {
		t.Fatalf(
			"len(DevMountInfo): want: %d, got: %d",
			len(wantDevMountinfo), len(mountinfo.DevMountInfo),
		)
	}

	diffBuf := &bytes.Buffer{}

	buf := mountinfo.content.Bytes()
	for i, wantInfo := range wantDevMountinfo {
		gotInfo := mountinfo.DevMountInfo[i]
		if len(wantInfo) != len(gotInfo) {
			fmt.Fprintf(
				diffBuf,
				"\nlen(DevMountInfo[%d]): want: %d, got: %d",
				i, len(wantInfo), len(gotInfo),
			)
			continue
		}
		for j, wantOpt := range wantInfo {
			startEnd := gotInfo[j]
			gotOpt := string(buf[startEnd.Start:startEnd.End])
			if wantOpt != gotOpt {
				fmt.Fprintf(
					diffBuf,
					"\nDevMountInfo[%d][%d (%s)]: want: %q, got: %q",
					i, j, mountinfoIndexName[j], wantOpt, gotOpt,
				)
			}
		}
	}

	wantDevMountInfoIndex := tc.wantDevMountInfoIndex
	if wantDevMountInfoIndex != nil {
		for dev, wantI := range wantDevMountInfoIndex {
			gotI, ok := mountinfo.DevMountInfoIndex[dev]
			if !ok {
				fmt.Fprintf(
					diffBuf,
					"\nDevMountInfoIndex[%s]: missing device",
					dev,
				)
				continue
			} else if wantI != gotI {
				fmt.Fprintf(
					diffBuf,
					"\nDevMountInfoIndex[%s]: want: %d, got: %d",
					dev, wantI, gotI,
				)
			}
		}
		for dev := range mountinfo.DevMountInfoIndex {
			_, ok := wantDevMountInfoIndex[dev]
			if !ok {
				fmt.Fprintf(
					diffBuf,
					"\nDevMountInfoIndex[%s]: unexpected device",
					dev,
				)
			}
		}
	}
	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf.String())
	}
}

func TestMountinfoParser(t *testing.T) {
	for _, tc := range []*MountinfoTestCase{
		&MountinfoTestCase{
			procfsRoot: path.Join(mountinfoTestdataDir, "field_mapping"),
			wantDevMountInfo: [][]string{
				{"10", "1", "20:0", "/root200", "/mount_point200", "mount,options=200", "value200:tag200", "-", "fstype200", "dev200", "super,options=200"},
				{"21", "1", "21:0", "/root210", "/mount_point210", "mount,options=210", "", "-", "fstype20", "dev20", "super,options=210"},
			},
			wantDevMountInfoIndex: map[string]int{
				"20:0": 0,
				"21:0": 1,
			},
		},
	} {
		var name string
		if tc.name != "" {
			name = fmt.Sprintf("name=%s,procfsRoot=%s", tc.name, tc.procfsRoot)
		} else {
			name = fmt.Sprintf("procfsRoot=%s", tc.procfsRoot)
		}
		t.Run(
			name,
			func(t *testing.T) { testMountinfoParser(tc, t) },
		)
	}
}
