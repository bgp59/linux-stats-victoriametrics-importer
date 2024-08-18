#! /usr/bin/env python3

from collections import OrderedDict
from copy import deepcopy
from dataclasses import dataclass, field
from typing import List, Optional, Tuple

import procfs

from . import (
    DEFAULT_TEST_HOSTNAME,
    DEFAULT_TEST_INSTANCE,
    HOSTNAME_LABEL_NAME,
    INSTANCE_LABEL_NAME,
    TEST_BOOTTIME_SEC,
    TEST_LINUX_CLKTCK_SEC,
    uint64_delta,
)

DEFAULT_BOOTTIME_MSEC = TEST_BOOTTIME_SEC * 1000

# Should match lsvmi/proc_pid_metrics.go:
DEFAULT_PROC_PID_INTERVAL_SEC = 1
DEFAULT_PROC_PID_FULL_METRICS_FACTOR = 15

PROC_PID_METRICS_CYCLE_NUM_COUNTERS = 1 << 4

# All metrics will have the following labels:
PROC_PID_PID_LABEL_NAME = "pid"
PROC_PID_TID_LABEL_NAME = "tid"  # TID only

# /proc/PID/stat:
PROC_PID_STAT_STATE_METRIC = "proc_pid_stat_state"  # PID + TID
PROC_PID_STAT_STATE_LABEL_NAME = "state"
PROC_PID_STAT_STARTTIME_LABEL_NAME = "starttime_msec"

PROC_PID_STAT_INFO_METRIC = "proc_pid_stat_info"  # PID only
PROC_PID_STAT_COMM_LABEL_NAME = "comm"
PROC_PID_STAT_PPID_LABEL_NAME = "ppid"
PROC_PID_STAT_PGRP_LABEL_NAME = "pgrp"
PROC_PID_STAT_SESSION_LABEL_NAME = "session"
PROC_PID_STAT_TTY_NR_LABEL_NAME = "tty"
PROC_PID_STAT_TPGID_LABEL_NAME = "tpgid"
PROC_PID_STAT_FLAGS_LABEL_NAME = "flags"
PROC_PID_STAT_PRIORITY_LABEL_NAME = "prio"
PROC_PID_STAT_NICE_LABEL_NAME = "nice"
PROC_PID_STAT_RT_PRIORITY_LABEL_NAME = "rt_prio"
PROC_PID_STAT_POLICY_LABEL_NAME = "policy"

PROC_PID_STAT_VSIZE_METRIC = "proc_pid_stat_vsize"  # PID only
PROC_PID_STAT_RSS_METRIC = "proc_pid_stat_rss"  # PID only
PROC_PID_STAT_RSSLIM_METRIC = "proc_pid_stat_rsslim"  # PID only

PROC_PID_STAT_MINFLT_METRIC = "proc_pid_stat_minflt_delta"  # PID + TID
PROC_PID_STAT_MAJFLT_METRIC = "proc_pid_stat_majflt_delta"  # PID + TID

PROC_PID_STAT_UTIME_PCT_METRIC = "proc_pid_stat_utime_pcpu"  # PID + TID
PROC_PID_STAT_STIME_PCT_METRIC = "proc_pid_stat_stime_pcpu"  # PID + TID
PROC_PID_STAT_TIME_PCT_METRIC = "proc_pid_stat_pcpu"  # PID + TID

PROC_PID_STAT_CPU_NUM_METRIC = "proc_pid_stat_cpu_num"  # PID + TID

# /proc/PID/status:
PROC_PID_STATUS_INFO_METRIC = "proc_pid_status_info"  # PID only
PROC_PID_STATUS_UID_LABEL_NAME = "uid"
PROC_PID_STATUS_GID_LABEL_NAME = "gid"
PROC_PID_STATUS_GROUPS_LABEL_NAME = "groups"
PROC_PID_STATUS_CPUS_ALLOWED_LIST_LABEL_NAME = "cpus_allowed"
PROC_PID_STATUS_MEMS_ALLOWED_LIST_LABEL_NAME = "mems_allowed"

