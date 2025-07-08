FROM --platform=$BUILDPLATFORM golang:1.24-bullseye AS builder
COPY ./collector/ /go/src/collector
# Need to copy the common folder to the collector for custom connector that uses it
COPY ./common/ /go/src/common
WORKDIR /go/src/collector

ARG TARGETARCH
RUN GOOS=linux GOARCH=$TARGETARCH make build-odigoscol

FROM gcr.io/distroless/base:latest
COPY --from=builder /go/src/collector/odigosotelcol/odigosotelcol /odigosotelcol
CMD ["/odigosotelcol"]
