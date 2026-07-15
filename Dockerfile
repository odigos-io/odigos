FROM --platform=$BUILDPLATFORM golang:1.26.4 AS builder
ARG SERVICE_NAME

# Copy only go.mod/go.sum for local modules to cache dependency downloads
WORKDIR /workspace
COPY api/go.mod api/go.sum api/
COPY common/go.mod common/go.sum common/
COPY k8sutils/go.mod k8sutils/go.sum k8sutils/
COPY profiles/go.mod profiles/go.sum profiles/
COPY distros/go.mod distros/go.sum distros/
COPY destinations/go.mod destinations/go.sum destinations/
COPY config/go.mod config/go.sum config/
COPY status/go.mod status/go.sum status/
COPY $SERVICE_NAME/go.mod $SERVICE_NAME/go.sum $SERVICE_NAME/

# go mod download must run from the service module so it resolves that go.mod
WORKDIR /workspace/$SERVICE_NAME
RUN mkdir -p /workspace/build
RUN --mount=type=cache,target=/go/pkg \
    go mod download && go mod verify

# Copy full local modules and service source
WORKDIR /workspace
COPY api/ api/
COPY common/ common/
COPY k8sutils/ k8sutils/
COPY profiles/ profiles/
COPY distros/ distros/
COPY destinations/ destinations/
COPY config/ config/
COPY status/ status/
COPY $SERVICE_NAME/ $SERVICE_NAME/
WORKDIR /workspace/$SERVICE_NAME
# Build for target architecture
ARG TARGETARCH
ARG LD_FLAGS
ARG RHEL=false
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