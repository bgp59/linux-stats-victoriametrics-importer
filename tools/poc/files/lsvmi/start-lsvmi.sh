#!/bin/bash --noprofile

this_script=${0##*/}
case "$0" in
    /*|*/*) this_dir=$(cd $(dirname $0) && pwd);;
    *) this_dir=$(cd $(dirname $(which $0)) && pwd);;
esac

set -e
. $this_dir/../common.sh
check_os_arch

for d in /volumes/linux-stats-victoriametrics-importer $(realpath $this_dir/../../../..); do
    if [[ -x $d/bin/$os_arch ]]; then
        ls -1 $d/bin/$os_arch >/dev/null # Hack required for run-lsvmi-in-container.sh (stale cache?!)
        export PATH="$d/bin/$os_arch:$PATH"
        break
    fi
done


case "$this_script" in
    start*)         
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
        set -x
        cd $this_dir
        exec linux-stats-victoriametrics-importer -log-file=stderr "$@"
    ;;
    stop*)
        kill_wait_proc linux-stats-victoriametrics-importer
    ;;
esac
