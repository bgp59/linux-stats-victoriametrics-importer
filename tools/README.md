# Tools For Testing, Development And Showcasing

**Note:** All relative paths next are relative to the root of this project, `linux-stats-victoriametrics-importer`.

## Prerequisites

Have python 3.8 or better executable as python3 and run:

    cd tools/py
    ./py_prerequisites.sh

## Testing Data

Testing data consists of recorded real life `/proc` files, used for testing `procfs` parsers and of test cases in JSON format used for testing `lsvmi` metrics generators. The latter are created by the set of `tools/test/generate*_test_cases.py` scripts. 

The resulting `testdata` directory is git ignored, so it should be created before `go test` can be invoked:

    cd tools/test
    ./prepare_testdata.sh

## Support During Development

Running `linux-stats-victoriametrics-importer` requires a Linux environment to run, a [VictoriaMetrics](https://docs.victoriametrics.com/single-server-victoriametrics/) infra to send metrics to and a [Grafana](https://grafana.com/docs/grafana/latest/setup-grafana/) server for time series visualization.

In practice the above should run on the same host or container. Depending on whether the development is carried on a Linux host or under a different OS, the following solutions are suggested.

### Developing On A Linux Host

* run once:

        cd tools/poc
        ./install-local-poc.sh

    It will install **VictoriaMetrics** and **Grafana** under `$HOME/lsvmi-poc` with a runtime directory (for logs and databases) under `$HOME/lsvmi-poc/runtime`. The installation and runtime directory can be changed via command line args:

        ./install-local-poc.sh -h
        Usage: install-local-poc.sh [-r POC_ROOT_DIR] [-R POC_RUNTIME_DIR]

        Install VictoriaMetrics & Grafana under POC_ROOT_DIR, default: $HOME/lsvmi-poc,
        using POC_RUNTIME_DIR as runtime dir, default: POC_ROOT_DIR/runtime.

* to start the supporting infra:

        cd $HOME/lsvmi-poc      # or the alternate POC_ROOT_DIR above
        ./start-poc.sh

* to stop the supporting infra:

        cd $HOME/lsvmi-poc      # or the alternate POC_ROOT_DIR above
        ./stop-poc.sh

* to run a freshly built [^1] binary:

        cd tools/poc/files/lsvmi
        ./run-lsvmi.sh          # log -> stderr

### Developing Under A Different OS

This project was developed on an Intel MacBook Pro running macOS. While **go** can cross-compile for `linux/amd64`, the environment for  **VictoriaMetrics**, **Grafana** and `linux-stats-victoriametrics-importer` will run a in [Docker](https://docs.docker.com/get-started/get-docker/) container.

The next steps assume that **Docker** is installed and running:

* build the container image, once:

        cd tools/poc/docker/dev
        ./build-container

* to start the container, running **VictoriaMetrics**, **Grafana**:

        cd tools/poc/docker/dev
        ./start-container

* to stop the container:

        cd tools/poc/docker/dev
        ./stop-container

* to run a freshly built [^1] binary:

        cd tools/poc/docker/dev
        ./run-lsvmi-in-container.sh


[^1]: to build a binary, run:

    cd .../linux-stats-victoriametrics-importer # project root, that is
    ./build
