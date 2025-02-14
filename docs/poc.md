# LSVMI Proof Of Concept Demo

<!-- TOC tocDepth:2..3 chapterDepth:2..6 -->

- [Note](#note)
- [TL;DR: Quick Start For PoC](#tldr-quick-start-for-poc)
  - [Using A Linux Server](#using-a-linux-server)
  - [Using A Containerized Solution](#using-a-containerized-solution)
- [Browsing The Reference Dashboards](#browsing-the-reference-dashboards)

<!-- /TOC -->

## Note

While this is intended as a `TL;DR`, some familiarity with the following is required:

- [Prometheus Metrics](https://prometheus.io/docs/concepts/data_model/)
- [VictoriaMetrics / Single version](https://docs.victoriametrics.com/single-server-victoriametrics/)
- [Grafana](https://grafana.com/docs/grafana/latest/getting-started/)
- [LSVI Metrics](metrics.md)

Or one could just dive in and get all the way to the reference dashboards just by following the instructions.

## TL;DR: Quick Start For PoC

The **PoC** requires an instance of [VictoriaMetrics](https://docs.victoriametrics.com/single-server-victoriametrics/), [Grafana](https://grafana.com/docs/grafana/latest/getting-started/) and [LSVMI](../README.md) running on the same Linux server or container.

### Using A Linux Server

- **NOTE!** For the security conscious tester, create a PoC acct:

    ```bash

    sudo useradd -U -d /home/lsvmi -m -s /bin/bash lsvmi

    ```

- extract the infra installation archive:

    ```bash
    
    [[ $(whoami) != "lsvmi" ]]  && id lsvmi 2>/dev/null && sudo -n su - lsvmi

    cd /tmp
    curl \
            -s \
            -L \
            https://github.com/bgp59/linux-stats-victoriametrics-importer/releases/latest/download/lsvmi-poc-infra.tgz | \
        tar  -xzf -

    ```

- install the **PoC** supporting [VictoriaMetrics](https://docs.victoriametrics.com/single-server-victoriametrics/) and [Grafana](https://grafana.com/docs/grafana/latest/getting-started/) under `$HOME/lsvmi-poc`, using `$HOME/lsvmi-poc/runtime` as working area for databases, logs, etc.:

    ```bash
    
    [[ $(whoami) != "lsvmi" ]]  && id lsvmi 2>/dev/null && sudo -n su - lsvmi

    cd /tmp/lsvmi-poc-infra
    ./install-lsvmi-infra.sh

    ```

    If a different location is desired, the installer supports specific locations for both the directories above:

    ```text
    Usage: install-lsvmi-infra.sh [-r POC_ROOT_DIR] [-R POC_RUNTIME_DIR]

    Install VictoriaMetrics & Grafana under POC_ROOT_DIR, default: HOME/lsvmi-poc,
    using POC_RUNTIME_DIR as runtime dir, default: POC_ROOT_DIR/runtime.
    
    ```

    Optional cleanup:

    ```bash
    
    [[ $(whoami) != "lsvmi" ]]  && id lsvmi 2>/dev/null && sudo -n su - lsvmi

    cd
    rm -rf /tmp/lsvmi-poc-infra*
    
    ```

- install the desired release for OS, architecture and version under the same **PoC** location:

    ```bash

    [[ $(whoami) != "lsvmi" ]]  && id lsvmi 2>/dev/null && sudo -n su - lsvmi

    cd $HOME/lsvmi-poc
    curl \
            -s \
            -L \
            https://github.com/bgp59/linux-stats-victoriametrics-importer/releases/latest/download/lsvmi-linux-amd64.tgz | \
        tar xzf -
    ln -fs lsvmi-linux-amd64 lsvmi

    ```

- start everything:

    ```bash

    [[ $(whoami) != "lsvmi" ]]  && id lsvmi 2>/dev/null && sudo -n su - lsvmi

    cd $HOME/lsvmi-poc      # or POC_ROOT_DIR if custom dir
    ./start-poc.sh          # logs and output under runtime/

    ```

    The relevant 3 processes that should be running are:

  - `victoria-metrics`
  - `grafana`
  - `linux-stats-victoriametrics-importer`

    e.g.

    ```bash

    pgrep -fa 'victoria-metrics|grafana|linux-stats-victoriametrics-importer'

    ```

    should produce the following output:

    ```text

    131 victoria-metrics -storageDataPath data -retentionPeriod 2d -selfScrapeInterval=10s
    156 grafana server
    188 linux-stats-victoriametrics-importer -log-file=log/linux-stats-victoriametrics-importer.log


    ```

- point a browser to <http://localhost:3000> (or http://_poc_linux_host_:3000, if the PoC is not running on the local host) for [Grafana](https://grafana.com/docs/grafana/latest/getting-started/) UI, user: `admin`, password: `lsvmi`. Private browsing works too, e.g. `Chrome` in `Incognito` mode.

- gracefully shutdown **PoC** to save [VictoriaMetrics](https://docs.victoriametrics.com/single-server-victoriametrics/) time series and.or [Grafana](https://grafana.com/docs/grafana/latest/getting-started/) custom dashboards:

    ```bash

    [[ $(whoami) != "lsvmi" ]]  && id lsvmi 2>/dev/null && sudo -n su - lsvmi

    cd $HOME/lsvmi-poc
    ./stop-poc.sh

    ```

Cleanup:

   ```bash

    [[ $(whoami) != "lsvmi" ]]  && id lsvmi 2>/dev/null && sudo -n su - lsvmi

    rm -rf $HOME/lsvmi-poc   # or rm -rf POC_ROOT_DIR POC_RUNTIME_DIR if custom dirs

   ```

### Using A Containerized Solution

- have  [Docker](https://docs.docker.com/get-started/get-docker/) installed
- run the demo image:
  - without persistence (neither [VictoriaMetrics](https://docs.victoriametrics.com/single-server-victoriametrics/) time series nor [Grafana's](https://grafana.com/docs/grafana/latest/getting-started/) custom dashboards will be saved between container restarts):

    ```bash
    platform=linux/amd64

    docker run \
        --platform $platform \
        --rm \
        --detach \
        --publish 3000:3000 \
        --publish 8428:8428 \
        --name lsvmi-demo \
        emypar/lsvmi-demo

    ```

  - with persistence:
    - select a convenient location:

        ```bash

        lsvmi_poc_dir=$HOME/docker/volumes/lsvmi-poc

        ```

    - start the container with a volume:

        ```bash
        platform=linux/amd64

        mkdir -p $lsvmi_poc_dir/$platform/runtime
        docker run \
            --platform $platform \
            --rm \
            --detach \
            --publish 3000:3000 \
            --publish 8428:8428 \
            --name lsvmi-demo \
            --volume $lsvmi_poc_dir/$platform/runtime:/volumes/runtime \
            emypar/lsvmi-demo

        ```

    - log files are now accessible on the host running **Docker**:

        ```bash

        cd $lsvmi_poc_dir/$platform/runtime/victoria-metrics/out
        cat victoria-metrics.err

        ```

        ```bash

        cd $lsvmi_poc_dir/$platform/runtime/grafana/log
        cat grafana.log

        ```

        ```bash

        cd $lsvmi_poc_dir/$platform/runtime/lsvmi/log
        cat linux-stats-victoriametrics-importer.log

        ```

- point a browser to <http://localhost:3000> (or http://_docker_host_:3000, if the PoC is not running on the local host) for [Grafana](https://grafana.com/docs/grafana/latest/getting-started/) UI, user: `admin`, password: `lsvmi`. Private browsing works too, e.g. `Chrome` in `Incognito` mode.

- it is a good practice to stop the container gracefully, really required if the persistent volume is used:

    ```bash

    docker \
        kill --signal='SIGTERM' \
        $(docker ps --filter name=lsvmi-demo --format "{{.ID}}")

    ```

- to log on the container, run:

    ```bash

    docker exec -it lsvmi-demo bash --login

    ```

## Browsing The Reference Dashboards

Once the **PoC** is up and running, the [LSVMI](../README.md) relevant dashboards can be found under `lsvmi-reference` folder. Note that they are  [provisioned dashboards](https://grafana.com/docs/grafana/latest/administration/provisioning/#dashboards) and as such they cannot be modified directly.

<!-- markdownlint-disable MD033 -->
<img src="images/lsvmi-ref-dashes.jpg" alt="lsvmi-reference" width="800">
<!-- markdownlint-enable -->