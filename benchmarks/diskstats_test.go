package benchmarks

import (
	"path"
	"testing"

	"github.com/eparparita/linux-stats-victoriametrics-importer/procfs"

	// Reference for performance comparison:
	prom_procfs_blockdevice "github.com/prometheus/procfs/blockdevice"
)

func BenchmarkDiskstatsParserIO(b *testing.B) {
	benchmarkFileRead(path.Join(LSVMI_TESTDATA_PROCFS_ROOT, "diskstats"), BENCH_FILE_READ, b)
}

func BenchmarkDiskstatsParser(b *testing.B) {
	diskstats := procfs.NewDiskstats(LSVMI_TESTDATA_PROCFS_ROOT)
	for n := 0; n < b.N; n++ {
		err := diskstats.Parse()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDiskstatsParserProm(b *testing.B) {
	fs, err := prom_procfs_blockdevice.NewFS(LSVMI_TESTDATA_PROCFS_ROOT, LSVMI_TESTDATA_PROCFS_ROOT)
	if err != nil {
		b.Fatal(err)
	}

	for n := 0; n < b.N; n++ {
		_, err := fs.ProcDiskstats()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// goos: darwin
// goarch: amd64
// pkg: github.com/eparparita/linux-stats-victoriametrics-importer/benchmarks
// cpu: Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz
// BenchmarkDiskstatsParserIO   	   70484	     16256 ns/op	     152 B/op	       3 allocs/op
// BenchmarkDiskstatsParser     	   62272	     19699 ns/op	     336 B/op	      38 allocs/op
// BenchmarkDiskstatsParserProm 	   12075	    101451 ns/op	   14744 B/op	     176 allocs/op

func BenchmarkDiskstatsFileRead(b *testing.B) {
	for op, name := range benchFileReadOpMap {
		b.Run(
			name,
			func(b *testing.B) {
				benchmarkFileRead(path.Join(LSVMI_TESTDATA_PROCFS_ROOT, "diskstats"), op, b)
			},
		)
	}
}

// goos: darwin
// goarch: amd64
// pkg: github.com/eparparita/linux-stats-victoriametrics-importer/benchmarks
// cpu: Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz
// BenchmarkDiskstatsFileRead/BENCH_FILE_SCAN_TEXT         	   60577	     19901 ns/op	    5320 B/op	      19 allocs/op
// BenchmarkDiskstatsFileRead/BENCH_FILE_READ              	   74121	     15888 ns/op	     152 B/op	       3 allocs/op
// BenchmarkDiskstatsFileRead/BENCH_FILE_READ_SCAN_BYTES   	   66908	     18515 ns/op	    4248 B/op	       4 allocs/op
// BenchmarkDiskstatsFileRead/BENCH_FILE_READ_SCAN_TEXT    	   59155	     19851 ns/op	    5320 B/op	      19 allocs/op
// BenchmarkDiskstatsFileRead/BENCH_FILE_SCAN_BYTES        	   66093	     18547 ns/op	    4248 B/op	       4 allocs/op
