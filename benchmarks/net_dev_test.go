package benchmarks

import (
	"testing"

	"github.com/eparparita/linux-stats-victoriametrics-importer/procfs"

	// Reference for performance comparison:
	prom_procfs "github.com/prometheus/procfs"
)

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
