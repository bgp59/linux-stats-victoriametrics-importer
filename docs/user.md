# LSVMI User Guide

## Command Line Args

```text

Usage of ./linux-stats-victoriametrics-importer:
  -config string
     Config file to load (default "lsvmi-config.yaml")
  -hostname string
     Override the the value returned by hostname syscall
  -http-pool-endpoints string
     Override the "http_endpoint_pool_config.endpoints" config
     setting
  -instance string
     Override the "global_config.instance" config setting
  -log-file string
     Override the config "log_config.log_file" config setting
  -log-level string
     Override the "log_config.log_level" config setting, it
     should be one of the ["panic" "fatal" "error" "warning"
     "info" "debug" "trace"] values
  -procfs-root string
     Override the "global_config.procfs_root" config setting
  -use-stdout-metrics-queue
     Print metrics to stdout instead of sending to import
     endpoints
  -version
     Print version and other build info and exit

```

## Configuration

The configuration file is in `YAML` format (it supports comments!) and the file [lsvmi-config-reference.yaml](../lsvmi/lsvmi-config-reference.yaml), distributed with the release package, is a good primer. The file itself is self-explanatory and most likely it requires little to no changes.

The most likely variants from one host to another are HTTP Pool endpoints used for import and those can be accommodated at invocation time via a command line argument.

## Deployment

The deployment of Victoria Metrics [VictoriaMetrics](https://docs.victoriametrics.com) and [Grafana](https://grafana.com/grafana/) are outside the scope of this document.

The use of [Victoria Metrics / Cluster version](https://docs.victoriametrics.com/cluster-victoriametrics/) is recommended, with a pool of [vmagent](https://docs.victoriametrics.com/vmagent/) as import endpoints.

Using the 1.5 million sample/sec [ingestion rate](https://docs.victoriametrics.com/single-server-victoriametrics/#capacity-planning) stated for the [Victoria Metrics / Single version](https://docs.victoriametrics.com/single-server-victoriametrics/) as a conservative baseline, a [vmagent](https://docs.victoriametrics.com/vmagent/) should be provisioned for each 0.5 million sample/sec typical load.

**Note:** the infra setting described in [LSVMI Proof Of Concept Demo](poc.md) can be used for a quick assessment of the sample rate from a given host. The agent should be started with:

```text
    -http-pool-endpoints http://<poc_host>:8428
```

arg. This works even for the container demo, since the port is exposed to the host.

The rate can then be determined from the `lsvmi-reference/internal_metrics_ref` dashboard, the `Generators` section, `Metrics Generator Sample Rate` panel.

Once **N** (the number of [vmagent](https://docs.victoriametrics.com/vmagent/) instances) is determined, the deployment should use **N+1** or more instances for resilience purpose.

Depending upon how the [vmagent](https://docs.victoriametrics.com/vmagent/) pool is setup, the agent configuration should be as follows:

* [vmagent](https://docs.victoriametrics.com/vmagent/) as a DNS pool with **M** members, presented as a virtual hostname. At any given times **N** (<= **M** ) members are supposed to be functional.

    ```yaml

    http_endpoint_pool_config:
        endpoints:
            - url: http://<pool_name>:8429/api/v1/import/prometheus
        mark_unhealthy_threshold: <N>
        shuffle: false

    ```

* [vmagent](https://docs.victoriametrics.com/vmagent/) as pool with **M** individual members, each with its own hostname. At any given times **N** (<= **M** ) members are supposed to be functional.

    ```yaml

    http_endpoint_pool_config:
        endpoints:
            - url: http://<vmagent_1>:8429/api/v1/import/prometheus
            - url: http://<vmagent_2>:8429/api/v1/import/prometheus
            #...
            - url: http://<vmagent_M>:8429/api/v1/import/prometheus
        mark_unhealthy_threshold: 1
        shuffle: true

    ```

Note the `shuffle: true` above which will ensure that the active connections will spread (pseudo-)randomly and hopefully evenly across all the members.

## [Grafana](https://grafana.com/docs/grafana/latest) Dashboards

[Provisioned](https://grafana.com/docs/grafana/latest/administration/provisioning/#dashboards)  dashboards can be found under [tools/poc/files/update/grafana/dashboards/lsvmi-reference](../tools/poc/files/update/grafana/dashboards/lsvmi-reference), or they are included into `LSVMI PoC Infra ...` [releases](https://github.com/emypar/linux-stats-victoriametrics-importer/releases).

They illustrate all the available metrics and they can be used as a starting point for actual dashboards.
