#! /usr/bin/env python3

import os
import time
from collections import OrderedDict
from copy import deepcopy
from dataclasses import dataclass, field
from typing import Generator, List, Optional, Tuple

import procfs

from . import (
    DEFAULT_TEST_HOSTNAME,
    DEFAULT_TEST_INSTANCE,
    HOSTNAME_LABEL_NAME,
    INSTANCE_LABEL_NAME,
    TEST_BOOTTIME_SEC,
    TEST_LINUX_CLKTCK_SEC,
    TEST_OS_PAGE_SIZE,
    lsvmi_test_cases_root_dir,
    save_test_cases,
    uint64_delta,
)

DEFAULT_BOOTTIME_MSEC = TEST_BOOTTIME_SEC * 1000
DEFAULT_PROCFS_ROOT = "/PROC_PID_METRICS_PROCFS"

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

PROC_PID_STAT_COMM_METRIC = "proc_pid_stat_comm"
PROC_PID_STAT_COMM_LABEL_NAME = "comm"
PROC_PID_STAT_STARTTIME_LABEL_NAME = "starttime_msec"

PROC_PID_STAT_INFO_METRIC = "proc_pid_stat_info"  # PID only
PROC_PID_STAT_PPID_LABEL_NAME = "ppid"
PROC_PID_STAT_PGRP_LABEL_NAME = "pgrp"
PROC_PID_STAT_SESSION_LABEL_NAME = "session"
PROC_PID_STAT_TTY_NR_LABEL_NAME = "tty"
PROC_PID_STAT_TPGID_LABEL_NAME = "tpgid"
PROC_PID_STAT_FLAGS_LABEL_NAME = "flags"

PROC_PID_STAT_PRIORITY_METRIC = "proc_pid_stat_prio"  # PID + TID
PROC_PID_STAT_PRIORITY_LABEL_NAME = "prio"
PROC_PID_STAT_NICE_LABEL_NAME = "nice"
PROC_PID_STAT_RT_PRIORITY_LABEL_NAME = "rt_prio"
PROC_PID_STAT_POLICY_LABEL_NAME = "policy"

PROC_PID_STAT_VSIZE_METRIC = "proc_pid_stat_vsize_bytes"  # PID only
PROC_PID_STAT_RSS_METRIC = "proc_pid_stat_rss_bytes"  # PID only
PROC_PID_STAT_RSSLIM_METRIC = "proc_pid_stat_rsslim_bytes"  # PID only

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
PROC_PID_CMDLINE_CMD_PATH_LABEL_NAME = "cmd_path"
PROC_PID_CMDLINE_CMD_LABEL_NAME = "cmd"
PROC_PID_CMDLINE_ARGS_LABEL_NAME = "args"

# This generator's specific metrics, i.e. in addition to those described in
# metrics_common.go:

# They all have the following label:
PROC_PID_PART_LABEL_NAME = "part"  # partition

# PID counts:
PROC_PID_TOTAL_COUNT_METRIC = "proc_pid_total_count"
PROC_PID_PARSE_OK_COUNT_METRIC = "proc_pid_parse_ok_count"
PROC_PID_PARSE_ERR_COUNT_METRIC = "proc_pid_parse_err_count"
PROC_PID_ACTIVE_COUNT_METRIC = "proc_pid_active_count"
PROC_PID_NEW_COUNT_METRIC = "proc_pid_new_count"
PROC_PID_DEL_COUNT_METRIC = "proc_pid_del_count"

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
    CmdPath: Optional[str] = None
    Args: Optional[str] = None


@dataclass
class TestPidParserStateData:
    PidStat: Optional[TestPidStatParsedData] = None
    PidStatus: Optional[TestPidStatusParsedData] = None
    PidCmdline: Optional[TestPidCmdlineParsedData] = None
    UnixMilli: Optional[int] = None
    Active: bool = False
    PidStatFltZeroDelta: Optional[List[bool]] = field(
        default_factory=lambda: [False] * PID_STAT_FLT_ZERO_DELTA_SIZE
    )
    PidStatusCtxZeroDelta: Optional[List[bool]] = field(
        default_factory=lambda: [False] * PID_STATUS_CTX_ZERO_DELTA_SIZE
    )
    CycleNum: int = 0
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
pm_generate_test_cases_file = "proc_pid_metrics_generate.json"
pm_execute_test_cases_file = "proc_pid_metrics_execute.json"


@dataclass
class ProcPidMetricsGenerateTestCase:
    Name: Optional[str] = None
    Description: Optional[str] = None
    ProcfsRoot: Optional[str] = DEFAULT_PROCFS_ROOT

    PageSize: int = TEST_OS_PAGE_SIZE

    Instance: Optional[str] = DEFAULT_TEST_INSTANCE
    Hostname: Optional[str] = DEFAULT_TEST_HOSTNAME
    LinuxClktckSec: float = TEST_LINUX_CLKTCK_SEC
    BoottimeMsec: int = DEFAULT_BOOTTIME_MSEC

    PidTidMetricsInfo: Optional[TestPidParserStateData] = None
    ParserData: Optional[TestPidParserStateData] = None
    FullMetrics: bool = False

    WantMetricsCount: int = 0
    WantMetrics: Optional[str] = None
    ReportExtra: bool = True
    WantZeroDelta: Optional[TestPidParserStateData] = None


@dataclass
class ProcPidMetricsExecuteTestCase:
    Name: Optional[str] = None
    Description: Optional[str] = None

    PartNo: int = 0
    FullMetricsFactor: int = 15
    UsePidStatus: bool = False
    ScanNum: int = 0

    PageSize: int = TEST_OS_PAGE_SIZE

    Instance: Optional[str] = DEFAULT_TEST_INSTANCE
    Hostname: Optional[str] = DEFAULT_TEST_HOSTNAME
    LinuxClktckSec: float = TEST_LINUX_CLKTCK_SEC
    BoottimeMsec: int = DEFAULT_BOOTTIME_MSEC

    PidTidListResult: Optional[List[procfs.PidTid]] = None
    PidTidMetricsInfoList: Optional[List[TestPidParserStateData]] = None
    PidParsersDataList: Optional[List[TestPidParserStateData]] = None

    CurrUnixMilli: int = 0
    PrevUnixMilli: int = 0

    WantMetricsCount: int = 0
    WantMetrics: Optional[str] = None
    ReportExtra: bool = True
    WantZeroDeltaList: Optional[List[TestPidParserStateData]] = None


# Use an ordered dict to match the expected label order:
pid_stat_info_index_to_label_map = OrderedDict(
    [
        (procfs.PID_STAT_PPID, PROC_PID_STAT_PPID_LABEL_NAME),
        (procfs.PID_STAT_PGRP, PROC_PID_STAT_PGRP_LABEL_NAME),
        (procfs.PID_STAT_SESSION, PROC_PID_STAT_SESSION_LABEL_NAME),
        (procfs.PID_STAT_TTY_NR, PROC_PID_STAT_TTY_NR_LABEL_NAME),
        (procfs.PID_STAT_TPGID, PROC_PID_STAT_TPGID_LABEL_NAME),
        (procfs.PID_STAT_FLAGS, PROC_PID_STAT_FLAGS_LABEL_NAME),
    ]
)

