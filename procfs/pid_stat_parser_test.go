package procfs

import (
	"bytes"
	"fmt"
	"path"

	"testing"
)

var pidStatTestdataDir = path.Join(PROCFS_TESTDATA_ROOT, "pid_stat")

var pidStatByteSliceFieldsIndexName = []string{
	"PID_STAT_COMM",
	"PID_STAT_STATE",
	"PID_STAT_PPID",
	"PID_STAT_PGRP",
	"PID_STAT_SESSION",
	"PID_STAT_TTY_NR",
	"PID_STAT_TPGID",
	"PID_STAT_FLAGS",
	"PID_STAT_PRIORITY",
	"PID_STAT_NICE",
	"PID_STAT_NUM_THREADS",
	"PID_STAT_STARTTIME",
	"PID_STAT_VSIZE",
	"PID_STAT_RSS",
	"PID_STAT_RSSLIM",
	"PID_STAT_PROCESSOR",
	"PID_STAT_RT_PRIORITY",
	"PID_STAT_POLICY",
}

var pidStatNumericFieldsIndexName = []string{
	"PID_STAT_MINFLT",
	"PID_STAT_MAJFLT",
	"PID_STAT_UTIME",
	"PID_STAT_STIME",
}

type PidStatTestCase struct {
	name                string
	procfsRoot          string
	pid, tid            int
	primeProcfsRoot     string
	primePid, primeTid  int
	wantByteSliceFields map[int]string
	wantNumericFields   map[int]uint64
	wantError           error
}

func testPidStatParser(tc *PidStatTestCase, t *testing.T) {
	t.Logf(`
name=%q
procfsRoot=%q, pid=%d, tid=%d
primeProcfsRoot=%q, primePid=%d, PrimeTid=%d
`,
		tc.name,
		tc.procfsRoot, tc.pid, tc.tid,
		tc.primeProcfsRoot, tc.primePid, tc.primeTid,
	)

	var pidStat, usePathFrom *PidStat
	if tc.primePid > 0 {
		primeProcfsRoot := tc.primeProcfsRoot
		if primeProcfsRoot == "" {
			primeProcfsRoot = tc.procfsRoot
		}
		pidStat = NewPidStat(primeProcfsRoot, tc.primePid, tc.primeTid)
		err := pidStat.Parse(nil)
		if err != nil {
			t.Fatal(err)
		}
		usePathFrom = NewPidStat(tc.procfsRoot, tc.pid, tc.tid)
	} else {
		pidStat = NewPidStat(tc.procfsRoot, tc.pid, tc.tid)
	}
	err := pidStat.Parse(usePathFrom)
	if tc.wantError == nil && err != nil {
		t.Fatal(err)
	}
	if tc.wantError != nil {
		wantError := fmt.Errorf("%s: %v", *pidStat.path, tc.wantError)
		if err == nil || wantError.Error() != err.Error() {
			t.Fatalf("error: want: %v, got: %v", wantError, err)
		}
	}
	diffBuf := &bytes.Buffer{}
	if tc.wantByteSliceFields != nil {
		for i, wantValue := range tc.wantByteSliceFields {
			gotValue := string(pidStat.ByteSliceFields[i])
			if wantValue != gotValue {
				fmt.Fprintf(
					diffBuf,
					"\nfield[%s]: want: %q, got: %q",
					pidStatByteSliceFieldsIndexName[i],
					wantValue,
					gotValue,
				)
			}
		}
	}
	if tc.wantNumericFields != nil {
		for i, wantValue := range tc.wantNumericFields {
			gotValue := pidStat.NumericFields[i]
			if wantValue != gotValue {
				fmt.Fprintf(diffBuf, "\nfield[%s]: want: %d, got: %d", pidStatNumericFieldsIndexName[i], wantValue, gotValue)
			}
		}
	}
	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf.String())
	}
}

