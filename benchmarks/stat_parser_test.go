// Benchmarks for /proc/stat parser

package benchmarks

import (
	"testing"

	"github.com/eparparita/linux-stats-victoriametrics-importer/procfs"

	// Reference for performance comparison:
	prom_procfs "github.com/prometheus/procfs"
)

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
