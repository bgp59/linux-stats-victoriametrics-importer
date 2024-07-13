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
	ProcfsRoot string
	flags      uint32
	PidTidList [][2]int
}

var pidListTestCaseFile = path.Join(
	"..", testutils.ProcfsTestDataSubdir,
	"pid_list_test_case.json",
)

func testPidListCache(
	t *testing.T,
	nPart int,
	tc *PidListTestCase,
) {

	pidListCache := NewPidListCache(nPart, PID_LIST_TEST_VALID_DURATION, tc.ProcfsRoot, tc.flags)
	for part := 0; part < nPart; part++ {
		want := make(map[PidTidPair]bool)
		for _, pidTidArray := range tc.PidTidList {
			pidTid := PidTidPair{pidTidArray[0], pidTidArray[1]}
			if (pidListCache.IsEnabledFor(PID_LIST_CACHE_PID_ENABLED) &&
				pidTid.Tid == 0 && (pidTid.Pid%nPart == part)) ||
				(pidListCache.IsEnabledFor(PID_LIST_CACHE_TID_ENABLED) &&
					pidTid.Tid > 0 && pidTid.Tid%nPart == part) {
				want[pidTid] = true
			}
		}

		pidList, err := pidListCache.GetPidTidList(part, nil)
		if err != nil {
			t.Error(err)
		}
		if pidList == nil {
			t.Errorf("%s: no list for  part %d", t.Name(), part)
		}
		for _, pidTid := range pidList {
			_, exists := want[pidTid]
			if exists {
				delete(want, pidTid)
			} else {
				t.Errorf("%s: unexpected pidTid %v for part %d", t.Name(), pidTid, part)
			}
		}
		for pidTid, _ := range want {
			t.Errorf("%s: missing pidTid %v for part %d", t.Name(), pidTid, part)
		}
	}
}

func TestPidListCache(t *testing.T) {
	tc := PidListTestCase{}
	err := testutils.LoadJsonFile(pidListTestCaseFile, &tc)

	if err != nil {
		t.Fatal(err)
	}
	for nPart := 1; nPart <= 16; nPart++ {
		flags_list := []uint32{
			0,
			PID_LIST_CACHE_PID_ENABLED,
			PID_LIST_CACHE_TID_ENABLED,
			PID_LIST_CACHE_ALL_ENABLED | PID_LIST_CACHE_TID_ENABLED,
		}
		for _, flags := range flags_list {
			tc.flags = flags
			pidEnabled := tc.flags&PID_LIST_CACHE_PID_ENABLED > 0
			tidEnabled := tc.flags&PID_LIST_CACHE_TID_ENABLED > 0
			t.Run(
				fmt.Sprintf("nPart=%d,pidEnabled=%v,tidEnabled=%v", nPart, pidEnabled, tidEnabled),
				func(t *testing.T) {
					testPidListCache(t, nPart, &tc)
				},
			)
		}
	}
}
