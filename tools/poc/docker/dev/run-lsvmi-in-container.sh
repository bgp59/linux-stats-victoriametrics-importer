#!/bin/bash --noprofile

this_script=${0##*/}
case "$0" in
    /*|*/*) 
        this_dir=$(cd $(dirname $0) && pwd)
    ;;
    *) 
        this_dir=$(cd $(dirname $(which $0)) && pwd)
    ;;
esac


cd $this_dir || exit 1
./exec-in-container /volumes/linux-stats-victoriametrics-importer/tools/poc/files/lsvmi/run-lsvmi.sh
