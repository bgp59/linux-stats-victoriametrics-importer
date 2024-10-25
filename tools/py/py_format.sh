#! /bin/bash

# Apply Python code formatting tools:

this_script=${0##*/}

case "$0" in
    /*|*/*) script_dir=$(dirname $(realpath $0));;
    *) script_dir=$(dirname $(realpath $(which $0)));;
esac

case "$1" in
    -h|--h*)
        echo >&2 "Usage: $this_script DIR ..."
        exit 1
    ;;
esac

set -e
for d in ${@:-.}; do
    (
        set -x
        autoflake --in-place --remove-all-unused-imports --ignore-init-module-imports --recursive $d
        isort --settings-path $script_dir $d
        black $d
    )
done


