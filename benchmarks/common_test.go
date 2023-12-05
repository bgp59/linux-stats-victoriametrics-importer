// Definitions common to all benchmarks:

package benchmarks

import (
	"bytes"
	"os"
	"testing"
)

const (
	LSVMI_TESTDATA_PROCFS_ROOT = "../testdata/lsvmi/proc"
)

func benchmarkFileRead(path string, b *testing.B) {
	buf := &bytes.Buffer{}
	for n := 0; n < b.N; n++ {
		f, err := os.Open(path)
		if err != nil {
			b.Fatal(err)
		}
		buf.Reset()
		_, err = buf.ReadFrom(f)
		if err != nil {
			b.Fatal(err)
		}
		f.Close()
	}
}
