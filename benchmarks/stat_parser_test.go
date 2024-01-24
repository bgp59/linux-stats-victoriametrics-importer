// Benchmarks for /proc/stat parser

package benchmarks

import (
	"path"
	"testing"

	"github.com/eparparita/linux-stats-victoriametrics-importer/procfs"

	// Reference for performance comparison:
	prom_procfs "github.com/prometheus/procfs"
)

func BenchmarkStatParserIO(b *testing.B) {
	benchmarkFileRead(path.Join(LSVMI_TESTDATA_PROCFS_ROOT, "stat"), BENCH_FILE_READ, b)
}

func BenchmarkStatParser(b *testing.B) {
	pidStat := procfs.NewStat(LSVMI_TESTDATA_PROCFS_ROOT)
	for n := 0; n < b.N; n++ {
		err := pidStat.Parse()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStatParserProm(b *testing.B) {
	fs, err := prom_procfs.NewFS(LSVMI_TESTDATA_PROCFS_ROOT)
	if err != nil {
		b.Fatal(err)
	}

	for n := 0; n < b.N; n++ {
		_, err := fs.Stat()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// goos: darwin
// goarch: amd64
// pkg: github.com/eparparita/linux-stats-victoriametrics-importer/benchmarks
// cpu: Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz
// BenchmarkStatParserIO   	   73706	     16417 ns/op	     136 B/op	       3 allocs/op
// BenchmarkStatParser     	   67137	     18738 ns/op	     160 B/op	       4 allocs/op
// BenchmarkStatParserProm 	   19458	     64293 ns/op	   47666 B/op	      78 allocs/op
