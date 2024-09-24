#! /bin/bash

this_script=${0##*/}
case "$0" in
    /*|*/*) this_dir=$(cd $(dirname $0) && pwd);;
    *) this_dir=$(cd $(dirname $(which $0)) && pwd);;
esac
. $this_dir/common.sh

case "$this_script" in
    start*)
        set -e
        check_os_arch 
        check_if_running victoria-metrics
        (
            set -ex
            cd $this_dir
            mkdir -p log out victoria-metrics-data
            victoria-metrics \
                -storageDataPath victoria-metrics-data \
                -retentionPeriod 2d \
                -selfScrapeInterval=10s \
                > out/victoria-metrics.out 2>out/victoria-metrics.err < /dev/null &
        )
    ;;
    stop*)
        kill_wait_proc victoria-metrics
    ;;
esac
