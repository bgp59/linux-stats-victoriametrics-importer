#!/bin/bash

# Create LSVMI release(s):
this_script=${0##*/}
    
usage="
Usage: $this_script [-f|--force]

Apply SEMVER tag locally and to the remote. Requires
a clean git status. Use --force to reapply the tag.

"


# All paths below are relative to project's root dir:
semver_file=semver.txt
tools_dir=tools
tools_devutils_dir=$tools_dir/devutils

force=
case "$1" in
    -h|--h*)
        echo >&2 "$usage"
        exit 1
        ;;
    -f|--force)
        force="--force"
        shift
        ;;
esac

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
    echo >&2 "$this_script: missing mandatory $semver"
    exit 1
fi

# Must be in in proper git state:
if ! check-git-state.sh; then
    echo >&2 "$this_script: cannot continue"
    exit 1
fi

git tag $force $semver
git push $force origin tag $semver 


 

