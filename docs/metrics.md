# LSVMI Metrics

<!-- TOC tocDepth:2..6 chapterDepth:2..6 -->

- [Common Labels](#common-labels)
- [Timestamps](#timestamps)
- [The Complete List Of Metrics](#the-complete-list-of-metrics)

<!-- /TOC -->

## Common Labels

All metrics have the following labels:

- `instance` with the associated value identifying a specific [LSVMI](../README.md). The value, in decreasing order of precedence:
  - `-instance INSTANCE` command line arg
  - `global_config.instance` in config file
  - `lsvmi` built-in default
- `hostname` with the associated value identifying a host where [LSVMI](../README.md) runs. The value, in decreasing order of precedence:
  - `-hostname HOSTNAME` command line arg
  - the value returned by `hostname` syscall
  The value may be stripped of domain part, depending upon `global_config.use_short_hostname: true|false` config

## Timestamps

The generated metrics use specific timestamps from when the parser(s) returned the new data. If multiple sources were involved, the timestamp is from when **all** the needed data was parsed. For instance `PID` metrics may require `/proc/[PID]/stat`, `/proc/[PID]/status` and `/proc/[PID]/cmdline` parsing; the timestamp is from when all 3 of them returned.

## The Complete List Of Metrics

- [by generator](metrics_by_generator.md)

- [alphabetically](metrics_alphabetically.md)
