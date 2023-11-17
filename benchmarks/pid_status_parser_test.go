// Benchmarks for /proc/pid/status parser Invoke the parser and additionally
// simulate real life usage; the parsed data will be printed to a bytes.Buffer.

package benchmarks

import (
	"testing"

	"github.com/eparparita/linux-stats-victoriametrics-importer/procfs"

	// Reference for performance comparison:
	prom_procfs "github.com/prometheus/procfs"
)

var pidStatusTestPid, pidStatusTestTid int = 468, 486

func BenchmarkPidStatusParser(b *testing.B) {
	pidStatus := procfs.NewPidStatus(LSVMI_TESTDATA_PROCFS_ROOT, pidStatusTestPid, pidStatusTestTid)
	for n := 0; n < b.N; n++ {
		err := pidStatus.Parse()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPidStatusParserProm(b *testing.B) {
	var proc prom_procfs.Proc
	fs, err := prom_procfs.NewFS(LSVMI_TESTDATA_PROCFS_ROOT)
	if err != nil {
		b.Fatal(err)
	}

	if pidStatTestTid != 0 {
		proc, err = fs.Thread(pidStatTestPid, pidStatTestTid)
	} else {
		proc, err = fs.Proc(pidStatTestPid)
	}
	if err != nil {
		b.Fatal(err)
	}

	for n := 0; n < b.N; n++ {
		_, err := proc.Stat()
		if err != nil {
			b.Fatal(err)
		}
	}
}
