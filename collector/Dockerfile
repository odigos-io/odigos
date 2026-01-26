FROM --platform=$BUILDPLATFORM golang:1.25-trixie AS builder
COPY ./collector/ /go/src/collector
# Need to copy the common folder to the collector for custom connector that uses it
COPY ./common/ /go/src/common
WORKDIR /go/src/collector

ARG TARGETARCH
ARG RHEL=false
RUN GOOS=linux GOARCH=$TARGETARCH make build-odigoscol
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
