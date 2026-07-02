# Build context is the repository root (BUILD_DIR=.), matching the other Odigos components.
# Unlike the shared root Dockerfile, the browser-proxy image is alpine-based because the init
# mode needs the `iptables` binary to install the inbound traffic redirect.

FROM --platform=$BUILDPLATFORM golang:1.26.2 AS builder
WORKDIR /workspace

# browser-proxy is a self-contained, dependency-free module (stdlib only), so no sibling
# modules need to be copied.
COPY browser-proxy/ browser-proxy/

WORKDIR /workspace/browser-proxy
ARG TARGETARCH
ARG LD_FLAGS
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH \
    go build -ldflags="${LD_FLAGS}" -o /workspace/build/odigos-browser-proxy ./cmd

######### Final image #########
FROM alpine:3.20
ARG VERSION
ARG RELEASE
ARG SUMMARY
ARG DESCRIPTION
LABEL "name"="odigos-browser-proxy"
LABEL "vendor"="Odigos"
LABEL "maintainer"="Odigos"
LABEL "version"=$VERSION
LABEL "release"=$RELEASE
LABEL "summary"=$SUMMARY
LABEL "description"=$DESCRIPTION

# iptables is required by the `init` mode to redirect inbound traffic to the sidecar.
RUN apk add --no-cache iptables

COPY --from=builder /workspace/build/odigos-browser-proxy /usr/local/bin/odigos-browser-proxy

# The serve (sidecar) mode runs as this fixed UID; the iptables redirect excludes traffic owned by
# it so the sidecar can reach the application on loopback. The init mode is run as root with
# CAP_NET_ADMIN via the pod securityContext set by the instrumentor webhook.
USER 1337:1337
ENTRYPOINT ["/usr/local/bin/odigos-browser-proxy"]
