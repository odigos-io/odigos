FROM --platform=$BUILDPLATFORM golang:1.26.2 AS builder
ARG TARGETARCH
ARG TARGETOS=linux
ARG ODIGOS_TAG

WORKDIR /workspace
RUN apt-get update && apt-get install -y --no-install-recommends curl ca-certificates && rm -rf /var/lib/apt/lists/*
RUN curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

COPY api/ api/
COPY autoscaler/ autoscaler/
COPY common/ common/
COPY k8sutils/ k8sutils/
COPY odigosauth/ odigosauth/
COPY profiles/ profiles/
COPY distros/ distros/
COPY destinations/ destinations/
COPY config/ config/
COPY helm/ helm/
COPY cli/ cli/

WORKDIR /workspace/cli
RUN TAG_NO_V="${ODIGOS_TAG#v}" && \
    cp -r ../helm/odigos /tmp/odigos && \
    sed -i -E "s/^version:.*/version: ${TAG_NO_V}/" /tmp/odigos/Chart.yaml && \
    helm package /tmp/odigos -d pkg/helm/embedded && \
    cp -r ../helm/odigos-central /tmp/odigos-central && \
    sed -i -E "s/^version:.*/version: ${TAG_NO_V}/" /tmp/odigos-central/Chart.yaml && \
    helm package /tmp/odigos-central -d pkg/helm/embedded && \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -tags=embed_manifests \
      -ldflags="-X github.com/odigos-io/odigos/cli/cmd.OdigosVersion=${TAG_NO_V} -X github.com/odigos-io/odigos/cli/pkg/helm.OdigosChartVersion=${TAG_NO_V}" \
      -o /workspace/build/odigos . && \
    rm -rf /tmp/odigos /tmp/odigos-central

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
COPY --from=builder /workspace/build/odigos /odigos
USER 65532:65532
ENTRYPOINT ["/odigos"]

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /workspace/build/odigos /odigos
USER 65532:65532
ENTRYPOINT ["/odigos"]
