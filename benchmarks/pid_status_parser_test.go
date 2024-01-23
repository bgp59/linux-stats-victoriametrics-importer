// Benchmarks for /proc/pid/status parser Invoke the parser and additionally
// simulate real life usage; the parsed data will be printed to a bytes.Buffer.

package benchmarks

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/eparparita/linux-stats-victoriametrics-importer/procfs"

	// Reference for performance comparison:
	prom_procfs "github.com/prometheus/procfs"
)

var pidStatusTestProcfsRoot = LSVMI_TESTDATA_PROCFS_ROOT
var pidStatusTestPid, pidStatusTestTid int = 468, 486

var testPidStatusPidTidList []PidTidPair
var testPidStatusFilePathList []string
var testPidStatusList []*procfs.PidStatus
var testPidStatusLock = &sync.Mutex{}

func getTestPidStatusPidTidListNoLock() ([]PidTidPair, error) {
	if testPidStatusPidTidList == nil {
		pidTidList, err := getPidTidList(pidStatusTestProcfsRoot, false)
		if err != nil {
			return nil, err
		}
		testPidStatusPidTidList = pidTidList
	}
	return testPidStatusPidTidList, nil
}

func getTestPidStatusPidTidList() ([]PidTidPair, error) {
	testPidStatusLock.Lock()
	defer testPidStatusLock.Unlock()
	return getTestPidStatusPidTidListNoLock()
}

func getTestPidStatusFilePathList() ([]string, error) {
	testPidStatusLock.Lock()
	defer testPidStatusLock.Unlock()
	if testPidStatusFilePathList == nil {
		pidTidList, err := getTestPidStatusPidTidListNoLock()
		if err != nil {
			return nil, err
		}
		testPidStatusFilePathList = make([]string, len(pidTidList))
		for i, pidTid := range pidTidList {
			testPidStatusFilePathList[i] = pidTidPath(pidStatusTestProcfsRoot, pidTid[0], pidTid[1], "status")
		}
	}
	return testPidStatusFilePathList, nil
}

func getTestPidStatusList() ([]*procfs.PidStatus, error) {
	testPidStatusLock.Lock()
	defer testPidStatusLock.Unlock()
	if testPidStatusList == nil {
		pidTidList, err := getTestPidStatusPidTidListNoLock()
		if err != nil {
			return nil, err
		}
		testPidStatusList = make([]*procfs.PidStatus, len(pidTidList))
		for i, pidTid := range pidTidList {
			testPidStatusList[i] = procfs.NewPidStatus(pidStatusTestProcfsRoot, pidTid[0], pidTid[1])
		}
	}
	return testPidStatusList, nil
}

// Benchmark single file:

func BenchmarkPidStatusParserIO(b *testing.B) {
	benchmarkFileRead(
		pidTidPath(pidStatusTestProcfsRoot, pidStatTestPid, pidStatTestTid, "status"),
		BENCH_FILE_READ,
		b,
	)
}

