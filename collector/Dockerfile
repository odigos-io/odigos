FROM --platform=$BUILDPLATFORM golang:1.23-bullseye AS builder
COPY . /go/src/
WORKDIR /go/src/

ARG TARGETARCH
RUN GOOS=linux GOARCH=$TARGETARCH make build-odigoscol

# Choose a minimal image with sh + acl utils
FROM registry.access.redhat.com/ubi9:9.5-1736404036
COPY --from=builder /go/src/odigosotelcol/odigosotelcol /odigosotelcol
USER 65532:65532
CMD ["/odigosotelcol"]