PROC_PID_STATUS_VM_PEAK_METRIC = "proc_pid_status_vm_peak"  # PID only
PROC_PID_STATUS_VM_SIZE_METRIC = "proc_pid_status_vm_size"  # PID only
PROC_PID_STATUS_VM_LCK_METRIC = "proc_pid_status_vm_lck"  # PID only
PROC_PID_STATUS_VM_PIN_METRIC = "proc_pid_status_vm_pin"  # PID only
PROC_PID_STATUS_VM_HWM_METRIC = "proc_pid_status_vm_hwm"  # PID only
PROC_PID_STATUS_VM_RSS_METRIC = "proc_pid_status_vm_rss"  # PID only
PROC_PID_STATUS_RSS_ANON_METRIC = "proc_pid_status_rss_anon"  # PID only
PROC_PID_STATUS_RSS_FILE_METRIC = "proc_pid_status_rss_file"  # PID only
PROC_PID_STATUS_RSS_SHMEM_METRIC = "proc_pid_status_rss_shmem"  # PID only
PROC_PID_STATUS_VM_DATA_METRIC = "proc_pid_status_vm_data"  # PID only
PROC_PID_STATUS_VM_STK_METRIC = "proc_pid_status_vm_stk"  # PID + TID
PROC_PID_STATUS_VM_EXE_METRIC = "proc_pid_status_vm_exe"  # PID only
PROC_PID_STATUS_VM_LIB_METRIC = "proc_pid_status_vm_lib"  # PID only
PROC_PID_STATUS_VM_PTE_METRIC = "proc_pid_status_vm_pte"  # PID only
PROC_PID_STATUS_VM_PMD_METRIC = "proc_pid_status_vm_pmd"  # PID only
PROC_PID_STATUS_VM_SWAP_METRIC = "proc_pid_status_vm_swap"  # PID only
PROC_PID_STATUS_HUGETLBPAGES_METRIC = "proc_pid_status_hugetlbpages"  # PID only
PROC_PID_STATUS_VM_UNIT_LABEL_NAME = "unit"

proc_pid_status_pid_tid_vm_metrics = {
    PROC_PID_STATUS_VM_STK_METRIC,
}

PROC_PID_STATUS_VOLUNTARY_CTXT_SWITCHES_METRIC = (
    "proc_pid_status_vol_ctx_switch_delta"  # PID + TID
)
PROC_PID_STATUS_NONVOLUNTARY_CTXT_SWITCHES_METRIC = (
    "proc_pid_status_nonvol_ctx_switch_delta"  # PID + TID
)

# /proc/PID/cmdline.
PROC_PID_CMDLINE_METRIC = (
    "proc_pid_cmdline"  # PID only, well behaved threads don't change their command line
)
PROC_PID_CMDLINE_LABEL_NAME = "cmdline"

# This generator's specific metrics, i.e. in addition to those described in
# metrics_common.go:

# They all have the following label:
PROC_PID_PART_LABEL_NAME = "part"  # partition

# Active/total PID counts:
PROC_PID_ACTIVE_COUNT_METRIC = "proc_pid_active_count"
PROC_PID_TOTAL_COUNT_METRIC = "proc_pid_total_count"

# Interval since last generation, i.e. the interval underlying the deltas.
# Normally this should be close to scan interval, but this is the actual
# value, rather than the desired one:
PROC_PID_INTERVAL_METRIC = "proc_pid_metrics_delta_sec"

# Based on lsvmi/proc_pid_metrics_utils_test.go:
@dataclass
class TestPidStatParsedData:
    ByteSliceFields: Optional[List[str]] = None
    NumericFields: Optional[List[int]] = None


@dataclass
class TestPidStatusParsedData:
    ByteSliceFields: Optional[List[str]] = None
    ByteSliceFieldUnit: Optional[List[str]] = None
    NumericFields: Optional[List[int]] = None


@dataclass
class TestPidCmdlineParsedData:
    Cmdline: str = None


@dataclass
class TestPidParserData:
    PidStat: Optional[TestPidStatParsedData] = None
    PidStatus: Optional[TestPidStatusParsedData] = None
    PidCmdline: Optional[TestPidCmdlineParsedData] = None
    PidTid: Optional[procfs.PidTid] = None


@dataclass
class TestPidParsersTestCaseData:
    ParserData: Optional[List[TestPidParserData]] = None
    ProcfsRoot: str = "DUMMY_PROCFS_ROOT_DIR"


