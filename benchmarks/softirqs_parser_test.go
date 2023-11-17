package benchmarks

import (
	"testing"

	"github.com/eparparita/linux-stats-victoriametrics-importer/procfs"

	// Reference for performance comparison:
	prom_procfs "github.com/prometheus/procfs"
)

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
