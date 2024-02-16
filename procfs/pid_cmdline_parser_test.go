package procfs

import (
	"os"
	"path"
	"testing"

	"github.com/eparparita/linux-stats-victoriametrics-importer/internal/utils"
)

type PidCmdlineTestCase struct {
	name         string
	procfsRoot   string
	pid, tid     int
	poolReadSize int64
	cmdline      string
	wantCmdline  string
	wantError    error
}

var pidCmdlineTestdataDir = path.Join(PROCFS_TESTDATA_ROOT, "pid_cmdline")

func testPidCmdlineParser(tc *PidCmdlineTestCase, t *testing.T) {
	t.Logf(`
procfsRoot:=%q, pid=%d, tid=%d
cmdline=%q (%d bytes)
poolReadSize=%d
`,
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

	pidCmdline := NewPidCmdline(tc.procfsRoot, tc.pid, tc.tid)

	if tc.cmdline != "" {
		err := os.MkdirAll(path.Dir(pidCmdline.path), os.ModePerm)
		if err != nil {
			t.Fatal(err)
		}
		file, err := os.Create(pidCmdline.path)
		if err != nil {
			t.Fatal(err)
		}
		_, err = file.WriteString(tc.cmdline)
		if err != nil {
			t.Fatal(err)
		}
		file.Close()
	}

	err := pidCmdline.Parse(0, 0)
	if tc.wantError == nil && err != nil {
		t.Fatal(err)
	}
	if tc.wantError != nil {
		if err == nil || tc.wantError.Error() != err.Error() {
			t.Fatalf("error: want: %v, got: %v", tc.wantError, err)
		}
	}
	gotCmdline := pidCmdline.Cmdline.String()
	if tc.wantCmdline != gotCmdline {
		t.Fatalf("cmdline: want: %q, got: %q", tc.wantCmdline, gotCmdline)
	}
}

func TestPidCmdlineParser(t *testing.T) {
	for _, tc := range []*PidCmdlineTestCase{
		{
			procfsRoot:  pidCmdlineTestdataDir,
			pid:         1,
			tid:         PID_STAT_PID_ONLY_TID,
			wantCmdline: `/sbin/init`,
		},
		{
			procfsRoot:  pidCmdlineTestdataDir,
			pid:         101,
			tid:         PID_STAT_PID_ONLY_TID,
			cmdline:     "arg0\x00arg1\x00",
			wantCmdline: `arg0 arg1`,
		},
		{
			procfsRoot:  pidCmdlineTestdataDir,
			pid:         101,
			tid:         101,
			cmdline:     "arg0\x00arg1\x00\x00",
			wantCmdline: `arg0 arg1`,
		},
		{
			procfsRoot:  pidCmdlineTestdataDir,
			pid:         102,
			tid:         PID_STAT_PID_ONLY_TID,
			cmdline:     "arg0\x00arg1\x00arg\n2\x00",
			wantCmdline: `arg0 arg1 arg\n2`,
		},
		{
			procfsRoot:  pidCmdlineTestdataDir,
			pid:         103,
			tid:         PID_STAT_PID_ONLY_TID,
			cmdline:     "arg0\x00arg1\x00\"arg 2\"\x00",
			wantCmdline: `arg0 arg1 \"arg 2\"`,
		},
		{
			procfsRoot:   pidCmdlineTestdataDir,
			pid:          1001,
			tid:          PID_STAT_PID_ONLY_TID,
			poolReadSize: 10,
			cmdline:      "arg0\x00arg1\x00arg2\x00",
			wantCmdline:  `arg0 ar...`,
		},
		{
			procfsRoot:   pidCmdlineTestdataDir,
			pid:          1002,
			tid:          8,
			poolReadSize: 8,
			cmdline:      "Hello\x00世界\x00",
			wantCmdline:  `Hello...`,
		},
		{
			procfsRoot:   pidCmdlineTestdataDir,
			pid:          1002,
			tid:          9,
			poolReadSize: 9,
			cmdline:      "Hello\x00世界\x00",
			wantCmdline:  `Hello ...`,
		},
		{
			procfsRoot:   pidCmdlineTestdataDir,
			pid:          1002,
			tid:          10,
			poolReadSize: 10,
			cmdline:      "Hello\x00世界\x00",
			wantCmdline:  `Hello ...`,
		},
		{
			procfsRoot:   pidCmdlineTestdataDir,
			pid:          1002,
			tid:          11,
			poolReadSize: 11,
			cmdline:      "Hello\x00世界\x00",
			wantCmdline:  `Hello ...`,
		},
		{
			procfsRoot:   pidCmdlineTestdataDir,
			pid:          1002,
			tid:          12,
			poolReadSize: 12,
			cmdline:      "Hello\x00世界\x00",
			wantCmdline:  `Hello 世...`,
		},
		{
			procfsRoot:   pidCmdlineTestdataDir,
			pid:          1002,
			tid:          13,
			poolReadSize: 13,
			cmdline:      "Hello\x00世界\x00",
			wantCmdline:  `Hello 世...`,
		},
		{
			procfsRoot:   pidCmdlineTestdataDir,
			pid:          1002,
			tid:          14,
			poolReadSize: 14,
			cmdline:      "Hello\x00世界\x00",
			wantCmdline:  `Hello 世界`,
		},
	} {
		t.Run(
			tc.name,
			func(t *testing.T) { testPidCmdlineParser(tc, t) },
		)
	}
}
