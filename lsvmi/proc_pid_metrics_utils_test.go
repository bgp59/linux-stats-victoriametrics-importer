// Utils for pid metrics tests:

package lsvmi

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/emypar/linux-stats-victoriametrics-importer/procfs"
)

// The test case data should be JSON loadable, they should emulate the pid parsers
// and they should be indexable by PID,TID:
type TestPidStatParsedData struct {
	ByteSliceFields []string
	NumericFields   []uint64
}

type TestPidStatusParsedData struct {
	ByteSliceFields    []string
	ByteSliceFieldUnit []string
	NumericFields      []uint64
}

type TestPidCmdlineParsedData struct {
	CmdPath, Args string
}

// A structure to be used as test case data for both parsed results and
// previous state cache:
type TestPidParserStateData struct {
	// Parsed data:
	PidStat    *TestPidStatParsedData
	PidStatus  *TestPidStatusParsedData
	PidCmdline *TestPidCmdlineParsedData
	// Timestamp for the above, milliseconds since the epoch, similar to
	// Prometheus timestamp:
	UnixMilli int64
	// Active state, as per last scan:
	Active bool
	// Zero delta flags,for state cache:
	PidStatFltZeroDelta   []bool
	PidStatusCtxZeroDelta []bool
	// Cycle#:
	CycleNum int
	// PID,TID for which the above:
	PidTid *procfs.PidTid
}

// PID parsers test case data, it underlies test parsers implementing the
// expected interface:
type TestPidParsers struct {
	Data       []*TestPidParserStateData
	ProcfsRoot string
	// Index by pidTidPath for a given procfsRoot, as expected by parsers:
	byPidTidPath map[string]*TestPidParserStateData
	// Keep track of PID,TID in Data w/ a lookup error to exclude them from
	// consistency checks; this happens for simulated parser errors via
	// PidStat|PidStatus|PidCmdline set to nil.
	failedPidTid map[procfs.PidTid]bool
	// The timestamp from the most recent successful lookup and the fallback
	// value:
	lastUnixMilli, fallbackUnixMilli int64
}

func NewTestPidParsers(pidParserAndStateData []*TestPidParserStateData, procfsRoot string, fallbackUnixMilli int64) *TestPidParsers {
	tpp := &TestPidParsers{
		Data:              pidParserAndStateData,
		ProcfsRoot:        procfsRoot,
		byPidTidPath:      make(map[string]*TestPidParserStateData),
		failedPidTid:      make(map[procfs.PidTid]bool),
		lastUnixMilli:     fallbackUnixMilli,
		fallbackUnixMilli: fallbackUnixMilli,
	}
	for _, parserData := range tpp.Data {
		pidTidPath := procfs.BuildPidTidPath(tpp.ProcfsRoot, parserData.PidTid.Pid, parserData.PidTid.Tid)
		tpp.byPidTidPath[pidTidPath] = parserData
	}
	return tpp
}

func (tpp *TestPidParsers) get(pidTidPath string) *TestPidParserStateData {
	parserData := tpp.byPidTidPath[pidTidPath]
	if parserData != nil {
		tpp.lastUnixMilli = parserData.UnixMilli
	} else {
		tpp.lastUnixMilli = tpp.fallbackUnixMilli
	}
	return parserData
}

func (tpp *TestPidParsers) timeNow() time.Time {
	// Last timestamp can be used only once after a successful lookup:
	unixMilli := tpp.lastUnixMilli
	tpp.lastUnixMilli = tpp.fallbackUnixMilli
	return time.UnixMilli(unixMilli)
}

// TestPidParsers gets loaded from the test case file and it is used
// to emulate parsers.

type TestPidStat struct {
	// The most recent call to Parse result:
	parsedData *TestPidStatParsedData
	// Underlying test data:
	pidParsers *TestPidParsers
}

func (testPidStat *TestPidStat) Parse(pidTidPath string) error {
	testPidStat.parsedData = nil
	pidParsers := testPidStat.pidParsers
	if pidParsers != nil {
		if testPidParserData := pidParsers.get(pidTidPath); testPidParserData != nil {
			testPidStat.parsedData = testPidParserData.PidStat
			if testPidStat.parsedData == nil {
				pidParsers.failedPidTid[*testPidParserData.PidTid] = true
			}
		}
	}
	if testPidStat.parsedData != nil {
		return nil
	}
	pidParsers.lastUnixMilli = pidParsers.fallbackUnixMilli
	return fmt.Errorf("%s/stat: no such (test case) file", pidTidPath)
}

func (testPidStat *TestPidStat) GetData() (byteSliceFields [][]byte, numericFields []uint64) {
	if testPidStat.parsedData == nil {
		return
	}

	if testPidStat.parsedData.ByteSliceFields != nil {
		byteSliceFields = make([][]byte, len(testPidStat.parsedData.ByteSliceFields))
		for i, s := range testPidStat.parsedData.ByteSliceFields {
			byteSliceFields[i] = []byte(s)
		}
	}
	numericFields = testPidStat.parsedData.NumericFields
	return
}

