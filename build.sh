#! /bin/bash --noprofile

case "$0" in
    /*|*/*) this_dir=$(cd $(dirname $0) && pwd);;
    *) this_dir=$(cd $(dirname $(which $0) && pwd));;
esac

if [[ -z "$this_dir" ]]; then
    echo >&2 "Cannot infer dir for $0"
    exit 1
fi

do_build() {
    (
        set -e
        out_dir=bin/$(go env GOOS)-$(go env GOARCH)
        out_file=$(basename $(pwd))
        cd $this_dir
        if [[ -x pre-build ]]; then
            ./pre-build
        fi
        semver_suffix=
        if [[ -r semver.txt ]]; then
            semver_suffix=.$(cat semver.txt)
        fi
        (
            set -x
            mkdir -p $out_dir
            go build -o $out_dir/$out_file$semver_suffix
        )
        if [[ -n "$semver_suffix" ]]; then
            (set -x; ln -fs $out_file$semver_suffix $out_dir/$out_file)
        fi
        if [[ -x post-build ]]; then
            ./post-build
        fi
    )
}

set -e
cd $this_dir
do_build
native_goos=$(go env GOOS)
native_goarch=$(go env GOARCH)
if [[ "$native_goos" != linux || "$native_goarch" != "amd64" ]]; then
    GOOS=linux GOARCH=amd64 do_build
fi