@dataclass
class TestProcPidTidMetricsInfoData:
    PidStatData: Optional[TestPidStatParsedData] = None
    PidStatusData: Optional[TestPidStatusParsedData] = None
    PidStatFltZeroDelta: Optional[List[bool]] = None
    PidStatusCtxZeroDelta: Optional[List[bool]] = None
    PidTid: Optional[procfs.PidTid] = None


# Indexes in PidStatFltZeroDelta and PidStatusCtxZeroDelta should match the
# order the ProcPidMetrics.pidStatFltMetricFmt and
# ProcPidMetrics.pidStatusCtxMetricFmt are built:
PID_STAT_MINFLT_ZERO_DELTA_INDEX = 0
PID_STAT_MAJFLT_ZERO_DELTA_INDEX = 1
PID_STAT_FLT_ZERO_DELTA_SIZE = 2

PID_STATUS_VOLUNTARY_CTXT_SWITCHES_ZERO_DELTA_INDEX = 0
PID_STATUS_NONVOLUNTARY_CTXT_SWITCHES_ZERO_DELTA_INDEX = 1
PID_STATUS_CTX_ZERO_DELTA_SIZE = 2

# Based on lsvmi/proc_pid_metrics_test.go:
@dataclass
class ProcPidMetricsGenerateTestCase:
    Name: Optional[str] = None
    Description: Optional[str] = None
    ProcfsRoot: Optional[str] = None

    Instance: Optional[str] = DEFAULT_TEST_INSTANCE
    Hostname: Optional[str] = DEFAULT_TEST_HOSTNAME
    LinuxClktckSec: float = TEST_LINUX_CLKTCK_SEC
    BoottimeMsec: int = DEFAULT_BOOTTIME_MSEC

    PidTidMetricsInfo: Optional[TestProcPidTidMetricsInfoData] = None
    ParserData: Optional[TestPidParserData] = None
    FullMetrics: bool = False

    CurrPromTs: int = 0
    PrevPromTs: int = 0

    WantMetricsCount: int = 0
    WantMetrics: Optional[str] = None
    ReportExtra: bool = True
    WantZeroDelta: Optional[TestProcPidTidMetricsInfoData] = None


@dataclass
class ProcPidMetricsExecuteTestCase:
    Name: Optional[str] = None
    Description: Optional[str] = None

    NPart: int = 0
    FullMetricsFactor: int = 15
    UsePidStatus: bool = False
    CycleNum: List[int] = field(
        default_factory=lambda: [0] * PROC_PID_METRICS_CYCLE_NUM_COUNTERS
    )
    ScanNum: int = 0

    Instance: Optional[str] = None
    Hostname: Optional[str] = None
    LinuxClktckSec: float = TEST_LINUX_CLKTCK_SEC
    BoottimeMsec: int = DEFAULT_BOOTTIME_MSEC

    PidTidListResult: Optional[List[procfs.PidTid]] = None
    PidTidMetricsInfo: Optional[List[TestProcPidTidMetricsInfoData]] = None
    TestCaseData: Optional[TestPidParsersTestCaseData] = None

    CurrPromTs: int = 0
    PrevPromTs: int = 0

    WantMetricsCount: int = 0
    WantMetrics: Optional[str] = None
    ReportExtra: bool = True
    WantZeroDelta: Optional[List[TestProcPidTidMetricsInfoData]] = None


# Use an ordered dict to match the expected label order:
pid_stat_info_index_to_label_map = OrderedDict(
    [
        (procfs.PID_STAT_COMM, PROC_PID_STAT_COMM_LABEL_NAME),
        (procfs.PID_STAT_PPID, PROC_PID_STAT_PPID_LABEL_NAME),
        (procfs.PID_STAT_PGRP, PROC_PID_STAT_PGRP_LABEL_NAME),
        (procfs.PID_STAT_SESSION, PROC_PID_STAT_SESSION_LABEL_NAME),
        (procfs.PID_STAT_TTY_NR, PROC_PID_STAT_TTY_NR_LABEL_NAME),
        (procfs.PID_STAT_TPGID, PROC_PID_STAT_TPGID_LABEL_NAME),
        (procfs.PID_STAT_FLAGS, PROC_PID_STAT_FLAGS_LABEL_NAME),
        (procfs.PID_STAT_PRIORITY, PROC_PID_STAT_PRIORITY_LABEL_NAME),
        (procfs.PID_STAT_NICE, PROC_PID_STAT_NICE_LABEL_NAME),
        (procfs.PID_STAT_RT_PRIORITY, PROC_PID_STAT_RT_PRIORITY_LABEL_NAME),
        (procfs.PID_STAT_POLICY, PROC_PID_STAT_POLICY_LABEL_NAME),
    ]
)


