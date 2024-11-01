#! /bin/bash --noprofile

this_script=${0##*/}

usage="
Usage: $this_script [PARSER]...
"

case "$1" in
    -h|--h*)
        echo >&2 "$usage"
        exit 1
    ;;
esac

case "$0" in
    /*|*/*) this_dir=$(cd $(dirname $0) && pwd);;
    *) this_dir=$(cd $(dirname $(which $0)) && pwd);;
esac

parser_reg_exp=
while [[ $# -gt 0 ]]; do
    parser_reg_exp="$parser_reg_exp${parser_reg_exp:+|}$1"
    shift
done
if [[ -n "$parser_reg_exp" ]]; then
    parser_reg_exp="($parser_reg_exp)"
else
    parser_reg_exp=".*"
fi


out_file=local/parser-bench-$(uname -srm | tr 'A-Z  ' a-z-).txt

set -ex
cd $this_dir
mkdir -p $(dirname $out_file)
go test -benchmem -cpu 1 -bench "Benchmark${parser_reg_exp}Parser"  | tee $out_file

