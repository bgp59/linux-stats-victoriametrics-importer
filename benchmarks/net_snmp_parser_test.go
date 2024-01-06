package benchmarks

import (
	"path"
	"testing"

	"github.com/eparparita/linux-stats-victoriametrics-importer/procfs"

	// Reference for performance comparison:
	prom_procfs "github.com/prometheus/procfs"
)

func BenchmarkNetSnmpParserIO(b *testing.B) {
	benchmarkFileRead(path.Join(LSVMI_TESTDATA_PROCFS_ROOT, "net", "snmp"), BENCH_FILE_READ, b)
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

// goos: darwin
// goarch: amd64
// pkg: github.com/eparparita/linux-stats-victoriametrics-importer/benchmarks
// cpu: Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz
// BenchmarkNetSnmpParserIO   	   49680	     23647 ns/op	     136 B/op	       3 allocs/op
// BenchmarkNetSnmpParser     	   48039	     24677 ns/op	     160 B/op	       4 allocs/op
// BenchmarkNetSnmpParserProm 	   28135	     44407 ns/op	   11960 B/op	     117 allocs/op
func BenchmarkNetSnmpFileRead(b *testing.B) {
	for op, name := range benchFileReadOpMap {
		b.Run(
			name,
			func(b *testing.B) {
				benchmarkFileRead(path.Join(LSVMI_TESTDATA_PROCFS_ROOT, "net", "snmp"), op, b)
			},
		)
	}
}

// goos: darwin
// goarch: amd64
// pkg: github.com/eparparita/linux-stats-victoriametrics-importer/benchmarks
// cpu: Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz
// BenchmarkNetSnmpFileRead/BENCH_FILE_READ         			   49275	     24469 ns/op	     136 B/op	       3 allocs/op
// BenchmarkNetSnmpFileRead/BENCH_FILE_SCAN_BYTES              	   45414	     26200 ns/op	    4232 B/op	       4 allocs/op
// BenchmarkNetSnmpFileRead/BENCH_FILE_READ_SCAN_TEXT          	   45985	     26449 ns/op	    5384 B/op	      12 allocs/op
// BenchmarkNetSnmpFileRead/BENCH_FILE_READ_SCAN_BYTES         	   44721	     26475 ns/op	    4232 B/op	       4 allocs/op
// BenchmarkNetSnmpFileRead/BENCH_FILE_SCAN_TEXT               	   45552	     26968 ns/op	    5384 B/op	      12 allocs/op