def generate_pid_stat_info_labels(pid_stat_bsf: List[str]) -> str:
    return ",".join(
        [
            f'{label}="{pid_stat_bsf[index]}"'
            for index, label in pid_stat_info_index_to_label_map.items()
        ]
    )


# Use an ordered dict to match the expected label order:
pid_status_info_index_to_label_map = OrderedDict(
    [
        (procfs.PID_STATUS_UID, PROC_PID_STATUS_UID_LABEL_NAME),
        (procfs.PID_STATUS_GID, PROC_PID_STATUS_GID_LABEL_NAME),
        (procfs.PID_STATUS_GROUPS, PROC_PID_STATUS_GROUPS_LABEL_NAME),
        (
            procfs.PID_STATUS_CPUS_ALLOWED_LIST,
            PROC_PID_STATUS_CPUS_ALLOWED_LIST_LABEL_NAME,
        ),
        (
            procfs.PID_STATUS_MEMS_ALLOWED_LIST,
            PROC_PID_STATUS_MEMS_ALLOWED_LIST_LABEL_NAME,
        ),
    ]
)


def generate_pid_status_info_labels(pid_status_bsf: List[str]) -> str:
    return ",".join(
        [
            f'{label}="{pid_status_bsf[index]}"'
            for index, label in pid_status_info_index_to_label_map.items()
        ]
    )


