// Benchmarks for /proc/pid/status parser Invoke the parser and additionally
// simulate real life usage; the parsed data will be printed to a bytes.Buffer.

package benchmarks

import (
	"testing"

	"github.com/eparparita/linux-stats-victoriametrics-importer/procfs"
)

var pidStatusTestPid, pidStatusTestTid int = 468, 486

func benchmarkPidStatusParser(b *testing.B) {
	pidStatus := procfs.NewPidStatus(TESTDATA_PROCFS_ROOT, pidStatusTestPid, pidStatusTestTid)
	for n := 0; n < b.N; n++ {
		err := pidStatus.Parse()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPidStatusParser(b *testing.B) {
	b.Run(
		"",
		func(b *testing.B) { benchmarkPidStatusParser(b) },
	)
}
