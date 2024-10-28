#!/bin/bash --noprofile

this_script=${0##*/}

usage="
Usage: $this_script [OS_ARCH_FILE]

List GOOS GOARCH specs, one per line based on OS_ARCH_FILE. The latter
is expected to contain GOOS GOARCH GOARCH... specifications, one per line;
a '-' in the spec denotes the native GOOS or GOARCH.

If no file is specified then the default GOOS or GOARCH are listed.

Default file: $go_os_arch_targets_file

"
native_os=$(go env GOOS)
native_arch=$(go env GOARCH)

case "$1" in
    -h|--h*)
        echo >&2 "$usage"
        exit 1
        ;;
    "") 
        echo $native_os $native_arch
        exit 0
        ;;
    *) 
        go_os_arch_targets_file="$1"
        shift
        ;;
esac


awk '
(NF > 1) && ($1 !~ /^#/) {
    os=$1
    if (os == "-") {
        os="'$native_os'"
    }
    for (k=2; k<=NF; k++) {
        arch=$k
        if (arch == "-") {
            arch="'$native_arch'"
        }
        os_arch = os " " arch
        if (! found_os_arch[os_arch]) {
            print os_arch
        }
        found_os_arch[os_arch] = 1
    }
}
' $go_os_arch_targets_file