func (tpp *TestPidParsers) NewPidStat() procfs.PidStatParser {
	return &TestPidStat{pidParsers: tpp}
}

func setTestPidStatData(pidStatParser procfs.PidStatParser, data *TestPidStatParsedData) {
	parsedData := &TestPidStatParsedData{}
	if data.ByteSliceFields != nil {
		parsedData.ByteSliceFields = make([]string, len(data.ByteSliceFields))
		copy(parsedData.ByteSliceFields, data.ByteSliceFields)
	}
	if data.NumericFields != nil {
		parsedData.NumericFields = make([]uint64, len(data.NumericFields))
		copy(parsedData.NumericFields, data.NumericFields)
	}
	pidStatParser.(*TestPidStat).parsedData = parsedData
}

// Test PidStatusParser:
type TestPidStatus struct {
	// The most recent call to Parse result:
	parsedData *TestPidStatusParsedData
	// Underlying test data:
	pidParsers *TestPidParsers
}

func (testPidStatus *TestPidStatus) Parse(pidTidPath string) error {
	testPidStatus.parsedData = nil
	pidParsers := testPidStatus.pidParsers
	if pidParsers != nil {
		if testPidParserData := pidParsers.get(pidTidPath); testPidParserData != nil {
			testPidStatus.parsedData = testPidParserData.PidStatus
			if testPidStatus.parsedData == nil {
				pidParsers.failedPidTid[*testPidParserData.PidTid] = true
			}
		}
	}
	if testPidStatus.parsedData != nil {
		return nil
	}
	pidParsers.lastUnixMilli = pidParsers.fallbackUnixMilli
	return fmt.Errorf("%s/status: no such (test case) file", pidTidPath)
}

func (testPidStatus *TestPidStatus) GetData() (byteSliceFields [][]byte, byteSliceFieldUnit [][]byte, numericFields []uint64) {
	if testPidStatus.parsedData == nil {
		return
	}
	if testPidStatus.parsedData != nil {
		if testPidStatus.parsedData.ByteSliceFields != nil {
			byteSliceFields = make([][]byte, len(testPidStatus.parsedData.ByteSliceFields))
			for i, s := range testPidStatus.parsedData.ByteSliceFields {
				byteSliceFields[i] = []byte(s)
			}
		}
		if testPidStatus.parsedData.ByteSliceFieldUnit != nil {
			byteSliceFieldUnit = make([][]byte, len(testPidStatus.parsedData.ByteSliceFieldUnit))
			for i, s := range testPidStatus.parsedData.ByteSliceFieldUnit {
				byteSliceFieldUnit[i] = []byte(s)
			}
		}
	}
	numericFields = testPidStatus.parsedData.NumericFields
	return
}

func (tpp *TestPidParsers) NewPidStatus() procfs.PidStatusParser {
	return &TestPidStatus{pidParsers: tpp}
}

func setTestPidStatusData(pidStatParser procfs.PidStatusParser, data *TestPidStatusParsedData) {
	parsedData := &TestPidStatusParsedData{}
	if data.ByteSliceFields != nil {
		parsedData.ByteSliceFields = make([]string, len(data.ByteSliceFields))
		copy(parsedData.ByteSliceFields, data.ByteSliceFields)
	}
	if data.ByteSliceFieldUnit != nil {
		parsedData.ByteSliceFieldUnit = make([]string, len(data.ByteSliceFieldUnit))
		copy(parsedData.ByteSliceFieldUnit, data.ByteSliceFieldUnit)
	}
	if data.NumericFields != nil {
		parsedData.NumericFields = make([]uint64, len(data.NumericFields))
		copy(parsedData.NumericFields, data.NumericFields)
	}

	pidStatParser.(*TestPidStatus).parsedData = parsedData
}

// Test PidCmdlineParser:
type TestPidCmdline struct {
	// The most recent call to Parse result:
	parsedData *TestPidCmdlineParsedData
	// Underlying test data:
	pidParsers *TestPidParsers
}

func (testPidCmdline *TestPidCmdline) Parse(pidTidPath string) error {
	testPidCmdline.parsedData = nil
	pidParsers := testPidCmdline.pidParsers
	if pidParsers != nil {
		if testPidParserData := pidParsers.get(pidTidPath); testPidParserData != nil {
			testPidCmdline.parsedData = testPidParserData.PidCmdline
			if testPidCmdline.parsedData == nil {
				pidParsers.failedPidTid[*testPidParserData.PidTid] = true
			}
		}
	}
	if testPidCmdline.parsedData != nil {
		return nil
	}
	pidParsers.lastUnixMilli = pidParsers.fallbackUnixMilli
	return fmt.Errorf("%s/cmdline: no such (test case) file", pidTidPath)
}

