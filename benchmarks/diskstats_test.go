package benchmarks

import (
	"testing"

	"github.com/eparparita/linux-stats-victoriametrics-importer/procfs"

	// Reference for performance comparison:

	prom_procfs_blockdevice "github.com/prometheus/procfs/blockdevice"
)

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

// BenchmarkDiskstatsParser-12        	   61227	     18761 ns/op
// BenchmarkDiskstatsParserProm-12    	   12240	     98174 ns/op