def generate_proc_pid_metrics(
    pid_parser_data: TestPidParserData,
    curr_prom_ts: int,
    pid_metrics_info_data: Optional[
        TestProcPidTidMetricsInfoData
    ] = None,  # i.e. no prev
    interval: float = DEFAULT_PROC_PID_INTERVAL_SEC,
    full_metrics: bool = False,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    boottime_msec: int = DEFAULT_BOOTTIME_MSEC,
    linux_clktck_sec: float = TEST_LINUX_CLKTCK_SEC,
) -> Tuple[List[str], TestProcPidTidMetricsInfoData]:
    metrics = []
    want_zero_delta = TestProcPidTidMetricsInfoData(
        PidStatFltZeroDelta=[False] * PID_STAT_FLT_ZERO_DELTA_SIZE,
        PidStatusCtxZeroDelta=[False] * PID_STATUS_CTX_ZERO_DELTA_SIZE,
    )

    is_pid = pid_parser_data.PidTid.Tid == procfs.PID_ONLY_TID
    # Labels common to all metrics:
    common_labels = ",".join(
        [
            f'{INSTANCE_LABEL_NAME}="{instance}"',
            f'{HOSTNAME_LABEL_NAME}="{hostname}"',
            f'{PROC_PID_PID_LABEL_NAME}="{pid_parser_data.PidTid.Pid}"',
        ]
    )
    if not is_pid:
        common_labels += f',{PROC_PID_TID_LABEL_NAME}="{pid_parser_data.PidTid.Tid}"'

    # /proc/PID/stat metrics:
    if pid_parser_data.PidStat is not None:
        has_prev = (
            pid_metrics_info_data is not None
            and pid_metrics_info_data.PidStatData is not None
        )
        pid_stat_full_metrics = full_metrics or not has_prev
        curr_pid_stat_bsf = pid_parser_data.PidStat.ByteSliceFields
        curr_pid_stat_nf = pid_parser_data.PidStat.NumericFields
        if has_prev:
            prev_pid_stat_bsf = pid_metrics_info_data.PidStatData.ByteSliceFields
            prev_pid_stat_nf = pid_metrics_info_data.PidStat.NumericFields
        else:
            prev_pid_stat_bsf = None
            prev_pid_stat_nf = None
        starttime_msec = boottime_msec + int(
            float(curr_pid_stat_bsf[procfs.PID_STAT_STARTTIME])
            * linux_clktck_sec
            * 1000
        )

        ## PID+TID:
        ### PROC_PID_STAT_STATE_METRIC:
        has_changed = (
            prev_pid_stat_bsf is not None
            and curr_pid_stat_bsf[procfs.PID_STAT_STATE]
            != prev_pid_stat_bsf[procfs.PID_STAT_STATE]
        )
        if has_changed:
            metrics.append(
                f"{PROC_PID_STAT_STATE_METRIC}{{"
                + ",".join(
                    [
                        common_labels,
                        f'{PROC_PID_STAT_STARTTIME_LABEL_NAME}="{starttime_msec}"',
                        f'{PROC_PID_STAT_STATE_LABEL_NAME}="{prev_pid_stat_bsf[procfs.PID_STAT_STATE]}"',
                    ]
                )
                + f"}} 0 {curr_prom_ts}"
            )
        if pid_stat_full_metrics or has_changed:
            metrics.append(
                f"{PROC_PID_STAT_STATE_METRIC}{{"
                + ",".join(
                    [
                        common_labels,
                        f'{PROC_PID_STAT_STARTTIME_LABEL_NAME}="{starttime_msec}"',
                        f'{PROC_PID_STAT_STATE_LABEL_NAME}="{curr_pid_stat_bsf[procfs.PID_STAT_STATE]}"',
                    ]
                )
                + f"}} 1 {curr_prom_ts}"
            )

        if has_prev:
            ### PROC_PID_STAT_*FLT_METRIC:
            for index, metric_name, zd_index in [
                (
                    procfs.PID_STAT_MINFLT,
                    PROC_PID_STAT_MINFLT_METRIC,
                    PID_STAT_MINFLT_ZERO_DELTA_INDEX,
                ),
                (
                    procfs.PID_STAT_MAJFLT,
                    PROC_PID_STAT_MAJFLT_METRIC,
                    PID_STAT_MAJFLT_ZERO_DELTA_INDEX,
                ),
            ]:
                delta = uint64_delta(curr_pid_stat_nf[index], prev_pid_stat_nf[index])
                if (
                    delta != 0
                    or pid_stat_full_metrics
                    or not pid_metrics_info_data.PidStatFltZeroDelta[zd_index]
                ):
                    metrics.append(
                        f"{metric_name}{{{common_labels}}} {delta} {curr_prom_ts}"
                    )
                want_zero_delta.PidStatFltZeroDelta[zd_index] = delta == 0

            ### PROC_PID_STAT_*TIME_PCT_METRIC:
            pcpu_factor = linux_clktck_sec / interval * 100.0
            cpu_ticks = 0
            for index, metric_name in [
                (procfs.PID_STAT_UTIME, PROC_PID_STAT_UTIME_PCT_METRIC),
                (procfs.PID_STAT_STIME, PROC_PID_STAT_STIME_PCT_METRIC),
            ]:
                delta_ticks = uint64_delta(
                    curr_pid_stat_nf[index], prev_pid_stat_nf[index]
                )
                cpu_ticks += delta_ticks
                metrics.append(
                    f"{metric_name}{{{common_labels}}} {delta_ticks*pcpu_factor:.1f} {curr_prom_ts}"
                )
            metrics.append(
                f"{PROC_PID_STAT_TIME_PCT_METRIC}{{{common_labels}}} {delta_ticks*pcpu_factor:.1f} {curr_prom_ts}"
            )

            ### PROC_PID_STAT_CPU_NUM_METRIC:
            metrics.append(
                f"{PROC_PID_STAT_CPU_NUM_METRIC}{{{common_labels}}} {curr_pid_stat_bsf[procfs.PID_STAT_PROCESSOR]} {curr_prom_ts}"
            )

        ## PID only:
        if is_pid:
            ### PROC_PID_STAT_INFO_METRIC:
            has_changed = False
            if has_prev:
                for i in pid_stat_info_index_to_label_map:
                    has_changed = curr_pid_stat_bsf[i] != prev_pid_stat_bsf[i]
                    if has_changed:
                        break
                if has_changed:
                    pid_stat_info_labels = generate_pid_stat_info_labels(
                        prev_pid_stat_bsf
                    )
                    metrics.append(
                        f"{PROC_PID_STAT_INFO_METRIC}{{{common_labels},{pid_stat_info_labels}}} 0 {curr_prom_ts}"
                    )
            if pid_stat_full_metrics or has_changed:
                pid_stat_info_labels = generate_pid_stat_info_labels(curr_pid_stat_bsf)
                metrics.append(
                    f"{PROC_PID_STAT_INFO_METRIC}{{{common_labels},{pid_stat_info_labels}}} 1 {curr_prom_ts}"
                )
            ### PROC_PID_STAT_(VSIZE|RSS*)_METRIC:
            for index, metric_name in [
                (procfs.PID_STAT_VSIZE, PROC_PID_STAT_VSIZE_METRIC),
                (procfs.PID_STAT_RSS, PROC_PID_STAT_RSS_METRIC),
                (procfs.PID_STAT_RSSLIM, PROC_PID_STAT_RSSLIM_METRIC),
            ]:
                crt_val = curr_pid_stat_bsf[index]
                if (
                    pid_stat_full_metrics
                    or has_prev
                    and crt_val != prev_pid_stat_bsf[index]
                ):
                    metrics.append(
                        f"{PROC_PID_STAT_INFO_METRIC}{{{common_labels}}} {crt_val} {curr_prom_ts}"
                    )

    # /proc/PID/status:
    if pid_parser_data.PidStatus is not None:
        curr_pid_status_bsf = pid_parser_data.PidStatus.ByteSliceFields
        curr_pid_status_bsu = pid_parser_data.PidStatus.ByteSliceFieldUnit
        curr_pid_status_nf = pid_parser_data.PidStat.NumericFields
        has_prev = (
            pid_metrics_info_data is not None
            and pid_metrics_info_data.PidStatusData is not None
        )
        pid_status_full_metrics = full_metrics or not has_prev
        if has_prev:
            prev_pid_status_bsf = pid_metrics_info_data.PidStatus.ByteSliceFields
            prev_pid_status_nf = pid_metrics_info_data.PidStat.NumericFields
        else:
            prev_pid_status_bsf = None
            prev_pid_status_nf = None
        ## PID+TID:
        if has_prev:
            ### PROC_PID_STATUS_*_CTXT_SWITCHES_METRIC:
            for index, metric_name, zd_index in [
                (
                    procfs.PID_STATUS_VOLUNTARY_CTXT_SWITCHES,
                    PROC_PID_STATUS_VOLUNTARY_CTXT_SWITCHES_METRIC,
                    PID_STATUS_VOLUNTARY_CTXT_SWITCHES_ZERO_DELTA_INDEX,
                ),
                (
                    procfs.PID_STATUS_NONVOLUNTARY_CTXT_SWITCHES,
                    PROC_PID_STATUS_NONVOLUNTARY_CTXT_SWITCHES_METRIC,
                    PID_STATUS_NONVOLUNTARY_CTXT_SWITCHES_ZERO_DELTA_INDEX,
                ),
            ]:
                delta = uint64_delta(
                    curr_pid_status_nf[index], prev_pid_status_nf[index]
                )
                if (
                    delta != 0
                    or pid_status_full_metrics
                    or not pid_metrics_info_data.PidStatusCtxZeroDelta[zd_index]
                ):
                    metrics.append(
                        f"{metric_name}{{{common_labels}}} {delta} {curr_prom_ts}"
                    )
                want_zero_delta.PidStatusCtxZeroDelta[zd_index] = delta == 0

        ### PROC_PID_STATUS_VM_*_METRIC, mix of PID+TID and PID only:
        for index, metric_name in [
            (procfs.PID_STATUS_VM_PEAK, PROC_PID_STATUS_VM_PEAK_METRIC),
            (procfs.PID_STATUS_VM_SIZE, PROC_PID_STATUS_VM_SIZE_METRIC),
            (procfs.PID_STATUS_VM_LCK, PROC_PID_STATUS_VM_LCK_METRIC),
            (procfs.PID_STATUS_VM_PIN, PROC_PID_STATUS_VM_PIN_METRIC),
            (procfs.PID_STATUS_VM_HWM, PROC_PID_STATUS_VM_HWM_METRIC),
            (procfs.PID_STATUS_VM_RSS, PROC_PID_STATUS_VM_RSS_METRIC),
            (procfs.PID_STATUS_RSS_ANON, PROC_PID_STATUS_RSS_ANON_METRIC),
            (procfs.PID_STATUS_RSS_FILE, PROC_PID_STATUS_RSS_FILE_METRIC),
            (procfs.PID_STATUS_RSS_SHMEM, PROC_PID_STATUS_RSS_SHMEM_METRIC),
            (procfs.PID_STATUS_VM_DATA, PROC_PID_STATUS_VM_DATA_METRIC),
            (procfs.PID_STATUS_VM_STK, PROC_PID_STATUS_VM_STK_METRIC),
            (procfs.PID_STATUS_VM_EXE, PROC_PID_STATUS_VM_EXE_METRIC),
            (procfs.PID_STATUS_VM_LIB, PROC_PID_STATUS_VM_LIB_METRIC),
            (procfs.PID_STATUS_VM_PTE, PROC_PID_STATUS_VM_PTE_METRIC),
            (procfs.PID_STATUS_VM_PMD, PROC_PID_STATUS_VM_PMD_METRIC),
            (procfs.PID_STATUS_VM_SWAP, PROC_PID_STATUS_VM_SWAP_METRIC),
            (procfs.PID_STATUS_HUGETLBPAGES, PROC_PID_STATUS_HUGETLBPAGES_METRIC),
        ]:
            if metric_name in proc_pid_status_pid_tid_vm_metrics or is_pid:
                crt_val, unit = curr_pid_status_bsf[index], curr_pid_status_bsu[index]
                if (
                    pid_status_full_metrics
                    or has_prev
                    and crt_val != prev_pid_status_bsf[index]
                ):
                    metrics.append(
                        f'{metric_name}{{{common_labels},{PROC_PID_STATUS_VM_UNIT_LABEL_NAME}="{unit}"}} {crt_val} {curr_prom_ts}'
                    )

        ## PID only:
        if is_pid:
            ### PROC_PID_STATUS_INFO_METRIC:
            has_changed = False
            if has_prev:
                for i in pid_status_info_index_to_label_map:
                    has_changed = curr_pid_status_bsf[i] != prev_pid_status_bsf[i]
                    if has_changed:
                        break
                if has_changed:
                    pid_status_info_labels = generate_pid_status_info_labels(
                        prev_pid_status_bsf
                    )
                    metrics.append(
                        f"{PROC_PID_STATUS_INFO_METRIC}{{{common_labels},{pid_status_info_labels}}} 0 {curr_prom_ts}"
                    )
            if pid_status_full_metrics or has_changed:
                pid_status_info_labels = generate_pid_status_info_labels(
                    curr_pid_status_bsf
                )
                metrics.append(
                    f"{PROC_PID_STATUS_INFO_METRIC}{{{common_labels},{pid_status_info_labels}}} 1 {curr_prom_ts}"
                )

    # /proc/PID/cmdline:
    if pid_parser_data.PidCmdline is not None:
        if is_pid and full_metrics:
            metrics.append(
                f'{PROC_PID_CMDLINE_METRIC}{{{common_labels},{PROC_PID_CMDLINE_LABEL_NAME}="{pid_parser_data.PidCmdline.Cmdline}"}} 1 {curr_prom_ts}'
            )

    return metrics, want_zero_delta


