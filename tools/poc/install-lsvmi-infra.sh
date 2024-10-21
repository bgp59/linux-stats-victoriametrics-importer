#!/bin/bash

this_script=${0##*/}

# Common functions, etc:
case "$0" in
    /*|*/*) this_dir=$(dirname $(realpath $0));;
    *) this_dir=$(dirname $(realpath $(which $0)));;
esac

if [[ -n "$HOME" && -d "$HOME" ]]; then
    lsvmi_poc_root_dir="$HOME/lsvmi-poc"
else
    lsvmi_poc_root_dir="/tmp/${USER:-$UID}/lsvmi-poc"
fi

lsvmi_poc_runtime_dir=

usage="
Usage: $this_script [-r POC_ROOT_DIR] [-R POC_RUNTIME_DIR]

Install VictoriaMetrics & Grafana under POC_ROOT_DIR, default: $lsvmi_poc_root_dir,
using POC_RUNTIME_DIR as runtime dir, default: POC_ROOT_DIR/runtime.
"

while [[ $# -gt 0 ]]; do
    case "$1" in
        -h*|--h*)
            echo >&2 "$usage"
            exit 1
            ;;
        -r*|--root*)
            shift
            lsvmi_poc_root_dir="$1"
            ;;
        -R|--runtime*)
            shift
            lsvmi_poc_runtime_dir="$1"
            ;;
    esac
    shift
done

if [[ -z "$lsvmi_poc_root_dir" ]]; then
    echo >&2 "$usage"
    exit 1
fi

set -ex
mkdir -p $lsvmi_poc_root_dir
lsvmi_poc_root_dir=$(realpath $lsvmi_poc_root_dir)

cd $this_dir/files
rsync -plrSH base/ $lsvmi_poc_root_dir
rsync -plrSH update/ $lsvmi_poc_root_dir

cd $lsvmi_poc_root_dir

./victoria-metrics/download-victoria-metrics.sh
./victoria-metrics/create-victoria-metrics-runtime-symlinks.sh ${lsvmi_poc_runtime_dir:-../runtime}/victoria-metrics

./grafana/download-grafana.sh
./grafana/create-grafana-runtime-symlinks.sh ${lsvmi_poc_runtime_dir:-../runtime}/grafana


