#! /bin/bash --noprofile

this_script=${0##*/}
case "$0" in
    /*|*/*) this_dir=$(dirname $(realpath $0));;
    *) this_dir=$(dirname $(realpath $(which $0)));;
esac

set -x
cd $this_dir || exit 1

case "$this_script" in
    start*)
        ./victoria-metrics/start-victoria-metrics.sh
        ./grafana/start-grafana.sh
        [[ -d lsvmi ]] && ./lsvmi/start-lsvmi.sh
        ;;
    run*)
        trap ./stop-poc.sh HUP INT TERM
        ./victoria-metrics/start-victoria-metrics.sh
        ./grafana/start-grafana.sh
        [[ -d lsvmi ]] && ./lsvmi/start-lsvmi.sh
        sleep infinity
        ;;
    stop*)
        [[ -d lsvmi ]] && ./lsvmi/stop-lsvmi.sh
        ./grafana/stop-grafana.sh
        ./victoria-metrics/stop-victoria-metrics.sh
        ;;
esac