pid_stat_priority_index_to_label_map = OrderedDict(
    [
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


def generate_pid_stat_priority_labels(pid_stat_bsf: List[str]) -> str:
    return ",".join(
        [
            f'{label}="{pid_stat_bsf[index]}"'
            for index, label in pid_stat_priority_index_to_label_map.items()
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
    pid_parser_data: TestPidParserStateData,
    pid_metrics_info_data: Optional[TestPidParserStateData] = None,  # i.e. no prev
    full_metrics: bool = False,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    boottime_msec: int = DEFAULT_BOOTTIME_MSEC,
    linux_clktck_sec: float = TEST_LINUX_CLKTCK_SEC,
    page_size: int = TEST_OS_PAGE_SIZE,
) -> Tuple[List[str], TestPidParserStateData]:
    metrics = []
    want_zero_delta = TestPidParserStateData(
        PidTid=pid_parser_data.PidTid,
    )

    curr_prom_ts = pid_parser_data.UnixMilli

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
            and pid_metrics_info_data.PidStat is not None
        )
        pid_stat_full_metrics = full_metrics or not has_prev
        curr_pid_stat_bsf = pid_parser_data.PidStat.ByteSliceFields
        curr_pid_stat_nf = pid_parser_data.PidStat.NumericFields
        if has_prev:
            prev_pid_stat_bsf = pid_metrics_info_data.PidStat.ByteSliceFields
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
        changed = (
            prev_pid_stat_bsf is not None
            and curr_pid_stat_bsf[procfs.PID_STAT_STATE]
            != prev_pid_stat_bsf[procfs.PID_STAT_STATE]
        )
        if changed:
            metrics.append(
                f"{PROC_PID_STAT_STATE_METRIC}{{"
                + ",".join(
                    [
                        common_labels,
                        f'{PROC_PID_STAT_STATE_LABEL_NAME}="{prev_pid_stat_bsf[procfs.PID_STAT_STATE]}"',
                    ]
                )
                + f"}} 0 {curr_prom_ts}"
            )
        if pid_stat_full_metrics or changed:
            metrics.append(
                f"{PROC_PID_STAT_STATE_METRIC}{{"
                + ",".join(
                    [
                        common_labels,
                        f'{PROC_PID_STAT_STATE_LABEL_NAME}="{curr_pid_stat_bsf[procfs.PID_STAT_STATE]}"',
                    ]
                )
                + f"}} 1 {curr_prom_ts}"
            )

        ### PROC_PID_STAT_COMM_METRIC:
        comm_changed = (
            prev_pid_stat_bsf is not None
            and curr_pid_stat_bsf[procfs.PID_STAT_COMM]
            != prev_pid_stat_bsf[procfs.PID_STAT_COMM]
        )
        if comm_changed:
            metrics.append(
                f"{PROC_PID_STAT_COMM_METRIC}{{"
                + ",".join(
                    [
                        common_labels,
                        f'{PROC_PID_STAT_STARTTIME_LABEL_NAME}="{starttime_msec}"',
                        f'{PROC_PID_STAT_COMM_LABEL_NAME}="{prev_pid_stat_bsf[procfs.PID_STAT_COMM]}"',
                    ]
                )
                + f"}} 0 {curr_prom_ts}"
            )
        if pid_stat_full_metrics or comm_changed:
            metrics.append(
                f"{PROC_PID_STAT_COMM_METRIC}{{"
                + ",".join(
                    [
                        common_labels,
                        f'{PROC_PID_STAT_STARTTIME_LABEL_NAME}="{starttime_msec}"',
                        f'{PROC_PID_STAT_COMM_LABEL_NAME}="{curr_pid_stat_bsf[procfs.PID_STAT_COMM]}"',
                    ]
                )
                + f"}} 1 {curr_prom_ts}"
            )

        ### PROC_PID_STAT_PRIORITY_METRIC:
        changed = False
        if has_prev:
            for i in pid_stat_priority_index_to_label_map:
                changed = curr_pid_stat_bsf[i] != prev_pid_stat_bsf[i]
                if changed:
                    break
            if changed:
                pid_stat_priority_labels = generate_pid_stat_priority_labels(
                    prev_pid_stat_bsf
                )
                metrics.append(
                    f"{PROC_PID_STAT_PRIORITY_METRIC}{{{common_labels},{pid_stat_priority_labels}}} 0 {curr_prom_ts}"
                )
        if pid_stat_full_metrics or changed:
            pid_stat_priority_labels = generate_pid_stat_priority_labels(
                curr_pid_stat_bsf
            )
            metrics.append(
                f"{PROC_PID_STAT_PRIORITY_METRIC}{{{common_labels},{pid_stat_priority_labels}}} 1 {curr_prom_ts}"
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
            pcpu_factor = (
                linux_clktck_sec
                / ((curr_prom_ts - pid_metrics_info_data.UnixMilli) / 1000)
                * 100
            )
            total_delta_ticks = 0
            for index, metric_name in [
                (procfs.PID_STAT_UTIME, PROC_PID_STAT_UTIME_PCT_METRIC),
                (procfs.PID_STAT_STIME, PROC_PID_STAT_STIME_PCT_METRIC),
            ]:
                delta_ticks = uint64_delta(
                    curr_pid_stat_nf[index], prev_pid_stat_nf[index]
                )
                total_delta_ticks += delta_ticks
                metrics.append(
                    f"{metric_name}{{{common_labels}}} {delta_ticks*pcpu_factor:.1f} {curr_prom_ts}"
                )
            metrics.append(
                f"{PROC_PID_STAT_TIME_PCT_METRIC}{{{common_labels}}} {total_delta_ticks*pcpu_factor:.1f} {curr_prom_ts}"
            )

        ### PROC_PID_STAT_CPU_NUM_METRIC:
        metrics.append(
            f"{PROC_PID_STAT_CPU_NUM_METRIC}{{{common_labels}}} {curr_pid_stat_bsf[procfs.PID_STAT_PROCESSOR]} {curr_prom_ts}"
        )

        ## PID only:
        if is_pid:
            ### PROC_PID_STAT_INFO_METRIC:
            changed = False
            if has_prev:
                for i in pid_stat_info_index_to_label_map:
                    changed = curr_pid_stat_bsf[i] != prev_pid_stat_bsf[i]
                    if changed:
                        break
                if changed:
                    pid_stat_info_labels = generate_pid_stat_info_labels(
                        prev_pid_stat_bsf
                    )
                    metrics.append(
                        f"{PROC_PID_STAT_INFO_METRIC}{{{common_labels},{pid_stat_info_labels}}} 0 {curr_prom_ts}"
                    )
            if pid_stat_full_metrics or changed:
                pid_stat_info_labels = generate_pid_stat_info_labels(curr_pid_stat_bsf)
                metrics.append(
                    f"{PROC_PID_STAT_INFO_METRIC}{{{common_labels},{pid_stat_info_labels}}} 1 {curr_prom_ts}"
                )
            #### PROC_PID_STAT_RSS_METRIC:
            crt_val = curr_pid_stat_nf[procfs.PID_STAT_RSS]
            if (
                pid_stat_full_metrics
                or has_prev
                and crt_val != prev_pid_stat_nf[procfs.PID_STAT_RSS]
            ):
                metrics.append(
                    f"{PROC_PID_STAT_RSS_METRIC}{{{common_labels}}} {crt_val*page_size} {curr_prom_ts}"
                )

            ### PROC_PID_STAT_(VSIZE|RSSLIM)_METRIC:
            for index, metric_name in [
                (procfs.PID_STAT_VSIZE, PROC_PID_STAT_VSIZE_METRIC),
                (procfs.PID_STAT_RSSLIM, PROC_PID_STAT_RSSLIM_METRIC),
            ]:
                crt_val = curr_pid_stat_bsf[index]
                if (
                    pid_stat_full_metrics
                    or has_prev
                    and crt_val != prev_pid_stat_bsf[index]
                ):
                    metrics.append(
                        f"{metric_name}{{{common_labels}}} {crt_val} {curr_prom_ts}"
                    )

    # /proc/PID/status:
    if pid_parser_data.PidStatus is not None:
        curr_pid_status_bsf = pid_parser_data.PidStatus.ByteSliceFields
        curr_pid_status_bsu = pid_parser_data.PidStatus.ByteSliceFieldUnit
        curr_pid_status_nf = pid_parser_data.PidStatus.NumericFields
        has_prev = (
            pid_metrics_info_data is not None
            and pid_metrics_info_data.PidStatus is not None
        )
        pid_status_full_metrics = full_metrics or not has_prev
        if has_prev:
            prev_pid_status_bsf = pid_metrics_info_data.PidStatus.ByteSliceFields
            prev_pid_status_nf = pid_metrics_info_data.PidStatus.NumericFields
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
            changed = False
            if has_prev:
                for i in pid_status_info_index_to_label_map:
                    changed = curr_pid_status_bsf[i] != prev_pid_status_bsf[i]
                    if changed:
                        break
                if changed:
                    pid_status_info_labels = generate_pid_status_info_labels(
                        prev_pid_status_bsf
                    )
                    metrics.append(
                        f"{PROC_PID_STATUS_INFO_METRIC}{{{common_labels},{pid_status_info_labels}}} 0 {curr_prom_ts}"
                    )
            if pid_status_full_metrics or changed:
                pid_status_info_labels = generate_pid_status_info_labels(
                    curr_pid_status_bsf
                )
                metrics.append(
                    f"{PROC_PID_STATUS_INFO_METRIC}{{{common_labels},{pid_status_info_labels}}} 1 {curr_prom_ts}"
                )

    # /proc/PID/cmdline:
    if (
        pid_parser_data.PidCmdline is not None
        and is_pid
        and (full_metrics or pid_metrics_info_data is None)
    ):
        cmd_path = pid_parser_data.PidCmdline.CmdPath or ""
        if cmd_path != "":
            cmd = os.path.basename(cmd_path)
            args = pid_parser_data.PidCmdline.Args or ""
            metrics.append(
                f"{PROC_PID_CMDLINE_METRIC}{{{common_labels},"
                + ",".join(
                    [
                        f'{PROC_PID_CMDLINE_CMD_PATH_LABEL_NAME}="{cmd_path}"',
                        f'{PROC_PID_CMDLINE_ARGS_LABEL_NAME}="{args}"',
                        f'{PROC_PID_CMDLINE_CMD_LABEL_NAME}="{cmd}"',
                    ]
                )
                + f"}} 1 {curr_prom_ts}"
            )
        elif pid_parser_data.PidStat is not None:
            # Fallback on comm:
            if comm_changed:
                metrics.append(
                    f"{PROC_PID_CMDLINE_METRIC}{{{common_labels},"
                    + f'{PROC_PID_CMDLINE_CMD_LABEL_NAME}="[{prev_pid_stat_bsf[procfs.PID_STAT_COMM]}]"'
                    + f"}} 0 {curr_prom_ts}",
                )
            metrics.append(
                f"{PROC_PID_CMDLINE_METRIC}{{{common_labels},"
                + f'{PROC_PID_CMDLINE_CMD_LABEL_NAME}="[{curr_pid_stat_bsf[procfs.PID_STAT_COMM]}]"'
                + f"}} 1 {curr_prom_ts}",
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
    pid_stat_bsf[procfs.PID_STAT_RSSLIM] = f"100000{num_suffix}"
    pid_stat_bsf[procfs.PID_STAT_PROCESSOR] = f"13{num_suffix}"
    pid_stat_bsf[procfs.PID_STAT_RT_PRIORITY] = f"0{num_suffix}"
    pid_stat_bsf[procfs.PID_STAT_POLICY] = "POLICY" + str_suffix

    pid_stat_nf = [0] * procfs.PID_STAT_ULONG_NUM_FIELDS
    pid_stat_nf[procfs.PID_STAT_MINFLT] = 1000 + num_offset
    pid_stat_nf[procfs.PID_STAT_MAJFLT] = 100 + num_offset
    pid_stat_nf[procfs.PID_STAT_UTIME] = 0 + num_offset
    pid_stat_nf[procfs.PID_STAT_STIME] = 0 + num_offset
    pid_stat_nf[procfs.PID_STAT_RSS] = 2000 + num_offset

    return TestPidStatParsedData(
        ByteSliceFields=pid_stat_bsf,
        NumericFields=pid_stat_nf,
    )


ALL_INDEXES = object()


def make_prev_proc_pid_stat(
    curr_pid_stat: TestPidStatParsedData,
    bsf_indexes: Optional[List[int]] = None,
    nf_indexes: Optional[List[int]] = None,
    target_utime_pcpu: float = 12,
    target_stime_pcpu: float = 13,
    interval: float = DEFAULT_PROC_PID_INTERVAL_SEC,
    linux_clktck_sec: float = TEST_LINUX_CLKTCK_SEC,
) -> TestPidStatParsedData:
    alt_pid_stat = deepcopy(curr_pid_stat)

    if bsf_indexes is not None:
        bsf_fields = alt_pid_stat.ByteSliceFields
        if bsf_indexes is ALL_INDEXES:
            bsf_indexes = range(procfs.PID_STAT_BYTE_SLICE_NUM_FIELDS)
        for index in bsf_indexes:
            if index in {procfs.PID_STAT_STARTTIME}:
                continue
            try:
                num = int(bsf_fields[index])
                bsf_fields[index] = str(num + 13 * (index + 1))
            except Exception:
                bsf_fields[index] += "_"
    if nf_indexes is not None:
        nf_fields = alt_pid_stat.NumericFields
        if nf_indexes is ALL_INDEXES:
            nf_indexes = range(procfs.PID_STAT_ULONG_NUM_FIELDS)
        for index in nf_indexes:
            val = nf_fields[index]
            if index == procfs.PID_STAT_UTIME:
                delta = int(target_utime_pcpu / 100 * interval / linux_clktck_sec)
            elif index == procfs.PID_STAT_STIME:
                delta = int(target_stime_pcpu / 100 * interval / linux_clktck_sec)
            else:
                delta = 2 * (val + index + 1)
            nf_fields[index] = uint64_delta(val, delta)
    return alt_pid_stat


def make_ref_proc_pid_status(
    pid_tid: Optional[procfs.PidTid] = None,
) -> TestPidStatParsedData:
    str_suffix, num_offset, num_suffix = get_pid_tid_variants(pid_tid)

    pid_status_bsf = [""] * procfs.PID_STATUS_BYTE_SLICE_NUM_FIELDS
    pid_status_bsf[procfs.PID_STATUS_UID] = f"UID{str_suffix}"
    pid_status_bsf[procfs.PID_STATUS_GID] = f"GID{str_suffix}"
    pid_status_bsf[procfs.PID_STATUS_GROUPS] = f"GID{str_suffix}"
    pid_status_bsf[procfs.PID_STATUS_VM_PEAK] = f"113{num_suffix}"
    pid_status_bsf[procfs.PID_STATUS_VM_SIZE] = f"114{num_suffix}"
    pid_status_bsf[procfs.PID_STATUS_VM_LCK] = f"115{num_suffix}"
    pid_status_bsf[procfs.PID_STATUS_VM_PIN] = f"116{num_suffix}"
    pid_status_bsf[procfs.PID_STATUS_VM_HWM] = f"117{num_suffix}"
    pid_status_bsf[procfs.PID_STATUS_VM_RSS] = f"118{num_suffix}"
    pid_status_bsf[procfs.PID_STATUS_RSS_ANON] = f"119{num_suffix}"
    pid_status_bsf[procfs.PID_STATUS_RSS_FILE] = f"120{num_suffix}"
    pid_status_bsf[procfs.PID_STATUS_RSS_SHMEM] = f"121{num_suffix}"
    pid_status_bsf[procfs.PID_STATUS_VM_DATA] = f"122{num_suffix}"
    pid_status_bsf[procfs.PID_STATUS_VM_STK] = f"123{num_suffix}"
    pid_status_bsf[procfs.PID_STATUS_VM_EXE] = f"124{num_suffix}"
    pid_status_bsf[procfs.PID_STATUS_VM_LIB] = f"125{num_suffix}"
    pid_status_bsf[procfs.PID_STATUS_VM_PTE] = f"126{num_suffix}"
    pid_status_bsf[procfs.PID_STATUS_VM_PMD] = f"127{num_suffix}"
    pid_status_bsf[procfs.PID_STATUS_VM_SWAP] = f"128{num_suffix}"
    pid_status_bsf[procfs.PID_STATUS_HUGETLBPAGES] = f"129{num_suffix}"
    pid_status_bsf[procfs.PID_STATUS_CPUS_ALLOWED_LIST] = f"CPUS_ALLOWED{str_suffix}"
    pid_status_bsf[procfs.PID_STATUS_MEMS_ALLOWED_LIST] = f"MEMS_ALLOWED{str_suffix}"

    pid_status_bsu = [""] * procfs.PID_STATUS_BYTE_SLICE_NUM_FIELDS
    pid_status_bsu[procfs.PID_STATUS_VM_PEAK] = f"VM_PEAK_UNIT{str_suffix}"
    pid_status_bsu[procfs.PID_STATUS_VM_SIZE] = f"VM_SIZE_UNIT{str_suffix}"
    pid_status_bsu[procfs.PID_STATUS_VM_LCK] = f"VM_LCK_UNIT{str_suffix}"
    pid_status_bsu[procfs.PID_STATUS_VM_PIN] = f"VM_PIN_UNIT{str_suffix}"
    pid_status_bsu[procfs.PID_STATUS_VM_HWM] = f"VM_HWM_UNIT{str_suffix}"
    pid_status_bsu[procfs.PID_STATUS_VM_RSS] = f"VM_RSS_UNIT{str_suffix}"
    pid_status_bsu[procfs.PID_STATUS_RSS_ANON] = f"RSS_ANON_UNIT{str_suffix}"
    pid_status_bsu[procfs.PID_STATUS_RSS_FILE] = f"RSS_FILE_UNIT{str_suffix}"
    pid_status_bsu[procfs.PID_STATUS_RSS_SHMEM] = f"RSS_SHMEM_UNIT{str_suffix}"
    pid_status_bsu[procfs.PID_STATUS_VM_DATA] = f"VM_DATA_UNIT{str_suffix}"
    pid_status_bsu[procfs.PID_STATUS_VM_STK] = f"VM_STK_UNIT{str_suffix}"
    pid_status_bsu[procfs.PID_STATUS_VM_EXE] = f"VM_EXE_UNIT{str_suffix}"
    pid_status_bsu[procfs.PID_STATUS_VM_LIB] = f"VM_LIB_UNIT{str_suffix}"
    pid_status_bsu[procfs.PID_STATUS_VM_PTE] = f"VM_PTE_UNIT{str_suffix}"
    pid_status_bsu[procfs.PID_STATUS_VM_PMD] = f"VM_PMD_UNIT{str_suffix}"
    pid_status_bsu[procfs.PID_STATUS_VM_SWAP] = f"VM_SWAP_UNIT{str_suffix}"
    pid_status_bsu[procfs.PID_STATUS_HUGETLBPAGES] = f"HUGETLBPAGES_UNIT{num_suffix}"

    pid_status_nf = [0] * procfs.PID_STATUS_ULONG_NUM_FIELDS
    pid_status_nf[procfs.PID_STATUS_VOLUNTARY_CTXT_SWITCHES] = 1021 + num_offset
    pid_status_nf[procfs.PID_STATUS_NONVOLUNTARY_CTXT_SWITCHES] = 1022 + num_offset

    return TestPidStatusParsedData(
        ByteSliceFields=pid_status_bsf,
        ByteSliceFieldUnit=pid_status_bsu,
        NumericFields=pid_status_nf,
    )


def make_prev_proc_pid_status(
    curr_pid_status: TestPidStatusParsedData,
    bsf_indexes: Optional[List[int]] = None,
    nf_indexes: Optional[List[int]] = None,
) -> TestPidStatusParsedData:
    alt_pid_status = deepcopy(curr_pid_status)

    if bsf_indexes is not None:
        bsf_fields = alt_pid_status.ByteSliceFields
        if bsf_indexes is ALL_INDEXES:
            bsf_indexes = range(procfs.PID_STATUS_BYTE_SLICE_NUM_FIELDS)
        for index in bsf_indexes:
            try:
                num = int(bsf_fields[index])
                bsf_fields[index] = str(num + 13 * (index + 1))
            except Exception:
                bsf_fields[index] += "_"
    if nf_indexes is not None:
        nf_fields = alt_pid_status.NumericFields
        if nf_indexes is ALL_INDEXES:
            nf_indexes = range(procfs.PID_STATUS_ULONG_NUM_FIELDS)
        for index in nf_indexes:
            val = nf_fields[index]
            delta = 2 * (val + index + 1)
            nf_fields[index] = uint64_delta(val, delta)
    return alt_pid_status


def make_ref_proc_pid_cmdline(
    pid_tid: Optional[procfs.PidTid] = None,
) -> TestPidCmdlineParsedData:
    cmd_path = "/pa/th/to/exec"
    args = []
    if pid_tid is not None:
        args.append(f"--pid={pid_tid.Pid}")
        if pid_tid.Tid > 0:
            args.append(f"--tid={pid_tid.Tid}")
    args.extend(["arg1", "arg2"])
    return TestPidCmdlineParsedData(CmdPath=cmd_path, Args=" ".join(args))


def generate_proc_pid_metrics_generate_test_case(
    name: str,
    pid_parser_data: TestPidParserStateData,
    pid_metrics_info_data: Optional[TestPidParserStateData] = None,  # i.e. no prev
    interval: float = DEFAULT_PROC_PID_INTERVAL_SEC,
    full_metrics: bool = False,
    procfs_root: str = DEFAULT_PROCFS_ROOT,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    boottime_msec: int = DEFAULT_BOOTTIME_MSEC,
    linux_clktck_sec: float = TEST_LINUX_CLKTCK_SEC,
    page_size: int = TEST_OS_PAGE_SIZE,
    description: Optional[str] = None,
) -> ProcPidMetricsGenerateTestCase:
    if pid_parser_data.UnixMilli is None:
        pid_parser_data = deepcopy(pid_parser_data)
        pid_parser_data.UnixMilli = int(time.time() * 1000)
        if pid_metrics_info_data is not None:
            pid_metrics_info_data = deepcopy(pid_metrics_info_data)
            pid_metrics_info_data.UnixMilli = pid_parser_data.UnixMilli - int(
                interval * 1000
            )
    elif pid_metrics_info_data is not None and pid_metrics_info_data.UnixMilli is None:
        pid_metrics_info_data = deepcopy(pid_metrics_info_data)
        pid_metrics_info_data.UnixMilli = pid_parser_data.UnixMilli - int(
            interval * 1000
        )
    metrics, want_zero_delta = generate_proc_pid_metrics(
        pid_parser_data,
        pid_metrics_info_data=pid_metrics_info_data,
        full_metrics=full_metrics,
        instance=instance,
        hostname=hostname,
        boottime_msec=boottime_msec,
        linux_clktck_sec=linux_clktck_sec,
    )
    # .generateMetrics does not change active flag, preserve the input:
    want_zero_delta.Active = (
        pid_metrics_info_data.Active if pid_metrics_info_data is not None else False
    )
    return ProcPidMetricsGenerateTestCase(
        Name=name,
        Description=description,
        ProcfsRoot=procfs_root,
        PageSize=page_size,
        Instance=instance,
        Hostname=hostname,
        BoottimeMsec=boottime_msec,
        PidTidMetricsInfo=pid_metrics_info_data,
        ParserData=pid_parser_data,
        FullMetrics=full_metrics,
        WantMetricsCount=len(metrics),
        WantMetrics=metrics,
        ReportExtra=True,
        WantZeroDelta=want_zero_delta,
    )


def generate_proc_pid_metrics_generate_test_cases(
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    test_cases_root_dir: Optional[str] = lsvmi_test_cases_root_dir,
):
    test_cases = []
    tc_num = 0

    # Initial metrics:
    curr_pid_stat = make_ref_proc_pid_stat()
    curr_pid_status = make_ref_proc_pid_status()
    curr_pid_cmdline = make_ref_proc_pid_cmdline()
    pid = 100
    for tid in [101, procfs.PID_ONLY_TID]:
        pid_tid = procfs.PidTid(Pid=pid, Tid=tid)
        for comm_fallback in [False, True]:
            if comm_fallback:
                pid_cmdline = TestPidCmdlineParsedData(Args="should be ignored")
            else:
                pid_cmdline = curr_pid_cmdline
            pid_parser_data = TestPidParserStateData(
                PidStat=curr_pid_stat,
                PidStatus=curr_pid_status,
                PidCmdline=pid_cmdline,
                PidTid=pid_tid,
            )
            for full_metrics in [False, True]:
                test_cases.append(
                    generate_proc_pid_metrics_generate_test_case(
                        f"initial/{tc_num:04d}",
                        pid_parser_data,
                        full_metrics=full_metrics,
                        instance=instance,
                        hostname=hostname,
                        description=f"is_pid={tid==procfs.PID_ONLY_TID}, comm_fallback={comm_fallback}, full_metrics={full_metrics}",
                    )
                )
                tc_num += 1

    # All changed:
    prev_pid_stat = make_prev_proc_pid_stat(
        curr_pid_stat,
        bsf_indexes=ALL_INDEXES,
        nf_indexes=ALL_INDEXES,
    )
    prev_pid_status = make_prev_proc_pid_status(
        curr_pid_status,
        bsf_indexes=ALL_INDEXES,
        nf_indexes=ALL_INDEXES,
    )
    for tid in [101, procfs.PID_ONLY_TID]:
        pid_tid = procfs.PidTid(Pid=pid, Tid=tid)
        for comm_fallback in [False, True]:
            if comm_fallback:
                pid_cmdline = TestPidCmdlineParsedData(Args="should be ignored")
            else:
                pid_cmdline = curr_pid_cmdline
            pid_parser_data = TestPidParserStateData(
                PidStat=curr_pid_stat,
                PidStatus=curr_pid_status,
                PidCmdline=pid_cmdline,
                PidTid=procfs.PidTid(Pid=pid, Tid=tid),
            )
            pid_metrics_info_data = TestPidParserStateData(
                PidStat=prev_pid_stat,
                PidStatus=prev_pid_status,
                PidTid=pid_tid,
            )
            for full_metrics in [False, True]:
                test_cases.append(
                    generate_proc_pid_metrics_generate_test_case(
                        f"all_change/{tc_num:04d}",
                        pid_parser_data,
                        pid_metrics_info_data=pid_metrics_info_data,
                        full_metrics=full_metrics,
                        instance=instance,
                        hostname=hostname,
                        description=f"is_pid={tid==procfs.PID_ONLY_TID}, comm_fallback={comm_fallback}, full_metrics={full_metrics}",
                    )
                )
                tc_num += 1

    # No change:
    for tid in [101, procfs.PID_ONLY_TID]:
        pid_tid = procfs.PidTid(Pid=pid, Tid=tid)
        for comm_fallback in [False, True]:
            if comm_fallback:
                pid_cmdline = TestPidCmdlineParsedData(Args="should be ignored")
            else:
                pid_cmdline = curr_pid_cmdline
            pid_parser_data = TestPidParserStateData(
                PidStat=curr_pid_stat,
                PidStatus=curr_pid_status,
                PidCmdline=pid_cmdline,
                PidTid=procfs.PidTid(Pid=pid, Tid=tid),
            )
            for zero_delta in [False, True]:
                pid_metrics_info_data = TestPidParserStateData(
                    PidStat=curr_pid_stat,
                    PidStatus=curr_pid_status,
                    PidStatFltZeroDelta=[zero_delta] * PID_STAT_FLT_ZERO_DELTA_SIZE,
                    PidStatusCtxZeroDelta=[zero_delta] * PID_STATUS_CTX_ZERO_DELTA_SIZE,
                    PidTid=pid_tid,
                )
                for full_metrics in [False, True]:
                    test_cases.append(
                        generate_proc_pid_metrics_generate_test_case(
                            f"no_change/{tc_num:04d}",
                            pid_parser_data,
                            pid_metrics_info_data=pid_metrics_info_data,
                            full_metrics=full_metrics,
                            instance=instance,
                            hostname=hostname,
                            description=f"is_pid={tid==procfs.PID_ONLY_TID}, comm_fallback={comm_fallback}, zero_delta={zero_delta}, full_metrics={full_metrics}",
                        )
                    )
                    tc_num += 1

    # Single change:
    for index in range(procfs.PID_STAT_BYTE_SLICE_NUM_FIELDS):
        if index in {procfs.PID_STAT_STARTTIME}:
            continue
        prev_pid_stat = make_prev_proc_pid_stat(
            curr_pid_stat, bsf_indexes=[index], nf_indexes=None
        )
        for tid in [101, procfs.PID_ONLY_TID]:
            pid_tid = procfs.PidTid(Pid=pid, Tid=tid)
            for comm_fallback in [False, True]:
                if comm_fallback:
                    pid_cmdline = TestPidCmdlineParsedData(Args="should be ignored")
                else:
                    pid_cmdline = curr_pid_cmdline
                pid_parser_data = TestPidParserStateData(
                    PidStat=curr_pid_stat,
                    PidStatus=curr_pid_status,
                    PidCmdline=pid_cmdline,
                    PidTid=procfs.PidTid(Pid=pid, Tid=tid),
                )
                pid_metrics_info_data = TestPidParserStateData(
                    PidStat=prev_pid_stat,
                    PidStatus=curr_pid_status,
                    PidStatFltZeroDelta=[True] * PID_STAT_FLT_ZERO_DELTA_SIZE,
                    PidStatusCtxZeroDelta=[True] * PID_STATUS_CTX_ZERO_DELTA_SIZE,
                    PidTid=pid_tid,
                )
                test_cases.append(
                    generate_proc_pid_metrics_generate_test_case(
                        f"pid_stat_bsf_one/{tc_num:04d}",
                        pid_parser_data,
                        pid_metrics_info_data=pid_metrics_info_data,
                        full_metrics=False,
                        instance=instance,
                        hostname=hostname,
                        description=f"is_pid={tid==procfs.PID_ONLY_TID}, comm_fallback={comm_fallback}, index={index}",
                    )
                )
                tc_num += 1
    for index in range(procfs.PID_STAT_ULONG_NUM_FIELDS):
        prev_pid_stat = make_prev_proc_pid_stat(
            curr_pid_stat, bsf_indexes=None, nf_indexes=[index]
        )
        for tid in [101, procfs.PID_ONLY_TID]:
            pid_tid = procfs.PidTid(Pid=pid, Tid=tid)
            pid_parser_data = TestPidParserStateData(
                PidStat=curr_pid_stat,
                PidStatus=curr_pid_status,
                PidCmdline=curr_pid_cmdline,
                PidTid=procfs.PidTid(Pid=pid, Tid=tid),
            )
            pid_metrics_info_data = TestPidParserStateData(
                PidStat=prev_pid_stat,
                PidStatus=curr_pid_status,
                PidStatFltZeroDelta=[True] * PID_STAT_FLT_ZERO_DELTA_SIZE,
                PidStatusCtxZeroDelta=[True] * PID_STATUS_CTX_ZERO_DELTA_SIZE,
                PidTid=pid_tid,
            )
            test_cases.append(
                generate_proc_pid_metrics_generate_test_case(
                    f"pid_stat_nf_one/{tc_num:04d}",
                    pid_parser_data,
                    pid_metrics_info_data=pid_metrics_info_data,
                    full_metrics=False,
                    instance=instance,
                    hostname=hostname,
                    description=f"is_pid={tid==procfs.PID_ONLY_TID}, index={index}",
                )
            )
            tc_num += 1

    for index in range(procfs.PID_STATUS_BYTE_SLICE_NUM_FIELDS):
        prev_pid_status = make_prev_proc_pid_status(
            curr_pid_status, bsf_indexes=[index], nf_indexes=None
        )
        for tid in [101, procfs.PID_ONLY_TID]:
            pid_tid = procfs.PidTid(Pid=pid, Tid=tid)
            pid_parser_data = TestPidParserStateData(
                PidStat=curr_pid_stat,
                PidStatus=curr_pid_status,
                PidCmdline=curr_pid_cmdline,
                PidTid=procfs.PidTid(Pid=pid, Tid=tid),
            )
            pid_metrics_info_data = TestPidParserStateData(
                PidStat=curr_pid_stat,
                PidStatus=prev_pid_status,
                PidStatFltZeroDelta=[True] * PID_STAT_FLT_ZERO_DELTA_SIZE,
                PidStatusCtxZeroDelta=[True] * PID_STATUS_CTX_ZERO_DELTA_SIZE,
                PidTid=pid_tid,
            )
            test_cases.append(
                generate_proc_pid_metrics_generate_test_case(
                    f"pid_status_bsf_one/{tc_num:04d}",
                    pid_parser_data,
                    pid_metrics_info_data=pid_metrics_info_data,
                    full_metrics=False,
                    instance=instance,
                    hostname=hostname,
                    description=f"is_pid={tid==procfs.PID_ONLY_TID}, index={index}",
                )
            )
            tc_num += 1
    for index in range(procfs.PID_STATUS_ULONG_NUM_FIELDS):
        prev_pid_status = make_prev_proc_pid_status(
            curr_pid_status, bsf_indexes=None, nf_indexes=[index]
        )
        for tid in [101, procfs.PID_ONLY_TID]:
            pid_tid = procfs.PidTid(Pid=pid, Tid=tid)
            pid_parser_data = TestPidParserStateData(
                PidStat=curr_pid_stat,
                PidStatus=curr_pid_status,
                PidCmdline=curr_pid_cmdline,
                PidTid=procfs.PidTid(Pid=pid, Tid=tid),
            )
            pid_metrics_info_data = TestPidParserStateData(
                PidStat=curr_pid_stat,
                PidStatus=prev_pid_status,
                PidStatFltZeroDelta=[True] * PID_STAT_FLT_ZERO_DELTA_SIZE,
                PidStatusCtxZeroDelta=[True] * PID_STATUS_CTX_ZERO_DELTA_SIZE,
                PidTid=pid_tid,
            )
            test_cases.append(
                generate_proc_pid_metrics_generate_test_case(
                    f"pid_status_nf_one/{tc_num:04d}",
                    pid_parser_data,
                    pid_metrics_info_data=pid_metrics_info_data,
                    full_metrics=False,
                    instance=instance,
                    hostname=hostname,
                    description=f"is_pid={tid==procfs.PID_ONLY_TID}, index={index}",
                )
            )
            tc_num += 1

    save_test_cases(
        test_cases, pm_generate_test_cases_file, test_cases_root_dir=test_cases_root_dir
    )


def generate_proc_pid_metrics_execute_test_case(
    name: str,
    pid_parsers_data_list: List[TestPidParserStateData],
    ts: Optional[float] = None,
    pid_metrics_info_data_list: Optional[
        List[TestPidParserStateData]
    ] = None,  # i.e. no prev
    part_no: int = 1,
    full_metrics_factor: int = DEFAULT_PROC_PID_FULL_METRICS_FACTOR,
    scan_num: int = 1,
    use_pid_status: bool = True,
    interval: float = DEFAULT_PROC_PID_INTERVAL_SEC,
    procfs_root: str = DEFAULT_PROCFS_ROOT,
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    boottime_msec: int = DEFAULT_BOOTTIME_MSEC,
    linux_clktck_sec: float = TEST_LINUX_CLKTCK_SEC,
    page_size: int = TEST_OS_PAGE_SIZE,
    description: Optional[str] = None,
) -> ProcPidMetricsExecuteTestCase:

    if ts is None:
        ts = time.time()
    ts_inc = interval / len(pid_parsers_data_list) * 0.8
    interval_msec = int(interval * 1000)

    pid_parsers_data_list = deepcopy(pid_parsers_data_list)
    pid_metrics_info_data_list = deepcopy(pid_metrics_info_data_list)

    pid_metrics_info_data_by_pid_tid = {}
    if pid_metrics_info_data_list:
        for pid_metrics_info_data in pid_metrics_info_data_list:
            pid_metrics_info_data_by_pid_tid[
                pid_metrics_info_data.PidTid
            ] = pid_metrics_info_data

    pid_tid_list_result = []
    all_metrics = []
    want_zero_delta_list = []
    active_pid_tid_count = 0
    parse_err_pid_tid_count = 0
    new_pid_tid_count = 0
    del_pid_tid_count = 0
    use_pid_status = True
    for pid_parser_data in pid_parsers_data_list:
        pid_tid = pid_parser_data.PidTid
        pid_tid_list_result.append(pid_tid)
        pid_parser_data.UnixMilli = int(ts * 1000)
        ts += ts_inc

        is_pid = pid_tid.Tid == procfs.PID_ONLY_TID
        pid_metrics_info_data = pid_metrics_info_data_by_pid_tid.get(pid_tid)
        full_metrics = False
        active = False
        if pid_metrics_info_data is None:
            # By definition this counts as active:
            active = True
            active_pid_tid_count += 1
        else:
            full_metrics = pid_metrics_info_data.CycleNum == 0
            # If either of the needed parser data is None then this PID,TID will
            # generate an error and it will be deleted:
            if (
                pid_parser_data.PidStat is None
                or use_pid_status
                and pid_parser_data.PidStatus is None
                or full_metrics
                and is_pid
                and pid_parser_data.PidCmdline is None
            ):
                parse_err_pid_tid_count += 1
                del_pid_tid_count += 1
                continue

            pid_metrics_info_data.UnixMilli = pid_parser_data.UnixMilli - interval_msec

            # Furthermore if starttime has changed then this is fact a new PID,TID:
            if (
                pid_parser_data.PidStat.ByteSliceFields[procfs.PID_STAT_STARTTIME]
                != pid_metrics_info_data.PidStat.ByteSliceFields[
                    procfs.PID_STAT_STARTTIME
                ]
            ):
                del_pid_tid_count += 1
                pid_metrics_info_data = None
            else:
                curr_pid_stat_nf = pid_parser_data.PidStat.NumericFields
                prev_pid_stat_nf = pid_metrics_info_data.PidStat.NumericFields
                active = (
                    curr_pid_stat_nf[procfs.PID_STAT_UTIME]
                    != prev_pid_stat_nf[procfs.PID_STAT_UTIME]
                    or curr_pid_stat_nf[procfs.PID_STAT_STIME]
                    != prev_pid_stat_nf[procfs.PID_STAT_STIME]
                )
                if active:
                    # Active:
                    active_pid_tid_count += 1
                elif not full_metrics and not pid_metrics_info_data.Active:
                    # Inactive after inactive, non-full metrics, no metrics will be generated:
                    continue
        metrics, want_zero_delta = generate_proc_pid_metrics(
            pid_parser_data,
            pid_metrics_info_data=pid_metrics_info_data,
            full_metrics=full_metrics,
            instance=instance,
            hostname=hostname,
            boottime_msec=boottime_msec,
            linux_clktck_sec=linux_clktck_sec,
        )
        want_zero_delta.Active = active
        all_metrics.extend(metrics)
        want_zero_delta_list.append(want_zero_delta)
        if pid_metrics_info_data is None:
            new_pid_tid_count += 1

    # PID,TID in metrics info (prev, that is) but not in parser data will also
    # be deleted, count them:
    paser_data_pid_tid = set(
        pid_parser_data.PidTid for pid_parser_data in pid_parsers_data_list
    )
    for pid_tid in pid_metrics_info_data_by_pid_tid:
        if pid_tid not in paser_data_pid_tid:
            del_pid_tid_count += 1

    # Generator specific metrics:
    curr_prom_ts = int(ts * 1000)
    gen_spec_labels = ",".join(
        [
            f'{INSTANCE_LABEL_NAME}="{instance}"',
            f'{HOSTNAME_LABEL_NAME}="{hostname}"',
            f'{PROC_PID_PART_LABEL_NAME}="{part_no}"',
        ]
    )
    all_metrics.append(
        f"{PROC_PID_TOTAL_COUNT_METRIC}{{{gen_spec_labels}}} {len(pid_tid_list_result)} {curr_prom_ts}"
    )
    all_metrics.append(
        f"{PROC_PID_PARSE_OK_COUNT_METRIC}{{{gen_spec_labels}}} {len(pid_tid_list_result) - parse_err_pid_tid_count} {curr_prom_ts}"
    )
    all_metrics.append(
        f"{PROC_PID_PARSE_ERR_COUNT_METRIC}{{{gen_spec_labels}}} {parse_err_pid_tid_count} {curr_prom_ts}"
    )
    all_metrics.append(
        f"{PROC_PID_ACTIVE_COUNT_METRIC}{{{gen_spec_labels}}} {active_pid_tid_count} {curr_prom_ts}"
    )
    all_metrics.append(
        f"{PROC_PID_NEW_COUNT_METRIC}{{{gen_spec_labels}}} {new_pid_tid_count} {curr_prom_ts}"
    )
    all_metrics.append(
        f"{PROC_PID_DEL_COUNT_METRIC}{{{gen_spec_labels}}} {del_pid_tid_count} {curr_prom_ts}"
    )
    if pid_metrics_info_data_list:
        all_metrics.append(
            f"{PROC_PID_INTERVAL_METRIC}{{{gen_spec_labels}}} {interval:.06f} {curr_prom_ts}"
        )

    return ProcPidMetricsExecuteTestCase(
        Name=name,
        Description=description,
        PartNo=part_no,
        FullMetricsFactor=full_metrics_factor,
        UsePidStatus=use_pid_status,
        ScanNum=scan_num,
        PageSize=page_size,
        Instance=instance,
        Hostname=hostname,
        LinuxClktckSec=linux_clktck_sec,
        BoottimeMsec=boottime_msec,
        PidTidListResult=pid_tid_list_result,
        PidTidMetricsInfoList=pid_metrics_info_data_list,
        PidParsersDataList=pid_parsers_data_list,
        CurrUnixMilli=curr_prom_ts,
        PrevUnixMilli=curr_prom_ts - interval_msec,
        WantMetricsCount=len(all_metrics),
        WantMetrics=all_metrics,
        ReportExtra=True,
        WantZeroDeltaList=want_zero_delta_list,
    )


def pid_tid_list_generator(
    start_pid: int = 100,
    tid_offset: int = 1000,
    num_pids: int = 1,
    tid_mod: int = 3,
) -> Generator:
    for pid in range(start_pid, start_pid + num_pids):
        yield procfs.PidTid(Pid=pid)
        num_tids = (pid - start_pid) % tid_mod
        start_tid = tid_offset + pid
        for tid in range(start_tid, start_tid + num_tids):
            yield procfs.PidTid(Pid=pid, Tid=tid)


def generate_proc_pid_metrics_execute_test_cases(
    instance: str = DEFAULT_TEST_INSTANCE,
    hostname: str = DEFAULT_TEST_HOSTNAME,
    test_cases_root_dir: Optional[str] = lsvmi_test_cases_root_dir,
):
    test_cases = []
    tc_num = 0

    max_n_pid = PROC_PID_METRICS_CYCLE_NUM_COUNTERS
    tid_mod = 3

    # All new:
    name = "new"
    start_pid, tid_offset = 100, 1000
    for num_pids in range(1, max_n_pid + 1):
        pid_parsers_data_list = []
        for pid_tid in pid_tid_list_generator(
            start_pid=start_pid,
            tid_offset=tid_offset,
            num_pids=num_pids,
            tid_mod=tid_mod,
        ):
            curr_pid_stat = make_ref_proc_pid_stat(pid_tid)
            curr_pid_status = make_ref_proc_pid_status(pid_tid)
            curr_pid_cmdline = make_ref_proc_pid_cmdline(pid_tid)
            pid_parsers_data_list.append(
                TestPidParserStateData(
                    PidStat=curr_pid_stat,
                    PidStatus=curr_pid_status,
                    PidCmdline=curr_pid_cmdline,
                    PidTid=pid_tid,
                )
            )
        test_cases.append(
            generate_proc_pid_metrics_execute_test_case(
                f"{name}/{tc_num:04d}",
                pid_parsers_data_list=pid_parsers_data_list,
                instance=instance,
                hostname=hostname,
                description=f"len(parsers_data)={len(pid_parsers_data_list)}",
            )
        )
        tc_num += 1

    # Existent, (in)active -> (in)active transitions:
    name = "active_transition"
    pid_tid = procfs.PidTid(Pid=200, Tid=2000)
    curr_pid_stat = make_ref_proc_pid_stat(pid_tid)
    curr_pid_status = make_ref_proc_pid_status(pid_tid)
    curr_pid_cmdline = make_ref_proc_pid_cmdline(pid_tid)
    pid_parsers_data_list = [
        TestPidParserStateData(
            PidStat=curr_pid_stat,
            PidStatus=curr_pid_status,
            PidCmdline=curr_pid_cmdline,
            PidTid=pid_tid,
        )
    ]
    for prev_active in [False, True]:
        for curr_active in [False, True]:
            if curr_active:
                prev_pid_stat = make_prev_proc_pid_stat(
                    curr_pid_stat,
                    nf_indexes=[procfs.PID_STAT_UTIME],
                )
            else:
                prev_pid_stat = make_prev_proc_pid_stat(
                    curr_pid_stat,
                    bsf_indexes=[procfs.PID_STAT_STATE],
                )
            for cycle_num in [0, 1]:
                pid_metrics_info_data_list = [
                    TestPidParserStateData(
                        PidStat=prev_pid_stat,
                        PidStatus=curr_pid_status,
                        PidTid=pid_tid,
                        CycleNum=cycle_num,
                        Active=prev_active,
                    )
                ]
                test_cases.append(
                    generate_proc_pid_metrics_execute_test_case(
                        f"{name}/{tc_num:04d}",
                        pid_parsers_data_list=pid_parsers_data_list,
                        pid_metrics_info_data_list=pid_metrics_info_data_list,
                        instance=instance,
                        hostname=hostname,
                        description=f"active={prev_active}->{curr_active}, full_cyle={cycle_num==0}",
                    )
                )
                tc_num += 1

    # Existent + new:
    name = "existent_new"
    pid_stat_changed_indexes = [
        ([procfs.PID_STAT_STATE], None),
        (None, [procfs.PID_STAT_UTIME]),
        (None, [procfs.PID_STAT_STIME]),
    ]
    pid_status_changed_indexes = [
        ([procfs.PID_STATUS_VM_DATA], None),
        ([procfs.PID_STATUS_VM_STK], None),
        (None, [procfs.PID_STATUS_VOLUNTARY_CTXT_SWITCHES]),
    ]

    start_pid, tid_offset = 300, 3000
    full_metrics_factor = 3
    for num_pids in range(1, max_n_pid + 1):
        k = 0
        pid_parsers_data_list = []
        pid_metrics_info_data_list = []
        for pid_tid in pid_tid_list_generator(
            start_pid=start_pid,
            tid_offset=tid_offset,
            num_pids=num_pids,
            tid_mod=tid_mod,
        ):
            curr_pid_stat = make_ref_proc_pid_stat(pid_tid)
            curr_pid_status = make_ref_proc_pid_status(pid_tid)
            curr_pid_cmdline = make_ref_proc_pid_cmdline(pid_tid)
            if k & 1:
                bsf_indexes, nf_indexes = pid_stat_changed_indexes[
                    k % len(pid_stat_changed_indexes)
                ]
                prev_pid_stat = make_prev_proc_pid_stat(
                    curr_pid_stat,
                    bsf_indexes=bsf_indexes,
                    nf_indexes=nf_indexes,
                )
                bsf_indexes, nf_indexes = pid_status_changed_indexes[
                    k % len(pid_status_changed_indexes)
                ]
                prev_pid_status = make_prev_proc_pid_status(
                    curr_pid_status,
                    bsf_indexes=bsf_indexes,
                    nf_indexes=nf_indexes,
                )
                pid_metrics_info_data_list.append(
                    TestPidParserStateData(
                        PidStat=prev_pid_stat,
                        PidStatus=prev_pid_status,
                        PidTid=pid_tid,
                        CycleNum=k % full_metrics_factor,
                    )
                )
            k += 1
            pid_parsers_data_list.append(
                TestPidParserStateData(
                    PidStat=curr_pid_stat,
                    PidStatus=curr_pid_status,
                    PidCmdline=curr_pid_cmdline,
                    PidTid=pid_tid,
                )
            )
        test_cases.append(
            generate_proc_pid_metrics_execute_test_case(
                f"{name}/{tc_num:04d}",
                pid_parsers_data_list=pid_parsers_data_list,
                pid_metrics_info_data_list=pid_metrics_info_data_list,
                full_metrics_factor=full_metrics_factor,
                instance=instance,
                hostname=hostname,
                description=f"len(info_data)={len(pid_metrics_info_data_list)}, len(parsers_data)={len(pid_parsers_data_list)}",
            )
        )
        tc_num += 1

    # Out-of-scope PID,TID:
    name = "out_of_scope"
    start_pid, tid_offset = 400, 4000
    for num_pids in range(1, max_n_pid + 1):
        pid_metrics_info_data_list = []
        for pid_tid in pid_tid_list_generator(
            start_pid=start_pid,
            tid_offset=tid_offset,
            num_pids=num_pids,
            tid_mod=tid_mod,
        ):
            curr_pid_stat = make_ref_proc_pid_stat(pid_tid)
            curr_pid_status = make_ref_proc_pid_status(pid_tid)
            curr_pid_cmdline = make_ref_proc_pid_cmdline(pid_tid)
            pid_metrics_info_data_list.append(
                TestPidParserStateData(
                    PidStat=curr_pid_stat,
                    PidStatus=curr_pid_status,
                    PidCmdline=curr_pid_cmdline,
                    PidTid=pid_tid,
                )
            )
        pid_parsers_data_list = [
            pid_parser_state_data
            for i, pid_parser_state_data in enumerate(pid_metrics_info_data_list)
            if i % 2 == 0
        ]
        test_cases.append(
            generate_proc_pid_metrics_execute_test_case(
                f"{name}/{tc_num:04d}",
                pid_parsers_data_list=pid_parsers_data_list,
                pid_metrics_info_data_list=pid_metrics_info_data_list,
                instance=instance,
                hostname=hostname,
                description=f"len(info_data)={len(pid_metrics_info_data_list)}, len(parsers_data)={len(pid_parsers_data_list)}",
            )
        )
        tc_num += 1

    # Deleted PID,TID; they are reported in the list but they vanish by the time
    # they are parsed and they are reported as parse error:
    name = "parse_error"
    start_pid, tid_offset = 500, 5000
    # Simulate parse failure for every 3 out of N:
    fail_mod = 5
    for num_pids in range(1, max_n_pid + 1):
        pid_metrics_info_data_list = []
        for pid_tid in pid_tid_list_generator(
            start_pid=start_pid,
            tid_offset=tid_offset,
            num_pids=num_pids,
            tid_mod=tid_mod,
        ):
            curr_pid_stat = make_ref_proc_pid_stat(pid_tid)
            curr_pid_status = make_ref_proc_pid_status(pid_tid)
            curr_pid_cmdline = make_ref_proc_pid_cmdline(pid_tid)
            pid_metrics_info_data_list.append(
                TestPidParserStateData(
                    PidStat=curr_pid_stat,
                    PidStatus=curr_pid_status,
                    PidCmdline=curr_pid_cmdline,
                    PidTid=pid_tid,
                )
            )
        pid_parsers_data_list = deepcopy(pid_metrics_info_data_list)
        for i, pid_parsers_data in enumerate(pid_parsers_data_list):
            i_mod = i % fail_mod
            if i_mod == 0:
                pid_parsers_data.PidStat = None
            elif i_mod == 1:
                pid_parsers_data.PidStatus = None
            elif i_mod == 2:
                pid_parsers_data.PidCmdline = None
        test_cases.append(
            generate_proc_pid_metrics_execute_test_case(
                f"{name}/{tc_num:04d}",
                pid_parsers_data_list=pid_parsers_data_list,
                pid_metrics_info_data_list=pid_metrics_info_data_list,
                instance=instance,
                hostname=hostname,
                description=f"len(info_data)={len(pid_metrics_info_data_list)}, len(parsers_data)={len(pid_parsers_data_list)}",
            )
        )
        tc_num += 1

    save_test_cases(
        test_cases, pm_execute_test_cases_file, test_cases_root_dir=test_cases_root_dir
    )
