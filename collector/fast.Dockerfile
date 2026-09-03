# syntax=docker/dockerfile:1.20
FROM --platform=$BUILDPLATFORM golang:1.26.6-trixie AS builder
# install the required tooling for calling `setcap` on the compiled binary
RUN apt-get update && apt-get install -y --no-install-recommends libcap2-bin
WORKDIR /go/src

# copy dependency metadata first for better layer caching
# supported from docker syntax 1.20+ (2024)
COPY --parents collector/odigosotelcol/go.mod collector/odigosotelcol/go.sum ./
COPY --parents collector/**/go.mod collector/**/go.sum common/go.mod common/go.sum ./

# download dependencies before branching out to cross-compile for better caching
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd collector/odigosotelcol && go mod download

COPY ./common/ ./common
COPY ./collector/ ./collector

ARG TARGETARCH
ARG RHEL=false
WORKDIR /go/src/collector
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH make build-odigoscol

# Add file capabilities so the binary can use eBPF privileges when run as non-root.
# Caps match odiglet.dataCollection.capabilities.
# only setting the capabilities as permitted (p) so that the binary can be shared between the node collector (needs eBPF) and the cluster collector (does not need eBPF).
# this means that the node collector needs to set its effective capabilities to match the permitted ones, which is done by the odigoscapabilitiesextension.
RUN setcap cap_sys_admin,cap_bpf,cap_perfmon,cap_ipc_lock=p ./odigosotelcol/odigosotelcol

RUN if [ "$RHEL" = "true" ] ; then \
      make licenses ; \
    fi

FROM registry.access.redhat.com/ubi9/ubi-micro:latest AS rhel
ARG VERSION
ARG RELEASE
ARG SUMMARY
ARG DESCRIPTION
LABEL "name"="collector"
LABEL "vendor"="Odigos"
LABEL "maintainer"="Odigos"
LABEL "version"=$VERSION
LABEL "release"=$RELEASE
LABEL "summary"=$SUMMARY
LABEL "description"=$DESCRIPTION
COPY --from=builder /go/src/collector/odigosotelcol/odigosotelcol /odigosotelcol
COPY --from=builder /go/src/collector/odigosotelcol/licenses /licenses
COPY --from=builder /go/src/collector/odigosotelcol/LICENSE /licenses/.
USER 65532:65532
CMD ["/odigosotelcol"]

FROM gcr.io/distroless/base:latest AS distroless
COPY --from=builder /go/src/collector/odigosotelcol/odigosotelcol /odigosotelcol
# make sure we run as non root user
# this is required when this container is added to a pod that has runAsNonRoot: true
USER 65532:65532
CMD ["/odigosotelcol"]
