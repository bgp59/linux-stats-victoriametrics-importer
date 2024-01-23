// Benchmarks for /proc/pid/stat parser Invoke the parser and additionally
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

var pidStatTestProcfsRoot = LSVMI_TESTDATA_PROCFS_ROOT
var pidStatTestPid, pidStatTestTid int = 468, 486

var testPidStatPidTidList []PidTidPair
var testPidStatFilePathList []string
var testPidStatList []*procfs.PidStat
var testPidStatLock = &sync.Mutex{}

func getTestPidStatPidTidListNoLock() ([]PidTidPair, error) {
	if testPidStatPidTidList == nil {
		pidTidList, err := getPidTidList(pidStatTestProcfsRoot, false)
		if err != nil {
			return nil, err
		}
		testPidStatPidTidList = pidTidList
	}
	return testPidStatPidTidList, nil
}

func getTestPidStatPidTidList() ([]PidTidPair, error) {
	testPidStatLock.Lock()
	defer testPidStatLock.Unlock()
	return getTestPidStatPidTidListNoLock()
}

func getTestPidStatFilePathList() ([]string, error) {
	testPidStatLock.Lock()
	defer testPidStatLock.Unlock()
	if testPidStatFilePathList == nil {
		pidTidList, err := getTestPidStatPidTidListNoLock()
		if err != nil {
			return nil, err
		}
		testPidStatFilePathList = make([]string, len(pidTidList))
		for i, pidTid := range pidTidList {
			testPidStatFilePathList[i] = pidTidPath(pidStatTestProcfsRoot, pidTid[0], pidTid[1], "stat")
		}
	}
	return testPidStatFilePathList, nil
}

func getTestPidStatList() ([]*procfs.PidStat, error) {
	testPidStatLock.Lock()
	defer testPidStatLock.Unlock()
	if testPidStatList == nil {
		pidTidList, err := getTestPidStatPidTidListNoLock()
		if err != nil {
			return nil, err
		}
		testPidStatList = make([]*procfs.PidStat, len(pidTidList))
		for i, pidTid := range pidTidList {
			testPidStatList[i] = procfs.NewPidStat(pidStatTestProcfsRoot, pidTid[0], pidTid[1])
		}
	}
	return testPidStatList, nil
}

func BenchmarkPidStatParserIO(b *testing.B) {
	benchmarkFileRead(
		pidTidPath(pidStatTestProcfsRoot, pidStatTestPid, pidStatTestTid, "stat"),
		BENCH_FILE_READ,
		b,
	)
}

func BenchmarkPidStatParser(b *testing.B) {
	pidStat := procfs.NewPidStat(pidStatTestProcfsRoot, pidStatTestPid, pidStatTestTid)
	for n := 0; n < b.N; n++ {
		err := pidStat.Parse(nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPidStatParserProm(b *testing.B) {
	var proc prom_procfs.Proc
	fs, err := prom_procfs.NewFS(pidStatTestProcfsRoot)
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
		_, err := proc.Stat()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// goos: darwin
// goarch: amd64
// pkg: github.com/eparparita/linux-stats-victoriametrics-importer/benchmarks
// cpu: Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz
// BenchmarkPidStatParserIO   	   69865	     16900 ns/op	     152 B/op	       3 allocs/op
// BenchmarkPidStatParser     	   66681	     17642 ns/op	     152 B/op	       3 allocs/op
// BenchmarkPidStatParserProm 	   45513	     26569 ns/op	    1336 B/op	      31 allocs/op

func benchmarkPidStatAllParserIO(fPathList []string, b *testing.B) {
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

func BenchmarkPidStatAllParserIO(b *testing.B) {
	fPathList, err := getTestPidStatFilePathList()
	if err != nil {
		b.Fatal(err)
	}
	b.Run(
		fmt.Sprintf("NFiles=%d", len(fPathList)),
		func(b *testing.B) { benchmarkPidStatAllParserIO(fPathList, b) },
	)
}

func benchmarkPidStatAllParser(pidStatList []*procfs.PidStat, b *testing.B) {
	pidStat := procfs.NewPidStat(pidStatTestProcfsRoot, pidStatTestPid, pidStatTestTid)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for i := 0; i < len(pidStatList); i++ {
			err := pidStat.Parse(pidStatList[i])
			if err != nil {
				b.Fatal(err)
			}
			pidStatList[i], pidStat = pidStat, pidStatList[i]
		}
	}
}

func BenchmarkPidStatAllParser(b *testing.B) {
	pidStatList, err := getTestPidStatList()
	if err != nil {
		b.Fatal(err)
	}
	b.Run(
		fmt.Sprintf("NPidTid=%d", len(pidStatList)),
		func(b *testing.B) { benchmarkPidStatAllParser(pidStatList, b) },
	)
}

func benchmarkPidStatAllParserProm(fs prom_procfs.FS, pidTidList []PidTidPair, b *testing.B) {
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
			_, err := proc.Stat()
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkPidStatAllParserProm(b *testing.B) {
	fs, err := prom_procfs.NewFS(pidStatTestProcfsRoot)
	if err != nil {
		b.Fatal(err)
	}
	pidTidList, err := getTestPidStatPidTidList()
	if err != nil {
		b.Fatal(err)
	}

	b.Run(
		fmt.Sprintf("NPidTid=%d", len(pidTidList)),
		func(b *testing.B) { benchmarkPidStatAllParserProm(fs, pidTidList, b) },
	)
}

// goos: darwin
// goarch: amd64
// pkg: github.com/eparparita/linux-stats-victoriametrics-importer/benchmarks
// cpu: Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz
// BenchmarkPidStatAllParserIO/NFiles=241         	     265	   4287871 ns/op	   35229 B/op	     723 allocs/op
// BenchmarkPidStatAllParser/NPidTid=241          	     278	   4358144 ns/op	   35229 B/op	     723 allocs/op
// BenchmarkPidStatAllParserProm/NPidTid=241      	     146	   8092249 ns/op	  381361 B/op	    7232 allocs/op
