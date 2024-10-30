# LSVMI Metrics

- [LSVMI Metrics](#lsvmi-metrics)
  - [Applicable To All Metrics](#applicable-to-all-metrics)
  - [Internal Metrics](#internal-metrics)
    - [lsvmi_uptime_sec](#lsvmi_uptime_sec)
    - [os_info](#os_info)
    - [os_btime_sec](#os_btime_sec)
    - [os_uptime_sec](#os_uptime_sec)
    - [lsvmi_internal_metrics_delta_sec](#lsvmi_internal_metrics_delta_sec)

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

### lsvmi_uptime_sec

- labels:

  | name | value |
  | ---   | --- |
  | instance | _instance_ |
  | hostname | _hostname_ |
  | version | semver of the agent |
  | gitinfo | git describe based |

- value: time, in seconds, since the agent was started

### os_info

- labels:

  | name | value |
  | ---   | --- |
  | instance | _instance_ |
  | hostname | _hostname_ |
  | sys_name | \`uname -s\` |
  | sys_release | \`uname -r\` |
  | sys_version | \'uname -v\` |

- value: `1`

### os_btime_sec

- labels:

  | name | value |
  | ---   | --- |
  | instance | _instance_ |
  | hostname | _hostname_ |

- value: boot time, in seconds

### os_uptime_sec

- labels:

  | name | value |
  | ---   | --- |
  | instance | _instance_ |
  | hostname | _hostname_ |

- value: time, in seconds, since OS boot

### lsvmi_internal_metrics_delta_sec

- labels:

  | name | value |
  | ---   | --- |
  | instance | _instance_ |
  | hostname | _hostname_ |

- value: the actual time delta, in seconds, since last internal metrics generation. This may be different than the scan `interval`
