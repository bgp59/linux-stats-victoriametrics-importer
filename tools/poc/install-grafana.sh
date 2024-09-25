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

usage="
Usage: $this_script [-r POC_ROOT_DIR]
Install Grafana under POC_ROOT_DIR, default: $lsvmi_poc_root_dir
"

case "$1" in
    -h*|--h*)
        echo >&2 "$usage"
        exit 1
        ;;
    -r*|--root*)
        shift
        lsvmi_poc_root_dir="$1"
        shift
        ;;
esac

if [[ -z "$lsvmi_poc_root_dir" ]]; then
    echo >&2 "$usage"
fi

set -ex
mkdir -p $lsvmi_poc_root_dir
lsvmi_poc_root_dir=$(realpath $lsvmi_poc_root_dir)

cd $this_dir/files
rsync -plrSH common.sh grafana $lsvmi_poc_root_dir
$lsvmi_poc_root_dir/grafana/download-grafana.sh
