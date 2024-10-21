#!/bin/bash

# Create LSVMI release(s):

this_script=${0##*/}

# Common functions, etc:
case "$0" in
    /*|*/*) this_dir=$(dirname $(realpath $0));;
    *) this_dir=$(dirname $(realpath $(which $0)));;
esac


bin_dir=../../bin
poc_dir=../poc
release_dir=../../releases
semver=$(cat ../../semver.txt)
staging_root_dir=../../staging

os_arch_list=linux-amd64

set -e
cd $this_dir

mkdir -p $release_dir

for os_arch in $os_arch_list; do
    staging_dir=$staging_root_dir/lsvmi-$os_arch${semver+-}$semver

    rm -rf $staging_dir
    mkdir -p $staging_dir

    mkdir -p $staging_dir/bin
    rsync -plrtHS \
        $bin_dir/$os_arch \
        $(realpath $bin_dir/$os_arch/linux-stats-victoriametrics-importer) \
        $staging_dir/bin

    rsync -plrtHS $poc_dir/files/lsvmi/ $staging_dir
    archive=$release_dir/$(basename $staging_dir).tgz
    tar czf $archive -C $(dirname $staging_dir) $(basename $staging_dir)
    rm -rf $staging_dir
    echo "$this_script: $(realpath $archive) created"
done