func TestPidStatParser(t *testing.T) {
	for _, tc := range []*PidStatTestCase{
		{
			name:       "field_mapping",
			procfsRoot: pidStatTestdataDir,
			pid:        1000,
			tid:        PID_STAT_PID_ONLY_TID,
			wantByteSliceFields: map[int]string{
				PID_STAT_COMM:        "comm",
				PID_STAT_STATE:       "state",
				PID_STAT_PPID:        "ppid",
				PID_STAT_PGRP:        "pgrp",
				PID_STAT_SESSION:     "session",
				PID_STAT_TTY_NR:      "tty_nr",
				PID_STAT_TPGID:       "tpgid",
				PID_STAT_FLAGS:       "flags",
				PID_STAT_PRIORITY:    "priority",
				PID_STAT_NICE:        "nice",
				PID_STAT_NUM_THREADS: "num_threads",
				PID_STAT_STARTTIME:   "starttime",
				PID_STAT_VSIZE:       "vsize",
				PID_STAT_RSS:         "rss",
				PID_STAT_RSSLIM:      "rsslim",
				PID_STAT_PROCESSOR:   "processor",
				PID_STAT_RT_PRIORITY: "rt_priority",
				PID_STAT_POLICY:      "policy",
			},
			wantNumericFields: map[int]uint64{
				PID_STAT_MINFLT: 1000,
				PID_STAT_MAJFLT: 1001,
				PID_STAT_UTIME:  10000,
				PID_STAT_STIME:  10001,
			},
		},
		{
			name:       "reuse",
			procfsRoot: pidStatTestdataDir,
			pid:        1000,
			tid:        PID_STAT_PID_ONLY_TID,
			wantByteSliceFields: map[int]string{
				PID_STAT_COMM:        "comm",
				PID_STAT_STATE:       "state",
				PID_STAT_PPID:        "ppid",
				PID_STAT_PGRP:        "pgrp",
				PID_STAT_SESSION:     "session",
				PID_STAT_TTY_NR:      "tty_nr",
				PID_STAT_TPGID:       "tpgid",
				PID_STAT_FLAGS:       "flags",
				PID_STAT_PRIORITY:    "priority",
				PID_STAT_NICE:        "nice",
				PID_STAT_NUM_THREADS: "num_threads",
				PID_STAT_STARTTIME:   "starttime",
				PID_STAT_VSIZE:       "vsize",
				PID_STAT_RSS:         "rss",
				PID_STAT_RSSLIM:      "rsslim",
				PID_STAT_PROCESSOR:   "processor",
				PID_STAT_RT_PRIORITY: "rt_priority",
				PID_STAT_POLICY:      "policy",
			},
			wantNumericFields: map[int]uint64{
				PID_STAT_MINFLT: 1000,
				PID_STAT_MAJFLT: 1001,
				PID_STAT_UTIME:  10000,
				PID_STAT_STIME:  10001,
			},
		},
		{
			name:       "real_life",
			procfsRoot: pidStatTestdataDir,
			pid:        468,
			tid:        486,
			primePid:   1000,
			primeTid:   PID_STAT_PID_ONLY_TID,
			wantByteSliceFields: map[int]string{
				PID_STAT_COMM:        "rs:main Q:Reg",
				PID_STAT_STATE:       "S",
				PID_STAT_PPID:        "1",
				PID_STAT_PGRP:        "468",
				PID_STAT_SESSION:     "468",
				PID_STAT_TTY_NR:      "0",
				PID_STAT_TPGID:       "-1",
				PID_STAT_FLAGS:       "1077936192",
				PID_STAT_PRIORITY:    "20",
				PID_STAT_NICE:        "0",
				PID_STAT_NUM_THREADS: "4",
				PID_STAT_STARTTIME:   "898",
				PID_STAT_VSIZE:       "227737600",
				PID_STAT_RSS:         "1340",
				PID_STAT_RSSLIM:      "18446744073709551615",
				PID_STAT_PROCESSOR:   "0",
				PID_STAT_RT_PRIORITY: "0",
				PID_STAT_POLICY:      "0",
			},
			wantNumericFields: map[int]uint64{
				PID_STAT_MINFLT: 44,
				PID_STAT_MAJFLT: 0,
				PID_STAT_UTIME:  0,
				PID_STAT_STIME:  2,
			},
		},
		{
			name:       "comm_too_long",
			procfsRoot: pidStatTestdataDir,
			pid:        1001,
			tid:        PID_STAT_PID_ONLY_TID,
			wantByteSliceFields: map[int]string{
				PID_STAT_COMM:        "command longer than sixteen bytes",
				PID_STAT_STATE:       "state",
				PID_STAT_PPID:        "ppid",
				PID_STAT_PGRP:        "pgrp",
				PID_STAT_SESSION:     "session",
				PID_STAT_TTY_NR:      "tty_nr",
				PID_STAT_TPGID:       "tpgid",
				PID_STAT_FLAGS:       "flags",
				PID_STAT_PRIORITY:    "priority",
				PID_STAT_NICE:        "nice",
				PID_STAT_NUM_THREADS: "num_threads",
				PID_STAT_STARTTIME:   "starttime",
				PID_STAT_VSIZE:       "vsize",
				PID_STAT_RSS:         "rss",
				PID_STAT_RSSLIM:      "rsslim",
				PID_STAT_PROCESSOR:   "processor",
				PID_STAT_RT_PRIORITY: "rt_priority",
				PID_STAT_POLICY:      "policy",
			},
			wantNumericFields: map[int]uint64{
				PID_STAT_MINFLT: 1000,
				PID_STAT_MAJFLT: 1001,
				PID_STAT_UTIME:  10000,
				PID_STAT_STIME:  10001,
			},
		},
		{
			name:       "comm_utf8",
			procfsRoot: pidStatTestdataDir,
			pid:        1002,
			tid:        PID_STAT_PID_ONLY_TID,
			wantByteSliceFields: map[int]string{
				PID_STAT_COMM:        "Nǐ hǎo shìjiè 你好世界",
				PID_STAT_STATE:       "state",
				PID_STAT_PPID:        "ppid",
				PID_STAT_PGRP:        "pgrp",
				PID_STAT_SESSION:     "session",
				PID_STAT_TTY_NR:      "tty_nr",
				PID_STAT_TPGID:       "tpgid",
				PID_STAT_FLAGS:       "flags",
				PID_STAT_PRIORITY:    "priority",
				PID_STAT_NICE:        "nice",
				PID_STAT_NUM_THREADS: "num_threads",
				PID_STAT_STARTTIME:   "starttime",
				PID_STAT_VSIZE:       "vsize",
				PID_STAT_RSS:         "rss",
				PID_STAT_RSSLIM:      "rsslim",
				PID_STAT_PROCESSOR:   "processor",
				PID_STAT_RT_PRIORITY: "rt_priority",
				PID_STAT_POLICY:      "policy",
			},
			wantNumericFields: map[int]uint64{
				PID_STAT_MINFLT: 1000,
				PID_STAT_MAJFLT: 1001,
				PID_STAT_UTIME:  10000,
				PID_STAT_STIME:  10001,
			},
		},
		{
			name:       "comm_missing_open_par",
			procfsRoot: pidStatTestdataDir,
			pid:        10000,
			tid:        PID_STAT_PID_ONLY_TID,
			wantError:  fmt.Errorf("cannot locate '('"),
		},
		{
			name:       "comm_missing_close_par",
			procfsRoot: pidStatTestdataDir,
			pid:        10001,
			tid:        PID_STAT_PID_ONLY_TID,
			wantError:  fmt.Errorf("cannot locate ')'"),
		},
		{
			name:       "conversion_error",
			procfsRoot: pidStatTestdataDir,
			pid:        10002,
			tid:        PID_STAT_PID_ONLY_TID,
			wantError:  fmt.Errorf(`field# 10: "_1000": invalid numerical value`),
		},
		{
			name:       "not_enough_fields",
			procfsRoot: pidStatTestdataDir,
			pid:        10003,
			tid:        PID_STAT_PID_ONLY_TID,
			wantError:  fmt.Errorf("not enough fields: want: %d, got: %d", PID_STAT_MAX_FIELD_NUM, PID_STAT_MAX_FIELD_NUM-1),
		},
	} {
		t.Run(
			tc.name,
			func(t *testing.T) { testPidStatParser(tc, t) },
		)
	}
}
