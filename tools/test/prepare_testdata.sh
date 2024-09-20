#! /bin/bash --noprofile

# Prepare testdata/:

case "$0" in
    /*|*/*) this_dir=$(cd $(dirname $0) && pwd);;
    *) this_dir=$(cd $(dirname $(which $0)) && pwd);;
esac

tools_dir=$(dirname $this_dir)
root_dir=$(dirname $tools_dir)

set -ex
cd $root_dir
tar xzf testdata.tgz
cd $this_dir
./py_prerequisites.sh
./generate_pid_tid_list_cache_test_cases.py
./generate_metrics_test_cases.py
