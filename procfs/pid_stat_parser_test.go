package procfs

import (
	"bytes"
	"fmt"

	"testing"
)

var pidStatFieldNameToIndexMap = map[string]int{
	"PID_STAT_COMM":        PID_STAT_COMM,
	"PID_STAT_STATE":       PID_STAT_STATE,
	"PID_STAT_PPID":        PID_STAT_PPID,
	"PID_STAT_PGRP":        PID_STAT_PGRP,
	"PID_STAT_SESSION":     PID_STAT_SESSION,
	"PID_STAT_TTY_NR":      PID_STAT_TTY_NR,
	"PID_STAT_TPGID":       PID_STAT_TPGID,
	"PID_STAT_FLAGS":       PID_STAT_FLAGS,
	"PID_STAT_MINFLT":      PID_STAT_MINFLT,
	"PID_STAT_MAJLT":       PID_STAT_MAJLT,
	"PID_STAT_UTIME":       PID_STAT_UTIME,
	"PID_STAT_STIME":       PID_STAT_STIME,
	"PID_STAT_PRIORITY":    PID_STAT_PRIORITY,
	"PID_STAT_NICE":        PID_STAT_NICE,
	"PID_STAT_NUM_THREADS": PID_STAT_NUM_THREADS,
	"PID_STAT_STARTTIME":   PID_STAT_STARTTIME,
	"PID_STAT_VSIZE":       PID_STAT_VSIZE,
	"PID_STAT_RSS":         PID_STAT_RSS,
	"PID_STAT_RSSLIM":      PID_STAT_RSSLIM,
	"PID_STAT_PROCESSOR":   PID_STAT_PROCESSOR,
	"PID_STAT_RT_PRIORITY": PID_STAT_RT_PRIORITY,
	"PID_STAT_POLICY":      PID_STAT_POLICY,
}

type PidStatTestCase struct {
	name        string
	pidStatData string
	wantFields  map[string]string
	wantError   error
}

func testPidStartParser(tc *PidStatTestCase, t *testing.T) {
	pidStat := &PidStatByteFields{
		Buf: bytes.NewBufferString(tc.pidStatData),
	}
	err := pidStat.Parse()
	if tc.wantError == nil && err != nil {
		t.Fatal(err)
	}
	if tc.wantError != nil && tc.wantError.Error() != err.Error() {
		t.Fatalf("error: want: %v, got: %v", tc.wantError, err)
	}
	b := pidStat.Buf.Bytes()
	diffBuf := &bytes.Buffer{}
	for name, wantValue := range tc.wantFields {
		i := pidStatFieldNameToIndexMap[name]
		gotValue := string(b[pidStat.FieldStart[i]:pidStat.FieldEnd[i]])
		if wantValue != gotValue {
			gotValue := string(b[pidStat.FieldStart[i]:pidStat.FieldEnd[i]])
			fmt.Fprintf(diffBuf, "\nfield[%s]: want: %q, got: %q", name, wantValue, gotValue)
		}
	}
	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf.String())
	}
}

