#! /bin/bash

# Apply job code formatting tools:

case "$0" in
    /*|*/*) 
        this_dir=$(cd $(dirname $0) && pwd)
        real_dir=$(dirname $(realpath $0))
    ;;
    *) 
        this_dir=$(cd $(dirname $(which $0)) && pwd)
        real_dir=$(dirname $(realpath $(which $0)))
    ;;
esac

set -e
cd $this_dir
set -x
autoflake --in-place --remove-all-unused-imports --ignore-init-module-imports --recursive .
isort --settings-path $real_dir .
black .

