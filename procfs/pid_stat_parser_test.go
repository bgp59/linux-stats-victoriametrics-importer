package procfs

import (
	"bytes"
	"fmt"

	"testing"
)

var pidStatByteSliceFieldName = map[int]string{
	PID_STAT_COMM:        "PID_STAT_COMM",
	PID_STAT_STATE:       "PID_STAT_STATE",
	PID_STAT_PPID:        "PID_STAT_PPID",
	PID_STAT_PGRP:        "PID_STAT_PGRP",
	PID_STAT_SESSION:     "PID_STAT_SESSION",
	PID_STAT_TTY_NR:      "PID_STAT_TTY_NR",
	PID_STAT_TPGID:       "PID_STAT_TPGID",
	PID_STAT_FLAGS:       "PID_STAT_FLAGS",
	PID_STAT_PRIORITY:    "PID_STAT_PRIORITY",
	PID_STAT_NICE:        "PID_STAT_NICE",
	PID_STAT_NUM_THREADS: "PID_STAT_NUM_THREADS",
	PID_STAT_STARTTIME:   "PID_STAT_STARTTIME",
	PID_STAT_VSIZE:       "PID_STAT_VSIZE",
	PID_STAT_RSS:         "PID_STAT_RSS",
	PID_STAT_RSSLIM:      "PID_STAT_RSSLIM",
	PID_STAT_PROCESSOR:   "PID_STAT_PROCESSOR",
	PID_STAT_RT_PRIORITY: "PID_STAT_RT_PRIORITY",
	PID_STAT_POLICY:      "PID_STAT_POLICY",
}

var pidStatNumericFieldName = map[int]string{
	PID_STAT_MINFLT: "PID_STAT_MINFLT",
	PID_STAT_MAJLT:  "PID_STAT_MAJLT",
	PID_STAT_UTIME:  "PID_STAT_UTIME",
	PID_STAT_STIME:  "PID_STAT_STIME",
}

type PidStatTestCase struct {
	name                string
	pidStatData         string
	wantByteSliceFields map[int]string
	wantNumericFields   map[int]uint64
	wantError           error
	pid, tid            int
}

var pidStatTestPid, pidStatTestTid int = 468, 486

var pidStatTestWantByteSliceFields map[int]string = map[int]string{
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
}

var pidStatTestWantNumericFields map[int]uint64 = map[int]uint64{
	PID_STAT_MINFLT: 44,
	PID_STAT_MAJLT:  0,
	PID_STAT_UTIME:  0,
	PID_STAT_STIME:  2,
}

