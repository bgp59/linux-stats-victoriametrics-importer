# LSVMI Metrics

<!-- TOC tocDepth:2..3 chapterDepth:2..6 -->

- [Applicable To All Metrics](#applicable-to-all-metrics)
- [Internal Metrics](#internal-metrics)
  - [lsvmi_internal_metrics_delta_sec](#lsvmi_internal_metrics_delta_sec)
  - [lsvmi_uptime_sec](#lsvmi_uptime_sec)
  - [os_info](#os_info)
  - [os_uptime_sec](#os_uptime_sec)
  - [os_btime_sec](#os_btime_sec)

<!-- /TOC -->

## Applicable To All Metrics

All metrics have the following labels:

- `instance` with the associated value identifying a specific [LSVMI](../README.md). The value, in decreasing order of precedence:
  - `-instance INSTANCE` command line arg
  - `global_config.instance` in config file
  - `lsvmi` built-in default
- `hostname` with the associated value identifying a host where [LSVMI](../README.md) runs. The value, in decreasing order of precedence:
  - `-hostname HOSTNAME` command line arg
  - the value returned by `hostname` syscall
  The value may be stripped of domain part, depending upon `global_config.use_short_hostname: true|false` config

## Internal Metrics

### lsvmi_internal_metrics_delta_sec

  The actual time delta, in seconds, since last internal metrics generation. This may be different than the scan `interval`, the latter is the desired, theoretical value.

  | Label Name | Value(s)/Info |
  | --- | --- |
  | instance | _instance_ |
  | hostname | _hostname_ |

### lsvmi_uptime_sec

  Time, in seconds, since the agent was started.
  
  | Label Name | Value(s)/Info |
  | --- | --- |
  | instance | _instance_ |
  | hostname | _hostname_ |
  | version | semver of the agent |
  | gitinfo | git describe based |

### os_info

  Categorical (constant `1`) with [uname](https://linux.die.net/man/1/uname) like info:

  | Label Name | Value(s)/Info |
  | --- | --- |
  | instance | _instance_ |
  | hostname | _hostname_ |
  | sys_name | \`uname -s\` |
  | sys_release | \`uname -r\` |
  | sys_version | \'uname -v\` |

### os_uptime_sec

  Time, in seconds, since OS boot

  | Label Name | Value(s)/Info |
  | --- | --- |
  | instance | _instance_ |
  | hostname | _hostname_ |

### os_btime_sec

  Boot time, in seconds.

  | Label Name | Value(s)/Info |
  | --- | --- |
  | instance | _instance_ |
  | hostname | _hostname_ |
