// Utils for pid metrics tests:

package lsvmi

import (
	"fmt"

	"github.com/emypar/linux-stats-victoriametrics-importer/procfs"
)

// The test case data should be JSON loadable, they should emulate the pid parsers
// and they should be indexable by PID,TID:
type TestPidStatPayload struct {
	// As-is fields, JSON loadable:
	ByteSliceFields []string
	// Numeric fields, JSON loadable:
	NumericFields []uint64
}

type TestPidStatusPayload struct {
	// As-is fields, JSON loadable:
	ByteSliceFields    []string
	ByteSliceFieldUnit []string
	// Numeric fields, JSON loadable:
	NumericFields []uint64
}

type TestPidCmdlinePayload struct {
	// JSON loadable:
	Cmdline string
}

type TestPidParserPayloads struct {
	PidStat    *TestPidStatPayload
	PidStatus  *TestPidStatusPayload
	PidCmdline *TestPidCmdlinePayload
	PidTid     *procfs.PidTid
}

type TestPidParsersTestCaseData struct {
	// JSON loadable part:
	Payloads   []*TestPidParserPayloads
	ProcfsRoot string

	// Indexed by pidTidPath for a given procfsRoot, as expected by parsers:
	byPidTidPath map[string]*TestPidParserPayloads
}

func (tcd *TestPidParsersTestCaseData) get(pidTidPath string) *TestPidParserPayloads {
	if tcd.byPidTidPath == nil {
		tcd.byPidTidPath = make(map[string]*TestPidParserPayloads)
		for _, payload := range tcd.Payloads {
			pidTidPath := procfs.BuildPidTidPath(tcd.ProcfsRoot, payload.PidTid.Pid, payload.PidTid.Tid)
			tcd.byPidTidPath[pidTidPath] = payload
		}
	}
	return tcd.byPidTidPath[pidTidPath]
}

// TestPidParsersTestCaseData gets loaded from the test case file and it used to
// emulate parsers.

// Test PidStatParser:
type TestPidStat struct {
	// The most recent call to Parse result:
	parseResult *TestPidStatPayload
	// Underlying test data:
	data *TestPidParsersTestCaseData
}

func (testPidStat *TestPidStat) Parse(pidTidPath string) error {
	testPidStat.parseResult = nil
	if testPidStat.data != nil {
		if testPidPayload := testPidStat.data.get(pidTidPath); testPidPayload != nil {
			testPidStat.parseResult = testPidPayload.PidStat
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
	return &TestPidStat{data: tcd}
}

// Test PidStatusParser:
type TestPidStatus struct {
	// The most recent call to Parse result:
	parseResult *TestPidStatusPayload
	// Underlying test data:
	data *TestPidParsersTestCaseData
}

func (testPidStatus *TestPidStatus) Parse(pidTidPath string) error {
	testPidStatus.parseResult = nil
	if testPidStatus.data != nil {
		if testPidPayload := testPidStatus.data.get(pidTidPath); testPidPayload != nil {
			testPidStatus.parseResult = testPidPayload.PidStatus
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
	return &TestPidStatus{data: tcd}
}

// Test PidCmdlineParser:
type TestPidCmdline struct {
	// The most recent call to Parse result:
	parseResult *TestPidCmdlinePayload
	// Underlying test data:
	data *TestPidParsersTestCaseData
}

func (testPidCmdline *TestPidCmdline) Parse(pidTidPath string) error {
	testPidCmdline.parseResult = nil
	if testPidCmdline.data != nil {
		if testPidPayload := testPidCmdline.data.get(pidTidPath); testPidPayload != nil {
			testPidCmdline.parseResult = testPidPayload.PidCmdline
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
	return &TestPidCmdline{data: tcd}
}
