package benchmarks

import (
	"testing"

	"github.com/emypar/linux-stats-victoriametrics-importer/procfs"

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
