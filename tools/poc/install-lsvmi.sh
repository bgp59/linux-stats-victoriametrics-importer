#! /bin/bash --noprofile

this_script=${0##*/}

if [[ -n "$HOME" && -d "$HOME" ]]; then
    lsvmi_poc_root_dir="$HOME/lsvmi-poc"
else
    lsvmi_poc_root_dir="/tmp/${USER:-$UID}/lsvmi-poc"
fi

usage="
Usage: $this_script [-d ROOT_DIR]

Install PoC LSVMI at ROOT_DIR, default: $lsvmi_poc_root_dir
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
    -d*|--root*)
        shift
        lsvmi_poc_root_dir="$1"
        shift
        ;;
esac

if [[ -z "$lsvmi_poc_root_dir" ]]; then
    echo >&2 "$usage"
fi

set -e
mkdir -p $lsvmi_poc_root_dir/lsvmi
dst_dir=$(realpath $lsvmi_poc_root_dir/lsvmi)

set -x
cd $this_dir
rsync -plrSH files/*-lsvmi.sh files/lsvmi-config.yaml $dst_dir/
mkdir -p $dst_dir/bin
rsync -plrSH $project_root_dir/bin/ $dst_dir/bin

