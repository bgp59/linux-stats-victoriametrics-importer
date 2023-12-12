package benchmarks

import (
	"path"
	"testing"

	"github.com/eparparita/linux-stats-victoriametrics-importer/procfs"

	// Reference for performance comparison:
	prom_procfs "github.com/prometheus/procfs"
)

func BenchmarkSoftirqsParserIO(b *testing.B) {
	benchmarkFileRead(path.Join(LSVMI_TESTDATA_PROCFS_ROOT, "softirqs"), BENCH_FILE_READ, b)
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
// pkg: github.com/eparparita/linux-stats-victoriametrics-importer/benchmarks
// cpu: Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz
// BenchmarkSoftirqsParserIO   	   47666	     24957 ns/op	     136 B/op	       3 allocs/op
// BenchmarkSoftirqsParser     	   37726	     31667 ns/op	    4296 B/op	      14 allocs/op
// BenchmarkSoftirqsParserProm 	   26047	     46424 ns/op	   14992 B/op	      42 allocs/op

func BenchmarkSoftirqsFileRead(b *testing.B) {
	for op, name := range benchFileReadOpMap {
		b.Run(
			name,
			func(b *testing.B) {
				benchmarkFileRead(path.Join(LSVMI_TESTDATA_PROCFS_ROOT, "softirqs"), op, b)
			},
		)
	}
}

// goos: darwin
// goarch: amd64
// pkg: github.com/eparparita/linux-stats-victoriametrics-importer/benchmarks
// cpu: Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz
// BenchmarkSoftirqsFileRead/BENCH_FILE_SCAN_TEXT         	   41954	     27995 ns/op	    6344 B/op	      15 allocs/op
// BenchmarkSoftirqsFileRead/BENCH_FILE_READ              	   48457	     24798 ns/op	     136 B/op	       3 allocs/op
// BenchmarkSoftirqsFileRead/BENCH_FILE_READ_SCAN_BYTES   	   43401	     26891 ns/op	    4232 B/op	       4 allocs/op
// BenchmarkSoftirqsFileRead/BENCH_FILE_READ_SCAN_TEXT    	   42246	     28439 ns/op	    6344 B/op	      15 allocs/op
// BenchmarkSoftirqsFileRead/BENCH_FILE_SCAN_BYTES        	   43953	     26772 ns/op	    4232 B/op	       4 allocs/op
