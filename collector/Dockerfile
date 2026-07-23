# syntax=docker/dockerfile:1.7-labs
# Builder runs on the host/build platform and cross-compiles with GOARCH.
# With CGO_ENABLED=0 this does not need QEMU emulation for linux/amd64|arm64.
FROM --platform=$BUILDPLATFORM golang:1.26.2-trixie AS ocb
ARG BUILDER_VERSION=0.151.0
ARG BUILDARCH
# Fixed path so the consumer stage can COPY without ARG interpolation in the source.
RUN curl -fsSL -o /ocb \
    "https://github.com/open-telemetry/opentelemetry-collector-releases/releases/download/cmd%2Fbuilder%2Fv${BUILDER_VERSION}/ocb_${BUILDER_VERSION}_linux_${BUILDARCH}" \
 && chmod +x /ocb

FROM --platform=$BUILDPLATFORM golang:1.26.2-trixie AS builder

WORKDIR /go/src

# Copy only module manifests first so dependency download stays cached across
# source-only changes (mirrors the root Dockerfile pattern).
COPY --parents ./collector/**/go.mod ./collector/**/go.sum ./common/go.mod ./common/go.sum ./

WORKDIR /go/src/collector

# Download module graph before copying full sources for better layer caching.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd odigosotelcol && go mod download

# Full sources (collector + shared common module used by local connectors).
COPY ./collector/ /go/src/collector/
COPY ./common/ /go/src/common/

# OCB is fetched in a separate stage so the binary stays cached when only
# application sources change (and is not wiped by the COPY above).
ARG BUILDER_VERSION=0.151.0
RUN mkdir -p .tools
COPY --from=ocb /ocb /go/src/collector/.tools/builder_${BUILDER_VERSION}

ARG TARGETARCH
ARG RHEL=false
# build-odigoscol-docker skips unit tests (those run in CI via `make test`).
# CGO_ENABLED=0 keeps cross-compilation native — no QEMU required.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH make build-odigoscol-docker

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
CMD ["/odigosotelcol"]
