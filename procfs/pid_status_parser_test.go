package procfs

import (
	"bytes"
	"fmt"
	"path"

	"testing"
)

var pidStatusTestdataDir = path.Join(PROCFS_TESTDATA_ROOT, "pid_status")

type PidStatusTestCase struct {
	name                     string
	procfsRoot               string
	pid, tid                 int
	primeProcfsRoot          string
	primePid, primeTid       int
	wantByteSliceFieldValues map[int]string
	wantByteSliceFieldUnit   map[int]string
	wantNumericFields        map[int]uint64
	wantError                error
}

var pidStatusByteSliceFieldsIndexToName = []string{
	"PID_STATUS_UID",
	"PID_STATUS_GID",
	"PID_STATUS_GROUPS",
	"PID_STATUS_VM_PEAK",
	"PID_STATUS_VM_SIZE",
	"PID_STATUS_VM_LCK",
	"PID_STATUS_VM_PIN",
	"PID_STATUS_VM_HWM",
	"PID_STATUS_VM_RSS",
	"PID_STATUS_RSS_ANON",
	"PID_STATUS_RSS_FILE",
	"PID_STATUS_RSS_SHMEM",
	"PID_STATUS_VM_DATA",
	"PID_STATUS_VM_STK",
	"PID_STATUS_VM_EXE",
	"PID_STATUS_VM_LIB",
	"PID_STATUS_VM_PTE",
	"PID_STATUS_VM_PMD",
	"PID_STATUS_VM_SWAP",
	"PID_STATUS_HUGETLBPAGES",
	"PID_STATUS_CPUS_ALLOWED_LIST",
	"PID_STATUS_MEMS_ALLOWED_LIST",
}

var pidStatusNumericFieldsIndexToName = []string{
	"PID_STATUS_VOLUNTARY_CTXT_SWITCHES",
	"PID_STATUS_NONVOLUNTARY_CTXT_SWITCHES",
}

func pidStatusSubtestName(tc *PidStatusTestCase) string {
	name := ""
	if tc.name != "" {
		name += fmt.Sprintf("name=%s", tc.name)
	}
	if name != "" {
		name += ","
	}
	name += fmt.Sprintf("procfsRoot=%s,pid=%d", tc.procfsRoot, tc.pid)
	if tc.tid != PID_STAT_PID_ONLY_TID {
		name += fmt.Sprintf(",tid=%d", tc.tid)
	}
	if tc.primePid > 0 {
		if tc.primeProcfsRoot != "" {
			name += fmt.Sprintf(",primeProcfsRoot=%s", tc.primeProcfsRoot)
		}
		name += fmt.Sprintf(",primePid=%d", tc.primePid)
		if tc.primeTid != PID_STAT_PID_ONLY_TID {
			name += fmt.Sprintf(",primeTid=%d", tc.primeTid)
		}
	}
	return name
}

func testPidStatusParser(tc *PidStatusTestCase, t *testing.T) {
	var pidStatus, usePathFrom *PidStatus
	if tc.primePid > 0 {
		primeProcfsRoot := tc.primeProcfsRoot
		if primeProcfsRoot == "" {
			primeProcfsRoot = tc.procfsRoot
		}
		pidStatus = NewPidStatus(primeProcfsRoot, tc.primePid, tc.primeTid)
		err := pidStatus.Parse(nil)
		if err != nil {
			t.Fatal(err)
		}
		usePathFrom = NewPidStatus(tc.procfsRoot, tc.pid, tc.tid)
	} else {
		pidStatus = NewPidStatus(tc.procfsRoot, tc.pid, tc.tid)
	}
	err := pidStatus.Parse(usePathFrom)

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

	for index := range tc.wantByteSliceFieldValues {
		wantVal := tc.wantByteSliceFieldValues[index]
		gotVal := string(pidStatus.ByteSliceFields[index])
		if wantVal != gotVal {
			fmt.Fprintf(
				diffBuf,
				"\nBytesSliceFields[%s]: want: %q, got: %q",
				pidStatusByteSliceFieldsIndexToName[index],
				wantVal,
				gotVal,
			)
		}
	}

	for index := range tc.wantByteSliceFieldUnit {
		wantUnit := tc.wantByteSliceFieldUnit[index]
		gotUnit := string(pidStatus.ByteSliceFieldUnit[index])
		if wantUnit != gotUnit {
			fmt.Fprintf(
				diffBuf,
				"\nByteSliceFieldUnit[%s]: want: %q, got: %q",
				pidStatusByteSliceFieldsIndexToName[index],
				wantUnit,
				gotUnit,
			)
		}
	}

	for index := range tc.wantNumericFields {
		wantVal := tc.wantNumericFields[index]
		gotVal := pidStatus.NumericFields[index]
		if wantVal != gotVal {
			fmt.Fprintf(
				diffBuf,
				"\nuNumericFields[%s]: want: %d, got: %d",
				pidStatusNumericFieldsIndexToName[index],
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
			name:       "field_mapping",
			procfsRoot: pidStatusTestdataDir,
			pid:        1000,
			tid:        PID_STAT_PID_ONLY_TID,
			wantByteSliceFieldValues: map[int]string{
				PID_STATUS_UID:               "900,901,902,903",
				PID_STATUS_GID:               "1000,1001,1002,1003",
				PID_STATUS_GROUPS:            "1200,1201,1202",
				PID_STATUS_VM_PEAK:           "1700",
				PID_STATUS_VM_SIZE:           "1800",
				PID_STATUS_VM_LCK:            "1900",
				PID_STATUS_VM_PIN:            "2000",
				PID_STATUS_VM_HWM:            "2100",
				PID_STATUS_VM_RSS:            "2200",
				PID_STATUS_RSS_ANON:          "2300",
				PID_STATUS_RSS_FILE:          "2400",
				PID_STATUS_RSS_SHMEM:         "2500",
				PID_STATUS_VM_DATA:           "2600",
				PID_STATUS_VM_STK:            "2700",
				PID_STATUS_VM_EXE:            "2800",
				PID_STATUS_VM_LIB:            "2900",
				PID_STATUS_VM_PTE:            "3000",
				PID_STATUS_VM_PMD:            "",
				PID_STATUS_VM_SWAP:           "3100",
				PID_STATUS_HUGETLBPAGES:      "3200",
				PID_STATUS_CPUS_ALLOWED_LIST: "5300,5301,5302,5303",
				PID_STATUS_MEMS_ALLOWED_LIST: "5500,5501",
			},
			wantByteSliceFieldUnit: map[int]string{
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
			wantNumericFields: map[int]uint64{
				PID_STATUS_VOLUNTARY_CTXT_SWITCHES:    5600,
				PID_STATUS_NONVOLUNTARY_CTXT_SWITCHES: 5700,
			},
		},
		{
			name:       "real_life",
			procfsRoot: pidStatusTestdataDir,
			pid:        468,
			tid:        486,
			primePid:   1000,
			primeTid:   PID_STAT_PID_ONLY_TID,
			wantByteSliceFieldValues: map[int]string{
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
			wantByteSliceFieldUnit: map[int]string{
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
			wantNumericFields: map[int]uint64{
				PID_STATUS_VOLUNTARY_CTXT_SWITCHES:    2588,
				PID_STATUS_NONVOLUNTARY_CTXT_SWITCHES: 12,
			},
		},
	} {
		t.Run(
			pidStatusSubtestName(tc),
			func(t *testing.T) { testPidStatusParser(tc, t) },
		)
	}
}
