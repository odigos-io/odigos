FROM golang:1.22.5-bookworm AS builder

# fury is our registry for linux packages
RUN echo "deb [trusted=yes] https://apt.fury.io/cli/ * *" > /etc/apt/sources.list.d/fury-cli.list
# goreleaser is used to build vmagent
RUN echo "deb [trusted=yes] https://repo.goreleaser.com/apt/ /" > /etc/apt/sources.list.d/goreleaser.list
RUN apt-get update && apt-get install -y curl clang gcc llvm make libbpf-dev fury-cli goreleaser
