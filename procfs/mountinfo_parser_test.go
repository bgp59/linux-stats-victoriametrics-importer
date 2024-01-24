package procfs

import (
	"bytes"
	"fmt"
	"path"
	"testing"
)

type MountinfoTestCase struct {
	name             string
	procfsRoot       string
	wantDevMountInfo map[string][]string
	wantError        error
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

	wantDevMountInfo := tc.wantDevMountInfo
	if wantDevMountInfo == nil {
		return
	}

	diffBuf := &bytes.Buffer{}

	for majorMinor, wantInfo := range wantDevMountInfo {
		gotInfo := mountinfo.DevMountInfo[majorMinor]
		if gotInfo == nil {
			fmt.Fprintf(
				diffBuf,
				"\nDevMountInfo[%q]: missing device",
				majorMinor,
			)
			continue
		}
		if len(wantInfo) != len(gotInfo) {
			fmt.Fprintf(
				diffBuf,
				"\nlen(DevMountInfo[%q]): want: %d, got: %d",
				majorMinor, len(wantInfo), len(gotInfo),
			)
			continue
		}
		for j, wantOpt := range wantInfo {
			gotOpt := string(gotInfo[j])
			if wantOpt != gotOpt {
				fmt.Fprintf(
					diffBuf,
					"\nDevMountInfo[%q][%s]: want: %q, got: %q",
					majorMinor, mountinfoIndexName[j], wantOpt, gotOpt,
				)
			}
		}
	}

	for majorMinor := range mountinfo.DevMountInfo {
		_, ok := wantDevMountInfo[majorMinor]
		if !ok {
			fmt.Fprintf(
				diffBuf,
				"\nDevMountInfoIndex[%s]: unexpected device",
				majorMinor,
			)
		}
	}
	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf.String())
	}
}

func TestMountinfoParser(t *testing.T) {
	for i, tc := range []*MountinfoTestCase{
		&MountinfoTestCase{
			procfsRoot: path.Join(mountinfoTestdataDir, "field_mapping"),
			wantDevMountInfo: map[string][]string{
				"11:0": {"10", "1", "11:0", "/root200", "/mount_point200", "mount,options=200", "value200:tag200", "-", "fstype200", "dev200", "super,options=200"},
				"21:0": {"20", "1", "21:0", "/root210", "/mount_point210", "mount,options=210", "", "-", "fstype20", "dev20", "super,options=210"},
				"31:0": {"30", "1", "31:0", "/root310", "/mount_point310", "mount,options=310", "value310:tag310 value311:tag311", "-", "fstype30", "dev30", "super,options=310"},
			},
		},
	} {
		var name string
		if tc.name != "" {
			name = fmt.Sprintf("tc=%d,name=%s,procfsRoot=%s", i, tc.name, tc.procfsRoot)
		} else {
			name = fmt.Sprintf("tc=%d,procfsRoot=%s", i, tc.procfsRoot)
		}
		t.Run(
			name,
			func(t *testing.T) { testMountinfoParser(tc, t) },
		)
	}
}