func testPidStatParser(tc *PidStatTestCase, t *testing.T) {
	pidStat := &PidStatByteFields{
		Buf: bytes.NewBufferString(tc.pidStatData),
	}
	if tc.pid != 0 {
		pidStat.SetPath(TestDataProcDir, tc.pid, tc.tid)
	}
	err := pidStat.Parse()
	if tc.wantError == nil && err != nil {
		t.Fatal(err)
	}
	if tc.wantError != nil && (err == nil || tc.wantError.Error() != err.Error()) {
		t.Fatalf("error: want: %v, got: %v", tc.wantError, err)
	}
	b := pidStat.Buf.Bytes()
	diffBuf := &bytes.Buffer{}
	if tc.wantByteSliceFields != nil {
		for i, wantValue := range tc.wantByteSliceFields {
			gotValue := string(b[pidStat.FieldStart[i]:pidStat.FieldEnd[i]])
			if wantValue != gotValue {
				fmt.Fprintf(diffBuf, "\nfield[%s]: want: %q, got: %q", pidStatByteSliceFieldName[i], wantValue, gotValue)
			}
		}
	}
	if tc.wantNumericFields != nil {
		for i, wantValue := range tc.wantNumericFields {
			gotValue := pidStat.NumericFields[i]
			if wantValue != gotValue {
				fmt.Fprintf(diffBuf, "\nfield[%s]: want: %d, got: %d", pidStatNumericFieldName[i], wantValue, gotValue)
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
			name:        "field_mapping",
			pidStatData: "pid (comm) state ppid pgrp session tty_nr tpgid flags 1000 cminflt 1001 cmajflt 10000 10001 cutime cstime priority nice num_threads itrealvalue starttime vsize rss rsslim startcode endcode startstack kstkesp kstkeip signal blocked sigignore sigcatch wchan nswap cnswap exit_signal processor rt_priority policy delayacct_blkio_ticks guest_time cguest_time start_data end_data start_brk arg_start arg_end env_start env_end exit_code",
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
				PID_STAT_MAJLT:  1001,
				PID_STAT_UTIME:  10000,
				PID_STAT_STIME:  10001,
			},
		},
		{
			name:                "real_data",
			wantByteSliceFields: pidStatTestWantByteSliceFields,
			wantNumericFields:   pidStatTestWantNumericFields,
			pid:                 pidStatTestPid,
			tid:                 pidStatTestTid,
		},
		{
			name:        "comm_too_long",
			pidStatData: "pid (command longer than sixteen bytes) state ppid pgrp session tty_nr tpgid flags 1000 cminflt 1001 cmajflt 10000 10001 cutime cstime priority nice num_threads itrealvalue starttime vsize rss rsslim startcode endcode startstack kstkesp kstkeip signal blocked sigignore sigcatch wchan nswap cnswap exit_signal processor rt_priority policy delayacct_blkio_ticks guest_time cguest_time start_data end_data start_brk arg_start arg_end env_start env_end exit_code",
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
				PID_STAT_MAJLT:  1001,
				PID_STAT_UTIME:  10000,
				PID_STAT_STIME:  10001,
			},
		},
		{
			name:        "comm_utf8",
			pidStatData: "pid (Nǐ hǎo shìjiè 你好世界) state ppid pgrp session tty_nr tpgid flags 1000 cminflt 1001 cmajflt 10000 10001 cutime cstime priority nice num_threads itrealvalue starttime vsize rss rsslim startcode endcode startstack kstkesp kstkeip signal blocked sigignore sigcatch wchan nswap cnswap exit_signal processor rt_priority policy delayacct_blkio_ticks guest_time cguest_time start_data end_data start_brk arg_start arg_end env_start env_end exit_code",
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
				PID_STAT_MAJLT:  1001,
				PID_STAT_UTIME:  10000,
				PID_STAT_STIME:  10001,
			},
		},
		{
			name:        "comm_missin_open_par",
			pidStatData: "pid comm) state ppid pgrp session tty_nr tpgid flags 1000 cminflt 1001 cmajflt 10000 10001 cutime cstime priority nice num_threads itrealvalue starttime vsize rss rsslim startcode endcode startstack kstkesp kstkeip signal blocked sigignore sigcatch wchan nswap cnswap exit_signal processor rt_priority policy delayacct_blkio_ticks guest_time cguest_time start_data end_data start_brk arg_start arg_end env_start env_end exit_code",
			wantError:   fmt.Errorf(": cannot locate '('"),
		},
		{
			name:        "comm_missin_close_par",
			pidStatData: "pid (comm state ppid pgrp session tty_nr tpgid flags 1000 cminflt 1001 cmajflt 10000 10001 cutime cstime priority nice num_threads itrealvalue starttime vsize rss rsslim startcode endcode startstack kstkesp kstkeip signal blocked sigignore sigcatch wchan nswap cnswap exit_signal processor rt_priority policy delayacct_blkio_ticks guest_time cguest_time start_data end_data start_brk arg_start arg_end env_start env_end exit_code",
			wantError:   fmt.Errorf(": cannot locate ')'"),
		},
		{
			name:        "missing_numeric_field",
			pidStatData: "pid (comm) state ppid pgrp session tty_nr tpgid flags 1000 cminflt 1001 cmajflt 10000\n", // "10001 cutime cstime priority nice num_threads itrealvalue starttime vsize rss rsslim startcode endcode startstack kstkesp kstkeip signal blocked sigignore sigcatch wchan nswap cnswap exit_signal processor rt_priority\n", // "policy delayacct_blkio_ticks guest_time cguest_time start_data end_data start_brk arg_start arg_end env_start env_end exit_code",
			wantError: fmt.Errorf(
				": scan incomplete: byte slice got/want field#: %d/%d, numeric got/want field#: %d/%d",
				8, PID_STAT_BYTE_SLICE_FIELD_COUNT,
				PID_STAT_NUMERIC_FIELD_COUNT-1, PID_STAT_NUMERIC_FIELD_COUNT,
			),
		},
		{
			name:        "conversion_error",
			pidStatData: "pid (comm) state ppid pgrp session tty_nr tpgid flags _1000 cminflt 1001 cmajflt 10000 10001 cutime cstime priority nice num_threads itrealvalue starttime vsize rss rsslim startcode endcode startstack kstkesp kstkeip signal blocked sigignore sigcatch wchan nswap cnswap exit_signal processor rt_priority policy delayacct_blkio_ticks guest_time cguest_time start_data end_data start_brk arg_start arg_end env_start env_end exit_code",
			wantError:   fmt.Errorf(`: field# 10: strconv.ParseUint: parsing "_1000": invalid syntax`),
		},
	} {
		t.Run(tc.name, func(t *testing.T) { testPidStatParser(tc, t) })
	}
}
