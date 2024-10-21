#!/bin/bash

# Create a release w/ infra installer for PoC

this_script=${0##*/}

# Common functions, etc:
case "$0" in
    /*|*/*) this_dir=$(dirname $(realpath $0));;
    *) this_dir=$(dirname $(realpath $(which $0)));;
esac

poc_dir=../poc
release_dir=../../releases
staging_dir=../../staging/lsvmi-infra-install

set -e
cd $this_dir

rm -rf $staging_dir
mkdir -p $staging_dir

rsync -plrtHS --exclude=lsvmi/ $poc_dir/files $poc_dir/install-lsvmi-infra.sh $staging_dir
archive=$release_dir/$(basename $staging_dir).tgz
tar czf $archive -C $(dirname $staging_dir) $(basename $staging_dir)
rm -rf $staging_dir
echo "$this_script: $(realpath $archive) created"
