package benchmarks

import (
	"fmt"
	"testing"

	"github.com/eparparita/linux-stats-victoriametrics-importer/procfs"
)

var (
	pidCmdlineProcfsRoot = LSVMI_TESTDATA_PROCFS_ROOT
	pidCmdlineTestPid    = 586
	pidCmdlineTestTid    = procfs.PID_STAT_PID_ONLY_TID
)

func BenchmarkPidCmdlineParserIO(b *testing.B) {
	benchmarkFileRead(
		procfs.PidCmdlinePath(pidCmdlineProcfsRoot, pidCmdlineTestPid, pidCmdlineTestTid),
		BENCH_FILE_READ,
		b,
	)
}

func benchmarkPidCmdlineParser(retBuf bool, b *testing.B) {
	pidCmdline := procfs.NewPidCmdline(pidCmdlineProcfsRoot, pidCmdlineTestPid, pidCmdlineTestTid)
	for n := 0; n < b.N; n++ {
		err := pidCmdline.Parse(0, 0)
		if retBuf {
			pidCmdline.ReturnBuf()
		}
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPidCmdlineParser(b *testing.B) {
	for _, retBuf := range []bool{false, true} {
		b.Run(
			fmt.Sprintf("retBuf=%v", retBuf),
			func(b *testing.B) { benchmarkPidCmdlineParser(retBuf, b) },
		)
	}
}

// goos: darwin
// goarch: amd64
// pkg: github.com/eparparita/linux-stats-victoriametrics-importer/benchmarks
// cpu: Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz
// BenchmarkPidCmdlineParserIO 	 					   71112	     16463 ns/op	     152 B/op	       3 allocs/op
// BenchmarkPidCmdlineParser/retBuf=false         	   75175	     16509 ns/op	     176 B/op	       4 allocs/op
// BenchmarkPidCmdlineParser/retBuf=true          	   75226	     16790 ns/op	     176 B/op	       4 allocs/op
