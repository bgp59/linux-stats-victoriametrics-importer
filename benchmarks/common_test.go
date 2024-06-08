// Definitions common to all benchmarks:

package benchmarks

import (
	"bufio"
	"bytes"
	"os"
	"path"
	"strconv"
	"testing"

	"github.com/emypar/linux-stats-victoriametrics-importer/procfs"
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

type PidTidPair [2]int

var benchFileReadOpMap = map[int]string{
	BENCH_FILE_READ:            "BENCH_FILE_READ",
	BENCH_FILE_READ_SCAN_BYTES: "BENCH_FILE_READ_SCAN_BYTES",
	BENCH_FILE_READ_SCAN_TEXT:  "BENCH_FILE_READ_SCAN_TEXT",
	BENCH_FILE_SCAN_BYTES:      "BENCH_FILE_SCAN_BYTES",
	BENCH_FILE_SCAN_TEXT:       "BENCH_FILE_SCAN_TEXT",
}

func benchmarkFileRead(fPath string, op int, b *testing.B) {
	buf := &bytes.Buffer{}
	for n := 0; n < b.N; n++ {
		f, err := os.Open(fPath)
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

func getPidTidList(procfsRoot string, pidOnly bool) ([]PidTidPair, error) {
	dirEntries, err := os.ReadDir(procfsRoot)
	if err != nil {
		return nil, err
	}
	pidTidList := make([]PidTidPair, 0)
	for _, dirEntry := range dirEntries {
		name := dirEntry.Name()
		pid, err := strconv.Atoi(name)
		if err == nil && pid > 0 {
			pidTidList = append(pidTidList, PidTidPair{pid, procfs.PID_STAT_PID_ONLY_TID})
			if !pidOnly {
				dirEntries, err := os.ReadDir(path.Join(procfsRoot, name, "task"))
				if err == nil {
					for _, dirEntry := range dirEntries {
						tid, err := strconv.Atoi(dirEntry.Name())
						if err == nil && tid > 0 {
							pidTidList = append(pidTidList, PidTidPair{pid, tid})
						}
					}
				}
			}
		}
	}
	return pidTidList, nil
}
