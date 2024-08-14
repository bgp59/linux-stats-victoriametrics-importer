// Utils for pid metrics tests:

package lsvmi

import (
	"bytes"
	"fmt"

	"github.com/emypar/linux-stats-victoriametrics-importer/procfs"
)

// The test case data should be JSON loadable, they should emulate the pid parsers
// and they should be indexable by PID,TID:
type TestPidStatParsedData struct {
	// As-is fields, JSON loadable:
	ByteSliceFields []string
	// Numeric fields, JSON loadable:
	NumericFields []uint64
}

type TestPidStatusParsedData struct {
	// As-is fields, JSON loadable:
	ByteSliceFields    []string
	ByteSliceFieldUnit []string
	// Numeric fields, JSON loadable:
	NumericFields []uint64
}

type TestPidCmdlineParsedData struct {
	// JSON loadable:
	Cmdline string
}

type TestPidParserData struct {
	// Parsed data:
	PidStat    *TestPidStatParsedData
	PidStatus  *TestPidStatusParsedData
	PidCmdline *TestPidCmdlineParsedData
	// PID,TID for which the above applies:
	PidTid *procfs.PidTid
}

type TestPidParsersTestCaseData struct {
	// JSON loadable part:
	ParserData []TestPidParserData
	ProcfsRoot string
	// Index by pidTidPath for a given procfsRoot, as expected by parsers:
	byPidTidPath map[string]*TestPidParserData
}

func (tcd *TestPidParsersTestCaseData) get(pidTidPath string) *TestPidParserData {
	if tcd.byPidTidPath == nil {
		tcd.byPidTidPath = make(map[string]*TestPidParserData)
		for _, parserData := range tcd.ParserData {
			pidTidPath := procfs.BuildPidTidPath(tcd.ProcfsRoot, parserData.PidTid.Pid, parserData.PidTid.Tid)
			tcd.byPidTidPath[pidTidPath] = &parserData
		}
	}
	return tcd.byPidTidPath[pidTidPath]
}

// TestPidParsersTestCaseData gets loaded from the test case file and it is used
// to emulate parsers.

// Test PidStatParser:
type TestPidStat struct {
	// The most recent call to Parse result:
	parseResult *TestPidStatParsedData
	// Underlying test data:
	testCaseData *TestPidParsersTestCaseData
}

func (testPidStat *TestPidStat) Parse(pidTidPath string) error {
	testPidStat.parseResult = nil
	if testPidStat.testCaseData != nil {
		if testPidParserData := testPidStat.testCaseData.get(pidTidPath); testPidParserData != nil {
			testPidStat.parseResult = testPidParserData.PidStat
		}
	}
	if testPidStat.parseResult != nil {
		return nil
	}
	return fmt.Errorf("%s/stat: no such file", pidTidPath)
}

func (testPidStat *TestPidStat) GetByteSliceFields() [][]byte {
	if testPidStat.parseResult == nil || testPidStat.parseResult.ByteSliceFields == nil {
		return nil
	}
	byteSliceFields := make([][]byte, len(testPidStat.parseResult.ByteSliceFields))
	for i, s := range testPidStat.parseResult.ByteSliceFields {
		byteSliceFields[i] = []byte(s)
	}
	return byteSliceFields
}

func (testPidStat *TestPidStat) GetNumericFields() []uint64 {
	if testPidStat.parseResult != nil {
		return testPidStat.parseResult.NumericFields
	}
	return nil
}

func (tcd *TestPidParsersTestCaseData) NewPidStat() procfs.PidStatParser {
	return &TestPidStat{testCaseData: tcd}
}

func setTestPidStatData(pidStatParser procfs.PidStatParser, data *TestPidStatParsedData) {
	parseResult := &TestPidStatParsedData{}
	if data.ByteSliceFields != nil {
		parseResult.ByteSliceFields = make([]string, len(data.ByteSliceFields))
		copy(parseResult.ByteSliceFields, data.ByteSliceFields)
	}
	if data.NumericFields != nil {
		parseResult.NumericFields = make([]uint64, len(data.NumericFields))
		copy(parseResult.NumericFields, data.NumericFields)
	}
	pidStatParser.(*TestPidStat).parseResult = parseResult
}

