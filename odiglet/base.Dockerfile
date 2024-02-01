FROM golang:1.21.6-bullseye as builder

RUN apt-get update && apt-get install -y curl clang gcc llvm make libbpf-dev

# goreleaser is used by vmagent to build linux packages (deb, apk, rpm, etc)
RUN go install github.com/goreleaser/goreleaser@v1.23.0
