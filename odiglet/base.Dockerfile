FROM fedora:38 as builder
ARG TARGETARCH
# git is used by go mod download
RUN dnf install clang llvm make libbpf-devel git -y
RUN curl -LO https://go.dev/dl/go1.21.0.linux-${TARGETARCH}.tar.gz && tar -C /usr/local -xzf go*.linux-${TARGETARCH}.tar.g

# goreleaser is used by vmagent to build linux packages (deb, apk, rpm, etc)
RUN GOPATH=/usr/local/go go install github.com/goreleaser/goreleaser@v1.23.0

ENV PATH="/usr/local/go/bin:${PATH}"