// Test PidStatusParser:
type TestPidStatus struct {
	// The most recent call to Parse result:
	parseResult *TestPidStatusParsedData
	// Underlying test data:
	testCaseData *TestPidParsersTestCaseData
}

func (testPidStatus *TestPidStatus) Parse(pidTidPath string) error {
	testPidStatus.parseResult = nil
	if testPidStatus.testCaseData != nil {
		if testPidParserData := testPidStatus.testCaseData.get(pidTidPath); testPidParserData != nil {
			testPidStatus.parseResult = testPidParserData.PidStatus
		}
	}
	if testPidStatus.parseResult != nil {
		return nil
	}
	return fmt.Errorf("%s/status: no such file", pidTidPath)
}

func (testPidStatus *TestPidStatus) GetByteSliceFieldsAndUnits() ([][]byte, [][]byte) {
	byteSliceFields, byteSliceFieldUnit := [][]byte(nil), [][]byte(nil)
	if testPidStatus.parseResult != nil {
		if testPidStatus.parseResult.ByteSliceFields != nil {
			byteSliceFields = make([][]byte, len(testPidStatus.parseResult.ByteSliceFields))
			for i, s := range testPidStatus.parseResult.ByteSliceFields {
				byteSliceFields[i] = []byte(s)
			}
		}
		if testPidStatus.parseResult.ByteSliceFieldUnit != nil {
			byteSliceFieldUnit = make([][]byte, len(testPidStatus.parseResult.ByteSliceFieldUnit))
			for i, s := range testPidStatus.parseResult.ByteSliceFieldUnit {
				byteSliceFieldUnit[i] = []byte(s)
			}
		}
	}
	return byteSliceFields, byteSliceFieldUnit
}

func (testPidStatus *TestPidStatus) GetNumericFields() []uint64 {
	if testPidStatus.parseResult != nil {
		return testPidStatus.parseResult.NumericFields
	}
	return nil
}

func (tcd *TestPidParsersTestCaseData) NewPidStatus() procfs.PidStatusParser {
	return &TestPidStatus{testCaseData: tcd}
}

func setTestPidStatusData(pidStatParser procfs.PidStatusParser, data *TestPidStatusParsedData) {
	parseResult := &TestPidStatusParsedData{}
	if data.ByteSliceFields != nil {
		parseResult.ByteSliceFields = make([]string, len(data.ByteSliceFields))
		copy(parseResult.ByteSliceFields, data.ByteSliceFields)
	}
	if data.ByteSliceFieldUnit != nil {
		parseResult.ByteSliceFieldUnit = make([]string, len(data.ByteSliceFieldUnit))
		copy(parseResult.ByteSliceFieldUnit, data.ByteSliceFieldUnit)
	}
	if data.NumericFields != nil {
		parseResult.NumericFields = make([]uint64, len(data.NumericFields))
		copy(parseResult.NumericFields, data.NumericFields)
	}

	pidStatParser.(*TestPidStatus).parseResult = parseResult
}

// Test PidCmdlineParser:
type TestPidCmdline struct {
	// The most recent call to Parse result:
	parseResult *TestPidCmdlineParsedData
	// Underlying test data:
	testCaseData *TestPidParsersTestCaseData
}

func (testPidCmdline *TestPidCmdline) Parse(pidTidPath string) error {
	testPidCmdline.parseResult = nil
	if testPidCmdline.testCaseData != nil {
		if testPidParserData := testPidCmdline.testCaseData.get(pidTidPath); testPidParserData != nil {
			testPidCmdline.parseResult = testPidParserData.PidCmdline
		}
	}
	if testPidCmdline.parseResult != nil {
		return nil
	}
	return fmt.Errorf("%s/cmdline: no such file", pidTidPath)
}

func (testPidCmdline *TestPidCmdline) GetCmdlineBytes() []byte {
	if testPidCmdline.parseResult != nil {
		return []byte(testPidCmdline.parseResult.Cmdline)
	}
	return nil
}

func (testPidCmdline *TestPidCmdline) GetCmdlineString() string {
	if testPidCmdline.parseResult != nil {
		return testPidCmdline.parseResult.Cmdline
	}
	return ""
}

