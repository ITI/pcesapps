#!/bin/bash -e
# Copyright 2024 The Board of Trustees of the University of Illinois

# Usage: docker-build.sh [suffix [version ...]]
# Builds a docker image
# and pushes it with the tags
#   ghcr.io/iti/pocesapps-$suffix:$version1
#   ghcr.io/iti/pocesapps-$suffix:$version2
#   ...
# The default value of "$suffix" is test.
# Additionally, versions of $(date) and "latest"
# are always created
#
# To test your code from the command line:
#   ./docker-build.sh
#   ./docker-run.sh
# That creates ghcr.io.iti/pcesapps-test:latest
#
# Whenever you git push a branch, the ci/cd runs:
#   ./docker-build.sh dev
# This creates ghcr.io/iti/pcesappos-dev:latest
#
# Whever your git push a tag, the ci/cd runs:
#  ./docker-build.shy beta $tag
# This creates ghcr.io/iti/pcesapps-beta:$tag

pfx="ghcr.io/illinoisadams/doc-test"
pfx="ghcr.io/iti/pcesapps"

now=$(date -u +%F-%H-%M-%S)
image="$pfx-"${1:-"test"}
latest=( ${1:+"latest"} "$@" )

docker build --no-cache -t "$image:$now" .
docker push "$image:$now"
for v in "${latest[@]}" ; do
	docker tag "$image:$now" "$image:$v"
	docker push "$image:$v"
done