func BenchmarkPidStatusParser(b *testing.B) {
	pidStatus := procfs.NewPidStatus(pidStatusTestProcfsRoot, pidStatusTestPid, pidStatusTestTid)
	for n := 0; n < b.N; n++ {
		err := pidStatus.Parse(nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPidStatusParserProm(b *testing.B) {
	var proc prom_procfs.Proc
	fs, err := prom_procfs.NewFS(pidStatusTestProcfsRoot)
	if err != nil {
		b.Fatal(err)
	}

	if pidStatTestTid != 0 {
		proc, err = fs.Thread(pidStatTestPid, pidStatTestTid)
	} else {
		proc, err = fs.Proc(pidStatTestPid)
	}
	if err != nil {
		b.Fatal(err)
	}

	for n := 0; n < b.N; n++ {
		_, err := proc.NewStatus()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// goos: darwin
// goarch: amd64
// pkg: github.com/eparparita/linux-stats-victoriametrics-importer/benchmarks
// cpu: Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz
// BenchmarkPidStatusParserIO   	   69513	     16988 ns/op	     152 B/op	       3 allocs/op
// BenchmarkPidStatusParser     	   59361	     19700 ns/op	     176 B/op	       4 allocs/op
// BenchmarkPidStatusParserProm 	   35875	     32303 ns/op	    9224 B/op	     102 allocs/op

// Benchmark all files (closer to real life situation):

func benchmarkPidStatusAllParserIO(fPathList []string, b *testing.B) {
	buf := &bytes.Buffer{}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for _, fPath := range fPathList {
			f, err := os.Open(fPath)
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
}

func BenchmarkPidStatusAllParserIO(b *testing.B) {
	fPathList, err := getTestPidStatusFilePathList()
	if err != nil {
		b.Fatal(err)
	}
	b.Run(
		fmt.Sprintf("NFiles=%d", len(fPathList)),
		func(b *testing.B) { benchmarkPidStatusAllParserIO(fPathList, b) },
	)
}

func benchmarkPidStatusAllParser(pidStatusList []*procfs.PidStatus, b *testing.B) {
	pidStatus := procfs.NewPidStatus(pidStatusTestProcfsRoot, pidStatTestPid, pidStatTestTid)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for i := 0; i < len(pidStatusList); i++ {
			err := pidStatus.Parse(pidStatusList[i])
			if err != nil {
				b.Fatal(err)
			}
			pidStatusList[i], pidStatus = pidStatus, pidStatusList[i]
		}
	}
}

func BenchmarkPidStatusAllParser(b *testing.B) {
	pidStatusList, err := getTestPidStatusList()
	if err != nil {
		b.Fatal(err)
	}
	b.Run(
		fmt.Sprintf("NPidTid=%d", len(pidStatusList)),
		func(b *testing.B) { benchmarkPidStatusAllParser(pidStatusList, b) },
	)
}

func benchmarkPidStatusAllParserProm(fs prom_procfs.FS, pidTidList []PidTidPair, b *testing.B) {
	var (
		proc prom_procfs.Proc
		err  error
	)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for _, pidTid := range pidTidList {
			if pidTid[1] != 0 {
				proc, err = fs.Thread(pidTid[0], pidTid[1])
			} else {
				proc, err = fs.Proc(pidTid[0])
			}
			if err != nil {
				b.Fatal(err)
			}
			_, err := proc.NewStatus()
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkPidStatusAllParserProm(b *testing.B) {
	fs, err := prom_procfs.NewFS(pidStatTestProcfsRoot)
	if err != nil {
		b.Fatal(err)
	}
	pidTidList, err := getTestPidStatusPidTidList()
	if err != nil {
		b.Fatal(err)
	}

	b.Run(
		fmt.Sprintf("NPidTid=%d", len(pidTidList)),
		func(b *testing.B) { benchmarkPidStatusAllParserProm(fs, pidTidList, b) },
	)
}

// goos: darwin
// goarch: amd64
// pkg: github.com/eparparita/linux-stats-victoriametrics-importer/benchmarks
// cpu: Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz
// BenchmarkPidStatusAllParserIO/NFiles=241         	     265	   4986784 ns/op	   35229 B/op	     723 allocs/op
// BenchmarkPidStatusAllParser/NPidTid=241          	     217	   5776543 ns/op	   43204 B/op	    1149 allocs/op
// BenchmarkPidStatusAllParserProm/NPidTid=241      	     100	  10494631 ns/op	 2197619 B/op	   24787 allocs/op

// Benchmark file read strategies:

func BenchmarkPidStatusFileRead(b *testing.B) {
	path := pidTidPath(LSVMI_TESTDATA_PROCFS_ROOT, pidStatTestPid, pidStatTestTid, "status")
	for op, name := range benchFileReadOpMap {
		b.Run(
			name,
			func(b *testing.B) {
				benchmarkFileRead(path, op, b)
			},
		)
	}
}

// goos: darwin
// goarch: amd64
// pkg: github.com/eparparita/linux-stats-victoriametrics-importer/benchmarks
// cpu: Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz
// BenchmarkPidStatusFileRead/BENCH_FILE_READ         	   			   70216	     17076 ns/op	     152 B/op	       3 allocs/op
// BenchmarkPidStatusFileRead/BENCH_FILE_READ_SCAN_BYTES         	   62594	     19468 ns/op	    4248 B/op	       4 allocs/op
// BenchmarkPidStatusFileRead/BENCH_FILE_READ_SCAN_TEXT          	   59212	     20115 ns/op	    4664 B/op	       7 allocs/op
// BenchmarkPidStatusFileRead/BENCH_FILE_SCAN_BYTES              	   60781	     19285 ns/op	    4248 B/op	       4 allocs/op
// BenchmarkPidStatusFileRead/BENCH_FILE_SCAN_TEXT               	   61776	     19659 ns/op	    4664 B/op	       7 allocs/op
