package procfs

import (
	"fmt"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/emypar/linux-stats-victoriametrics-importer/internal/testutils"
)

type PidListTestCase struct {
	ProcfsRoot  string
	Flags       uint32
	NPart       int
	PidTidLists [][]PidTid
}

var pidListTestCaseFile = path.Join(
	"..", testutils.ProcfsTestCasesSubdir,
	"pid_list_test_case.json",
)

func testPidListCacheOnePart(
	pidListCache *PidListCache,
	nPart int,
	pidTidList []PidTid,
	wg *sync.WaitGroup,
	t *testing.T,
) {
	defer wg.Done()

	want := make(map[PidTid]bool)
	for _, pidTid := range pidTidList {
		want[pidTid] = true
	}
	pidTidList, err := pidListCache.GetPidTidList(nPart, nil)
	if err != nil {
		t.Error(err)
		return
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

func testPidListCache(tc *PidListTestCase, t *testing.T) {
	// Use an absurdly large validFor to ensure that refresh will occur only as
	// instructed:
	validFor := time.Hour
	pidListCache := NewPidListCache(tc.ProcfsRoot, tc.NPart, validFor, tc.Flags)
	wg := &sync.WaitGroup{}

	// Run twice, to test reusability:
	for k, forceRefresh := range []bool{false, true} {
		if forceRefresh {
			pidListCache.Invalidate()
		}
		for nPart, pidTidList := range tc.PidTidLists {
			wg.Add(1)
			go testPidListCacheOnePart(pidListCache, nPart, pidTidList, wg, t)
		}
		wg.Wait()
		wantRefreshCount, gotRefreshCount := uint64(k+1), pidListCache.GetRefreshCount()
		if wantRefreshCount != gotRefreshCount {
			t.Errorf("refreshCount: want: %d, got: %d", wantRefreshCount, gotRefreshCount)
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
				tc.Flags&PID_LIST_CACHE_PID_ENABLED != 0,
				tc.Flags&PID_LIST_CACHE_TID_ENABLED != 0,
			),
			func(t *testing.T) { testPidListCache(tc, t) },
		)
	}
}
