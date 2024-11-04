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

set -e
cd $this_dir
if [[ -x tag ]]; then
    tag=$(./tag)
else
    tag=$(cat tag)
fi

demo_tag=${tag/:demo*/:demo}

set -x
docker tag $tag $demo_tag
docker push $tag
docker push $demo_tag
