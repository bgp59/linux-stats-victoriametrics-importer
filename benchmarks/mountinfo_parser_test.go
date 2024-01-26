package benchmarks

import (
	"fmt"
	"testing"

	"github.com/eparparita/linux-stats-victoriametrics-importer/procfs"
)

func BenchmarkMountinfoParserIO(b *testing.B) {
	benchmarkFileRead(procfs.MountinfoPath(LSVMI_TESTDATA_PROCFS_ROOT, 1), BENCH_FILE_READ, b)
}

func benchmarkMountinfoParser(forceUpdate bool, b *testing.B) {
	mountinfo := procfs.NewMountinfo(LSVMI_TESTDATA_PROCFS_ROOT, 1)
	mountinfo.ForceUpdate = forceUpdate
	for n := 0; n < b.N; n++ {
		err := mountinfo.Parse()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMountinfoParser(b *testing.B) {
	for _, forceUpdate := range []bool{true, false} {
		b.Run(
			fmt.Sprintf("forceUpdate=%v", forceUpdate),
			func(b *testing.B) { benchmarkMountinfoParser(forceUpdate, b) },
		)
	}
}

// goos: darwin
// goarch: amd64
// pkg: github.com/eparparita/linux-stats-victoriametrics-importer/benchmarks
// cpu: Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz
// BenchmarkMountinfoParserIO 	   						   70509	     17010 ns/op	     152 B/op	       3 allocs/op
// BenchmarkMountinfoParser/forceUpdate=false        	   63097	     17405 ns/op	     176 B/op	       4 allocs/op
// BenchmarkMountinfoParser/forceUpdate=true         	   52868	     23762 ns/op	     312 B/op	      39 allocs/op

func BenchmarkMountinfoFileRead(b *testing.B) {
	for op, name := range benchFileReadOpMap {
		b.Run(
			name,
			func(b *testing.B) {
				benchmarkFileRead(procfs.MountinfoPath(LSVMI_TESTDATA_PROCFS_ROOT, 1), op, b)
			},
		)
	}
}

// goos: darwin
// goarch: amd64
// pkg: github.com/eparparita/linux-stats-victoriametrics-importer/benchmarks
// cpu: Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz
// BenchmarkMountinfoFileRead/BENCH_FILE_READ         	   			   68134	     17004 ns/op	     152 B/op	       3 allocs/op
// BenchmarkMountinfoFileRead/BENCH_FILE_READ_SCAN_BYTES         	   61975	     19412 ns/op	    4248 B/op	       4 allocs/op
// BenchmarkMountinfoFileRead/BENCH_FILE_SCAN_BYTES              	   62101	     19436 ns/op	    4248 B/op	       4 allocs/op
// BenchmarkMountinfoFileRead/BENCH_FILE_SCAN_TEXT               	   56610	     21636 ns/op	    8088 B/op	      39 allocs/op
// BenchmarkMountinfoFileRead/BENCH_FILE_READ_SCAN_TEXT          	   59616	     22286 ns/op	    8088 B/op	      39 allocs/op
