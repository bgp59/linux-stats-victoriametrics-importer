package benchmarks

import (
	"path"
	"testing"

	"github.com/eparparita/linux-stats-victoriametrics-importer/procfs"

	// Reference for performance comparison:
	prom_procfs "github.com/prometheus/procfs"
)

func BenchmarkInterruptsParserIO(b *testing.B) {
	benchmarkFileRead(path.Join(LSVMI_TESTDATA_PROCFS_ROOT, "interrupts"), BENCH_FILE_READ, b)
}

func BenchmarkAllInterruptsParserIO(b *testing.B) {
	for op, name := range benchFileReadOpMap {
		b.Run(
			name,
			func(b *testing.B) {
				benchmarkFileRead(path.Join(LSVMI_TESTDATA_PROCFS_ROOT, "interrupts"), op, b)
			},
		)
	}
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
// BenchmarkInterruptsParserIO   	   70122	     16215 ns/op	     152 B/op	       3 allocs/op
// BenchmarkInterruptsParser     	   53030	     22944 ns/op	    4360 B/op	      37 allocs/op
// BenchmarkInterruptsParserProm 	   25659	     50023 ns/op	   26336 B/op	     171 allocs/op
