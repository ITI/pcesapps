# Copyright 2024 The Board of Trustees of the University of Illinois
# 
# The resulting docker image will allow one to run
# the mrnesbits beta demonstration.
#
# Usage:
#   ...
#

# Build phase
FROM golang:bookworm AS builder
# Next line is just for debug
RUN ldd --version
WORKDIR /build
COPY . .
RUN cd bld-dir && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./bld.go
RUN cd bld-dir && ./bld -is args-bld
RUN cd db && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build cnvrtExec.go
RUN cd db && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build cnvrtDesc.go
RUN cd sim-dir && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./sim.go

# Production phase
FROM debian:bookworm
RUN apt update && apt upgrade -y && apt install -y \
	python3 \
	python3-tk \
	python3-pil \
	python3-pil.imagetk vim-nox \
	python3-yaml \
	python3-matplotlib
WORKDIR /app
COPY --from=builder /build/. .
