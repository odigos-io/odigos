FROM --platform=$BUILDPLATFORM golang:1.26 AS builder
ARG SERVICE_NAME

# Copy local modules required by the build
WORKDIR /workspace
COPY api/ api/
COPY common/ common/
COPY k8sutils/ k8sutils/
COPY profiles/ profiles/
COPY distros/ distros/
COPY destinations/ destinations/

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
ARG LD_FLAGS
ARG RHEL=false
RUN go mod tidy
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 GOARCH=$TARGETARCH \
    go build -ldflags="${LD_FLAGS}" -a -o /workspace/build/$SERVICE_NAME cmd/main.go
RUN if [ "$RHEL" = "true" ] ; then \
      make licenses ; \
    fi

######## RHEL Image ########
FROM registry.access.redhat.com/ubi9/ubi-micro:latest AS rhel
ARG SERVICE_NAME
ARG VERSION
ARG RELEASE
ARG SUMMARY
ARG DESCRIPTION
LABEL "name"=$SERVICE_NAME
LABEL "vendor"="Odigos"
LABEL "maintainer"="Odigos"
LABEL "version"=$VERSION
LABEL "release"=$RELEASE
LABEL "summary"=$SUMMARY
LABEL "description"=$DESCRIPTION

WORKDIR /
COPY --from=builder /workspace/build/$SERVICE_NAME ./app
COPY --from=builder /workspace/$SERVICE_NAME/licenses ./licenses
COPY --from=builder /workspace/$SERVICE_NAME/LICENSE ./licenses/.
USER 65532:65532
# TODO: calling the binary by SERVICE_NAME should be better for us in debugging
# but it does not work in distroless image
ENTRYPOINT ["/app"]

######### Final Image #########
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