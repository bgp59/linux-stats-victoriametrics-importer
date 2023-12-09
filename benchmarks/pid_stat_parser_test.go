// Benchmarks for /proc/pid/stat parser Invoke the parser and additionally
// simulate real life usage; the parsed data will be printed to a bytes.Buffer.

package benchmarks

import (
	"testing"

	"github.com/eparparita/linux-stats-victoriametrics-importer/procfs"

	// Reference for performance comparison:
	prom_procfs "github.com/prometheus/procfs"
)

var pidStatTestPid, pidStatTestTid int = 468, 486

func BenchmarkPidStatParserIO(b *testing.B) {
	benchmarkFileRead(
		pidTidPath(LSVMI_TESTDATA_PROCFS_ROOT, pidStatTestPid, pidStatTestTid, "stat"),
		BENCH_FILE_READ,
		b,
	)
}

func BenchmarkPidStatParser(b *testing.B) {
	pidStat := procfs.NewPidStat(LSVMI_TESTDATA_PROCFS_ROOT, pidStatTestPid, pidStatTestTid)
	for n := 0; n < b.N; n++ {
		err := pidStat.Parse()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPidStatParserProm(b *testing.B) {
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

// goos: darwin
// goarch: amd64
// pkg: github.com/eparparita/linux-stats-victoriametrics-importer/benchmarks
// cpu: Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz
// BenchmarkPidStatParserIO   	   70147	     16857 ns/op	     152 B/op	       3 allocs/op
// BenchmarkPidStatParser     	   73059	     17176 ns/op	    1070 B/op	       3 allocs/op
// BenchmarkPidStatParserProm 	   44281	     27217 ns/op	    1336 B/op	      31 allocs/op
