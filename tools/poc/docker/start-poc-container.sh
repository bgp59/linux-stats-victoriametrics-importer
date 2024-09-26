#! /bin/bash --noprofile

this_script=${0##*/}
case "$0" in
    /*|*/*) this_dir=$(dirname $(realpath $0));;
    *) this_dir=$(dirname $(realpath $(which $0)));;
esac

cd $this_dir
case "$this_script" in
    start-poc-container*)
        mkdir -p volumes/runtime
        ./start-container
        ./exec-in-container bash -c "
            echo 'Starting VictoriaMetrics'
            ./victoria-metrics/start-victoria-metrics.sh
            echo

            echo 'Starting Grafana'
            ./grafana/start-grafana.sh
            echo

            echo 'Check status'
            ps -f -p \$(pgrep -f '^(.*/)?(victoria-metrics|grafana)( |$)')
        "
    ;;
    stop-poc-container*)
        ./exec-in-container bash -c "
            ./grafana/stop-grafana.sh
            ./victoria-metrics/stop-victoria-metrics.sh
        "
        ./stop-container
    ;;
    run-poc-lsvmi*)
        ./exec-in-container ./lsvmi/run-lsvmi.sh
    ;;
esac