func (tcd *TestPidParsersTestCaseData) NewPidCmdline() procfs.PidCmdlineParser {
	return &TestPidCmdline{testCaseData: tcd}
}

func setTestPidCmdlineData(pidCmdlineParser procfs.PidCmdlineParser, data *TestPidCmdlineParsedData) {
	pidCmdlineParser.(*TestPidCmdline).parseResult = &TestPidCmdlineParsedData{
		Cmdline: data.Cmdline,
	}
}

// ProcPidTidMetricsInfo:
type TestProcPidTidMetricsInfoData struct {
	PidStatData           *TestPidStatParsedData
	PidStatusData         *TestPidStatusParsedData
	PidStatFltZeroDelta   []bool
	PidStatusCtxZeroDelta []bool
	// PID,TID for which the above applies:
	PidTid procfs.PidTid
}

func buildTestPidTidMetricsInfo(pm *ProcPidMetrics, data any) *ProcPidTidMetricsInfo {
	var pidTidMetricsInfo *ProcPidTidMetricsInfo

	savedPidStat := pm.pidStat
	pm.pidStat = pm.newPidStatParser()

	switch data := data.(type) {
	case *TestProcPidTidMetricsInfoData:
		setTestPidStatData(pm.pidStat, data.PidStatData)
		pidTid := data.PidTid
		pidTidPath := procfs.BuildPidTidPath(pm.procfsRoot, pidTid.Pid, pidTid.Tid)
		pidTidMetricsInfo = pm.initPidTidMetricsInfo(pidTid, pidTidPath)
		setTestPidStatData(pidTidMetricsInfo.pidStat, data.PidStatData)
		copy(pidTidMetricsInfo.pidStatFltZeroDelta, data.PidStatFltZeroDelta)
		if pm.usePidStatus {
			setTestPidStatusData(pidTidMetricsInfo.pidStatus, data.PidStatusData)
			copy(pidTidMetricsInfo.pidStatusCtxZeroDelta, data.PidStatusCtxZeroDelta)
		}
		pidTidMetricsInfo.prevTs = pm.prevTs
	case *TestPidParserData:
		setTestPidStatData(pm.pidStat, data.PidStat)
		pidTid := data.PidTid
		pidTidPath := procfs.BuildPidTidPath(pm.procfsRoot, pidTid.Pid, pidTid.Tid)
		pidTidMetricsInfo = pm.initPidTidMetricsInfo(*pidTid, pidTidPath)
	}

	pm.pidStat = savedPidStat
	return pidTidMetricsInfo
}

func cmpPidTidMetricsZeroDelta(pidTid *procfs.PidTid, pidTidMetricsInfo *ProcPidTidMetricsInfo, testData *TestProcPidTidMetricsInfoData, errBuf *bytes.Buffer) *bytes.Buffer {
	if errBuf == nil {
		errBuf = &bytes.Buffer{}
	}

	pidTidPrefix := ""
	if pidTid != nil {
		pidTidPrefix = fmt.Sprintf("pid: %d, tid: %d, ", pidTid.Pid, pidTid.Tid)
	}
	cmpOne := func(deltaName string, wantZeroDelta, gotZeroDelta []bool) {
		if len(wantZeroDelta) != len(gotZeroDelta) {
			fmt.Fprintf(
				errBuf,
				"%slen(%s): want: %d, got: %d",
				pidTidPrefix, deltaName, len(wantZeroDelta), len(gotZeroDelta),
			)
			return
		}
		for i, want := range wantZeroDelta {
			got := gotZeroDelta[i]
			if want != got {
				fmt.Fprintf(
					errBuf,
					"%s%s[%d]: want: %v, got: %v",
					pidTidPrefix, deltaName, i, want, got,
				)
			}
		}
	}

	if testData.PidStatFltZeroDelta != nil {
		cmpOne("pidStatFltZeroDelta", testData.PidStatFltZeroDelta, pidTidMetricsInfo.pidStatFltZeroDelta)
	}

	if testData.PidStatusCtxZeroDelta != nil {
		cmpOne("pidStatusCtxZeroDelta", testData.PidStatusCtxZeroDelta, pidTidMetricsInfo.pidStatusCtxZeroDelta)
	}

	return errBuf
}
