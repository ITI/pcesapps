#!/bin/sh -ex
# Copyright 2024 The Board of Trustees of the University of Illinois

TAG=ghcr.io/illinoisrobert/mrnesbitsapps-alpha:v0.2
docker build --no-cache -t $TAG .
docker push $TAG
