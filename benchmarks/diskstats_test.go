package benchmarks

import (
	"path"
	"testing"

	"github.com/eparparita/linux-stats-victoriametrics-importer/procfs"

	// Reference for performance comparison:
	prom_procfs_blockdevice "github.com/prometheus/procfs/blockdevice"
)

func BenchmarkDiskstatsParserIO(b *testing.B) {
	benchmarkFileRead(path.Join(LSVMI_TESTDATA_PROCFS_ROOT, "diskstats"), BENCH_FILE_READ, b)
}

func BenchmarkAllDiskstatsParserIO(b *testing.B) {
	for op, name := range benchFileReadOpMap {
		b.Run(
			name,
			func(b *testing.B) {
				benchmarkFileRead(path.Join(LSVMI_TESTDATA_PROCFS_ROOT, "diskstats"), op, b)
			},
		)
	}
}

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

// goos: darwin
// goarch: amd64
// pkg: github.com/eparparita/linux-stats-victoriametrics-importer/benchmarks
// cpu: Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz
// BenchmarkDiskstatsParserIO   	   71635	     16254 ns/op
// BenchmarkDiskstatsParser     	   60306	     20645 ns/op
// BenchmarkDiskstatsParserProm 	   10000	    104665 ns/op
