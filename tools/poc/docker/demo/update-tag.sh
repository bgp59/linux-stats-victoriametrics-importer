#! /bin/bash --noprofile

this_script=${0##*/}
case "$0" in
    /*|*/*) 
        this_dir=$(cd $(dirname $0) && pwd)
    ;;
    *) 
        this_dir=$(cd $(dirname $(which $0)) && pwd)
    ;;
esac

docker_hub_dir=../../../../.docker-hub
set -e
cd $this_dir
echo "$(cat $docker_hub_dir/repo.txt):demo-$(cat ../../../../semver.txt)" > $docker_hub_dir/tag
ln -fs $docker_hub_dir/tag .

