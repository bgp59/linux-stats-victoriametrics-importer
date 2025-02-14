package procfs

import (
	"os"
	"path"
	"testing"

	"github.com/bgp59/linux-stats-victoriametrics-importer/internal/utils"
)

type PidCmdlineTestCase struct {
	name                           string
	procfsRoot                     string
	pid, tid                       int
	poolReadSize                   int64
	cmdline                        string
	wantCmdPath, wantArgs, wantCmd string
	wantError                      error
}

var pidCmdlineTestDataDir = path.Join(PROCFS_TESTDATA_ROOT, "pid_cmdline")

func testPidCmdlineParser(tc *PidCmdlineTestCase, t *testing.T) {
	t.Logf("\nprocfsRoot:=%q, pid=%d, tid=%d\ncmdline=%q (%d bytes)\npoolReadSize=%d",
		tc.procfsRoot, tc.pid, tc.tid, tc.cmdline, len(tc.cmdline), tc.poolReadSize,
	)

	if tc.poolReadSize > 0 {
		orgPidCmdlineReadFileBufPool := pidCmdlineReadFileBufPool
		pidCmdlineReadFileBufPool = utils.NewReadFileBufPool(
			orgPidCmdlineReadFileBufPool.MaxPoolSize(),
			tc.poolReadSize,
		)
		defer func() { pidCmdlineReadFileBufPool = orgPidCmdlineReadFileBufPool }()
	}

	pidTidPath := BuildPidTidPath(tc.procfsRoot, tc.pid, tc.tid)

	if tc.cmdline != "" {
		err := os.MkdirAll(pidTidPath, os.ModePerm)
		if err != nil {
			t.Fatal(err)
		}
		file, err := os.Create(path.Join(pidTidPath, "cmdline"))
		if err != nil {
			t.Fatal(err)
		}
		_, err = file.WriteString(tc.cmdline)
		if err != nil {
			t.Fatal(err)
		}
		file.Close()
	}

	pidCmdline := NewPidCmdline()
	err := pidCmdline.Parse(pidTidPath)
	if tc.wantError == nil && err != nil {
		t.Fatal(err)
	}
	if tc.wantError != nil {
		if err == nil || tc.wantError.Error() != err.Error() {
			t.Fatalf("error: want: %v, got: %v", tc.wantError, err)
		}
	}
	gotCmdPath, gotArgs, gotCmd := pidCmdline.GetData()
	got := string(gotCmdPath)
	if tc.wantCmdPath != got {
		t.Fatalf("cmdPath: want: %q, got: %q", tc.wantCmdPath, got)
	}
	got = string(gotArgs)
	if tc.wantArgs != got {
		t.Fatalf("args: want: %q, got: %q", tc.wantArgs, got)
	}
	got = string(gotCmd)
	if tc.wantCmd != got {
		t.Fatalf("cmd: want: %q, got: %q", tc.wantCmd, got)
	}
	t.Logf("\ncmdPath=%q\nargs=%q\ncmd=%q", gotCmdPath, gotArgs, gotCmd)

}

func TestPidCmdlineParser(t *testing.T) {
	for _, tc := range []*PidCmdlineTestCase{
		{
			procfsRoot:  pidCmdlineTestDataDir,
			pid:         1,
			tid:         PID_ONLY_TID,
			wantCmdPath: `/sbin/init`,
			wantCmd:     `init`,
		},
		{
			procfsRoot:  pidCmdlineTestDataDir,
			pid:         101,
			tid:         PID_ONLY_TID,
			cmdline:     "arg0\x00arg1\x00",
			wantCmdPath: `arg0`,
			wantArgs:    `arg1`,
			wantCmd:     `arg0`,
		},
		{
			procfsRoot:  pidCmdlineTestDataDir,
			pid:         101,
			tid:         101,
			cmdline:     "arg0\x00arg1\x00\x00",
			wantCmdPath: `arg0`,
			wantArgs:    `arg1`,
			wantCmd:     `arg0`,
		},
		{
			procfsRoot:  pidCmdlineTestDataDir,
			pid:         102,
			tid:         PID_ONLY_TID,
			cmdline:     "arg0\x00arg1\x00arg\n2\x00",
			wantCmdPath: `arg0`,
			wantArgs:    `arg1 arg\n2`,
			wantCmd:     `arg0`,
		},
		{
			procfsRoot:  pidCmdlineTestDataDir,
			pid:         103,
			tid:         PID_ONLY_TID,
			cmdline:     "arg0\x00arg1\x00\"arg 2\"\x00",
			wantCmdPath: `arg0`,
			wantArgs:    `arg1 \"arg 2\"`,
			wantCmd:     `arg0`,
		},
		{
			procfsRoot:   pidCmdlineTestDataDir,
			pid:          1001,
			tid:          PID_ONLY_TID,
			poolReadSize: 10,
			cmdline:      "arg0\x00arg1\x00arg2\x00",
			wantCmdPath:  `arg0`,
			wantArgs:     `ar...`,
			wantCmd:      `arg0`,
		},
		{
			procfsRoot:   pidCmdlineTestDataDir,
			pid:          1002,
			tid:          8,
			poolReadSize: 8,
			cmdline:      "Hello\x00世界\x00",
			wantCmdPath:  `Hello...`,
			wantCmd:      `Hello...`,
		},
		{
			procfsRoot:   pidCmdlineTestDataDir,
			pid:          1002,
			tid:          9,
			poolReadSize: 9,
			cmdline:      "Hello\x00世界\x00",
			wantCmdPath:  `Hello`,
			wantArgs:     `...`,
			wantCmd:      `Hello`,
		},
		{
			procfsRoot:   pidCmdlineTestDataDir,
			pid:          1002,
			tid:          10,
			poolReadSize: 10,
			cmdline:      "Hello\x00世界\x00",
			wantCmdPath:  `Hello`,
			wantArgs:     `...`,
			wantCmd:      `Hello`,
		},
		{
			procfsRoot:   pidCmdlineTestDataDir,
			pid:          1002,
			tid:          11,
			poolReadSize: 11,
			cmdline:      "Hello\x00世界\x00",
			wantCmdPath:  `Hello`,
			wantArgs:     `...`,
			wantCmd:      `Hello`,
		},
		{
			procfsRoot:   pidCmdlineTestDataDir,
			pid:          1002,
			tid:          12,
			poolReadSize: 12,
			cmdline:      "Hello\x00世界\x00",
			wantCmdPath:  `Hello`,
			wantArgs:     `世...`,
			wantCmd:      `Hello`,
		},
		{
			procfsRoot:   pidCmdlineTestDataDir,
			pid:          1002,
			tid:          13,
			poolReadSize: 13,
			cmdline:      "Hello\x00世界\x00",
			wantCmdPath:  `Hello`,
			wantArgs:     `世...`,
			wantCmd:      `Hello`,
		},
		{
			procfsRoot:   pidCmdlineTestDataDir,
			pid:          1002,
			tid:          14,
			poolReadSize: 14,
			cmdline:      "Hello\x00世界\x00",
			wantCmdPath:  `Hello`,
			wantArgs:     `世界`,
			wantCmd:      `Hello`,
		},
	} {
		t.Run(
			tc.name,
			func(t *testing.T) { testPidCmdlineParser(tc, t) },
		)
	}
}
