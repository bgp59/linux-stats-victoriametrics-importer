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
// BenchmarkNetSnmpParserIO   	   50048	     23288 ns/op	     136 B/op	       3 allocs/op
// BenchmarkNetSnmpParser     	   43789	     26168 ns/op	    4232 B/op	       4 allocs/op
// BenchmarkNetSnmpParserProm 	   28854	     42945 ns/op	   11960 B/op	     117 allocs/op

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
// BenchmarkNetSnmpFileRead/BENCH_FILE_SCAN_TEXT         	   44066	     26367 ns/op	    5384 B/op	      12 allocs/op
// BenchmarkNetSnmpFileRead/BENCH_FILE_READ              	   48685	     23706 ns/op	     136 B/op	       3 allocs/op
// BenchmarkNetSnmpFileRead/BENCH_FILE_READ_SCAN_BYTES   	   46906	     26032 ns/op	    4232 B/op	       4 allocs/op
// BenchmarkNetSnmpFileRead/BENCH_FILE_READ_SCAN_TEXT    	   46956	     26926 ns/op	    5384 B/op	      12 allocs/op
// BenchmarkNetSnmpFileRead/BENCH_FILE_SCAN_BYTES        	   46678	     25833 ns/op	    4232 B/op	       4 allocs/op
