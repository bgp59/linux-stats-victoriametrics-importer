#! /bin/bash --noprofile

# Sourced by various scripts.

if [[ -z "$this_dir" ]]; then
    this_dir=$(cd $(dirname ${BASH_SOURCE}) && pwd)
fi

os=$(uname -s | tr A-Z a-z)
case "$os" in
    linux) go_os="$os";;
    *) go_os="";;
esac

arch=$(uname -m | tr A-Z a-z)
case "$arch" in
    x86_64) go_arch="amd64";;
    *) go_arch=;;
esac

if [[ -n "$go_os" && -n "$go_arch" && -d "$this_dir/bin/$go_os-$go_arch" ]]; then
    export PATH="$this_dir/bin/$go_os-$go_arch${PATH:+:}$PATH"
fi


check_os_arch() {
    local _this_script=${this_script:-${BASH_SOURCE##*/}}
    local os=$(uname -s | tr A-Z a-z)
    case "$os" in
        linux) go_os="$os";;
        *)
            echo >&2 "$_this_script - $os: unsupported OS"
            return 1
        ;;
    esac

    local arch=$(uname -m | tr A-Z a-z)
    case "$arch" in
        x86_64) go_arch="amd64";;
        *)
            echo >&2 "$_this_script - $arch: unsupported arch"
            return 1
        ;;
    esac

    if [[ -d "$this_dir/bin/$go_os-$go_arch" && "$PATH" != "$this_dir/bin/$go_os-$go_arch"* ]]; then
        export PATH="$this_dir/bin/$go_os-$go_arch${PATH:+:}$PATH"
    fi
    return 0
}

check_if_running() {
    local _this_script=${this_script:-${BASH_SOURCE##*/}}
    if pgrep -af "(.*/)?$*( |\$)" >&2; then
        echo >&2 "$_this_script - $@ already running"
        return 1
    fi
    return 0
}

kill_wait_proc() {
    local pids=$(pgrep -f "(.*/)?$*( |\$)")
    local _this_script=${this_script:-${BASH_SOURCE##*/}}
    if [[ -z "$pids" ]]; then
        echo >&2 "$_this_script - $@ not running"
        return 0
    fi

    echo >&2 "$_this_script - Killing $@..."

    local _max_pid_wait=${max_pid_wait:-8}
    local _kill_sig_list=${kill_sig_list:-TERM KILL}
    local sig
    local k
    for sig in $_kill_sig_list; do
        (set -x; kill -$sig $pids) || return 1
        for ((k=0; k<$_max_pid_wait; k++)); do
            sleep 1
            ps -p $pids > /dev/null || return 0
        done
    done
    return 1
}
