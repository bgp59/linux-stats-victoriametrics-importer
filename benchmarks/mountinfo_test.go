package benchmarks

import (
	"fmt"
	"path"
	"testing"

	"github.com/eparparita/linux-stats-victoriametrics-importer/procfs"
)

func BenchmarkMountinfoParserIO(b *testing.B) {
	benchmarkFileRead(path.Join(LSVMI_TESTDATA_PROCFS_ROOT, "1", "mountinfo"), b)
}

func benchmarkMountinfoParser(forceUpdate bool, b *testing.B) {
	mountinfo := procfs.NewMountInfo(LSVMI_TESTDATA_PROCFS_ROOT, 1)
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
