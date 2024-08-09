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

func TestProcPidMetricsInitPidStatMetricsCache(t *testing.T) {
	const (
		TEST_INSTANCE       = "INSTANCE"
		TEST_HOSTNAME       = "HOST"
		TEST_PID_TID_LABELS = `pid="PID",tid="TID"`
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
			`proc_pid_stat_state{instance="INSTANCE",hostname="HOST",pid="PID",tid="TID",starttime_msec="STARTTIME_MS",state="S"} 1 TIMESTAMP`,
		},
		{
			pm.pidStatInfoMetricFmt,
			[]any{TEST_PID_TID_LABELS, []byte("COMM"), []byte("PPID"), []byte("PGRP"), []byte("SESSION"), []byte("TTY_NR"), []byte("TPGID"), []byte("FLAGS"), []byte("PRIORITY"), []byte("NICE"), []byte("RT_PRIORITY"), []byte("POLICY"), '1', TEST_TIMESTAMP},
			`proc_pid_stat_info{instance="INSTANCE",hostname="HOST",pid="PID",tid="TID",comm="COMM",ppid="PPID",pgrp="PGRP",session="SESSION",tty="TTY_NR",tpgid="TPGID",flags="FLAGS",prio="PRIORITY",nice="NICE",rt_prio="RT_PRIORITY",policy="POLICY"} 1 TIMESTAMP`,
		},
		{
			pm.pidStatCpuNumMetricFmt,
			[]any{TEST_PID_TID_LABELS, []byte("CPU_N"), TEST_TIMESTAMP},
			`proc_pid_stat_cpu_num{instance="INSTANCE",hostname="HOST",pid="PID",tid="TID"} CPU_N TIMESTAMP`,
		},
		{
			pm.pidStatMemoryMetricFmt[0].fmt,
			[]any{TEST_PID_TID_LABELS, []byte("VSIZE"), TEST_TIMESTAMP},
			`proc_pid_stat_vsize{instance="INSTANCE",hostname="HOST",pid="PID",tid="TID"} VSIZE TIMESTAMP`,
		},
		{
			pm.pidStatMemoryMetricFmt[1].fmt,
			[]any{TEST_PID_TID_LABELS, []byte("RSS"), TEST_TIMESTAMP},
			`proc_pid_stat_rss{instance="INSTANCE",hostname="HOST",pid="PID",tid="TID"} RSS TIMESTAMP`,
		},
		{
			pm.pidStatMemoryMetricFmt[2].fmt,
			[]any{TEST_PID_TID_LABELS, []byte("RSSLIM"), TEST_TIMESTAMP},
			`proc_pid_stat_rsslim{instance="INSTANCE",hostname="HOST",pid="PID",tid="TID"} RSSLIM TIMESTAMP`,
		},
		{
			pm.pidStatPcpuMetricFmt[0].fmt,
			[]any{TEST_PID_TID_LABELS, 0.123454600, TEST_TIMESTAMP},
			`proc_pid_stat_pcpu{instance="INSTANCE",hostname="HOST",pid="PID",tid="TID"} 0.1 TIMESTAMP`,
		},
		{
			pm.pidStatPcpuMetricFmt[1].fmt,
			[]any{TEST_PID_TID_LABELS, 1.123454600, TEST_TIMESTAMP},
			`proc_pid_stat_stime_pcpu{instance="INSTANCE",hostname="HOST",pid="PID",tid="TID"} 1.1 TIMESTAMP`,
		},
		{
			pm.pidStatPcpuMetricFmt[2].fmt,
			[]any{TEST_PID_TID_LABELS, 2.123454600, TEST_TIMESTAMP},
			`proc_pid_stat_utime_pcpu{instance="INSTANCE",hostname="HOST",pid="PID",tid="TID"} 2.1 TIMESTAMP`,
		},
		{
			pm.pidStatFltMetricFmt[0].fmt,
			[]any{TEST_PID_TID_LABELS, 100, TEST_TIMESTAMP},
			`proc_pid_stat_minflt_delta{instance="INSTANCE",hostname="HOST",pid="PID",tid="TID"} 100 TIMESTAMP`,
		},
		{
			pm.pidStatFltMetricFmt[1].fmt,
			[]any{TEST_PID_TID_LABELS, 200, TEST_TIMESTAMP},
			`proc_pid_stat_majflt_delta{instance="INSTANCE",hostname="HOST",pid="PID",tid="TID"} 200 TIMESTAMP`,
		},
	} {
		wantMetric := tc.wantMetric + "\n"
		gotMetric := fmt.Sprintf(tc.fmt, tc.args...)
		if wantMetric != gotMetric {
			t.Errorf("\n\twant: %q\n\t got: %q", wantMetric, gotMetric)
		} else {
			t.Log(tc.wantMetric)
		}
	}

}
