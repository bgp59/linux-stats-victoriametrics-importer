package benchmarks

import (
	"testing"

	"github.com/eparparita/linux-stats-victoriametrics-importer/procfs"

	// Reference for performance comparison:
	prom_procfs "github.com/prometheus/procfs"
)

func BenchmarkInterruptsParserIO(b *testing.B) {
	benchmarkFileRead(procfs.InterruptsPath(LSVMI_TESTDATA_PROCFS_ROOT), BENCH_FILE_READ, b)
}

func BenchmarkInterruptsParser(b *testing.B) {
	interrupts := procfs.NewInterrupts(LSVMI_TESTDATA_PROCFS_ROOT)
	for n := 0; n < b.N; n++ {
		err := interrupts.Parse()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkInterruptsParserProm(b *testing.B) {
	fs, err := prom_procfs.NewFS(LSVMI_TESTDATA_PROCFS_ROOT)
	if err != nil {
		b.Fatal(err)
	}
	proc, err := fs.Proc(0)
	if err != nil {
		b.Fatal(err)
	}

	for n := 0; n < b.N; n++ {
		_, err := proc.Interrupts()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// goos: darwin
// goarch: amd64
// pkg: github.com/eparparita/linux-stats-victoriametrics-importer/benchmarks
// cpu: Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz
// BenchmarkInterruptsParserIO   	   74157	     16252 ns/op	     152 B/op	       3 allocs/op
// BenchmarkInterruptsParser     	   61410	     19693 ns/op	     240 B/op	      35 allocs/op
// BenchmarkInterruptsParserProm 	   25360	     48432 ns/op	   26327 B/op	     171 allocs/op

func BenchmarkInterruptsFileRead(b *testing.B) {
	for op, name := range benchFileReadOpMap {
		b.Run(
			name,
			func(b *testing.B) {
				benchmarkFileRead(procfs.InterruptsPath(LSVMI_TESTDATA_PROCFS_ROOT), op, b)
			},
		)
	}
}

// goos: darwin
// goarch: amd64
// pkg: github.com/eparparita/linux-stats-victoriametrics-importer/benchmarks
// cpu: Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz
// BenchmarkInterruptsFileRead/BENCH_FILE_READ         	   			   71082	     16179 ns/op	     152 B/op	       3 allocs/op
// BenchmarkInterruptsFileRead/BENCH_FILE_READ_SCAN_BYTES         	   63536	     18841 ns/op	    4248 B/op	       4 allocs/op
// BenchmarkInterruptsFileRead/BENCH_FILE_SCAN_BYTES              	   64528	     18986 ns/op	    4248 B/op	       4 allocs/op
// BenchmarkInterruptsFileRead/BENCH_FILE_SCAN_TEXT               	   57381	     20571 ns/op	    6072 B/op	      39 allocs/op
// BenchmarkInterruptsFileRead/BENCH_FILE_READ_SCAN_TEXT          	   61892	     20849 ns/op	    6072 B/op	      39 allocs/op
