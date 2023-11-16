package benchmarks

import (
	"testing"

	"github.com/eparparita/linux-stats-victoriametrics-importer/procfs"

	// Reference for performance comparison:
	prom_procfs "github.com/prometheus/procfs"
)

func BenchmarkNetSnmpParser(b *testing.B) {
	netSnmp := procfs.NewNetSnmp(TESTDATA_PROC_ROOT)
	for n := 0; n < b.N; n++ {
		err := netSnmp.Parse()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNetSnmpParserProm(b *testing.B) {
	fs, err := prom_procfs.NewFS(TESTDATA_PROC_ROOT)
	if err != nil {
		b.Fatal(err)
	}

	proc, err := fs.Proc(0)
	if err != nil {
		b.Fatal(err)
	}

	for n := 0; n < b.N; n++ {
		_, err := proc.Snmp()
		if err != nil {
			b.Fatal(err)
		}
	}
}
