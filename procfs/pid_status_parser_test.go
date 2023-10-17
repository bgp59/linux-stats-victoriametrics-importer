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
	wantUnit            map[int]string
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
				"\nbytesData[%s]: want: %q, got: %q",
				pidStatusTestByteDataIndexToName[index],
				wantVal,
				gotVal,
			)
		}
	}

	for index := range tc.wantUnit {
		wantVal := tc.wantUnit[index]
		gotVal := pidStatus.unit[index]
		if wantVal != gotVal {
			fmt.Fprintf(
				diffBuf,
				"\nunit[%s]: want: %q, got: %q",
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
				"\nulongData[%s]: want: %d, got: %d",
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
				PID_STATUS_UID:               "10400,10401,10402,10403",
				PID_STATUS_GID:               "11100,11101,11102,11103",
				PID_STATUS_GROUPS:            "4,111",
				PID_STATUS_VM_PEAK:           "2224000",
				PID_STATUS_VM_SIZE:           "2224001",
				PID_STATUS_VM_LCK:            "2",
				PID_STATUS_VM_PIN:            "3",
				PID_STATUS_VM_HWM:            "53604",
				PID_STATUS_VM_RSS:            "53605",
				PID_STATUS_RSS_ANON:          "10806",
				PID_STATUS_RSS_FILE:          "42707",
				PID_STATUS_RSS_SHMEM:         "8",
				PID_STATUS_VM_DATA:           "183409",
				PID_STATUS_VM_STK:            "13210",
				PID_STATUS_VM_EXE:            "43211",
				PID_STATUS_VM_LIB:            "510412",
				PID_STATUS_VM_PTE:            "7613",
				PID_STATUS_VM_PMD:            "",
				PID_STATUS_VM_SWAP:           "14",
				PID_STATUS_HUGETLBPAGES:      "15",
				PID_STATUS_CPUS_ALLOWED_LIST: "0,1,2,3",
				PID_STATUS_MEMS_ALLOWED_LIST: "0,1",
			},
			wantUnit: map[int]string{
				PID_STATUS_UID:               "",
				PID_STATUS_GID:               "",
				PID_STATUS_GROUPS:            "",
				PID_STATUS_VM_PEAK:           "kB",
				PID_STATUS_VM_SIZE:           "kB",
				PID_STATUS_VM_LCK:            "kB",
				PID_STATUS_VM_PIN:            "kB",
				PID_STATUS_VM_HWM:            "kB",
				PID_STATUS_VM_RSS:            "kB",
				PID_STATUS_RSS_ANON:          "kB",
				PID_STATUS_RSS_FILE:          "kB",
				PID_STATUS_RSS_SHMEM:         "kB",
				PID_STATUS_VM_DATA:           "kB",
				PID_STATUS_VM_STK:            "kB",
				PID_STATUS_VM_EXE:            "kB",
				PID_STATUS_VM_LIB:            "kB",
				PID_STATUS_VM_PTE:            "kB",
				PID_STATUS_VM_PMD:            "",
				PID_STATUS_VM_SWAP:           "kB",
				PID_STATUS_HUGETLBPAGES:      "kB",
				PID_STATUS_CPUS_ALLOWED_LIST: "",
				PID_STATUS_MEMS_ALLOWED_LIST: "",
			},
			wantUlongDataValues: map[int]uint64{
				PID_STATUS_VOLUNTARY_CTXT_SWITCHES:    2588,
				PID_STATUS_NONVOLUNTARY_CTXT_SWITCHES: 12,
			},
		},
	} {
		t.Run(
			fmt.Sprintf("pid=%d,tid=%d", tc.pid, tc.tid),
			func(t *testing.T) { testPidStatusParser(tc, t) },
		)
	}
}
