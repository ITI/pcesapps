#!/bin/sh -ex
# Copyright 2024 The Board of Trustees of the University of Illinois

TAG=ghcr.io/iti/mrnesbitsapps-beta:v0.1.1
TAG2=ghcr.io/iti/mrnesbitsapps-beta:latest
cp ../go.mod ../go.sum .
docker build --no-cache -t $TAG -t $TAG2 .
docker push $TAG
docker push $TAG2
