# Developer Guide

<!-- TOC tocDepth:2..3 chapterDepth:2..6 -->

- [Pre-requisites](#pre-requisites)
- [Coding Style](#coding-style)
  - [Go](#go)
  - [Python](#python)
  - [Others (Shell, etc)](#others-shell-etc)
- [Building The Code](#building-the-code)
  - [Semver](#semver)
  - [Build](#build)
- [Testing The Code](#testing-the-code)
- [Support For Running The Code During Development](#support-for-running-the-code-during-development)
  - [Developing On A Linux Host](#developing-on-a-linux-host)
  - [Developing Under A Different OS](#developing-under-a-different-os)
- [Creating Releases](#creating-releases)
  - [Prerequisites](#prerequisites)
  - [The Actual Releases](#the-actual-releases)

<!-- /TOC -->

## Pre-requisites

- [Go](https://go.dev/doc/install) >= `1.21.6`
- [Python](https://www.python.org/) \>= `3.8` w/ the following pre-requisites:

    ```bash

    cd tools/py
    ./py_prerequisites.sh

    ```

- (strongly) suggested [VSCode](https://code.visualstudio.com/download) \>= `1.95.0`:
  
  - recommended extensions:

    - [Code Spell Checker](https://marketplace.visualstudio.com/items?itemName=streetsidesoftware.code-spell-checker)
    - [Docker](https://marketplace.visualstudio.com/items?itemName=ms-azuretools.vscode-docker)
    - [Even Better TOML](https://marketplace.visualstudio.com/items?itemName=tamasfe.even-better-toml)
    - [Go](https://marketplace.visualstudio.com/items?itemName=golang.Go)
    - [isort](https://marketplace.visualstudio.com/items?itemName=ms-python.isort)
    - [Markdown TOC & Chapter Number](https://marketplace.visualstudio.com/items?itemName=TakumiI.markdown-toc-num)
    - [markdownlint](https://marketplace.visualstudio.com/items?itemName=DavidAnson.vscode-markdownlint)
    - [Python](https://marketplace.visualstudio.com/items?itemName=ms-python.python)
    - [Rewrap](https://marketplace.visualstudio.com/items?itemName=stkb.rewrap)
    - [Sort lines](https://marketplace.visualstudio.com/items?itemName=Tyriar.sort-lines)

  - recommended settings: see [.vscode-ref/settings.json](../.vscode-ref/settings.json) to prime `.vscode/` (git ignored) or merge its exclusion lists. The latter are very important, otherwise various plugins will become runaway when attempting to scan `testdata/`.

## Coding Style

There is no git [`pre-commit` hook](https://git-scm.com/book/ms/v2/Customizing-Git-Git-Hooks) in place yet.

### Go

[VSCode](https://code.visualstudio.com/download) users will benefit from the [Go](https://marketplace.visualstudio.com/items?itemName=golang.Go) extension in real time.

Additionally [gofmt](https://pkg.go.dev/cmd/gofmt) could be used.

### Python

For now the combination of [autoflake](https://pypi.org/project/autoflake/), [isort](https://pypi.org/project/isort/) and [black](https://pypi.org/project/black/) is applied, in that order, by an external script: [tools/py/py_format.sh](../tools/py/py_format.sh).

### Others (Shell, etc)

Relying on [VSCode](https://code.visualstudio.com/download)

## Building The Code

### Semver

Defined in [semver.txt](../semver.txt), it will be used for both tagging and for the version info included in the binary (`--version` command line arg)

### Build

  ```bash

  ./go-build

  ```

It will create:

- `bin/linux-stats-victoriametrics-importer-<GOOS>-<GOARCH>-<SEMVER>`
- `bin/<GOOS>-<GOARCH>/linux-stats-victoriametrics-importer` `->` above

for all `GOOS`, `GOARCH` defined in [go-os-arch.targets](../go-os-arch.targets) file.

The `buildinfo/buildinfo.go` file is generating dynamically at build time and it is git ignored.

## Testing The Code

The test data consists of:

- [procfs](https://linux.die.net/man/5/proc") files (recoded real-life examples or artificially created for edge cases), used for testing [procfs](../procfs) parsers. They are all archived in [testdata.tgz](../testdata.tgz) file.
- JSON format test cases, used for testing [lsvmi](../lsvmi) metrics generators. They are created by the set of [tools/test](../tools/test) `generate*_test_cases.py` scripts.

All of the above are placed under `testdata` directory which is git ignored, so it should be created before `go test` can be invoked:

  ```bash

    cd tools/test
    ./prepare_testdata.sh

  ```

If new [procfs](https://linux.die.net/man/5/proc") files are added to the `testdata` collection, the archive has to be regenerated:

  ```bash

  cd tools/test
  ./archive_testdata.sh

  ```

and committed into the git repo.

## Support For Running The Code During Development

Running [LSVMI](../README.md) requires a Linux environment to run, a [VictoriaMetrics](https://docs.victoriametrics.com/single-server-victoriametrics/) infra to send metrics to and a [Grafana](https://grafana.com/docs/grafana/latest/setup-grafana/) server for time series visualization.

In practice the above should run on the same host or container. Depending on whether the development is carried on a Linux host or under a different OS, the following solutions are suggested.

### Developing On A Linux Host

- run once:

  ```bash

    cd tools/poc
    ./install-lsvmi-infra.sh

  ````

  It will install [VictoriaMetrics](https://docs.victoriametrics.com/single-server-victoriametrics/) and [Grafana](https://grafana.com/docs/grafana/latest/setup-grafana/)  under `$HOME/lsvmi-poc` with a runtime directory (for logs and databases) under `$HOME/lsvmi-poc/runtime`. The installation and runtime directory can be changed via command line args:

  ```bash

    ./install-lsvmi-infra.sh -h
    Usage: install-lsvmi-infra.sh [-r POC_ROOT_DIR] [-R POC_RUNTIME_DIR]

    Install VictoriaMetrics & Grafana under POC_ROOT_DIR, default: $HOME/lsvmi-poc,
    using POC_RUNTIME_DIR as runtime dir, default: POC_ROOT_DIR/runtime.

  ```

- to start the supporting infra:

  ```bash

    cd $HOME/lsvmi-poc      # or the alternate POC_ROOT_DIR above
    ./start-poc.sh

  ```

- to run a freshly built binary:

  ```bash

    cd tools/poc/files/lsvmi
    ./run-lsvmi.sh          # log -> stderr

  ```

- to stop the supporting infra:

  ```bash

    cd $HOME/lsvmi-poc      # or the alternate POC_ROOT_DIR above
    ./stop-poc.sh

  ```

### Developing Under A Different OS

This project was developed on `MacBook (Intel)`/`MacAir (Apple Chip)` running
`macOS`. While [Go](https://go.dev/doc/install) can cross-compile for
`linux/amd64`/`linux/arm64`, the environment for
[VictoriaMetrics](https://docs.victoriametrics.com/single-server-victoriametrics/)
and [Grafana](https://grafana.com/docs/grafana/latest/setup-grafana/) and
[LSVMI](../README.md) will run in a
[Docker](https://docs.docker.com/get-started/get-docker/) container.

The next steps assume that [Docker](https://docs.docker.com/get-started/get-docker/) is installed and running:

- build the container multi-platform image, once:

  ```bash

    cd tools/poc/docker/dev
    ./build-image multi

  ```

  **NOTE!** The state, logs and output for [VictoriaMetrics](https://docs.victoriametrics.com/single-server-victoriametrics/) and [Grafana](https://grafana.com/docs/grafana/latest/setup-grafana/) will be persisted under `tools/poc/docker/dev/volumes/linux/ARCH/runtime` directory on the host. If a different location is desired, run once, before the container is started for the first time:

  ```bash

    alternate_runtime_dir=...  # set your location here

    cd tools/poc/docker/dev

    for arch in amd64 arm64; do
      mkdir -p $alternate_runtime_dir/$arch/runtime
      mkdir -p volumes/linux/$arch
      ln -fs $alternate_runtime_dir/$arch/runtime volumes/$arch/runtime
    done
  ```

- to start [VictoriaMetrics](https://docs.victoriametrics.com/single-server-victoriametrics/) and [Grafana](https://grafana.com/docs/grafana/latest/setup-grafana/) on the container:

  ```bash

    cd tools/poc/docker/dev
    ./start-multi-container

  ```

  This will start the container for the **preferred** platform, which is the one corresponding for the first entry in [go-os-arch.targets](../go-os-arch.targets); currently that is `linux/amd64`.

  If a different platform is desired, specify it at container start-up time, e.g. running for `linux/arm64`:

  ```bash

    cd tools/poc/docker/dev
    ./start-multi-container --platform linux/arm64

  ```

- to run a freshly built binary:

  ```bash

    cd tools/poc/docker/dev
    ./run-lsvmi-in-container.sh

  ```

- to stop the container:

  ```bash

    cd tools/poc/docker/dev
    ./stop-container

  ```

## Creating Releases

### Prerequisites

- [relnotes.txt](../relnotes.txt) updated (not checked or enforced as of yet)
- on the `main` branch, in a clean state
- the [semver.txt](../semver.txt) tag applied, which can be achieved by running:

  ```bash

    ./apply-semver-tag

  ```

  which will check the git state beforehand. Should the tag have existed before, it can be moved by `--force`:

    ```bash

    ./apply-semver-tag --force

  ```

### The Actual Releases

Run:

  ```bash

  ./create-git-releases

  ```

It will check for pre-requisites and if met, it will run an `os-build` command and it will create `.tgz` files under `releases/` dir (git ignored):

- `releases/SEMVER/lsvmi-GOOS-GOARCH.tgz`: one for for each `GOOS`, `GOARCH` pair
- `releases/SEMVER/lsvmi-poc-infra.tgz`: in support of [LSVMI Proof Of Concept Demo](poc.md), [Using A Linux Server](poc.md#using-a-linux-server)

e.g.:

  ```text

  releases/v0.0.2/lsvmi-linux-amd64.tgz
  releases/v0.0.2/lsvmi-linux-arm64.tgz
  releases/v0.0.2/lsvmi-poc-infra.tgz

  ```

## Grafana Support

The [PoC](poc.md) deployment includes [LSVMI](../README.md) specific [provisioned dashboards](https://grafana.com/docs/grafana/latest/administration/provisioning/#dashboards) under the `lsvmi-reference` folder. Such dashboards cannot be edited in place, but rather a copy has to be made first.

Assuming that the development support [Grafana](https://grafana.com/docs/grafana/latest/setup-grafana/) is running, the following steps describe how to update or create such dashboards:

- prepare a copy:

  ```bash

        cd tools/grafana
        ./prepare_grafana_wip_dashboard.py DASHBOARD_TITLE
        # .e.g.
        ./prepare_grafana_wip_dashboard.py internal_metrics_ref

  ```

  The copy will be renamed `internal_metrics_ref (WIP)` and it will be located under the `General` folder.

- once the copy has been modified in the [Grafana](https://grafana.com/docs/grafana/latest/setup-grafana/) UI <http://localhost:3000>, save it to the project:

  ```bash

      cd tools/grafana
      ./save_grafana_wip_dashboard.py DASHBOARD_TITLE
      # .e.g.
      ./save_grafana_wip_dashboard.py internal_metrics_ref

  ```

- rebuild the  [PoC](poc.md) and restart the container, if in use.

- for new dashboards, create them under the `General` folder with a name ending in `' (WIP)'` and save the to the project as above
