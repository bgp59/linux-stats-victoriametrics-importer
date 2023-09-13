// Benchmarks for /proc/pid/stat parser Invoke the parser and additionally
// simulate real life usage; the parsed data will be printed to a bytes.Buffer.

package benchmarks

import (
	"bytes"
	"fmt"

	"testing"

	"github.com/eparparita/linux-stats-victoriametrics-importer/procfs"

	// Reference for performance comparison:
	prom_procfs "github.com/prometheus/procfs"
)

var pidStatTestPid, pidStatTestTid int = 468, 486

// The following slice fields are expected to change often.
var pidStatByteSliceActiveFields = map[int]bool{
	procfs.PID_STAT_STATE:       true,
	procfs.PID_STAT_NUM_THREADS: true,
	procfs.PID_STAT_VSIZE:       true,
	procfs.PID_STAT_RSS:         true,
	procfs.PID_STAT_PROCESSOR:   true,
}

type PidStatParserBenchmarkCase struct {
	parseOnly     bool
	refreshFactor int
}

func benchmarkPidStatParser(bc *PidStatParserBenchmarkCase, b *testing.B) {
	var (
		prevPidStat *procfs.PidStatByteFields
		prevBuf     []byte
		wBuf        *bytes.Buffer
	)

	parseOnly, refreshFactor := bc.parseOnly, bc.refreshFactor

	if !parseOnly {
		// Prepare prev state based on the current file and modify the frequent
		// changing fields.
		prevPidStat = &procfs.PidStatByteFields{
			Buf: &bytes.Buffer{},
		}
		prevPidStat.SetPath(TestDataProcDir, pidStatTestPid, pidStatTestTid)
		err := prevPidStat.Parse()
		if err != nil {
			b.Fatal(err)
		}
		prevBuf = prevPidStat.Buf.Bytes()
		for i := range pidStatByteSliceActiveFields {
			prevBuf[prevPidStat.FieldEnd[i-1]] ^= 1
		}
		wBuf = &bytes.Buffer{}
	}

	pidStat := &procfs.PidStatByteFields{
		Buf: &bytes.Buffer{},
	}
	pidStat.SetPath(TestDataProcDir, pidStatTestPid, pidStatTestTid)

	for n := 0; n < b.N; n++ {
		err := pidStat.Parse()
		if err != nil {
			b.Fatal(err)
		}

		if parseOnly {
			continue
		}

		buf := pidStat.Buf.Bytes()
		fieldStart, fieldEnd := pidStat.FieldStart, pidStat.FieldEnd
		prevFieldStart, prevFieldEnd := prevPidStat.FieldStart, prevPidStat.FieldEnd
		wBuf.Reset()
		if refreshFactor <= 1 || (n%refreshFactor) == 0 {
			// Full cycle:
			for i := 0; i < procfs.PID_STAT_BYTE_SLICE_FIELD_COUNT; i++ {
				field := buf[fieldStart[i]:fieldEnd[i]]
				if !bytes.Equal(prevBuf[prevFieldStart[i]:prevFieldEnd[i]], field) {
					wBuf.Write(field)
				}
			}
		} else {
			for i := range pidStatByteSliceActiveFields {
				field := buf[fieldStart[i]:fieldEnd[i]]
				if !bytes.Equal(prevBuf[prevFieldStart[i]:prevFieldEnd[i]], field) {
					wBuf.Write(field)
				}
			}
		}
	}
}

func BenchmarkPidStatParser(b *testing.B) {
	for _, bc := range []*PidStatParserBenchmarkCase{
		{true, 0},
		{false, 0},
		{false, 15},
	} {
		b.Run(
			fmt.Sprintf("parseOnly=%v,refreshFactor=%d", bc.parseOnly, bc.refreshFactor),
			func(b *testing.B) { benchmarkPidStatParser(bc, b) },
		)
	}
}

func benchmarkPromPidStatParser(bc *PidStatParserBenchmarkCase, b *testing.B) {
	var (
		proc     prom_procfs.Proc
		prevStat prom_procfs.ProcStat
		wBuf     *bytes.Buffer
	)

	fs, err := prom_procfs.NewFS(TestDataProcDir)
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

	parseOnly, refreshFactor := bc.parseOnly, bc.refreshFactor

	if !parseOnly {
		prevStat, err = proc.Stat()
		if err != nil {
			b.Fatal(err)
		}
		prevStat.State += "_"
		prevStat.NumThreads += 100
		prevStat.VSize += 100
		prevStat.RSS += 100
		prevStat.Processor += 1024
		wBuf = &bytes.Buffer{}
	}

	for n := 0; n < b.N; n++ {
		stat, err := proc.Stat()
		if err != nil {
			b.Fatal(err)
		}
		if parseOnly {
			continue
		}
		wBuf.Reset()
		if refreshFactor <= 1 || (n%refreshFactor) == 0 {
			// Full cycle:
			if prevStat.Comm != stat.Comm {
				fmt.Fprintf(wBuf, "%s", stat.Comm)
			}
			if prevStat.State != stat.State {
				fmt.Fprintf(wBuf, "%s", stat.State)
			}
			if prevStat.PPID != stat.PPID {
				fmt.Fprintf(wBuf, "%d", stat.PPID)
			}
			if prevStat.PGRP != stat.PGRP {
				fmt.Fprintf(wBuf, "%d", stat.PGRP)
			}
			if prevStat.Session != stat.Session {
				fmt.Fprintf(wBuf, "%d", stat.Session)
			}
			if prevStat.TTY != stat.TTY {
				fmt.Fprintf(wBuf, "%d", stat.TTY)
			}
			if prevStat.TPGID != stat.TPGID {
				fmt.Fprintf(wBuf, "%d", stat.TPGID)
			}
			if prevStat.Priority != stat.Priority {
				fmt.Fprintf(wBuf, "%d", stat.Priority)
			}
			if prevStat.Nice != stat.Nice {
				fmt.Fprintf(wBuf, "%d", stat.Nice)
			}
			if prevStat.NumThreads != stat.NumThreads {
				fmt.Fprintf(wBuf, "%d", stat.NumThreads)
			}
			if prevStat.VSize != stat.VSize {
				fmt.Fprintf(wBuf, "%d", stat.VSize)
			}
			if prevStat.RSS != stat.RSS {
				fmt.Fprintf(wBuf, "%d", stat.RSS)
			}
			if prevStat.Processor != stat.Processor {
				fmt.Fprintf(wBuf, "%d", stat.Processor)
			}
		} else {
			if prevStat.State != stat.State {
				fmt.Fprintf(wBuf, "%s", stat.State)
			}
			if prevStat.NumThreads != stat.NumThreads {
				fmt.Fprintf(wBuf, "%d", stat.NumThreads)
			}
			if prevStat.VSize != stat.VSize {
				fmt.Fprintf(wBuf, "%d", stat.VSize)
			}
			if prevStat.RSS != stat.RSS {
				fmt.Fprintf(wBuf, "%d", stat.RSS)
			}
			if prevStat.Processor != stat.Processor {
				fmt.Fprintf(wBuf, "%d", stat.Processor)
			}
		}
	}
}

func BenchmarkPromPidStatParser(b *testing.B) {
	for _, bc := range []*PidStatParserBenchmarkCase{
		{true, 0},
		{false, 0},
		{false, 15},
	} {
		b.Run(
			fmt.Sprintf("parseOnly=%v,refreshFactor=%d", bc.parseOnly, bc.refreshFactor),
			func(b *testing.B) { benchmarkPromPidStatParser(bc, b) },
		)
	}
}
