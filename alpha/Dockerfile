# Copyright 2024 The Board of Trustees of the University of Illinois
# 
# The resulting docker image will allow one to run the mrnesbits alpha demonstration.
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
RUN cd mdfy-dir && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./mdfy.go
RUN cd sim-dir && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./sim.go

# Production phase
FROM debian:bookworm
# Next line is just for debug
RUN ldd; exit 0
RUN apt update && apt install -y python3 python3-tk vim-nox
WORKDIR /app
COPY --from=builder /build/. .
