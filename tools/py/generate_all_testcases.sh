#!/bin/bash --noprofile

this_script=${0##*/}

usage="
Usage: $this_script [ARG....]

Optional args to be passed to generate_*_testcases.py
"

case "$0" in
    /*|*/*) this_dir=$(cd $(dirname $0) && pwd);;
    *) this_dir=$(cd $(dirname $(which $0)) && pwd);;
esac

if [[ -z "$this_dir" ]]; then
    echo >&2 "$this_script: cannot infer location from invocation"
    exit 1
fi

case "$1" in
    -h|--help) echo >&2 "$usage"; exit 1;;
esac

set -e
cd $this_dir
for s in $(ls -1 generate_*_testcases.py); do
    ./$s $*
done


