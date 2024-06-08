package benchmarks

import (
	"testing"

	"github.com/emypar/linux-stats-victoriametrics-importer/procfs"

	// Reference for performance comparison:
	prom_procfs "github.com/prometheus/procfs"
)

func BenchmarkSoftirqsParserIO(b *testing.B) {
	benchmarkFileRead(procfs.SoftirqsPath(LSVMI_TESTDATA_PROCFS_ROOT), BENCH_FILE_READ, b)
}

func BenchmarkSoftirqsParser(b *testing.B) {
	softirqs := procfs.NewSoftirqs(LSVMI_TESTDATA_PROCFS_ROOT)
	for n := 0; n < b.N; n++ {
		err := softirqs.Parse()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSoftirqsParserProm(b *testing.B) {
	fs, err := prom_procfs.NewFS(LSVMI_TESTDATA_PROCFS_ROOT)
	if err != nil {
		b.Fatal(err)
	}

	for n := 0; n < b.N; n++ {
		_, err := fs.Softirqs()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// goos: darwin
// goarch: amd64
// pkg: github.com/emypar/linux-stats-victoriametrics-importer/benchmarks
// cpu: Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz
// BenchmarkSoftirqsParserIO   	   69418	     16342 ns/op	     136 B/op	       3 allocs/op
// BenchmarkSoftirqsParser     	   65466	     18810 ns/op	     200 B/op	      13 allocs/op
// BenchmarkSoftirqsParserProm 	   39843	     32084 ns/op	   14992 B/op	      42 allocs/op

func BenchmarkSoftirqsFileRead(b *testing.B) {
	for op, name := range benchFileReadOpMap {
		b.Run(
			name,
			func(b *testing.B) {
				benchmarkFileRead(procfs.SoftirqsPath(LSVMI_TESTDATA_PROCFS_ROOT), op, b)
			},
		)
	}
}

// goos: darwin
// goarch: amd64
// pkg: github.com/emypar/linux-stats-victoriametrics-importer/benchmarks
// cpu: Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz
// BenchmarkSoftirqsFileRead/BENCH_FILE_READ         	   			   67806	     16610 ns/op	     136 B/op	       3 allocs/op
// BenchmarkSoftirqsFileRead/BENCH_FILE_READ_SCAN_BYTES         	   61936	     18959 ns/op	    4232 B/op	       4 allocs/op
// BenchmarkSoftirqsFileRead/BENCH_FILE_SCAN_BYTES              	   61742	     19149 ns/op	    4232 B/op	       4 allocs/op
// BenchmarkSoftirqsFileRead/BENCH_FILE_READ_SCAN_TEXT          	   61707	     20052 ns/op	    6344 B/op	      15 allocs/op
// BenchmarkSoftirqsFileRead/BENCH_FILE_SCAN_TEXT               	   60798	     20153 ns/op	    6344 B/op	      15 allocs/op
