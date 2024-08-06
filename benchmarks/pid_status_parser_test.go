// Benchmarks for /proc/pid/status parser Invoke the parser and additionally
// simulate real life usage; the parsed data will be printed to a bytes.Buffer.

package benchmarks

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/emypar/linux-stats-victoriametrics-importer/procfs"

	// Reference for performance comparison:
	prom_procfs "github.com/prometheus/procfs"
)

var (
	benchPidStatusParserProcfsRoot = LSVMI_TESTDATA_PROCFS_ROOT
	benchPidStatusParserPid        = BENCH_PID
	benchPidStatusParserTid        = BENCH_TID
	benchPidStatusParserPidTidPath = procfs.BuildPidTidPath(
		benchPidStatusParserProcfsRoot, benchPidStatusParserPid, benchPidStatusParserTid)
	benchPidStatusParserPidTidStatusPath     = path.Join(benchPidStatusParserPidTidPath, "status")
	benchPidStatusParserPidTidList           = benchPidTidList
	benchPidStatusParserPidTidPathList       = benchPidTidPathList
	benchPidStatusParserPidTidStatusPathList = buildPidTidStatPathList(benchPidStatusParserPidTidPathList, "status")
)

// Benchmark single file:

func BenchmarkPidStatusParserIO(b *testing.B) {
	benchmarkFileRead(benchPidStatusParserPidTidStatusPath, BENCH_FILE_READ, b)
}

func BenchmarkPidStatusParser(b *testing.B) {
	pidStatus := procfs.NewPidStatus()
	for n := 0; n < b.N; n++ {
		err := pidStatus.Parse(benchPidStatusParserPidTidPath)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPidStatusParserProm(b *testing.B) {
	var proc prom_procfs.Proc
	fs, err := prom_procfs.NewFS(benchPidStatusParserProcfsRoot)
	if err != nil {
		b.Fatal(err)
	}

	if benchPidStatusParserTid > 0 {
		proc, err = fs.Thread(benchPidStatusParserPid, benchPidStatusParserTid)
	} else {
		proc, err = fs.Proc(benchPidStatusParserPid)
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
	b.Run(
		fmt.Sprintf("NFiles=%d", len(benchPidStatusParserPidTidStatusPathList)),
		func(b *testing.B) { benchmarkPidStatusAllParserIO(benchPidStatusParserPidTidStatusPathList, b) },
	)
}

func benchmarkPidStatusAllParser(pidTidPathList []string, b *testing.B) {
	pidStatus := procfs.NewPidStatus()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for _, pidTidPath := range pidTidPathList {
			err := pidStatus.Parse(pidTidPath)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkPidStatusAllParser(b *testing.B) {
	b.Run(
		fmt.Sprintf("NPidTid=%d", len(benchPidStatusParserPidTidPathList)),
		func(b *testing.B) { benchmarkPidStatusAllParser(benchPidStatusParserPidTidPathList, b) },
	)
}

func benchmarkPidStatusAllParserProm(fs prom_procfs.FS, pidTidList []PidTid, b *testing.B) {
	var (
		proc prom_procfs.Proc
		err  error
	)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for _, pidTid := range pidTidList {
			if pidTid[1] > 0 {
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
	fs, err := prom_procfs.NewFS(benchPidStatusParserProcfsRoot)
	if err != nil {
		b.Fatal(err)
	}

	b.Run(
		fmt.Sprintf("NPidTid=%d", len(benchPidStatusParserPidTidList)),
		func(b *testing.B) { benchmarkPidStatusAllParserProm(fs, benchPidStatusParserPidTidList, b) },
	)
}
