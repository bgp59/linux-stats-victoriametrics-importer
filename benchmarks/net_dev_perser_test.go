package benchmarks

import (
	"path"
	"testing"

	"github.com/eparparita/linux-stats-victoriametrics-importer/procfs"

	// Reference for performance comparison:
	prom_procfs "github.com/prometheus/procfs"
)

func BenchmarkNetDevParserIO(b *testing.B) {
	benchmarkFileRead(path.Join(LSVMI_TESTDATA_PROCFS_ROOT, "net", "dev"), BENCH_FILE_READ, b)
}

func BenchmarkNetDevParser(b *testing.B) {
	netDev := procfs.NewNetDev(LSVMI_TESTDATA_PROCFS_ROOT)
	for n := 0; n < b.N; n++ {
		err := netDev.Parse()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNetDevParserProm(b *testing.B) {
	fs, err := prom_procfs.NewFS(LSVMI_TESTDATA_PROCFS_ROOT)
	if err != nil {
		b.Fatal(err)
	}

	for n := 0; n < b.N; n++ {
		_, err := fs.NetDev()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// goarch: amd64
// pkg: github.com/eparparita/linux-stats-victoriametrics-importer/benchmarks
// cpu: Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz
// BenchmarkNetDevParserIO   	   48913	     23218 ns/op	     136 B/op	       3 allocs/op
// BenchmarkNetDevParser     	   50480	     23842 ns/op	     168 B/op	       6 allocs/op
// BenchmarkNetDevParserProm 	   39608	     27756 ns/op	    5896 B/op	      16 allocs/op

func BenchmarkNetDevFileRead(b *testing.B) {
	for op, name := range benchFileReadOpMap {
		b.Run(
			name,
			func(b *testing.B) {
				benchmarkFileRead(path.Join(LSVMI_TESTDATA_PROCFS_ROOT, "net", "dev"), op, b)
			},
		)
	}
}

// goos: darwin
// goarch: amd64
// pkg: github.com/eparparita/linux-stats-victoriametrics-importer/benchmarks
// cpu: Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz
// BenchmarkNetDevFileRead/BENCH_FILE_READ_SCAN_BYTES         	   42466	     26361 ns/op	    4232 B/op	       4 allocs/op
// BenchmarkNetDevFileRead/BENCH_FILE_READ_SCAN_TEXT          	   48958	     25747 ns/op	    4696 B/op	       8 allocs/op
// BenchmarkNetDevFileRead/BENCH_FILE_SCAN_BYTES              	   46430	     25920 ns/op	    4232 B/op	       4 allocs/op
// BenchmarkNetDevFileRead/BENCH_FILE_SCAN_TEXT               	   46657	     25461 ns/op	    4696 B/op	       8 allocs/op
// BenchmarkNetDevFileRead/BENCH_FILE_READ                    	   51642	     22890 ns/op	     136 B/op	       3 allocs/op
