package procfs

import (
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/emypar/linux-stats-victoriametrics-importer/internal/testutils"
)

const PID_LIST_TEST_VALID_DURATION = time.Second

type PidListTestCase struct {
	ProcfsRoot  string
	Flags       uint32
	NPart       int
	PidTidLists [][]PidTid
}

var pidListTestCaseFile = path.Join(
	"..", testutils.ProcfsTestDataSubdir,
	"pid_list_test_case.json",
)

func testPidListCache(tc *PidListTestCase, t *testing.T) {
	pidListCache := NewPidListCache(tc.ProcfsRoot, tc.NPart, PID_LIST_TEST_VALID_DURATION, tc.Flags)
	for nPart, pidTidList := range tc.PidTidLists {
		want := make(map[PidTid]bool)
		for _, pidTid := range pidTidList {
			want[pidTid] = true
		}
		pidTidList, err := pidListCache.GetPidTidList(nPart, nil)
		if err != nil {
			t.Error(err)
		}
		if pidTidList == nil {
			t.Errorf("%s: no list for  part %d", t.Name(), nPart)
		}

		for _, pidTid := range pidTidList {
			_, exists := want[pidTid]
			if exists {
				delete(want, pidTid)
			} else {
				t.Errorf("%s: unexpected pidTid %v for part %d", t.Name(), pidTid, nPart)
			}
		}
		for pidTid, _ := range want {
			t.Errorf("%s: missing pidTid %v for part %d", t.Name(), pidTid, nPart)
		}
	}
}

func TestPidListCache(t *testing.T) {
	testCases := make([]*PidListTestCase, 0)
	err := testutils.LoadJsonFile(pidListTestCaseFile, &testCases)

	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range testCases {
		t.Run(
			fmt.Sprintf(
				"nPart=%d,pidEnabled=%v,tidEnabled=%v",
				tc.NPart,
				tc.Flags&PID_LIST_CACHE_PID_ENABLED > 0,
				tc.Flags&PID_LIST_CACHE_TID_ENABLED > 0,
			),
			func(t *testing.T) { testPidListCache(tc, t) },
		)
	}
}
