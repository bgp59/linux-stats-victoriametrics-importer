# LSVMI Statfs Metrics (id: `statfs_metrics`)

<!-- TOC tocDepth:2..3 chapterDepth:2..6 -->

- [General Information](#general-information)
- [Metrics](#metrics)
  - [statfs_bsize](#statfs_bsize)
  - [statfs_blocks](#statfs_blocks)
  - [statfs_bfree](#statfs_bfree)
  - [statfs_bavail](#statfs_bavail)
  - [statfs_files](#statfs_files)
  - [statfs_ffree](#statfs_ffree)
  - [statfs_total_size_kb](#statfs_total_size_kb)
  - [statfs_free_size_kb](#statfs_free_size_kb)
  - [statfs_avail_size_kb](#statfs_avail_size_kb)
  - [statfs_free_pct](#statfs_free_pct)
  - [statfs_avail_pct](#statfs_avail_pct)
  - [statfs_present](#statfs_present)
  - [statfs_metrics_delta_sec](#statfs_metrics_delta_sec)

<!-- /TOC -->

## General Information

Based [statfs](https://man7.org/linux/man-pages/man2/statfs.2.html) like info, cross-referenced with mount points from [/proc/PID/mountinfo](https://man7.org/linux/man-pages/man5/proc_pid_mountinfo.5.html).

## Metrics

Unless otherwise specified all metrics in this section have the following label set:

| Label Name | Value(s)
| --- | --- |
| instance | _instance_ |
| hostname | _hostname_ |
| mount_point | _path_ for the mount |
| fs | filesystem specific info, e.g. device path <br>Called `mount source` in [/proc/PID/mountinfo](https://man7.org/linux/man-pages/man5/proc_pid_mountinfo.5.html) |
| fs_type | _type\[.subtype\]_ |

### statfs_bsize

Block size.

### statfs_blocks

Total number of blocks.

### statfs_bfree

Number of free blocks.

### statfs_bavail

Number of available blocks.

### statfs_files

Total inodes in the filesystem.

### statfs_ffree

Number of free inodes.

### statfs_total_size_kb

Total size in kB.

### statfs_free_size_kb

Free size in kB.

### statfs_avail_size_kb

Available size in kB.

### statfs_free_pct

Free size percentage.

### statfs_avail_pct

Available size percentage.

### statfs_present

[Pseudo-categorical](internals.md#pseudo-categorical-metrics) metric indicating the presence (value `1`) or disappearance (value `0`) of a mounted filesystem. This metric could be used as a qualifier for the values above, since filesystems can be mounted/unmounted at any time. When a filesystem is unmounted its last metrics may still linger in queries due to lookback interval. For a more precise cutoff point, the metrics can by qualified as follows:

  ```text
  
  statfs_free_size_kb \
  and on (instance, hostname, mount_point, fs, fs_type) \
  (last_over_time(statfs_present) > 0)

  ```

### statfs_metrics_delta_sec

Time in seconds since the last scan. The real life counterpart (i.e. measured value) to the desired (configured) `interval`.

| Label Name | Value(s)
| --- | --- |
| instance | _instance_ |
| hostname | _hostname_ |
