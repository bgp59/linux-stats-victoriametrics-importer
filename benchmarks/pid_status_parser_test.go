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

func BenchmarkPidStatusParserIO(b *testing.B) {
	benchmarkFileRead(
		pidTidPath(LSVMI_TESTDATA_PROCFS_ROOT, pidStatTestPid, pidStatTestTid, "status"),
		BENCH_FILE_READ,
		b,
	)
}

func BenchmarkPidStatusParser(b *testing.B) {
	pidStatus := procfs.NewPidStatus(LSVMI_TESTDATA_PROCFS_ROOT, pidStatusTestPid, pidStatusTestTid)
	for n := 0; n < b.N; n++ {
		err := pidStatus.Parse(nil)
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

// goos: darwin
// goarch: amd64
// pkg: github.com/eparparita/linux-stats-victoriametrics-importer/benchmarks
// cpu: Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz
// BenchmarkPidStatusParserIO   	   69541	     16960 ns/op	     152 B/op	       3 allocs/op
// BenchmarkPidStatusParser     	   67520	     18458 ns/op	     176 B/op	       4 allocs/op
// BenchmarkPidStatusParserProm 	   44492	     27708 ns/op	    1336 B/op	      31 allocs/op

func BenchmarkPidStatusFileRead(b *testing.B) {
	path := pidTidPath(LSVMI_TESTDATA_PROCFS_ROOT, pidStatTestPid, pidStatTestTid, "status")
	for op, name := range benchFileReadOpMap {
		b.Run(
			name,
			func(b *testing.B) {
				benchmarkFileRead(path, op, b)
			},
		)
	}
}

// goos: darwin
// goarch: amd64
// pkg: github.com/eparparita/linux-stats-victoriametrics-importer/benchmarks
// cpu: Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz
// BenchmarkPidStatusFileRead/BENCH_FILE_READ         	   			   70216	     17076 ns/op	     152 B/op	       3 allocs/op
// BenchmarkPidStatusFileRead/BENCH_FILE_READ_SCAN_BYTES         	   62594	     19468 ns/op	    4248 B/op	       4 allocs/op
// BenchmarkPidStatusFileRead/BENCH_FILE_READ_SCAN_TEXT          	   59212	     20115 ns/op	    4664 B/op	       7 allocs/op
// BenchmarkPidStatusFileRead/BENCH_FILE_SCAN_BYTES              	   60781	     19285 ns/op	    4248 B/op	       4 allocs/op
// BenchmarkPidStatusFileRead/BENCH_FILE_SCAN_TEXT               	   61776	     19659 ns/op	    4664 B/op	       7 allocs/op
