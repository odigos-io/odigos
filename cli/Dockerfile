FROM --platform=$BUILDPLATFORM golang:1.26.4 AS builder
ARG TARGETOS
ARG TARGETARCH
ARG VERSION
ARG SHORT_COMMIT
ARG DATE

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

WORKDIR /workspace/cli
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -tags=embed_manifests -a -o odigos \
    -ldflags="-X github.com/odigos-io/odigos/cli/cmd.OdigosVersion=${VERSION} \
              -X github.com/odigos-io/odigos/cli/pkg/helm.OdigosChartVersion=${VERSION} \
              -X github.com/odigos-io/odigos/cli/cmd.OdigosCommit=${SHORT_COMMIT} \
              -X github.com/odigos-io/odigos/cli/cmd.OdigosDate=${DATE}" \
    .

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/cli/odigos /odigos
USER 65532:65532
ENTRYPOINT ["/odigos"]
