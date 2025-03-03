package procfs

import (
	"bytes"
	"fmt"
	"path"

	"testing"

	"github.com/bgp59/linux-stats-victoriametrics-importer/internal/testutils"
)

var pidStatusTestDataDir = path.Join(PROCFS_TESTDATA_ROOT, "pid_status")

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

func testPidStatusParser(tc *PidStatusTestCase, t *testing.T) {
	t.Logf(`
name=%q
procfsRoot=%q, pid=%d, tid=%d
primeProcfsRoot=%q, primePid=%d, PrimeTid=%d
`,
		tc.name,
		tc.procfsRoot, tc.pid, tc.tid,
		tc.primeProcfsRoot, tc.primePid, tc.primeTid,
	)

	pidStatus := NewPidStatus()
	if tc.primePid > 0 {
		primeProcfsRoot := tc.primeProcfsRoot
		if primeProcfsRoot == "" {
			primeProcfsRoot = tc.procfsRoot
		}
		err := pidStatus.Parse(BuildPidTidPath(primeProcfsRoot, tc.primePid, tc.primeTid))
		if err != nil {
			t.Fatal(err)
		}
	}

	pidTidPath := BuildPidTidPath(tc.procfsRoot, tc.pid, tc.tid)
	err := pidStatus.Parse(pidTidPath)

	if tc.wantError != nil {
		if err == nil || tc.wantError.Error() != err.Error() {
			t.Fatalf("want: %v error, got: %v", tc.wantError, err)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	gotByteSliceFields, gotByteSliceFieldUnit, gotNumericFields := pidStatus.GetData()

	diffBuf := &bytes.Buffer{}

	for index := range tc.wantByteSliceFieldValues {
		wantVal := tc.wantByteSliceFieldValues[index]
		gotVal := string(gotByteSliceFields[index])
		if wantVal != gotVal {
			fmt.Fprintf(
				diffBuf,
				"\nBytesSliceFields[%s]: want: %q, got: %q",
				testutils.PidStatByteSliceFieldsIndexName[index],
				wantVal,
				gotVal,
			)
			continue
		}
		if gotByteSliceFields[index] != nil {
			// Check unit as well:
			wantUnit := tc.wantByteSliceFieldUnit[index]
			gotUnit := string(gotByteSliceFieldUnit[index])
			if wantUnit != gotUnit {
				fmt.Fprintf(
					diffBuf,
					"\nByteSliceFieldUnit[%s]: want: %q, got: %q",
					testutils.PidStatByteSliceFieldsIndexName[index],
					wantUnit,
					gotUnit,
				)
			}
		}
	}

	for index := range tc.wantNumericFields {
		wantVal := tc.wantNumericFields[index]
		gotVal := gotNumericFields[index]
		if wantVal != gotVal {
			fmt.Fprintf(
				diffBuf,
				"\nuNumericFields[%s]: want: %d, got: %d",
				testutils.PidStatNumericFieldsIndexName[index],
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
			procfsRoot: pidStatusTestDataDir,
			pid:        1000,
			tid:        PID_ONLY_TID,
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
			name:       "empty_fields",
			procfsRoot: pidStatusTestDataDir,
			pid:        1001,
			tid:        PID_ONLY_TID,
			wantByteSliceFieldValues: map[int]string{
				PID_STATUS_UID:               "900,901,902,903",
				PID_STATUS_GID:               "1000,1001,1002,1003",
				PID_STATUS_GROUPS:            "",
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
			procfsRoot: pidStatusTestDataDir,
			pid:        468,
			tid:        486,
			primePid:   1000,
			primeTid:   PID_ONLY_TID,
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
		{
			name:       "real_life_missing_fields",
			procfsRoot: pidStatusTestDataDir,
			pid:        98,
			tid:        PID_ONLY_TID,
			primePid:   1000,
			primeTid:   PID_ONLY_TID,
			wantByteSliceFieldValues: map[int]string{
				PID_STATUS_UID:               "0,0,0,0",
				PID_STATUS_GID:               "0,0,0,0",
				PID_STATUS_GROUPS:            "",
				PID_STATUS_VM_PEAK:           "",
				PID_STATUS_VM_SIZE:           "",
				PID_STATUS_VM_LCK:            "",
				PID_STATUS_VM_PIN:            "",
				PID_STATUS_VM_HWM:            "",
				PID_STATUS_VM_RSS:            "",
				PID_STATUS_RSS_ANON:          "",
				PID_STATUS_RSS_FILE:          "",
				PID_STATUS_RSS_SHMEM:         "",
				PID_STATUS_VM_DATA:           "",
				PID_STATUS_VM_STK:            "",
				PID_STATUS_VM_EXE:            "",
				PID_STATUS_VM_LIB:            "",
				PID_STATUS_VM_PTE:            "",
				PID_STATUS_VM_PMD:            "",
				PID_STATUS_VM_SWAP:           "",
				PID_STATUS_HUGETLBPAGES:      "",
				PID_STATUS_CPUS_ALLOWED_LIST: "0-14",
				PID_STATUS_MEMS_ALLOWED_LIST: "0",
			},
			wantByteSliceFieldUnit: map[int]string{
				PID_STATUS_UID:               "",
				PID_STATUS_GID:               "",
				PID_STATUS_GROUPS:            "",
				PID_STATUS_VM_PEAK:           "ignore",
				PID_STATUS_VM_SIZE:           "ignore",
				PID_STATUS_VM_LCK:            "ignore",
				PID_STATUS_VM_PIN:            "ignore",
				PID_STATUS_VM_HWM:            "ignore",
				PID_STATUS_VM_RSS:            "ignore",
				PID_STATUS_RSS_ANON:          "ignore",
				PID_STATUS_RSS_FILE:          "ignore",
				PID_STATUS_RSS_SHMEM:         "ignore",
				PID_STATUS_VM_DATA:           "ignore",
				PID_STATUS_VM_STK:            "ignore",
				PID_STATUS_VM_EXE:            "ignore",
				PID_STATUS_VM_LIB:            "ignore",
				PID_STATUS_VM_PTE:            "ignore",
				PID_STATUS_VM_PMD:            "ignore",
				PID_STATUS_VM_SWAP:           "ignore",
				PID_STATUS_HUGETLBPAGES:      "ignore",
				PID_STATUS_CPUS_ALLOWED_LIST: "",
				PID_STATUS_MEMS_ALLOWED_LIST: "",
			},
			wantNumericFields: map[int]uint64{
				PID_STATUS_VOLUNTARY_CTXT_SWITCHES:    4,
				PID_STATUS_NONVOLUNTARY_CTXT_SWITCHES: 0,
			},
		},
	} {
		t.Run(
			tc.name,
			func(t *testing.T) { testPidStatusParser(tc, t) },
		)
	}
}
