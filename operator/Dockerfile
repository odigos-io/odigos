# Build the manager binary
FROM --platform=$BUILDPLATFORM golang:1.26 AS builder
ARG TARGETOS
ARG TARGETARCH
ARG ODIGOS_VERSION

# Install helm for packaging the chart
RUN curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

WORKDIR /go/src/github.com/odigos-io/odigos
COPY api/ api/
COPY k8sutils/ k8sutils/
COPY cli/ cli/
COPY common/ common/
COPY odigosauth/ odigosauth/
COPY profiles/ profiles/
COPY helm/ helm/

WORKDIR /go/src/github.com/odigos-io/odigos/operator
COPY operator/ .
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Package the helm chart and embed it in the CLI package (which operator imports)
# The chart version is set to match the ODIGOS_VERSION (without 'v' prefix)
RUN CHART_VERSION="${ODIGOS_VERSION#v}" && \
    echo "Packaging helm chart with version: ${CHART_VERSION}, appVersion: ${ODIGOS_VERSION}" && \
    cp -r ../helm/odigos /tmp/odigos-chart && \
    sed -i -E 's/^version: "0.0.0"/version: "'"${CHART_VERSION}"'"/' /tmp/odigos-chart/Chart.yaml && \
    sed -i -E 's/^appVersion: "v0.0.0"/appVersion: "'"${ODIGOS_VERSION}"'"/' /tmp/odigos-chart/Chart.yaml && \
    cat /tmp/odigos-chart/Chart.yaml && \
    helm package /tmp/odigos-chart -d ../cli/pkg/helm/embedded/ && \
    rm -rf /tmp/odigos-chart

ARG LD_FLAGS
ARG RHEL=false
# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -ldflags="${LD_FLAGS}" -tags=embed_manifests -a -o manager cmd/main.go
RUN if [ "$RHEL" = "true" ] ; then \
      make licenses ; \
    fi

FROM registry.access.redhat.com/ubi9/ubi-micro:latest AS rhel
ARG VERSION
ARG RELEASE
ARG SUMMARY
ARG DESCRIPTION
LABEL "name"="operator"
LABEL "vendor"="Odigos"
LABEL "maintainer"="Odigos"
LABEL "version"=$VERSION
LABEL "release"=$RELEASE
LABEL "summary"=$SUMMARY
LABEL "description"=$DESCRIPTION
WORKDIR /
COPY --from=builder /go/src/github.com/odigos-io/odigos/operator/manager .
COPY --from=builder /go/src/github.com/odigos-io/odigos/operator/licenses /licenses
COPY --from=builder /go/src/github.com/odigos-io/odigos/operator/LICENSE /licenses/.
USER 65532:65532

ENTRYPOINT ["/manager"]

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /go/src/github.com/odigos-io/odigos/operator/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
