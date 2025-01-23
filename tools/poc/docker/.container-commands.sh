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

image_name=$(cat image_name)
if [[ -r semver ]]; then
    image_tag=$(cat semver)
fi
container_name=$(basename $image_name | sed -r -e 's/[^a-zA-Z0-9-]+/-/')

default_platform=
platforms=
if [[ -r platforms.txt ]]; then
    while read os _ platform; do
        if [[ "$os" != '#'* && -n "$platform" ]]; then
            if [[ -z "$platforms" ]]; then
                default_platform="${platform}"
                platforms="${platform}"
            else
                platforms="${platforms} ${platform}"
            fi
        fi
    done < platforms.txt
fi


case "$this_script" in
    build-container)
        case "$1" in
            all) platform_list="$platforms";;
            "") platform_list="$default_platform";;
            *) platform_list="$@";;
        esac
        if [[ -f context ]]; then
            context=$(cat context)
        else
            context="."
        fi
        for platform in ${platform_list:--}; do
            if [[ "$platform" = "-" ]]; then
                platform=
            fi
            if [[ -x pre-build-command ]]; then
                (set -x; ./pre-build-command $platform)
            fi
            (set -x; exec docker build ${platform:+--platform $platform} -t $image_name${platform:+-}${platform//\//-}${image_tag:+:}$image_tag -f $this_dir/Dockerfile $context)
        done
    ;;
    run-container|start-container)
        if [[ -f runargs ]]; then
            runargs=$(cat runargs)
        else
            runargs=
        fi
        if [[ "$1" == "--platform" ]]; then
            platform="$2"
            runargs="$runargs${runargs:+ }$1 $2"
            shift 2
        else
            prev_arg=
            platform=
            for arg in $runargs; do
                if [[ "$prev_arg" == "--platform" ]]; then
                    platform="$arg"
                    break
                fi
                prev_arg="$arg"
            done
        fi
        if [[ -z "$platform" && -n "$default_platform" ]]; then
            platform=$default_platform
            runargs="--platform $platform${runargs:+ }$runargs"
        fi
        if [[ "$this_script" == start* ]]; then
            runargs="$runargs${runargs:+ }--detach"
        fi
        if [[ -x ./pre-start-local-command ]]; then
            ./pre-start-local-command $platform
        fi
        host_volumes_dir=volumes${platform:+/}$platform
        docker_volumes_dir=/volumes
        for v in $(/bin/ls -1d volumes${platform:+/}$platform/* 2>/dev/null); do
            v_path=$(realpath $v)
            [[ -z "$v_path" ]] && continue
            runargs="$runargs${runargs:+ }--volume $v_path:$docker_volumes_dir/$(basename $v)"
        done
        if [[ -f ports ]]; then
            for p in $(cat ports); do
                runargs="$runargs${runargs:+ }--publish $p"
            done
        fi
        (set -x; exec docker run -it --rm $runargs --name $container_name $image_name${platform:+-}${platform//\//-}${image_tag:+:}$image_tag "$@")
    ;;
    stop-container)
        container_id=$(docker ps --filter name=$container_name --format "{{.ID}}")
        if [[ -n "$container_id" ]]; then
            set +e
            if [[ -f pre-stop-command ]]; then
                (
                    set -x
                    docker exec -it $container_name $(cat pre-stop-command)
                    docker kill $container_id
                )
            else
                if [[ -f killsig ]]; then
                    killsig=$(cat killsig)
                else
                    killsig=SIGTERM
                fi
                (set -x; docker kill --signal=$killsig $container_id)
                if [[ -f max_wait ]]; then
                    max_wait=$(cat max_wait)
                else
                    max_wait=5
                fi
                echo >&2 "$this_script - Waiting at most $max_wait sec for the container to terminate..."
                for ((k=1; k<=$max_wait; k++)); do
                    sleep 1
                    container_id=$(docker ps --filter name=$container_name --format "{{.ID}}")
                    [[ -z "$container_id" ]] && break
                done
                if [[ -n "$container_id" ]]; then
                    echo >&2 "$this_script - Force killing the container, not a clean shutdown"
                    (set -x; docker kill $container_id)
                fi
            fi
        fi
    ;;
    kill-container)
        container_id=$(docker ps --filter name=$container_name --format "{{.ID}}")
        if [[ -n "$container_id" ]]; then
            echo >&2 "$this_script - Force killing the container, not a clean shutdown"
            (set -x; docker kill $container_id)
        fi
    ;;
    login-container)
        (set -x; exec docker exec -it $container_name bash --login)
    ;;
    exec-in-container)
        if [[ -f exec-args ]]; then
            args=$(cat exec-args)
        else
            args="$@"
        fi
        (set -x; exec docker exec -it $container_name $args)
    ;;
esac