func (testPidCmdline *TestPidCmdline) GetData() ([]byte, []byte, []byte) {
	var cmdPath, args, cmd []byte
	if testPidCmdline.parsedData != nil {
		if len(testPidCmdline.parsedData.CmdPath) > 0 {
			cp := testPidCmdline.parsedData.CmdPath
			if i := strings.LastIndex(cp, "/"); i < 0 {
				cmd = []byte(cp)
			} else if i < len(cp)-1 {
				cmd = []byte(cp[i+1:])
			}
			cmdPath = []byte(cp)
		}
		if len(testPidCmdline.parsedData.Args) > 0 {
			args = []byte(testPidCmdline.parsedData.Args)
		}
	}
	return cmdPath, args, cmd
}

func (tpp *TestPidParsers) NewPidCmdline() procfs.PidCmdlineParser {
	return &TestPidCmdline{pidParsers: tpp}
}

func setTestPidCmdlineData(pidCmdlineParser procfs.PidCmdlineParser, data *TestPidCmdlineParsedData) {
	pidCmdlineParser.(*TestPidCmdline).parsedData = &TestPidCmdlineParsedData{
		CmdPath: data.CmdPath,
		Args:    data.Args,
	}
}

func buildTestPidTidMetricsInfo(pm *ProcPidMetrics, pidParserState *TestPidParserStateData) *ProcPidTidMetricsInfo {
	var pidTidMetricsInfo *ProcPidTidMetricsInfo

	savedPidStat := pm.pidStat
	pm.pidStat = pm.newPidStatParser()
	setTestPidStatData(pm.pidStat, pidParserState.PidStat)
	pidTid := *pidParserState.PidTid
	pidTidPath := procfs.BuildPidTidPath(pm.procfsRoot, pidTid.Pid, pidTid.Tid)
	pidTidMetricsInfo = pm.initPidTidMetricsInfo(pidTid, pidTidPath)
	setTestPidStatData(pidTidMetricsInfo.pidStat, pidParserState.PidStat)
	pidTidMetricsInfo.active = pidParserState.Active
	copy(pidTidMetricsInfo.pidStatFltZeroDelta, pidParserState.PidStatFltZeroDelta)
	if pm.usePidStatus {
		setTestPidStatusData(pidTidMetricsInfo.pidStatus, pidParserState.PidStatus)
		copy(pidTidMetricsInfo.pidStatusCtxZeroDelta, pidParserState.PidStatusCtxZeroDelta)
	}
	pidTidMetricsInfo.prevTs = time.UnixMilli(pidParserState.UnixMilli)
	pidTidMetricsInfo.cycleNum = pidParserState.CycleNum
	pidTidMetricsInfo.scanNum = pm.scanNum - 1
	pm.pidStat = savedPidStat
	return pidTidMetricsInfo
}

func cmpPidTidMetricsActiveZeroDelta(
	pidTidMetricsInfo *ProcPidTidMetricsInfo,
	pidParserState *TestPidParserStateData,
	errBuf *bytes.Buffer,
) *bytes.Buffer {
	if errBuf == nil {
		errBuf = &bytes.Buffer{}
	}

	cmpOne := func(deltaName string, wantZeroDelta, gotZeroDelta []bool) {
		if len(wantZeroDelta) != len(gotZeroDelta) {
			fmt.Fprintf(
				errBuf,
				"\n%#v: len(%s): want: %d, got: %d",
				pidTidMetricsInfo.pidTid, deltaName, len(wantZeroDelta), len(gotZeroDelta),
			)
			return
		}
		for i, want := range wantZeroDelta {
			got := gotZeroDelta[i]
			if want != got {
				fmt.Fprintf(
					errBuf,
					"\n%#v: %s[%d]: want: %v, got: %v",
					pidTidMetricsInfo.pidTid, deltaName, i, want, got,
				)
			}
		}
	}

	if pidParserState.Active != pidTidMetricsInfo.active {
		fmt.Fprintf(
			errBuf,
			"\n%#v: active: want: %v, got: %v",
			pidTidMetricsInfo.pidTid, pidParserState.Active, pidTidMetricsInfo.active,
		)
	}

	if pidParserState.PidStatFltZeroDelta != nil {
		cmpOne("pidStatFltZeroDelta", pidParserState.PidStatFltZeroDelta, pidTidMetricsInfo.pidStatFltZeroDelta)
	}

	if pidParserState.PidStatusCtxZeroDelta != nil {
		cmpOne("pidStatusCtxZeroDelta", pidParserState.PidStatusCtxZeroDelta, pidTidMetricsInfo.pidStatusCtxZeroDelta)
	}

	return errBuf
}
