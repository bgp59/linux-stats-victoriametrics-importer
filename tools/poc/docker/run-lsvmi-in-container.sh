#!/bin/bash --noprofile

this_script=${0##*/}
case "$0" in
    /*|*/*) 
        this_dir=$(dirname $(realpath $0))
    ;;
    *) 
        this_dir=$(dirname $(realpath $(which $0)))
    ;;
esac


cd $this_dir || exit 1
./exec-in-container /volumes/linux-stats-victoriametrics-importer/tools/poc/lsvmi/run-lsvmi.sh