def get_pid_tid_variants(pid_tid: Optional[procfs.PidTid] = None) -> Tuple:
    if pid_tid is None:
        return "", 0, ""
    str_suffix = f"-{pid_tid.Pid}"
    num_offset = pid_tid.Pid
    if pid_tid.Tid > 0:
        str_suffix += f"-{pid_tid.Tid}"
        num_offset = num_offset * 57 + pid_tid.Tid
    return str_suffix, num_offset, str(num_offset)


def make_ref_proc_pid_stat(
    pid_tid: Optional[procfs.PidTid] = None,
) -> TestPidStatParsedData:
    str_suffix, num_offset, num_suffix = get_pid_tid_variants(pid_tid)

    pid_stat_bsf = [""] * procfs.PID_STAT_BYTE_SLICE_NUM_FIELDS
    pid_stat_bsf[procfs.PID_STAT_COMM] = "COMM" + str_suffix
    pid_stat_bsf[procfs.PID_STAT_STATE] = "STATE" + str_suffix
    pid_stat_bsf[procfs.PID_STAT_PPID] = "PPID" + str_suffix
    pid_stat_bsf[procfs.PID_STAT_PGRP] = "PGRP" + str_suffix
    pid_stat_bsf[procfs.PID_STAT_SESSION] = "SESSION" + str_suffix
    pid_stat_bsf[procfs.PID_STAT_TTY_NR] = "TTY_NR" + str_suffix
    pid_stat_bsf[procfs.PID_STAT_TPGID] = "TPGID" + str_suffix
    pid_stat_bsf[procfs.PID_STAT_FLAGS] = "FLAGS" + str_suffix
    pid_stat_bsf[procfs.PID_STAT_PRIORITY] = "PRIORITY" + str_suffix
    pid_stat_bsf[procfs.PID_STAT_NICE] = f"10{num_suffix}"
    pid_stat_bsf[procfs.PID_STAT_NUM_THREADS] = f"7{num_suffix}"
    pid_stat_bsf[procfs.PID_STAT_STARTTIME] = f"33{num_suffix}"
    pid_stat_bsf[procfs.PID_STAT_VSIZE] = f"1000{num_suffix}"
    pid_stat_bsf[procfs.PID_STAT_RSS] = f"2000{num_suffix}"
    pid_stat_bsf[procfs.PID_STAT_RSSLIM] = f"100000{num_suffix}"
    pid_stat_bsf[procfs.PID_STAT_PROCESSOR] = f"13{num_suffix}"
    pid_stat_bsf[procfs.PID_STAT_RT_PRIORITY] = f"0{num_suffix}"
    pid_stat_bsf[procfs.PID_STAT_POLICY] = "POLICY" + str_suffix

    pid_stat_nf = [0] * procfs.PID_STAT_ULONG_FIELD_NUM_FIELDS
    pid_stat_nf[procfs.PID_STAT_MINFLT] = 1000 + num_offset
    pid_stat_nf[procfs.PID_STAT_MAJFLT] = 100 + num_offset
    pid_stat_nf[procfs.PID_STAT_UTIME] = 0 + num_offset
    pid_stat_nf[procfs.PID_STAT_STIME] = 0 + num_offset

    return TestPidStatParsedData(
        ByteSliceFields=pid_stat_bsf,
        NumericFields=pid_stat_nf,
    )


