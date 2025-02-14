package benchmarks

import (
	"fmt"
	"testing"

	"github.com/bgp59/linux-stats-victoriametrics-importer/procfs"
)

func BenchmarkMountinfoParserIO(b *testing.B) {
	benchmarkFileRead(procfs.MountinfoPath(LSVMI_TESTDATA_PROCFS_ROOT, 1), BENCH_FILE_READ, b)
}

func benchmarkMountinfoParser(forceUpdate bool, b *testing.B) {
	mountinfo := procfs.NewMountinfo(LSVMI_TESTDATA_PROCFS_ROOT, 1)
	mountinfo.ForceUpdate = forceUpdate
	for n := 0; n < b.N; n++ {
		err := mountinfo.Parse()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMountinfoParser(b *testing.B) {
	for _, forceUpdate := range []bool{true, false} {
		b.Run(
			fmt.Sprintf("forceUpdate=%v", forceUpdate),
			func(b *testing.B) { benchmarkMountinfoParser(forceUpdate, b) },
		)
	}
}

func BenchmarkMountinfoFileRead(b *testing.B) {
	for op, name := range benchFileReadOpMap {
		b.Run(
			name,
			func(b *testing.B) {
				benchmarkFileRead(procfs.MountinfoPath(LSVMI_TESTDATA_PROCFS_ROOT, 1), op, b)
			},
		)
	}
}
