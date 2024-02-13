#!/bin/bash --noprofile

# Run vm_import_endpoint in run/pause loop:

this_script=${0##*/}

usage="
Usage: $this_script RUN PAUSE [ARG...]
Run vm_import_endpoint ARG... in a loop, RUN sec active, PAUSE down
"

case "$1" in
    ""|-h|--h*) echo >&2 "$usage"; exit 1;;
    *) run="$1"; shift;;
esac
case "$1" in
    ""|-h|--h*) echo >&2 "$usage"; exit 1;;
    *) pause="$1"; shift;;
esac

case "$0" in
    /*|*/*) this_dir=$(cd $(dirname $0) && pwd);;
    *) this_dir=$(cd $(dirname $(which $0) && pwd));;
esac

if [[ -z "$this_dir" ]]; then
    echo >&2 "Cannot infer dir for $0"
    exit 1
fi

cd $this_dir/bin/$(go env GOOS)-$(go env GOARCH) || exit 1


cleanup() {
    (set -x; pkill -P $$ -f "vm_import_endpoint $@")
    exit 1
}

trap cleanup 1 2 3 15

while [[ true ]]; do
    (set -x; exec ./vm_import_endpoint "$@") &
    sleep $run
    (set -x; pkill -P $$ -f "vm_import_endpoint $@"; sleep $pause)
done
