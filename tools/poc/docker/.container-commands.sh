#!/bin/bash --noprofile

this_script=${0##*/}
case "$0" in
    /*|*/*) 
        this_dir=$(dirname $(realpath $0))
    ;;
    *) 
        this_dir=$(dirname $(realpath $(which $0)))
    ;;
esac


set -e
cd $this_dir

tag=$(cat $this_dir/tag)
if [[ -f name ]]; then
    name=$(cat name)
else
    name=${tag//:/_}
fi
case "$this_script" in
    build-container)
        docker build -t $tag -f $this_dir/Dockerfile ..
    ;;
    run-container|start-container)
        if [[ -f runargs ]]; then
            runargs=$(cat runargs)
        else
            runargs=
        fi
        if [[ "$this_script" == start* ]]; then
            runargs="$runargs${runargs:+ }--detach"
        fi
        for v in $(/bin/ls -1d volumes/*); do
            v_path=$(realpath $v)
            [[ -z "$v_path" ]] && continue
            runargs="$runargs${runargs:+ }--volume $v_path:/$v"
        done
        if [[ -f ports ]]; then
            for p in $(cat ports); do
                runargs="$runargs${runargs:+ }--publish $p"
            done
        fi
        (set -x; exec docker run -it --rm $runargs --name $name $tag "$@")
    ;;
    stop-container)
        container_id=$(docker ps --filter name=$(cat name ) --format "{{.ID}}")
        if [[ -n "$container_id" ]]; then
            (set -x; docker kill $container_id)
        fi
    ;;
    login-container)
        (set -x; exec docker exec -it $name bash --login)
    ;;
    exec-in-container)
        (set -x; exec docker exec -it $name "$@")
    ;;
    exec-args-in-container)
        (set -x; exec docker exec -it $name $(cat exec.args) $@)
    ;;
esac

