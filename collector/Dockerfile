FROM --platform=$BUILDPLATFORM golang:1.22-bullseye AS builder
COPY . /go/src/
WORKDIR /go/src/

ARG TARGETARCH
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    GOOS=linux GOARCH=$TARGETARCH make build-odigoscol

FROM gcr.io/distroless/base:latest
COPY --from=builder /go/src/odigosotelcol/odigosotelcol /odigosotelcol
CMD ["/odigosotelcol"]