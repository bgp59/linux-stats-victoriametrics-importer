package benchmarks

import (
	"fmt"
	"path"
	"testing"

	"github.com/emypar/linux-stats-victoriametrics-importer/procfs"
)

var (
	benchPidCmdlineParserProcfsRoot = LSVMI_TESTDATA_PROCFS_ROOT
	benchPidCmdlineParserPid        = 586
	benchPidCmdlineParserTid        = procfs.PID_ONLY_TID
	benchPidCmdlineParserPidTidPath = procfs.BuildPidTidPath(
		benchPidCmdlineParserProcfsRoot, benchPidCmdlineParserPid, benchPidCmdlineParserTid,
	)
)

func BenchmarkPidCmdlineParserIO(b *testing.B) {
	benchmarkFileRead(
		path.Join(benchPidCmdlineParserPidTidPath, "cmdline"),
		BENCH_FILE_READ,
		b,
	)
}

func benchmarkPidCmdlineParser(pidTidPath string, retrieveCmd bool, b *testing.B) {
	pidCmdline := procfs.NewPidCmdline()
	for n := 0; n < b.N; n++ {
		err := pidCmdline.Parse(pidTidPath)
		if retrieveCmd {
			pidCmdline.GetCmdlineString()
		}
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPidCmdlineParser(b *testing.B) {
	for _, retrieveCmd := range []bool{false, true} {
		b.Run(
			fmt.Sprintf("retrieveCmd=%v", retrieveCmd),
			func(b *testing.B) { benchmarkPidCmdlineParser(benchPidCmdlineParserPidTidPath, retrieveCmd, b) },
		)
	}
}
