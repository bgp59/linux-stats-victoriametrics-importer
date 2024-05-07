package procfs

import (
	"bytes"
	"fmt"
	"path"
	"testing"
)

type MountinfoTestCase struct {
	name            string
	procfsRoot      string
	wantParsedLines [][]string
	wantError       error
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
	t.Logf(`
name=%q
procfsRoot=%q
`,
		tc.name, tc.procfsRoot,
	)

	mountinfo := NewMountinfo(tc.procfsRoot, 1)
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

	wantParsedLines := tc.wantParsedLines
	if wantParsedLines == nil {
		return
	}

	gotParsedLines := mountinfo.ParsedLines
	if len(wantParsedLines) != len(gotParsedLines) {
		t.Fatalf("len(ParsedLines): want %d, got: %d", len(wantParsedLines), len(gotParsedLines))
	}

	diffBuf := &bytes.Buffer{}

	for i, wantParsedLine := range wantParsedLines {
		gotParsedLine := gotParsedLines[i]
		if len(wantParsedLine) != len(gotParsedLine) {
			fmt.Fprintf(
				diffBuf,
				"\nlen(ParsedLines[%d]): want: %d, got: %d",
				i, len(wantParsedLine), len(gotParsedLine),
			)
			continue
		}
		for j, wantOpt := range wantParsedLine {
			gotOpt := string(gotParsedLine[j])
			if wantOpt != gotOpt {
				fmt.Fprintf(
					diffBuf,
					"\nParsedLines[%d][%d (%s)]: want: %q, got: %q",
					i, j, mountinfoIndexName[j], wantOpt, gotOpt,
				)
			}
		}
	}
	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf.String())
	}

	// 2nd time around there should be no change.
	err = mountinfo.Parse()
	if err != nil {
		t.Fatal(err)
	}
	if mountinfo.Changed {
		t.Fatalf("Changed: %v", mountinfo.Changed)
	}
}

func TestMountinfoParser(t *testing.T) {
	for _, tc := range []*MountinfoTestCase{
		{
			name:       "field_mapping",
			procfsRoot: path.Join(mountinfoTestdataDir, "field_mapping"),
			wantParsedLines: [][]string{
				{"10", "1", "11:0", "/root200", "/mount_point200", "mount,options=200", "value200:tag200", "-", "fstype200", "dev200", "super,options=200"},
				{"20", "1", "21:0", "/root210", "/mount_point210", "mount,options=210", "", "-", "fstype20", "dev20", "super,options=210"},
				{"30", "1", "31:0", "/root310", "/mount_point310", "mount,options=310", "value310:tag310 value311:tag311", "-", "fstype30", "dev30", "super,options=310"},
			},
		},
	} {
		t.Run(
			tc.name,
			func(t *testing.T) { testMountinfoParser(tc, t) },
		)
	}
}
