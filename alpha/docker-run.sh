#!/bin/sh -ex

# Copyright 2024 The Board of Trustees of the University of Illinois.

# This program, when executed on the Docker HOST, will run the alpha demonstration
# with graphics.
#
# Ussing grphics in Docker is somewhat fiddly. the following command works for me.

TAG=ghcr.io/iti/mrnesbitsapps-alpha

docker run -it --rm  \
	--env DISPLAY=$DISPLAY \
	-v /tmp/.X11-unix:/tmp/.X11-unix \
	-v ${XAUTHORITY:-$HOME/.Xauthority}:/root/.Xauthority:ro \
	--net=host \
	$TAG

# The following command, when executed on the Docker HOST, will run the alpha demonstration
# without graphics.
#   docker run -it --rm $TAG

# The following command sequence will copy this file to the docker HOST.
#   docker container create --name $$ $TAG
#   docker container cp $$:/app/docker-run.sh /tmp/.
#   docker container rm $$
