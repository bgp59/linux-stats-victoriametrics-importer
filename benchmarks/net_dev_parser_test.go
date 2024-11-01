package benchmarks

import (
	"testing"

	"github.com/emypar/linux-stats-victoriametrics-importer/procfs"

	// Reference for performance comparison:
	prom_procfs "github.com/prometheus/procfs"
)

func BenchmarkNetDevParserIO(b *testing.B) {
	benchmarkFileRead(procfs.NetDevPath(LSVMI_TESTDATA_PROCFS_ROOT), BENCH_FILE_READ, b)
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

func BenchmarkNetDevFileRead(b *testing.B) {
	for op, name := range benchFileReadOpMap {
		b.Run(
			name,
			func(b *testing.B) {
				benchmarkFileRead(procfs.NetDevPath(LSVMI_TESTDATA_PROCFS_ROOT), op, b)
			},
		)
	}
}
