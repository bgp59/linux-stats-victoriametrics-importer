package benchmarks

import (
	"testing"

	"github.com/emypar/linux-stats-victoriametrics-importer/procfs"

	// Reference for performance comparison:
	prom_procfs "github.com/prometheus/procfs"
)

func BenchmarkNetSnmpParserIO(b *testing.B) {
	benchmarkFileRead(procfs.NetSnmpPath(LSVMI_TESTDATA_PROCFS_ROOT), BENCH_FILE_READ, b)
}

func BenchmarkNetSnmpParser(b *testing.B) {
	netSnmp := procfs.NewNetSnmp(LSVMI_TESTDATA_PROCFS_ROOT)
	for n := 0; n < b.N; n++ {
		err := netSnmp.Parse()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNetSnmpParserProm(b *testing.B) {
	fs, err := prom_procfs.NewFS(LSVMI_TESTDATA_PROCFS_ROOT)
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

func BenchmarkNetSnmpFileRead(b *testing.B) {
	for op, name := range benchFileReadOpMap {
		b.Run(
			name,
			func(b *testing.B) {
				benchmarkFileRead(procfs.NetSnmpPath(LSVMI_TESTDATA_PROCFS_ROOT), op, b)
			},
		)
	}
}
