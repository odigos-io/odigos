FROM --platform=$BUILDPLATFORM golang:1.27.0-trixie AS builder
# install the required tooling for calling `setcap` on the compiled binary
RUN apt-get update && apt-get install -y --no-install-recommends libcap2-bin
COPY ./collector/ /go/src/collector
# Need to copy the common folder to the collector for custom connector that uses it
COPY ./common/ /go/src/common
WORKDIR /go/src/collector

# download dependencies before branching out to cross-compile for better caching
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd odigosotelcol && go mod download

ARG TARGETARCH
ARG RHEL=false
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

FROM gcr.io/distroless/base:latest
COPY --from=builder /go/src/collector/odigosotelcol/odigosotelcol /odigosotelcol
# make sure we run as non root user
# this is required when this container is added to a pod that has runAsNonRoot: true
USER 65532:65532
CMD ["/odigosotelcol"]
