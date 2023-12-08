package benchmarks

import (
	"path"
	"testing"

	"github.com/eparparita/linux-stats-victoriametrics-importer/procfs"

	// Reference for performance comparison:
	prom_procfs "github.com/prometheus/procfs"
)

func BenchmarkNetSnmp6ParserIO(b *testing.B) {
	benchmarkFileRead(path.Join(LSVMI_TESTDATA_PROCFS_ROOT, "net", "snmp6"), BENCH_FILE_READ, b)
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

// goos: darwin
// goarch: amd64
// pkg: github.com/eparparita/linux-stats-victoriametrics-importer/benchmarks
// cpu: Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz
// BenchmarkNetSnmp6ParserIO   	   49329	     23998 ns/op	     152 B/op	       3 allocs/op
// BenchmarkNetSnmp6Parser     	   41250	     28727 ns/op	     176 B/op	       4 allocs/op
// BenchmarkNetSnmp6ParserProm 	   20584	     58837 ns/op	   20040 B/op	     275 allocs/op

func BenchmarkNetSnmp6FileRead(b *testing.B) {
	for op, name := range benchFileReadOpMap {
		b.Run(
			name,
			func(b *testing.B) {
				benchmarkFileRead(path.Join(LSVMI_TESTDATA_PROCFS_ROOT, "net", "snmp6"), op, b)
			},
		)
	}
}

// goos: darwin
// goarch: amd64
// pkg: github.com/eparparita/linux-stats-victoriametrics-importer/benchmarks
// cpu: Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz
// BenchmarkNetSnmp6FileRead/BENCH_FILE_READ         	   			   50306	     23615 ns/op	     152 B/op	       3 allocs/op
// BenchmarkNetSnmp6FileRead/BENCH_FILE_READ_SCAN_BYTES         	   43284	     27055 ns/op	    4248 B/op	       4 allocs/op
// BenchmarkNetSnmp6FileRead/BENCH_FILE_READ_SCAN_TEXT          	   39408	     29140 ns/op	    8424 B/op	      91 allocs/op
// BenchmarkNetSnmp6FileRead/BENCH_FILE_SCAN_BYTES              	   46892	     26018 ns/op	    4248 B/op	       4 allocs/op
// BenchmarkNetSnmp6FileRead/BENCH_FILE_SCAN_TEXT               	   40375	     33990 ns/op	    8424 B/op	      91 allocs/op
