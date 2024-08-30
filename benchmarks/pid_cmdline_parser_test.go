package benchmarks

import (
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

func BenchmarkPidCmdlineParser(b *testing.B) {
	pidCmdline := procfs.NewPidCmdline()
	for n := 0; n < b.N; n++ {
		err := pidCmdline.Parse(benchPidCmdlineParserPidTidPath)
		if err != nil {
			b.Fatal(err)
		}
	}
}
