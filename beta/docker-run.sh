#!/bin/sh -ex

# Copyright 2024 The Board of Trustees of the University of Illinois.

# This program, when executed on the Docker HOST, will run
# a shell inside the beta demonstration environmment.
#
# Inside the beta demonstration environment, the following command
# will run the graphical beta demonstration:
#   python3 gui.py -is args-gui
#
# N.b. Using grphics in Docker is somewhat fiddly.
# the following command works for me (Rob Adams).

TAG=ghcr.io/iti/pcesapps-beta

docker run -it --rm  \
	--env DISPLAY=$DISPLAY \
	-v /tmp/.X11-unix:/tmp/.X11-unix \
	-v ${XAUTHORITY:-$HOME/.Xauthority}:/root/.Xauthority:ro \
	--net=host \
	$TAG

# The following command will bring up the beta environment
# without graphical support
# docker run -it --rm --net=host $TAG

# The following command sequence will copy this file to the docker HOST.
#   docker container create --name $$ $TAG
#   docker container cp $$:/app/docker-run.sh /tmp/.
#   docker container rm $$
