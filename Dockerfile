# Usage:
#
# docker build -t pcesapps-dev:latest -t pcesapps:latest .
# docker run -it --rm -v ~/pcesapps/extern:/tmp/extern ghcr.io/iti/pcesapps-dev
# docker run -it --rm -v ~/pcesapps/extern:/tmp/extern ghcr.io/iti/pcesapps
#
FROM golang:1.23-bookworm

# Add whatever Debian packages you want here.
RUN apt-get -y update &&  \ 
    apt-get install --no-install-recommends -y \
    vim-nox && \
    rm -rf /var/lib/apt/lists/*

# Build the pcesapps app
WORKDIR /pcesapps
COPY . .
RUN cd embedded/sim-dir && go mod tidy && go build -o /bin/sim sim.go exp.go

# remember to use "-v" to map in /tmp/extern
WORKDIR /tmp/extern/input
