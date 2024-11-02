# Developer Guide

## Pre-requisites

* [go](https://go.dev/doc/install) >= `1.21.6`
* [Python](https://www.python.org/) \>= `3.8` w/ the following pre-requisites:

    ```bash

    cd tools/py
    ./py_prerequisites.sh

    ```

* (strongly) suggested [VSCode](https://code.visualstudio.com/download) \>= `1.95.0` with the following extensions:

  * [Docker](https://marketplace.visualstudio.com/items?itemName=ms-azuretools.vscode-docker)
  * [Even Better TOML](https://marketplace.visualstudio.com/items?itemName=tamasfe.even-better-toml)
  * [Go](https://marketplace.visualstudio.com/items?itemName=golang.Go)
  * [isort](https://marketplace.visualstudio.com/items?itemName=ms-python.isort)
  * [Markdown TOC & Chapter Number](https://marketplace.visualstudio.com/items?itemName=TakumiI.markdown-toc-num)
  * [markdownlint](https://marketplace.visualstudio.com/items?itemName=DavidAnson.vscode-markdownlint)
  * [Python](https://marketplace.visualstudio.com/items?itemName=ms-python.python)
  * [Rewrap](https://marketplace.visualstudio.com/items?itemName=stkb.rewrap)
  * [Sort lines](https://marketplace.visualstudio.com/items?itemName=Tyriar.sort-lines)

## Coding Style

There is no git [`pre-commit` hook](https://git-scm.com/book/ms/v2/Customizing-Git-Git-Hooks) in place yet.

### Go

[VSCode](https://code.visualstudio.com/download) users will benefit from the [Go](https://marketplace.visualstudio.com/items?itemName=golang.Go) extension in real time.

Additionally [gofmt](https://pkg.go.dev/cmd/gofmt) could be used.

### Python

For now the combination of [autoflake](https://pypi.org/project/autoflake/), [isort](https://pypi.org/project/isort/) and [black](https://pypi.org/project/black/) is applied, in that order, by an external script: [tools/py/py_format.sh](../tools/py/py_format.sh).

### Others (Shell, etc)

Relying on [VSCode](https://code.visualstudio.com/download)

## Building, Testing And Releasing The Code

### Semver

Defined in [semver.txt](../semver.txt), it will be used for both tagging and for the version info included in the binary (`--version` command line arg)

### Build

  ```bash

  ./go-build

  ```

It will create:

* `bin/linux-stats-victoriametrics-importer-<GOOS>-<GOARCH>-<SEMVER>`
* `bin/<GOOS>-<GOARCH>/linux-stats-victoriametrics-importer` `->` above

for all `GOOS`, `GOARCH` defined in [go-os-arch.targets](../go-os-arch.targets) file.

The `buildinfo/buildinfo.go` file is generating dynamically at build time and it is git ignored.

### Tests
