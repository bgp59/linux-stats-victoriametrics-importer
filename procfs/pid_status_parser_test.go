package procfs

import (
	"bytes"
	"fmt"
	"testing"
)

type PidStatusTestCase struct {
	procfsRoot          string
	pid, tid            int
	wantBytesDataValues map[int]string
	wantUlongDataValues map[int]uint64
	wantError           error
}

var pidStatusTestByteDataIndexToName = map[int]string{
	PID_STATUS_UID:               "PID_STATUS_UID",
	PID_STATUS_GID:               "PID_STATUS_GID",
	PID_STATUS_GROUPS:            "PID_STATUS_GROUPS",
	PID_STATUS_VM_PEAK:           "PID_STATUS_VM_PEAK",
	PID_STATUS_VM_SIZE:           "PID_STATUS_VM_SIZE",
	PID_STATUS_VM_LCK:            "PID_STATUS_VM_LCK",
	PID_STATUS_VM_PIN:            "PID_STATUS_VM_PIN",
	PID_STATUS_VM_HWM:            "PID_STATUS_VM_HWM",
	PID_STATUS_VM_RSS:            "PID_STATUS_VM_RSS",
	PID_STATUS_RSS_ANON:          "PID_STATUS_RSS_ANON",
	PID_STATUS_RSS_FILE:          "PID_STATUS_RSS_FILE",
	PID_STATUS_RSS_SHMEM:         "PID_STATUS_RSS_SHMEM",
	PID_STATUS_VM_DATA:           "PID_STATUS_VM_DATA",
	PID_STATUS_VM_STK:            "PID_STATUS_VM_STK",
	PID_STATUS_VM_EXE:            "PID_STATUS_VM_EXE",
	PID_STATUS_VM_LIB:            "PID_STATUS_VM_LIB",
	PID_STATUS_VM_PTE:            "PID_STATUS_VM_PTE",
	PID_STATUS_VM_PMD:            "PID_STATUS_VM_PMD",
	PID_STATUS_VM_SWAP:           "PID_STATUS_VM_SWAP",
	PID_STATUS_HUGETLBPAGES:      "PID_STATUS_HUGETLBPAGES",
	PID_STATUS_CPUS_ALLOWED_LIST: "PID_STATUS_CPUS_ALLOWED_LIST",
	PID_STATUS_MEMS_ALLOWED_LIST: "PID_STATUS_MEMS_ALLOWED_LIST",
}

var pidStatusTestUlongDataIndexToName = map[int]string{
	PID_STATUS_VOLUNTARY_CTXT_SWITCHES:    "PID_STATUS_VOLUNTARY_CTXT_SWITCHES",
	PID_STATUS_NONVOLUNTARY_CTXT_SWITCHES: "PID_STATUS_NONVOLUNTARY_CTXT_SWITCHES",
}

func testPidStatusParser(tc *PidStatusTestCase, t *testing.T) {
	pidStatus := NewPidStatus(tc.procfsRoot, tc.pid, tc.tid)
	err := pidStatus.Parse()

	if tc.wantError != nil {
		if err == nil || tc.wantError.Error() != err.Error() {
			t.Fatalf("want: %v error, got: %v", tc.wantError, err)
		}
		return
	}

	if err != nil {
		t.Fatal(err)
	}

	diffBuf := &bytes.Buffer{}

	for index := range tc.wantBytesDataValues {
		wantVal := tc.wantBytesDataValues[index]
		gotVal := string(pidStatus.bytesData.Bytes()[pidStatus.bytesStart[index]:pidStatus.bytesEnd[index]])
		if wantVal != gotVal {
			fmt.Fprintf(
				diffBuf,
				"\nbytesData[%s]: want: %q, got %q",
				pidStatusTestByteDataIndexToName[index],
				wantVal,
				gotVal,
			)
		}
	}

	for index := range tc.wantUlongDataValues {
		wantVal := tc.wantUlongDataValues[index]
		gotVal := pidStatus.ulongData[index]
		if wantVal != gotVal {
			fmt.Fprintf(
				diffBuf,
				"\nbytesData[%s]: want: %d, got %d",
				pidStatusTestUlongDataIndexToName[index],
				wantVal,
				gotVal,
			)
		}
	}

	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf)
	}
}

func TestPidStatusParser(t *testing.T) {
	for _, tc := range []*PidStatusTestCase{
		{
			procfsRoot: TESTDATA_PROCFS_ROOT,
			pid:        468,
			tid:        486,
			wantBytesDataValues: map[int]string{
				PID_STATUS_UID: "104,104,104,104",
				PID_STATUS_GID: "111,111,111,111",
			},
		},
	} {
		t.Run(
			fmt.Sprintf("pid=%d,tid=%d", tc.pid, tc.tid),
			func(t *testing.T) { testPidStatusParser(tc, t) },
		)
	}
}
