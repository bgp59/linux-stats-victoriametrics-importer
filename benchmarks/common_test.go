// Definitions common to all benchmarks:

package benchmarks

import (
	"bufio"
	"bytes"
	"os"
	"path"
	"strconv"
	"testing"

	"github.com/bgp59/linux-stats-victoriametrics-importer/procfs"
)

const (
	LSVMI_TESTDATA_PROCFS_ROOT = "../testdata/lsvmi/proc"
	BENCH_PID                  = 468
	BENCH_TID                  = 486
)

const (
	BENCH_FILE_READ = iota
	BENCH_FILE_READ_SCAN_BYTES
	BENCH_FILE_READ_SCAN_TEXT
	BENCH_FILE_SCAN_BYTES
	BENCH_FILE_SCAN_TEXT
)

type PidTid [2]int

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

func buildPidTidLists(procfsRoot string, pidOnly bool) ([]PidTid, []string, error) {
	dirEntries, err := os.ReadDir(procfsRoot)
	if err != nil {
		return nil, nil, err
	}
	pidTidList := make([]PidTid, 0)
	pidTidPathList := make([]string, 0)
	for _, dirEntry := range dirEntries {
		name := dirEntry.Name()
		pid, err := strconv.Atoi(name)
		if err == nil && pid > 0 {
			pidTidList = append(pidTidList, PidTid{pid, procfs.PID_ONLY_TID})
			pidRoot := path.Join(procfsRoot, name)
			pidTidPathList = append(pidTidPathList, pidRoot)
			taskRoot := path.Join(pidRoot, "task")
			if !pidOnly {
				dirEntries, err := os.ReadDir(taskRoot)
				if err == nil {
					for _, dirEntry := range dirEntries {
						name := dirEntry.Name()
						tid, err := strconv.Atoi(name)
						if err == nil && tid > 0 {
							pidTidList = append(pidTidList, PidTid{pid, tid})
							pidTidPathList = append(pidTidPathList, path.Join(taskRoot, name))
						}
					}
				}
			}
		}
	}
	return pidTidList, pidTidPathList, nil
}

func buildPidTidStatPathList(pidTidPathList []string, statPath string) []string {
	statPathList := make([]string, len(pidTidPathList))
	for i, pidTidPath := range pidTidPathList {
		statPathList[i] = path.Join(pidTidPath, statPath)
	}
	return statPathList
}

func initPidTidLists(procfsRoot string, pidOnly bool) ([]PidTid, []string) {
	pidTidList, pidTidPathList, err := buildPidTidLists(procfsRoot, pidOnly)
	if err != nil {
		panic(err)
	}
	return pidTidList, pidTidPathList
}

var benchPidTidList, benchPidTidPathList = initPidTidLists(LSVMI_TESTDATA_PROCFS_ROOT, false)
