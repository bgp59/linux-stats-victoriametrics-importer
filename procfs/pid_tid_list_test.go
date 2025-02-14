package procfs

import (
	"fmt"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/bgp59/linux-stats-victoriametrics-importer/internal/testutils"
)

type PidTidListTestCase struct {
	ProcfsRoot  string
	Flags       uint32
	NumPart     int
	PidTidLists [][]PidTid
}

var pidTidListTestCaseFile = path.Join(
	"..", testutils.ProcfsTestCasesSubdir,
	"pid_tid_list_test_case.json",
)

func testPidTidListCacheOnePart(
	pidTidListCache PidTidListCacheIF,
	partNo int,
	pidTidList []PidTid,
	wg *sync.WaitGroup,
	t *testing.T,
) {
	defer wg.Done()

	want := make(map[PidTid]bool)
	for _, pidTid := range pidTidList {
		want[pidTid] = true
	}
	pidTidList, err := pidTidListCache.GetPidTidList(partNo, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if pidTidList == nil {
		t.Errorf("no list for part# %d", partNo)
	}

	for _, pidTid := range pidTidList {
		_, exists := want[pidTid]
		if exists {
			delete(want, pidTid)
		} else {
			t.Errorf("unexpected pidTid %v for part# %d", pidTid, partNo)
		}
	}
	for pidTid := range want {
		t.Errorf("missing pidTid %v for part# %d", pidTid, partNo)
	}
}

func testPidTidListCache(tc *PidTidListTestCase, t *testing.T) {
	// Use an absurdly large validFor to ensure that refresh will occur only as
	// instructed:
	validFor := time.Hour
	pidTidListCache := NewPidTidListCache(tc.ProcfsRoot, tc.NumPart, validFor, tc.Flags)
	wg := &sync.WaitGroup{}

	// Run twice, to test reusability:
	for k, forceRefresh := range []bool{false, true} {
		if forceRefresh {
			pidTidListCache.Invalidate()
		}
		for partNo, pidTidList := range tc.PidTidLists {
			wg.Add(1)
			go testPidTidListCacheOnePart(pidTidListCache, partNo, pidTidList, wg, t)
		}
		wg.Wait()
		wantRefreshCount, gotRefreshCount := uint64(k+1), pidTidListCache.GetRefreshCount()
		if wantRefreshCount != gotRefreshCount {
			t.Errorf("refreshCount: want: %d, got: %d", wantRefreshCount, gotRefreshCount)
		}
	}
}

func TestPidTidListCache(t *testing.T) {
	testCases := make([]*PidTidListTestCase, 0)
	err := testutils.LoadJsonFile(pidTidListTestCaseFile, &testCases)

	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range testCases {
		t.Run(
			fmt.Sprintf(
				"numPart=%d,pidEnabled=%v,tidEnabled=%v",
				tc.NumPart,
				tc.Flags&PID_LIST_CACHE_PID_ENABLED != 0,
				tc.Flags&PID_LIST_CACHE_TID_ENABLED != 0,
			),
			func(t *testing.T) { testPidTidListCache(tc, t) },
		)
	}
}
