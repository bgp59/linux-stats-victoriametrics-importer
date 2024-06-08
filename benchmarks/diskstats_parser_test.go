package benchmarks

import (
	"testing"

	"github.com/emypar/linux-stats-victoriametrics-importer/procfs"

	// Reference for performance comparison:
	prom_procfs_blockdevice "github.com/prometheus/procfs/blockdevice"
)

func BenchmarkDiskstatsParserIO(b *testing.B) {
	benchmarkFileRead(procfs.DiskstatsPath(LSVMI_TESTDATA_PROCFS_ROOT), BENCH_FILE_READ, b)
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
// pkg: github.com/emypar/linux-stats-victoriametrics-importer/benchmarks
// cpu: Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz
// BenchmarkDiskstatsParserIO   	   71402	     17087 ns/op	     152 B/op	       3 allocs/op
// BenchmarkDiskstatsParser     	   58149	     21128 ns/op	     336 B/op	      38 allocs/op
// BenchmarkDiskstatsParserProm 	   10000	    101076 ns/op	   14744 B/op	     176 allocs/op

func BenchmarkDiskstatsFileRead(b *testing.B) {
	for op, name := range benchFileReadOpMap {
		b.Run(
			name,
			func(b *testing.B) {
				benchmarkFileRead(procfs.DiskstatsPath(LSVMI_TESTDATA_PROCFS_ROOT), op, b)
			},
		)
	}
}

// goos: darwin
// goarch: amd64
// pkg: github.com/emypar/linux-stats-victoriametrics-importer/benchmarks
// cpu: Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz
// BenchmarkDiskstatsFileRead/BENCH_FILE_READ         	   			   70636	     16880 ns/op	     152 B/op	       3 allocs/op
// BenchmarkDiskstatsFileRead/BENCH_FILE_SCAN_BYTES              	   66147	     18475 ns/op	    4248 B/op	       4 allocs/op
// BenchmarkDiskstatsFileRead/BENCH_FILE_READ_SCAN_BYTES         	   63024	     18860 ns/op	    4248 B/op	       4 allocs/op
// BenchmarkDiskstatsFileRead/BENCH_FILE_SCAN_TEXT               	   60080	     19526 ns/op	    5320 B/op	      19 allocs/op
// BenchmarkDiskstatsFileRead/BENCH_FILE_READ_SCAN_TEXT          	   58963	     20111 ns/op	    5320 B/op	      19 allocs/op
