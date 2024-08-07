FROM --platform=$BUILDPLATFORM golang:1.22 AS builder
ARG SERVICE_NAME

# Copyy local modules required by the build
WORKDIR /workspace
COPY api/ api/
COPY common/ common/
COPY k8sutils/ k8sutils/

WORKDIR /workspace/$SERVICE_NAME
RUN mkdir -p /workspace/build
# Pre-copy/cache go.mod for pre-downloading dependencies and only redownloading
COPY $SERVICE_NAME/go.mod $SERVICE_NAME/go.sum ./
RUN --mount=type=cache,target=/go/pkg \
    go mod download && go mod verify
# Copy rest of source code
COPY $SERVICE_NAME/ .
# Build for target architecture
ARG TARGETARCH
RUN go mod tidy
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 GOARCH=$TARGETARCH \
    go build -a -o /workspace/build/$SERVICE_NAME main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
ARG SERVICE_NAME
WORKDIR /
COPY --from=builder /workspace/build/$SERVICE_NAME ./app
USER 65532:65532
# TODO: calling the binary by SERVICE_NAME should be better for us in debugging
# but it does not work in distroless image
ENTRYPOINT ["/app"]