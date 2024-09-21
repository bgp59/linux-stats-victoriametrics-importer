#!/bin/bash --noprofile

this_script=${0##*/}
case "$0" in
    /*|*/*) this_dir=$(cd $(dirname $0) && pwd);;
    *) this_dir=$(cd $(dirname $(which $0)) && pwd);;
esac

os=$(uname -s | tr A-Z a-z)
case "$os" in
    linux) go_os="$os";;
    *)
        echo >&2 "$this_script: $os unsupported OS"
        exit 1
    ;;
esac

arch=$(uname -m | tr A-Z a-z)
case "$arch" in
    x86_64) go_arch="amd64";;
    *)
        echo >&2 "$this_script: $arch unsupported ARCH"
        exit 1
    ;;   
esac
export PATH="$this_dir/bin/$go_os-$go_arch:$this_dir/bin:$PATH"


case "$this_script" in
    start*)
        (
            set -ex
            cd $this_dir
            mkdir -p log out
            exec linux-stats-victoriametrics-importer \
                -log-file=log/linux-stats-victoriametrics-importer.log \
                >out/linux-stats-victoriametrics-importer.out \
                2>out/linux-stats-victoriametrics-importer.err \
                < /dev/null &
        )
    ;;
    run*)
        set -ex
        cd $this_dir
        exec linux-stats-victoriametrics-importer -log-file=stderr
    ;;
    stop*)
        set -ex
        pkill -f linux-stats-victoriametrics-importer
    ;;
esac
