#! /bin/bash --noprofile

# Archive testdata/:

case "$0" in
    /*|*/*) this_dir=$(cd $(dirname $0) && pwd);;
    *) this_dir=$(cd $(dirname $(which $0)) && pwd);;
esac

tools_dir=$(dirname $this_dir)
root_dir=$(dirname $tools_dir)

set -ex
cd $root_dir
tar --exclude=testcases/ --exclude=.DS_Store -czf testdata.tgz testdata
