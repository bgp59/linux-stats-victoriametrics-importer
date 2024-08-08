// Tests for pid metrics

package lsvmi

import (
	"fmt"
	"testing"

	"github.com/emypar/linux-stats-victoriametrics-importer/internal/testutils"
)

type TestProcPidMetricsFmtTestCase struct {
	fmt        string
	args       []any
	wantMetric string
}

func TestProcPidMetricsInitMetricsCache(t *testing.T) {
	const (
		TEST_INSTANCE       = "INSTANCE"
		TEST_HOSTNAME       = "HOST"
		TEST_PID            = 100
		TEST_TID            = 101
		TEST_PID_TID_LABELS = `pid="100",tid="101"`
		TEST_STARTTIME      = "STARTTIME_MS"
		TEST_TIMESTAMP      = "TIMESTAMP"
	)

	tlc := testutils.NewTestLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	pm, err := NewProcProcPidMetrics(nil, 0, nil)
	if err != nil {
		t.Fatal(err)
	}

	pm.instance = TEST_INSTANCE
	pm.hostname = TEST_HOSTNAME

	pm.initMetricsCache()

	for _, tc := range []*TestProcPidMetricsFmtTestCase{
		{
			pm.pidStatStateMetricFmt,
			[]any{TEST_PID_TID_LABELS, TEST_STARTTIME, []byte("S"), '1', TEST_TIMESTAMP},
			`proc_pid_stat_state{instance="INSTANCE",hostname="HOST",pid="100",tid="101",starttime_msec="STARTTIME_MS",state="S"} 1 TIMESTAMP` + "\n",
		},
		{
			pm.pidStatInfoMetricFmt,
			[]any{TEST_PID_TID_LABELS, []byte("COMM"), []byte("PPID"), []byte("PGRP"), []byte("SESSION"), []byte("TTY_NR"), []byte("TPGID"), []byte("FLAGS"), []byte("PRIORITY"), []byte("NICE"), []byte("RT_PRIORITY"), []byte("POLICY"), '1', TEST_TIMESTAMP},
			`proc_pid_stat_info{instance="INSTANCE",hostname="HOST",pid="100",tid="101",comm="COMM",ppid="PPID",pgrp="PGRP",session="SESSION",tty="TTY_NR",tpgid="TPGID",flags="FLAGS",prio="PRIORITY",nice="NICE",rt_prio="RT_PRIORITY",policy="POLICY"} 1 TIMESTAMP` + "\n",
		},
		{
			pm.pidStatCpuNumMetricFmt,
			[]any{TEST_PID_TID_LABELS, []byte("CPU_N"), TEST_TIMESTAMP},
			`proc_pid_stat_cpu_num{instance="INSTANCE",hostname="HOST",pid="100",tid="101"} CPU_N TIMESTAMP` + "\n",
		},
	} {
		gotMetric := fmt.Sprintf(tc.fmt, tc.args...)
		if tc.wantMetric != gotMetric {
			t.Errorf("\n\twant: %q\n\t got: %q", tc.wantMetric, gotMetric)
		}
	}

}