func TestPidStartParser(t *testing.T) {
	for _, tc := range []*PidStatTestCase{
		{
			name:        "field_mapping",
			pidStatData: "pid (comm) state ppid pgrp session tty_nr tpgid flags minflt cminflt majflt cmajflt utime stime cutime cstime priority nice num_threads itrealvalue starttime vsize rss rsslim startcode endcode startstack kstkesp kstkeip signal blocked sigignore sigcatch wchan nswap cnswap exit_signal processor rt_priority policy delayacct_blkio_ticks guest_time cguest_time start_data end_data start_brk arg_start arg_end env_start env_end exit_code",
			wantFields: map[string]string{
				"PID_STAT_COMM":        "comm",
				"PID_STAT_STATE":       "state",
				"PID_STAT_PPID":        "ppid",
				"PID_STAT_PGRP":        "pgrp",
				"PID_STAT_SESSION":     "session",
				"PID_STAT_TTY_NR":      "tty_nr",
				"PID_STAT_TPGID":       "tpgid",
				"PID_STAT_FLAGS":       "flags",
				"PID_STAT_MINFLT":      "minflt",
				"PID_STAT_MAJLT":       "majflt",
				"PID_STAT_UTIME":       "utime",
				"PID_STAT_STIME":       "stime",
				"PID_STAT_PRIORITY":    "priority",
				"PID_STAT_NICE":        "nice",
				"PID_STAT_NUM_THREADS": "num_threads",
				"PID_STAT_STARTTIME":   "starttime",
				"PID_STAT_VSIZE":       "vsize",
				"PID_STAT_RSS":         "rss",
				"PID_STAT_RSSLIM":      "rsslim",
				"PID_STAT_PROCESSOR":   "processor",
				"PID_STAT_RT_PRIORITY": "rt_priority",
				"PID_STAT_POLICY":      "policy",
			},
		},
		{
			name: "real_data",
			//           "pid (comm)          state ppid pgrp session tty_nr tpgid      flags minflt cminflt majflt cmajflt utime stime cutime cstime priority nice num_threads itrealvalue starttime     vsize  rss              rsslim       startcode        endcode      startstack kstkesp kstkeip signal    blocked sigignore sigcatch wchan nswap cnswap exit_signal processor rt_priority policy delayacct_blkio_ticks guest_time cguest_time      start_data       end_data      start_brk       arg_start         arg_end       env_start        env_end exit_code"
			pidStatData: "486 (rs:main Q:Reg)     S    1  468     468      0    -1 1077936192     44       0      0       0     0     2      0      0       20    0           4           0       898 227737600 1340 18446744073709551615 94649719967744 94649720406605 140724805212720       0       0      0 2146171647  16781830  3227649     1     0      0          -1         0           0      0                     0          0           0  94649720624720 94649720664912 94649728393216 140724805218000 140724805218029 140724805218029 140724805218277        0\n",
			wantFields: map[string]string{
				"PID_STAT_COMM":        "rs:main Q:Reg",
				"PID_STAT_STATE":       "S",
				"PID_STAT_PPID":        "1",
				"PID_STAT_PGRP":        "468",
				"PID_STAT_SESSION":     "468",
				"PID_STAT_TTY_NR":      "0",
				"PID_STAT_TPGID":       "-1",
				"PID_STAT_FLAGS":       "1077936192",
				"PID_STAT_MINFLT":      "44",
				"PID_STAT_MAJLT":       "0",
				"PID_STAT_UTIME":       "0",
				"PID_STAT_STIME":       "2",
				"PID_STAT_PRIORITY":    "20",
				"PID_STAT_NICE":        "0",
				"PID_STAT_NUM_THREADS": "4",
				"PID_STAT_STARTTIME":   "898",
				"PID_STAT_VSIZE":       "227737600",
				"PID_STAT_RSS":         "1340",
				"PID_STAT_RSSLIM":      "18446744073709551615",
				"PID_STAT_PROCESSOR":   "0",
				"PID_STAT_RT_PRIORITY": "0",
				"PID_STAT_POLICY":      "0",
			},
		},
		{
			name:        "comm_too_long",
			pidStatData: "pid (abcdefghijklmnopqrstuvwxyz) state ppid pgrp session tty_nr tpgid flags minflt cminflt majflt cmajflt utime stime cutime cstime priority nice num_threads itrealvalue starttime vsize rss rsslim startcode endcode startstack kstkesp kstkeip signal blocked sigignore sigcatch wchan nswap cnswap exit_signal processor rt_priority policy delayacct_blkio_ticks guest_time cguest_time start_data end_data start_brk arg_start arg_end env_start env_end exit_code",
			wantFields: map[string]string{
				"PID_STAT_COMM":        "abcdefghijklmnopqrstuvwxyz",
				"PID_STAT_STATE":       "state",
				"PID_STAT_PPID":        "ppid",
				"PID_STAT_PGRP":        "pgrp",
				"PID_STAT_SESSION":     "session",
				"PID_STAT_TTY_NR":      "tty_nr",
				"PID_STAT_TPGID":       "tpgid",
				"PID_STAT_FLAGS":       "flags",
				"PID_STAT_MINFLT":      "minflt",
				"PID_STAT_MAJLT":       "majflt",
				"PID_STAT_UTIME":       "utime",
				"PID_STAT_STIME":       "stime",
				"PID_STAT_PRIORITY":    "priority",
				"PID_STAT_NICE":        "nice",
				"PID_STAT_NUM_THREADS": "num_threads",
				"PID_STAT_STARTTIME":   "starttime",
				"PID_STAT_VSIZE":       "vsize",
				"PID_STAT_RSS":         "rss",
				"PID_STAT_RSSLIM":      "rsslim",
				"PID_STAT_PROCESSOR":   "processor",
				"PID_STAT_RT_PRIORITY": "rt_priority",
				"PID_STAT_POLICY":      "policy",
			},
		},
		{
			name:        "comm_utf8",
			pidStatData: "pid (Nǐ hǎo shìjiè 你好世界) state ppid pgrp session tty_nr tpgid flags minflt cminflt majflt cmajflt utime stime cutime cstime priority nice num_threads itrealvalue starttime vsize rss rsslim startcode endcode startstack kstkesp kstkeip signal blocked sigignore sigcatch wchan nswap cnswap exit_signal processor rt_priority policy delayacct_blkio_ticks guest_time cguest_time start_data end_data start_brk arg_start arg_end env_start env_end exit_code",
			wantFields: map[string]string{
				"PID_STAT_COMM":        "Nǐ hǎo shìjiè 你好世界",
				"PID_STAT_STATE":       "state",
				"PID_STAT_PPID":        "ppid",
				"PID_STAT_PGRP":        "pgrp",
				"PID_STAT_SESSION":     "session",
				"PID_STAT_TTY_NR":      "tty_nr",
				"PID_STAT_TPGID":       "tpgid",
				"PID_STAT_FLAGS":       "flags",
				"PID_STAT_MINFLT":      "minflt",
				"PID_STAT_MAJLT":       "majflt",
				"PID_STAT_UTIME":       "utime",
				"PID_STAT_STIME":       "stime",
				"PID_STAT_PRIORITY":    "priority",
				"PID_STAT_NICE":        "nice",
				"PID_STAT_NUM_THREADS": "num_threads",
				"PID_STAT_STARTTIME":   "starttime",
				"PID_STAT_VSIZE":       "vsize",
				"PID_STAT_RSS":         "rss",
				"PID_STAT_RSSLIM":      "rsslim",
				"PID_STAT_PROCESSOR":   "processor",
				"PID_STAT_RT_PRIORITY": "rt_priority",
				"PID_STAT_POLICY":      "policy",
			},
		},
		{
			name:        "comm_missin_open_par",
			pidStatData: "pid comm) state ppid pgrp session tty_nr tpgid flags minflt cminflt majflt cmajflt utime stime cutime cstime priority nice num_threads itrealvalue starttime vsize rss rsslim startcode endcode startstack kstkesp kstkeip signal blocked sigignore sigcatch wchan nswap cnswap exit_signal processor rt_priority policy delayacct_blkio_ticks guest_time cguest_time start_data end_data start_brk arg_start arg_end env_start env_end exit_code",
			wantError:   fmt.Errorf("cannot locate '('"),
		},
		{
			name:        "comm_missin_close_par",
			pidStatData: "pid (comm state ppid pgrp session tty_nr tpgid flags minflt cminflt majflt cmajflt utime stime cutime cstime priority nice num_threads itrealvalue starttime vsize rss rsslim startcode endcode startstack kstkesp kstkeip signal blocked sigignore sigcatch wchan nswap cnswap exit_signal processor rt_priority policy delayacct_blkio_ticks guest_time cguest_time start_data end_data start_brk arg_start arg_end env_start env_end exit_code",
			wantError:   fmt.Errorf("cannot locate ')'"),
		},
		{
			name:        "missing_fields/got=21",
			pidStatData: "pid (comm) state ppid pgrp session tty_nr tpgid flags minflt cminflt majflt cmajflt utime stime cutime cstime priority nice num_threads itrealvalue starttime vsize rss rsslim startcode endcode startstack kstkesp kstkeip signal blocked sigignore sigcatch wchan nswap cnswap exit_signal processor rt_priority\n",
			wantError:   fmt.Errorf("scan incomplete: field#: want: %d, got: %d", PID_STAT_NUM_FIELDS, 21),
		},
	} {
		t.Run(tc.name, func(t *testing.T) { testPidStartParser(tc, t) })
	}
}