def make_prev_proc_pid_stat(
    curr_pid_stat: TestPidStatParsedData,
    bsf_indexes: Optional[List[int]] = None,  # [] stands for all
    nf_indexes: Optional[List[int]] = None,  # [] stands for all
    target_pcpu: float = 12.3,
    interval: float = DEFAULT_PROC_PID_INTERVAL_SEC,
    linux_clktck_sec: float = TEST_LINUX_CLKTCK_SEC,
) -> TestPidStatParsedData:
    alt_pid_stat = deepcopy(curr_pid_stat)

    if bsf_indexes is not None:
        bsf_fields = alt_pid_stat.ByteSliceFields
        if not bsf_indexes:
            bsf_indexes = range(procfs.PID_STAT_BYTE_SLICE_NUM_FIELDS)
        for index in bsf_indexes:
            try:
                num = int(bsf_fields[index])
                bsf_fields[index] = str(num + 13 * (index + 1))
            except Exception:
                bsf_fields[index] += "_"
    if nf_indexes is not None:
        nf_fields = alt_pid_stat.NumericFields
        if not nf_indexes:
            nf_indexes = range(procfs.PID_STAT_ULONG_FIELD_NUM_FIELDS)
        for index in nf_indexes:
            val = nf_fields[index]
            if index in {procfs.PID_STAT_UTIME, procfs.PID_STAT_STIME}:
                delta = int(target_pcpu / 100 * interval / linux_clktck_sec)
            else:
                delta = 2 * (val + index + 1)
            nf_fields[index] = uint64_delta(val, delta)
    return alt_pid_stat


def make_ref_proc_pid_cmdline() -> TestPidCmdlineParsedData:
    return TestPidCmdlineParsedData(Cmdline="/pa/th/to/exec arg1 arg2 arg3=val3")


"""

from lsvmi import proc_pid_metrics as m
curr_pid_stat = m.make_ref_proc_pid_stat()
pid_tid = m.procfs.PidTid(10)
pid_parser_data = m.TestPidParserData(PidStat=curr_pid_stat, PidTid=pid_tid)
metrics, want_zero_delta = m.generate_proc_pid_metrics(pid_parser_data, 13)

"""
