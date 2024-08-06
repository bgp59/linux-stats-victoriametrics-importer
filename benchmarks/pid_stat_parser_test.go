// Benchmarks for /proc/pid/stat parser

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
	benchPidStatParserProcfsRoot = LSVMI_TESTDATA_PROCFS_ROOT
	benchPidStatParserPid        = BENCH_PID
	benchPidStatParserTid        = BENCH_TID
	benchPidStatParserPidTidPath = procfs.BuildPidTidPath(
		benchPidStatParserProcfsRoot, benchPidStatParserPid, benchPidStatParserTid)
	benchPidStatParserPidTidStatPath     = path.Join(benchPidStatParserPidTidPath, "stat")
	benchPidStatParserPidTidList         = benchPidTidList
	benchPidStatParserPidTidPathList     = benchPidTidPathList
	benchPidStatParserPidTidStatPathList = buildPidTidStatPathList(benchPidStatParserPidTidPathList, "stat")
)

// Benchmark a single file:

func BenchmarkPidStatParserIO(b *testing.B) {
	benchmarkFileRead(benchPidStatParserPidTidStatPath, BENCH_FILE_READ, b)
}

func BenchmarkPidStatParser(b *testing.B) {
	pidStat := procfs.NewPidStat()
	for n := 0; n < b.N; n++ {
		err := pidStat.Parse(benchPidStatParserPidTidPath)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPidStatParserProm(b *testing.B) {
	var proc prom_procfs.Proc
	fs, err := prom_procfs.NewFS(benchPidStatParserProcfsRoot)
	if err != nil {
		b.Fatal(err)
	}

	if benchPidStatParserTid != 0 {
		proc, err = fs.Thread(benchPidStatParserPid, benchPidStatParserTid)
	} else {
		proc, err = fs.Proc(benchPidStatParserPid)
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

// Benchmark all files (closer to real life situation):

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
	b.Run(
		fmt.Sprintf("NFiles=%d", len(benchPidStatParserPidTidStatPathList)),
		func(b *testing.B) { benchmarkPidStatAllParserIO(benchPidStatParserPidTidStatPathList, b) },
	)
}

func benchmarkPidStatAllParser(pidTidPathList []string, b *testing.B) {
	pidStat := procfs.NewPidStat()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for _, pidTidPath := range pidTidPathList {
			err := pidStat.Parse(pidTidPath)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkPidStatAllParser(b *testing.B) {
	b.Run(
		fmt.Sprintf("NPidTid=%d", len(benchPidStatParserPidTidPathList)),
		func(b *testing.B) { benchmarkPidStatAllParser(benchPidStatParserPidTidPathList, b) },
	)
}

func benchmarkPidStatAllParserProm(fs prom_procfs.FS, pidTidList []PidTid, b *testing.B) {
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
			_, err := proc.Stat()
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkPidStatAllParserProm(b *testing.B) {
	fs, err := prom_procfs.NewFS(benchPidStatParserProcfsRoot)
	if err != nil {
		b.Fatal(err)
	}

	b.Run(
		fmt.Sprintf("NPidTid=%d", len(benchPidStatParserPidTidList)),
		func(b *testing.B) { benchmarkPidStatAllParserProm(fs, benchPidStatParserPidTidList, b) },
	)
}
