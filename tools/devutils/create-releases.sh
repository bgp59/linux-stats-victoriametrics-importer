#!/bin/bash

# Create LSVMI release(s):
this_script=${0##*/}

# All paths below are relative to project's root dir:
bin_dir=bin
os_arch_file=go-os-arch.targets
release_root_dir=releases
semver_file=semver.txt
staging_dir=staging
tools_dir=tools
tools_devutils_dir=$tools_dir/devutils
tools_poc_dir=$tools_dir/poc

# Common functions, etc:
case "$0" in
    /*|*/*) this_dir=$(dirname $(realpath $0));;
    *) this_dir=$(dirname $(realpath $(which $0)));;
esac
project_root_dir=$(realpath $this_dir/../..)


set -e
set -x; cd $project_root_dir; set +x
export PATH="$(realpath $tools_devutils_dir)${PATH+:}${PATH}"

# Must have semver:
semver=$(cat $semver_file)
if [[ -z "$semver" ]]; then
    echo >&2 "$this_script - Missing mandatory semver"
    exit 1
fi

# Must be in proper git state and have semver tag applied at the HEAD:
if ! check-git-state.sh $semver; then
    echo >&2 "$this_script: cannot continue"
    exit 1
fi

# Ensure the latest build:
(set -x; ./go-build)

# Proceed w/ the release tarballs:
release_dir="$release_root_dir/$semver"
rm -rf $release_dir
mkdir -p $release_dir

# The executable:
list-os-arch.sh $os_arch_file | while read os arch; do
    os_arch="$os-$arch"
    release_prefix="lsvmi-$os_arch"
    release_subdir="$release_prefix-$semver"
    release_staging_dir="$staging_dir/$release_subdir"

    rm -rf $release_staging_dir
    mkdir -p $release_staging_dir

    mkdir -p $release_staging_dir/bin
    cp -p $bin_dir/$os_arch/linux-stats-victoriametrics-importer $release_staging_dir/bin

    rsync -plrtHS \
        --exclude=log/ \
        --exclude=out/ \
        --exclude-from=$this_dir/release-rsync.exclude \
        $tools_poc_dir/files/lsvmi/ \
        $release_staging_dir

    cp -p relnotes.txt lsvmi/lsvmi-config-reference.yaml $release_staging_dir

    archive=$release_dir/$release_prefix.tgz
    ln -fs $release_subdir $staging_dir/$release_prefix
    tar czf $archive -C $staging_dir $release_subdir $release_prefix
    rm -rf $release_staging_dir $staging_dir/$release_prefix
    echo "$this_script: $(realpath $archive) created"
done

# PoC supporting infra:
release_prefix="lsvmi-poc-infra"
release_subdir="$release_prefix-$semver"
release_staging_dir="$staging_dir/$release_subdir"

rm -rf $release_staging_dir
mkdir -p $release_staging_dir

rsync \
    -plrtHS \
    --exclude=lsvmi/ \
    --exclude-from=$this_dir/release-rsync.exclude \
    $tools_poc_dir/files \
    $tools_poc_dir/install-lsvmi-infra.sh \
    $release_staging_dir

cp -p relnotes.txt $release_staging_dir

archive=$release_dir/$release_prefix.tgz
ln -fs $release_subdir $staging_dir/$release_prefix
tar czf $archive -C $staging_dir $release_subdir $release_prefix
rm -rf $release_staging_dir $staging_dir/$release_prefix
echo "$this_script: $(realpath $archive) created"
