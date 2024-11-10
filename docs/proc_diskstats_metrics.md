# LSVMI Disk Stats And Mount Info Metrics (id: `proc_diskstats_metrics`)

<!-- TOC tocDepth:2..3 chapterDepth:2..6 -->

- [Disk Stats Metrics](#disk-stats-metrics)
  - [proc_diskstats_num_reads_completed_delta](#proc_diskstats_num_reads_completed_delta)
  - [proc_diskstats_num_reads_merged_delta](#proc_diskstats_num_reads_merged_delta)
  - [proc_diskstats_num_read_sectors_delta](#proc_diskstats_num_read_sectors_delta)
  - [proc_diskstats_read_pct](#proc_diskstats_read_pct)
  - [proc_diskstats_num_writes_completed_delta](#proc_diskstats_num_writes_completed_delta)
  - [proc_diskstats_num_writes_merged_delta](#proc_diskstats_num_writes_merged_delta)
  - [proc_diskstats_num_write_sectors_delta](#proc_diskstats_num_write_sectors_delta)
  - [proc_diskstats_write_pct](#proc_diskstats_write_pct)
  - [proc_diskstats_num_io_in_progress](#proc_diskstats_num_io_in_progress)
  - [proc_diskstats_io_pct](#proc_diskstats_io_pct)
  - [proc_diskstats_io_weigthed_pct](#proc_diskstats_io_weigthed_pct)
  - [proc_diskstats_num_discards_completed_delta](#proc_diskstats_num_discards_completed_delta)
  - [proc_diskstats_num_discards_merged_delta](#proc_diskstats_num_discards_merged_delta)
  - [proc_diskstats_num_discard_sectors_delta](#proc_diskstats_num_discard_sectors_delta)
  - [proc_diskstats_discard_pct](#proc_diskstats_discard_pct)
  - [proc_diskstats_num_flush_requests_delta](#proc_diskstats_num_flush_requests_delta)
  - [proc_diskstats_flush_pct](#proc_diskstats_flush_pct)
- [Mount Info Metrics](#mount-info-metrics)
  - [proc_mountinfo](#proc_mountinfo)
- [Generator Metrics](#generator-metrics)
  - [proc_diskstats_metrics_delta_sec](#proc_diskstats_metrics_delta_sec)

<!-- /TOC -->
## Disk Stats Metrics

Based on [/proc/diskstats](https://github.com/torvalds/linux/blob/master/Documentation/admin-guide/iostats.rst).

Unless otherwise stated, the metrics in this paragraph have the following label set:

| Label Name | Value(s)/Info |
| --- | --- |
| instance | _instance_ |
| hostname | _hostname_ |
| maj_min | _major:minor_ |
| name | _device name_ |

### proc_diskstats_num_reads_completed_delta

The number of reads completed successfully since the last scan.

### proc_diskstats_num_reads_merged_delta

The number of adjacent reads merged since the last scan.

### proc_diskstats_num_read_sectors_delta

The number of sectors read successfully since the last scan.

### proc_diskstats_read_pct

The percentage of time spent in reads over the interval since the last scan.

### proc_diskstats_num_writes_completed_delta

The number of writes completed successfully since the last scan.

### proc_diskstats_num_writes_merged_delta

The number of adjacent writes merged since the last scan.

### proc_diskstats_num_write_sectors_delta

The number of sectors written successfully since the last scan.

### proc_diskstats_write_pct

The percentage of time spent in writes over the interval since the last scan.

### proc_diskstats_num_io_in_progress

The number of I/Os currently in progress.

### proc_diskstats_io_pct

The percentage of time spent doing I/O over the interval since the last scan.

### proc_diskstats_io_weigthed_pct

The percentage of time spent measuring weighted I/O over the interval since the last scan.

### proc_diskstats_num_discards_completed_delta

The number of discards completed successfully since the last scan.

### proc_diskstats_num_discards_merged_delta

The number of adjacent discards merged since the last scan.

### proc_diskstats_num_discard_sectors_delta

The number of discarded sectors since the last scan.

### proc_diskstats_discard_pct

The percentage of time spent in discards over the interval since the last scan.

### proc_diskstats_num_flush_requests_delta

The number of flush requests completed successfully since the last scan

### proc_diskstats_flush_pct

The percentage of time spent in flush requests over the interval since the last scan.

## Mount Info Metrics

Based on [/proc/PID/mountinfo](https://man7.org/linux/man-pages/man5/proc_pid_mountinfo.5.html)

### proc_mountinfo

[Pseudo-categorical](internals.md#pseudo-categorical-metrics) metric containing information about the mounted file systems.

| Label Name | Value(s)/Info |
| --- | --- |
| instance | _instance_ |
| hostname | _hostname_ |
| pid | _PID_ for `/proc/PID/mountinfo`, `0` for self |
| maj_min | _major:minor_ |
| root | _path_ for the root of this mount |
| mount_point | _path_ for the mount, relative to _root_ |
| fs_type | _type\[.subtype\]_ |
| source | filesystem specific info, e.g. device path |

## Generator Metrics

### proc_diskstats_metrics_delta_sec

Time in seconds since the last scan. The real life counterpart (i.e. measured value) to the desired (configured) `interval`.

| Label Name | Value(s)/Info |
| --- | --- |
| instance | _instance_ |
| hostname | _hostname_ |
| id | `proc_diskstats_metrics` |
