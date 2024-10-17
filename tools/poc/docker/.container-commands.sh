#!/bin/bash --noprofile

this_script=${0##*/}
case "$0" in
    /*|*/*) 
        this_dir=$(cd $(dirname $0) && pwd)
    ;;
    *) 
        this_dir=$(cd $(dirname $(which $0)) && pwd)
    ;;
esac

set -e
cd $this_dir

tag=$(cat $this_dir/tag)
if [[ -f name ]]; then
    name=$(cat name)
else
    name=${tag//[^a-zA-Z0-9._-]/_}
fi
case "$this_script" in
    build-container)
        if [[ -f context ]]; then
            context=$(cat context)
        else
            context="."
        fi
        if [[ -x pre-build-command ]]; then
            ./pre-build-command
        fi
        (set -x; exec docker build -t $tag -f $this_dir/Dockerfile $context)
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
        if [[ -x ./pre-start-local-command ]]; then
            ./pre-start-local-command
        fi
        for v in $(/bin/ls -1d volumes/* 2>/dev/null); do
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
        container_id=$(docker ps --filter name=$name --format "{{.ID}}")
        if [[ -n "$container_id" ]]; then
            set +e
            if [[ -f pre-stop-command ]]; then
                (set -x; exec docker exec -it $name $(cat pre-stop-command))
            fi
            (set -x; docker kill $container_id)
        fi
    ;;
    login-container)
        (set -x; exec docker exec -it $name bash --login)
    ;;
    exec-in-container)
        if [[ -f exec-args ]]; then
            args=$(cat exec-args)
        else
            args="$@"
        fi
        (set -x; exec docker exec -it $name $args)
    ;;
esac

