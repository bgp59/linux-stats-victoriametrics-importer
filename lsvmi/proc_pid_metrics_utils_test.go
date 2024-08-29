// Utils for pid metrics tests:

package lsvmi

import (
	"bytes"
	"fmt"
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
	Cmdline string
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
	// Zero delta flags,for state cache:
	PidStatFltZeroDelta   []bool
	PidStatusCtxZeroDelta []bool
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
	// The timestamp from the most recent successful lookup and the fallback
	// value:
	lastUnixMilli, fallbackUnixMilli int64
}

func NewTestPidParsers(pidParserAndStateData []*TestPidParserStateData, procfsRoot string, fallbackUnixMilli int64) *TestPidParsers {
	tpp := &TestPidParsers{
		Data:              pidParserAndStateData,
		ProcfsRoot:        procfsRoot,
		byPidTidPath:      make(map[string]*TestPidParserStateData),
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
	pidParser *TestPidParsers
}

func (testPidStat *TestPidStat) Parse(pidTidPath string) error {
	testPidStat.parsedData = nil
	if testPidStat.pidParser != nil {
		if testPidParserData := testPidStat.pidParser.get(pidTidPath); testPidParserData != nil {
			testPidStat.parsedData = testPidParserData.PidStat
		}
	}
	if testPidStat.parsedData != nil {
		return nil
	}
	return fmt.Errorf("%s/stat: no such (test case) file", pidTidPath)
}

func (testPidStat *TestPidStat) GetByteSliceFields() [][]byte {
	if testPidStat.parsedData == nil || testPidStat.parsedData.ByteSliceFields == nil {
		return nil
	}
	byteSliceFields := make([][]byte, len(testPidStat.parsedData.ByteSliceFields))
	for i, s := range testPidStat.parsedData.ByteSliceFields {
		byteSliceFields[i] = []byte(s)
	}
	return byteSliceFields
}

func (testPidStat *TestPidStat) GetNumericFields() []uint64 {
	if testPidStat.parsedData != nil {
		return testPidStat.parsedData.NumericFields
	}
	return nil
}

func (tpp *TestPidParsers) NewPidStat() procfs.PidStatParser {
	return &TestPidStat{pidParser: tpp}
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
	pidParser *TestPidParsers
}

func (testPidStatus *TestPidStatus) Parse(pidTidPath string) error {
	testPidStatus.parsedData = nil
	if testPidStatus.pidParser != nil {
		if testPidParserData := testPidStatus.pidParser.get(pidTidPath); testPidParserData != nil {
			testPidStatus.parsedData = testPidParserData.PidStatus
		}
	}
	if testPidStatus.parsedData != nil {
		return nil
	}
	return fmt.Errorf("%s/status: no such (test case) file", pidTidPath)
}

func (testPidStatus *TestPidStatus) GetByteSliceFieldsAndUnits() ([][]byte, [][]byte) {
	byteSliceFields, byteSliceFieldUnit := [][]byte(nil), [][]byte(nil)
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
	return byteSliceFields, byteSliceFieldUnit
}

func (testPidStatus *TestPidStatus) GetNumericFields() []uint64 {
	if testPidStatus.parsedData != nil {
		return testPidStatus.parsedData.NumericFields
	}
	return nil
}

func (tpp *TestPidParsers) NewPidStatus() procfs.PidStatusParser {
	return &TestPidStatus{pidParser: tpp}
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
	pidParser *TestPidParsers
}

func (testPidCmdline *TestPidCmdline) Parse(pidTidPath string) error {
	testPidCmdline.parsedData = nil
	if testPidCmdline.pidParser != nil {
		if testPidParserData := testPidCmdline.pidParser.get(pidTidPath); testPidParserData != nil {
			testPidCmdline.parsedData = testPidParserData.PidCmdline
		}
	}
	if testPidCmdline.parsedData != nil {
		return nil
	}
	return fmt.Errorf("%s/cmdline: no such (test case) file", pidTidPath)
}

func (testPidCmdline *TestPidCmdline) GetCmdlineBytes() []byte {
	if testPidCmdline.parsedData != nil {
		return []byte(testPidCmdline.parsedData.Cmdline)
	}
	return nil
}

func (testPidCmdline *TestPidCmdline) GetCmdlineString() string {
	if testPidCmdline.parsedData != nil {
		return testPidCmdline.parsedData.Cmdline
	}
	return ""
}

func (tpp *TestPidParsers) NewPidCmdline() procfs.PidCmdlineParser {
	return &TestPidCmdline{pidParser: tpp}
}

func setTestPidCmdlineData(pidCmdlineParser procfs.PidCmdlineParser, data *TestPidCmdlineParsedData) {
	pidCmdlineParser.(*TestPidCmdline).parsedData = &TestPidCmdlineParsedData{
		Cmdline: data.Cmdline,
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
	copy(pidTidMetricsInfo.pidStatFltZeroDelta, pidParserState.PidStatFltZeroDelta)
	if pm.usePidStatus {
		setTestPidStatusData(pidTidMetricsInfo.pidStatus, pidParserState.PidStatus)
		copy(pidTidMetricsInfo.pidStatusCtxZeroDelta, pidParserState.PidStatusCtxZeroDelta)
	}
	pidTidMetricsInfo.prevTs = time.UnixMilli(pidParserState.UnixMilli)
	pidTidMetricsInfo.scanNum = pm.scanNum - 1
	pm.pidStat = savedPidStat
	return pidTidMetricsInfo
}

func cmpPidTidMetricsZeroDelta(
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

	if pidParserState.PidStatFltZeroDelta != nil {
		cmpOne("pidStatFltZeroDelta", pidParserState.PidStatFltZeroDelta, pidTidMetricsInfo.pidStatFltZeroDelta)
	}

	if pidParserState.PidStatusCtxZeroDelta != nil {
		cmpOne("pidStatusCtxZeroDelta", pidParserState.PidStatusCtxZeroDelta, pidTidMetricsInfo.pidStatusCtxZeroDelta)
	}

	return errBuf
}
