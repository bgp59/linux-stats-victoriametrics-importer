// Tests for pid metrics formats

package lsvmi

import (
	"fmt"
	"testing"

	"github.com/emypar/linux-stats-victoriametrics-importer/internal/testutils"
	"github.com/emypar/linux-stats-victoriametrics-importer/procfs"
)

type TestProcPidMetricsFmtTestCase struct {
	fmt        string
	args       []any
	wantMetric string
}

func TestProcPidMetricsInitPidStatMetricFmt(t *testing.T) {
	tlc := testutils.NewTestLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	pm, err := NewProcProcPidMetrics(nil, 0, nil)
	if err != nil {
		t.Fatal(err)
	}

	pm.instance = "INSTANCE"
	pm.hostname = "HOSTNAME"

	pm.initMetricsCache()

	for _, tc := range []*TestProcPidMetricsFmtTestCase{
		{
			pm.pidStatStateMetricFmt,
			[]any{`pid="PID",tid="TID"`, "STARTTIME_MS", []byte("S"), '1', []byte("TIMESTAMP")},
			`proc_pid_stat_state{instance="INSTANCE",hostname="HOSTNAME",pid="PID",tid="TID",starttime_msec="STARTTIME_MS",state="S"} 1 TIMESTAMP`,
		},
		{
			pm.pidStatInfoMetricFmt,
			[]any{`pid="PID",tid="TID"`, []byte("COMM"), []byte("PPID"), []byte("PGRP"), []byte("SESSION"), []byte("TTY_NR"), []byte("TPGID"), []byte("FLAGS"), '1', []byte("TIMESTAMP")},
			`proc_pid_stat_info{instance="INSTANCE",hostname="HOSTNAME",pid="PID",tid="TID",comm="COMM",ppid="PPID",pgrp="PGRP",session="SESSION",tty="TTY_NR",tpgid="TPGID",flags="FLAGS"} 1 TIMESTAMP`,
		},
		{
			pm.pidStatPriorityMetricFmt,
			[]any{`pid="PID",tid="TID"`, []byte("PRIORITY"), []byte("NICE"), []byte("RT_PRIORITY"), []byte("POLICY"), '1', []byte("TIMESTAMP")},
			`proc_pid_stat_prio{instance="INSTANCE",hostname="HOSTNAME",pid="PID",tid="TID",prio="PRIORITY",nice="NICE",rt_prio="RT_PRIORITY",policy="POLICY"} 1 TIMESTAMP`,
		},
		{
			pm.pidStatCpuNumMetricFmt,
			[]any{`pid="PID",tid="TID"`, []byte("CPU_N"), []byte("TIMESTAMP")},
			`proc_pid_stat_cpu_num{instance="INSTANCE",hostname="HOSTNAME",pid="PID",tid="TID"} CPU_N TIMESTAMP`,
		},
		{
			pm.pidStatMemoryMetricFmt[0].fmt,
			[]any{`pid="PID",tid="TID"`, []byte("VSIZE"), []byte("TIMESTAMP")},
			`proc_pid_stat_vsize{instance="INSTANCE",hostname="HOSTNAME",pid="PID",tid="TID"} VSIZE TIMESTAMP`,
		},
		{
			pm.pidStatMemoryMetricFmt[1].fmt,
			[]any{`pid="PID",tid="TID"`, []byte("RSS"), []byte("TIMESTAMP")},
			`proc_pid_stat_rss{instance="INSTANCE",hostname="HOSTNAME",pid="PID",tid="TID"} RSS TIMESTAMP`,
		},
		{
			pm.pidStatMemoryMetricFmt[2].fmt,
			[]any{`pid="PID",tid="TID"`, []byte("RSSLIM"), []byte("TIMESTAMP")},
			`proc_pid_stat_rsslim{instance="INSTANCE",hostname="HOSTNAME",pid="PID",tid="TID"} RSSLIM TIMESTAMP`,
		},
		{
			pm.pidStatPcpuMetricFmt[0].fmt,
			[]any{`pid="PID",tid="TID"`, 0.123454600, []byte("TIMESTAMP")},
			`proc_pid_stat_pcpu{instance="INSTANCE",hostname="HOSTNAME",pid="PID",tid="TID"} 0.1 TIMESTAMP`,
		},
		{
			pm.pidStatPcpuMetricFmt[1].fmt,
			[]any{`pid="PID",tid="TID"`, 1.123454600, []byte("TIMESTAMP")},
			`proc_pid_stat_stime_pcpu{instance="INSTANCE",hostname="HOSTNAME",pid="PID",tid="TID"} 1.1 TIMESTAMP`,
		},
		{
			pm.pidStatPcpuMetricFmt[2].fmt,
			[]any{`pid="PID",tid="TID"`, 2.123454600, []byte("TIMESTAMP")},
			`proc_pid_stat_utime_pcpu{instance="INSTANCE",hostname="HOSTNAME",pid="PID",tid="TID"} 2.1 TIMESTAMP`,
		},
		{
			pm.pidStatFltMetricFmt[0].fmt,
			[]any{`pid="PID",tid="TID"`, 100, []byte("TIMESTAMP")},
			`proc_pid_stat_minflt_delta{instance="INSTANCE",hostname="HOSTNAME",pid="PID",tid="TID"} 100 TIMESTAMP`,
		},
		{
			pm.pidStatFltMetricFmt[1].fmt,
			[]any{`pid="PID",tid="TID"`, 200, []byte("TIMESTAMP")},
			`proc_pid_stat_majflt_delta{instance="INSTANCE",hostname="HOSTNAME",pid="PID",tid="TID"} 200 TIMESTAMP`,
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

func TestProcPidMetricsInitPidStatusMetricFmt(t *testing.T) {
	tlc := testutils.NewTestLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	pm, err := NewProcProcPidMetrics(nil, 0, nil)
	if err != nil {
		t.Fatal(err)
	}

	pm.instance = "INSTANCE"
	pm.hostname = "HOSTNAME"
	pm.usePidStatus = true

	pm.initMetricsCache()

	for _, tc := range []*TestProcPidMetricsFmtTestCase{
		{
			pm.pidStatusInfoMetricFmt,
			[]any{`pid="PID",tid="TID"`, "UID", "GID", "GROUPS", "CPUS_ALLOWED", "MEMS_ALLOWED", '1', []byte("TIMESTAMP")},
			`proc_pid_status_info{instance="INSTANCE",hostname="HOSTNAME",pid="PID",tid="TID",uid="UID",gid="GID",groups="GROUPS",cpus_allowed="CPUS_ALLOWED",mems_allowed="MEMS_ALLOWED"} 1 TIMESTAMP`,
		},
		{
			pm.pidStatusPidOnlyMemoryMetricFmt[0].fmt,
			[]any{`pid="PID",tid="TID"`, []byte("UNIT"), []byte("VM_PEAK"), []byte("TIMESTAMP")},
			`proc_pid_status_vm_peak{instance="INSTANCE",hostname="HOSTNAME",pid="PID",tid="TID",unit="UNIT"} VM_PEAK TIMESTAMP`,
		},
		{
			pm.pidStatusPidOnlyMemoryMetricFmt[1].fmt,
			[]any{`pid="PID",tid="TID"`, []byte("UNIT"), []byte("VM_SIZE"), []byte("TIMESTAMP")},
			`proc_pid_status_vm_size{instance="INSTANCE",hostname="HOSTNAME",pid="PID",tid="TID",unit="UNIT"} VM_SIZE TIMESTAMP`,
		},
		{
			pm.pidStatusPidOnlyMemoryMetricFmt[2].fmt,
			[]any{`pid="PID",tid="TID"`, []byte("UNIT"), []byte("VM_LCK"), []byte("TIMESTAMP")},
			`proc_pid_status_vm_lck{instance="INSTANCE",hostname="HOSTNAME",pid="PID",tid="TID",unit="UNIT"} VM_LCK TIMESTAMP`,
		},
		{
			pm.pidStatusPidOnlyMemoryMetricFmt[3].fmt,
			[]any{`pid="PID",tid="TID"`, []byte("UNIT"), []byte("VM_PIN"), []byte("TIMESTAMP")},
			`proc_pid_status_vm_pin{instance="INSTANCE",hostname="HOSTNAME",pid="PID",tid="TID",unit="UNIT"} VM_PIN TIMESTAMP`,
		},
		{
			pm.pidStatusPidOnlyMemoryMetricFmt[4].fmt,
			[]any{`pid="PID",tid="TID"`, []byte("UNIT"), []byte("VM_HWM"), []byte("TIMESTAMP")},
			`proc_pid_status_vm_hwm{instance="INSTANCE",hostname="HOSTNAME",pid="PID",tid="TID",unit="UNIT"} VM_HWM TIMESTAMP`,
		},
		{
			pm.pidStatusPidOnlyMemoryMetricFmt[5].fmt,
			[]any{`pid="PID",tid="TID"`, []byte("UNIT"), []byte("VM_RSS"), []byte("TIMESTAMP")},
			`proc_pid_status_vm_rss{instance="INSTANCE",hostname="HOSTNAME",pid="PID",tid="TID",unit="UNIT"} VM_RSS TIMESTAMP`,
		},
		{
			pm.pidStatusPidOnlyMemoryMetricFmt[6].fmt,
			[]any{`pid="PID",tid="TID"`, []byte("UNIT"), []byte("RSS_ANON"), []byte("TIMESTAMP")},
			`proc_pid_status_rss_anon{instance="INSTANCE",hostname="HOSTNAME",pid="PID",tid="TID",unit="UNIT"} RSS_ANON TIMESTAMP`,
		},
		{
			pm.pidStatusPidOnlyMemoryMetricFmt[7].fmt,
			[]any{`pid="PID",tid="TID"`, []byte("UNIT"), []byte("RSS_FILE"), []byte("TIMESTAMP")},
			`proc_pid_status_rss_file{instance="INSTANCE",hostname="HOSTNAME",pid="PID",tid="TID",unit="UNIT"} RSS_FILE TIMESTAMP`,
		},
		{
			pm.pidStatusPidOnlyMemoryMetricFmt[8].fmt,
			[]any{`pid="PID",tid="TID"`, []byte("UNIT"), []byte("RSS_SHMEM"), []byte("TIMESTAMP")},
			`proc_pid_status_rss_shmem{instance="INSTANCE",hostname="HOSTNAME",pid="PID",tid="TID",unit="UNIT"} RSS_SHMEM TIMESTAMP`,
		},
		{
			pm.pidStatusPidOnlyMemoryMetricFmt[9].fmt,
			[]any{`pid="PID",tid="TID"`, []byte("UNIT"), []byte("VM_DATA"), []byte("TIMESTAMP")},
			`proc_pid_status_vm_data{instance="INSTANCE",hostname="HOSTNAME",pid="PID",tid="TID",unit="UNIT"} VM_DATA TIMESTAMP`,
		},
		{
			pm.pidStatusPidTidMemoryMetricFmt[0].fmt,
			[]any{`pid="PID",tid="TID"`, []byte("UNIT"), []byte("VM_STK"), []byte("TIMESTAMP")},
			`proc_pid_status_vm_stk{instance="INSTANCE",hostname="HOSTNAME",pid="PID",tid="TID",unit="UNIT"} VM_STK TIMESTAMP`,
		},
		{
			pm.pidStatusPidOnlyMemoryMetricFmt[10].fmt,
			[]any{`pid="PID",tid="TID"`, []byte("UNIT"), []byte("VM_EXE"), []byte("TIMESTAMP")},
			`proc_pid_status_vm_exe{instance="INSTANCE",hostname="HOSTNAME",pid="PID",tid="TID",unit="UNIT"} VM_EXE TIMESTAMP`,
		},
		{
			pm.pidStatusPidOnlyMemoryMetricFmt[11].fmt,
			[]any{`pid="PID",tid="TID"`, []byte("UNIT"), []byte("VM_LIB"), []byte("TIMESTAMP")},
			`proc_pid_status_vm_lib{instance="INSTANCE",hostname="HOSTNAME",pid="PID",tid="TID",unit="UNIT"} VM_LIB TIMESTAMP`,
		},
		{
			pm.pidStatusPidOnlyMemoryMetricFmt[12].fmt,
			[]any{`pid="PID",tid="TID"`, []byte("UNIT"), []byte("VM_PTE"), []byte("TIMESTAMP")},
			`proc_pid_status_vm_pte{instance="INSTANCE",hostname="HOSTNAME",pid="PID",tid="TID",unit="UNIT"} VM_PTE TIMESTAMP`,
		},
		{
			pm.pidStatusPidOnlyMemoryMetricFmt[13].fmt,
			[]any{`pid="PID",tid="TID"`, []byte("UNIT"), []byte("VM_PMD"), []byte("TIMESTAMP")},
			`proc_pid_status_vm_pmd{instance="INSTANCE",hostname="HOSTNAME",pid="PID",tid="TID",unit="UNIT"} VM_PMD TIMESTAMP`,
		},
		{
			pm.pidStatusPidOnlyMemoryMetricFmt[14].fmt,
			[]any{`pid="PID",tid="TID"`, []byte("UNIT"), []byte("VM_SWAP"), []byte("TIMESTAMP")},
			`proc_pid_status_vm_swap{instance="INSTANCE",hostname="HOSTNAME",pid="PID",tid="TID",unit="UNIT"} VM_SWAP TIMESTAMP`,
		},
		{
			pm.pidStatusPidOnlyMemoryMetricFmt[15].fmt,
			[]any{`pid="PID",tid="TID"`, []byte("UNIT"), []byte("HUGETBLPAGES"), []byte("TIMESTAMP")},
			`proc_pid_status_hugetlbpages{instance="INSTANCE",hostname="HOSTNAME",pid="PID",tid="TID",unit="UNIT"} HUGETBLPAGES TIMESTAMP`,
		},
		{
			pm.pidStatusCtxMetricFmt[0].fmt,
			[]any{`pid="PID",tid="TID"`, 100, []byte("TIMESTAMP")},
			`proc_pid_status_vol_ctx_switch_delta{instance="INSTANCE",hostname="HOSTNAME",pid="PID",tid="TID"} 100 TIMESTAMP`,
		},
		{
			pm.pidStatusCtxMetricFmt[1].fmt,
			[]any{`pid="PID",tid="TID"`, 101, []byte("TIMESTAMP")},
			`proc_pid_status_nonvol_ctx_switch_delta{instance="INSTANCE",hostname="HOSTNAME",pid="PID",tid="TID"} 101 TIMESTAMP`,
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

func TestProcPidMetricsInitSpecificMetricFmt(t *testing.T) {
	tlc := testutils.NewTestLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	pm, err := NewProcProcPidMetrics(nil, 0, nil)
	if err != nil {
		t.Fatal(err)
	}

	pm.partNo = 13
	pm.instance = "INSTANCE"
	pm.hostname = "HOSTNAME"

	pm.initMetricsCache()

	for _, tc := range []*TestProcPidMetricsFmtTestCase{
		{
			pm.pidActiveCountMetricFmt,
			[]any{113, []byte("TIMESTAMP")},
			`proc_pid_active_count{instance="INSTANCE",hostname="HOSTNAME",part="13"} 113 TIMESTAMP`,
		},
		{
			pm.pidTotalCountMetricFmt,
			[]any{1113, []byte("TIMESTAMP")},
			`proc_pid_total_count{instance="INSTANCE",hostname="HOSTNAME",part="13"} 1113 TIMESTAMP`,
		},
		{
			pm.intervalMetricFmt,
			[]any{0.1234567, []byte("TIMESTAMP")},
			`proc_pid_metrics_delta_sec{instance="INSTANCE",hostname="HOSTNAME",part="13"} 0.123457 TIMESTAMP`,
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

func TestProcPidMetricsInitPidTidMetricsInfo(t *testing.T) {
	tlc := testutils.NewTestLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	pm, err := NewProcProcPidMetrics(nil, 0, nil)
	if err != nil {
		t.Fatal(err)
	}

	pm.instance = "INSTANCE"
	pm.hostname = "HOSTNAME"
	pm.procfsRoot = "PROCFS"

	tcd := &TestPidParsersTestCaseData{}
	pm.newPidStatParser = tcd.NewPidStat
	pm.newPidStatusParser = tcd.NewPidStatus
	pm.usePidStatus = true
	pm.boottimeMsec = 9_800_000
	pm.linuxClktckSec = .01

	pidStatByteSliceFields := make([]string, procfs.PID_STAT_BYTE_SLICE_NUM_FIELDS)
	pidStatByteSliceFields[procfs.PID_STAT_STARTTIME] = "1234"
	pm.pidStat = &TestPidStat{
		parseResult: &TestPidStatParsedData{
			ByteSliceFields: pidStatByteSliceFields,
		},
	}

	pidTidPath := "PID_TID_PATH"
	pidTidMetricsInfo := pm.initPidTidMetricsInfo(procfs.PidTid{Pid: 100, Tid: 101}, pidTidPath)

	wantPidTidPath, gotPidTidPath := pidTidPath, pidTidMetricsInfo.pidTidPath
	if wantPidTidPath != gotPidTidPath {
		t.Errorf("pidTidMetricsInfo.pidTidPath: want: %q, got: %q", wantPidTidPath, gotPidTidPath)
	} else {
		t.Logf("pidTidMetricsInfo.pidTidPath: %q", gotPidTidPath)
	}

	wantPidTidLabels, gotPidTidLabels := `pid="100",tid="101"`, pidTidMetricsInfo.pidTidLabels
	if wantPidTidLabels != gotPidTidLabels {
		t.Errorf("pidTidMetricsInfo.pidTidLabels: want: %q, got: %q", wantPidTidLabels, gotPidTidLabels)
	} else {
		t.Logf("pidTidMetricsInfo.pidTidLabels: %q", gotPidTidLabels)
	}

	wantStarttimeMsec, gotStarttimeMsec := "9812340", pidTidMetricsInfo.starttimeMsec
	if wantStarttimeMsec != gotStarttimeMsec {
		t.Errorf("pidTidMetricsInfo.starttimeMsec: want: %q, got: %q", wantStarttimeMsec, gotStarttimeMsec)
	} else {
		t.Logf("pidTidMetricsInfo.starttimeMsec: %q", gotStarttimeMsec)
	}
}
