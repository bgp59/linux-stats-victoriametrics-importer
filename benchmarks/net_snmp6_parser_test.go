package benchmarks

import (
	"testing"

	"github.com/bgp59/linux-stats-victoriametrics-importer/procfs"

	// Reference for performance comparison:
	prom_procfs "github.com/prometheus/procfs"
)

func BenchmarkNetSnmp6ParserIO(b *testing.B) {
	benchmarkFileRead(procfs.NetSnmp6Path(LSVMI_TESTDATA_PROCFS_ROOT), BENCH_FILE_READ, b)
}

func BenchmarkNetSnmp6Parser(b *testing.B) {
	netSnmp6 := procfs.NewNetSnmp6(LSVMI_TESTDATA_PROCFS_ROOT)
	for n := 0; n < b.N; n++ {
		err := netSnmp6.Parse()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNetSnmp6ParserProm(b *testing.B) {
	fs, err := prom_procfs.NewFS(LSVMI_TESTDATA_PROCFS_ROOT)
	if err != nil {
		b.Fatal(err)
	}

	proc, err := fs.Proc(0)
	if err != nil {
		b.Fatal(err)
	}

	for n := 0; n < b.N; n++ {
		_, err := proc.Snmp6()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNetSnmp6FileRead(b *testing.B) {
	for op, name := range benchFileReadOpMap {
		b.Run(
			name,
			func(b *testing.B) {
				benchmarkFileRead(procfs.NetSnmp6Path(LSVMI_TESTDATA_PROCFS_ROOT), op, b)
			},
		)
	}
}
