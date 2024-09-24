#!/bin/bash

vm_ver=1.90.0
vm_os=linux
vm_arch=amd64

this_script=${0##*/}

if [[ -n "$HOME" && -d "$HOME" ]]; then
    lsvmi_poc_root_dir="$HOME/lsvmi-poc"
else
    lsvmi_poc_root_dir="/tmp/${USER:-$UID}/lsvmi-poc"
fi

usage="
Usage: $this_script [-r ROOT_DIR]

Install PoC VictoriaMetrics at ROOT_DIR, default: $lsvmi_poc_root_dir
"

case "$0" in
    /*|*/*) this_dir=$(dirname $(realpath $0));;
    *) this_dir=$(dirname $(realpath $(which $0)));;
esac
project_root_dir=$(realpath $this_dir/../..)

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

set -e
mkdir -p $lsvmi_poc_root_dir
dst_dir=$(realpath $lsvmi_poc_root_dir)


set -x

mkdir -p $dst_dir/bin/$vm_os-$vm_arch
cd $dst_dir/bin/$vm_os-$vm_arch

curl -s -L https://github.com/VictoriaMetrics/VictoriaMetrics/releases/download/v$vm_ver/vmutils-$vm_os-$vm_arch-v$vm_ver.tar.gz | tar xzf -
curl -s -L https://github.com/VictoriaMetrics/VictoriaMetrics/releases/download/v$vm_ver/victoria-metrics-$vm_os-$vm_arch-v$vm_ver.tar.gz | tar xzf -

ln -fs victoria-metrics-prod victoria-metrics
ln -fs vmagent-prod vmagent

cd $this_dir
rsync -plrSH files/common.sh files/*-victoria-metrics.sh $dst_dir/
