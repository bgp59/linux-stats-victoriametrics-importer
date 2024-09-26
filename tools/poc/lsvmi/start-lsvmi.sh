#!/bin/bash --noprofile

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
        check_if_not_running linux-stats-victoriametrics-importer
        (
            set -x
            cd $this_dir
            create_dir_maybe_symlink log out
            exec linux-stats-victoriametrics-importer \
                -log-file=log/linux-stats-victoriametrics-importer.log \
                "$@" \
                >out/linux-stats-victoriametrics-importer.out \
                2>out/linux-stats-victoriametrics-importer.err \
                < /dev/null &
        )
    ;;
    run*)
        set -e
        check_os_arch 
        set -x
        cd $this_dir
        exec linux-stats-victoriametrics-importer -log-file=stderr "$@"
    ;;
    stop*)
        kill_wait_proc linux-stats-victoriametrics-importer
    ;;
esac
