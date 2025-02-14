package benchmarks

import (
	"testing"

	"github.com/bgp59/linux-stats-victoriametrics-importer/procfs"

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
