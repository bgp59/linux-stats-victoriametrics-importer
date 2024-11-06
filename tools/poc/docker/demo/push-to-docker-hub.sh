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

(set -x; docker push $tag)

semver=$(cat ../../../../semver.txt)
demo_tag=${tag%%-$semver}
if [[ "$demo_tag" != "$tag" ]]; then
    (
        set -x
        docker tag $tag $demo_tag
        docker push $demo_tag
    )
fi
