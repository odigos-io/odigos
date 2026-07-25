FROM --platform=$BUILDPLATFORM alpine/helm:3.15.2 AS chart-builder
USER root
ARG CHART_VERSION
WORKDIR /workspace
COPY helm/ helm/
RUN set -e && \
    CHART_VER="${CHART_VERSION#v}" && \
    echo "Packaging Helm charts for version ${CHART_VER}" && \
    for chart in helm/odigos helm/odigos-central; do \
      echo "Updating ${chart}/Chart.yaml with version ${CHART_VER}" && \
      sed -i -E 's/0.0.0/'"${CHART_VER}"'/' "${chart}/Chart.yaml"; \
    done && \
    helm package helm/odigos helm/odigos-central -d /workspace/chart-artifacts

FROM --platform=$BUILDPLATFORM golang:1.26.4 AS builder
ARG TARGETOS
ARG TARGETARCH
ARG VERSION
ARG CHART_VERSION
ARG SHORT_COMMIT
ARG DATE
ARG RHEL=false

WORKDIR /workspace

COPY api/go.mod api/go.sum api/
COPY common/go.mod common/go.sum common/
COPY k8sutils/go.mod k8sutils/go.sum k8sutils/
COPY profiles/go.mod profiles/go.sum profiles/
COPY odigosauth/go.mod odigosauth/go.sum odigosauth/
COPY cli/go.mod cli/go.sum cli/

WORKDIR /workspace/cli
RUN --mount=type=cache,target=/go/pkg \
    go mod download

WORKDIR /workspace
COPY api/ api/
COPY common/ common/
COPY k8sutils/ k8sutils/
COPY profiles/ profiles/
COPY odigosauth/ odigosauth/
COPY cli/ cli/
COPY --from=chart-builder /workspace/chart-artifacts/ cli/pkg/helm/embedded/

WORKDIR /workspace/cli
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CHART_VER="${CHART_VERSION#v}" && \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -tags=embed_manifests -a -o odigos \
    -ldflags="-s -w \
              -X github.com/odigos-io/odigos/cli/cmd.OdigosVersion=${VERSION} \
              -X github.com/odigos-io/odigos/cli/pkg/helm.OdigosChartVersion=$${CHART_VER} \
              -X github.com/odigos-io/odigos/cli/cmd.OdigosCommit=${SHORT_COMMIT} \
              -X github.com/odigos-io/odigos/cli/cmd.OdigosDate=${DATE}" \
    .
RUN if [ "$RHEL" = "true" ] ; then \
      make licenses ; \
    fi

######## RHEL Image ########
FROM registry.access.redhat.com/ubi9/ubi-micro:latest AS rhel
ARG VERSION
ARG RELEASE
ARG SUMMARY
ARG DESCRIPTION
LABEL "name"="odigos-cli"
LABEL "vendor"="Odigos"
LABEL "maintainer"="Odigos"
LABEL "version"=$VERSION
LABEL "release"=$RELEASE
LABEL "summary"=$SUMMARY
LABEL "description"=$DESCRIPTION
WORKDIR /
COPY --from=builder /workspace/cli/odigos /odigos
COPY --from=builder /workspace/cli/licenses /licenses
COPY --from=builder /workspace/cli/LICENSE /licenses/.
USER 65532:65532
ENTRYPOINT ["/odigos"]

######### Final Image #########
# Use distroless as minimal base image to package the CLI binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/cli/odigos /odigos
USER 65532:65532
ENTRYPOINT ["/odigos"]
