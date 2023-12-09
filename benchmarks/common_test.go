// Definitions common to all benchmarks:

package benchmarks

import (
	"bufio"
	"bytes"
	"os"
	"path"
	"strconv"
	"testing"
)

const (
	LSVMI_TESTDATA_PROCFS_ROOT = "../testdata/lsvmi/proc"
)

const (
	BENCH_FILE_READ = iota
	BENCH_FILE_READ_SCAN_BYTES
	BENCH_FILE_READ_SCAN_TEXT
	BENCH_FILE_SCAN_BYTES
	BENCH_FILE_SCAN_TEXT
)

var benchFileReadOpMap = map[int]string{
	BENCH_FILE_READ:            "BENCH_FILE_READ",
	BENCH_FILE_READ_SCAN_BYTES: "BENCH_FILE_READ_SCAN_BYTES",
	BENCH_FILE_READ_SCAN_TEXT:  "BENCH_FILE_READ_SCAN_TEXT",
	BENCH_FILE_SCAN_BYTES:      "BENCH_FILE_SCAN_BYTES",
	BENCH_FILE_SCAN_TEXT:       "BENCH_FILE_SCAN_TEXT",
}

func pidTidPath(procfsRoot string, pid, tid int, statFile string) string {
	if tid > 0 {
		return path.Join(
			procfsRoot, strconv.Itoa(pid), "task", strconv.Itoa(tid), statFile,
		)
	} else {
		return path.Join(
			procfsRoot, strconv.Itoa(pid), statFile,
		)
	}
}

func benchmarkFileRead(path string, op int, b *testing.B) {
	buf := &bytes.Buffer{}
	for n := 0; n < b.N; n++ {
		f, err := os.Open(path)
		if err != nil {
			b.Fatal(err)
		}
		switch op {
		case BENCH_FILE_READ, BENCH_FILE_READ_SCAN_BYTES, BENCH_FILE_READ_SCAN_TEXT:
			buf.Reset()
			_, err = buf.ReadFrom(f)
			if err != nil {
				b.Fatal(err)
			}
			if op != BENCH_FILE_READ {
				scanner := bufio.NewScanner(buf)
				for scanner.Scan() {
					if op == BENCH_FILE_READ_SCAN_BYTES {
						_ = scanner.Bytes()
					} else {
						_ = scanner.Text()
					}
				}
				err = scanner.Err()
				if err != nil {
					b.Fatal(err)
				}
			}
		case BENCH_FILE_SCAN_BYTES, BENCH_FILE_SCAN_TEXT:
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				if op == BENCH_FILE_SCAN_BYTES {
					_ = scanner.Bytes()
				} else {
					_ = scanner.Text()
				}
			}
			err = scanner.Err()
			if err != nil {
				b.Fatal(err)
			}
		}
		f.Close()
	}
}
