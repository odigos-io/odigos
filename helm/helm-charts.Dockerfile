# Packages Odigos Helm charts into /charts/*.tgz for Depot save/pull/extract.
# Build context must be the odigos repo root (contains helm/).
# Example:
#   depot build --project ... --file docker/Dockerfile.helm-charts --build-arg CHART_VERSION=e2e-tests \
#     --save --save-tag amir-charts-1234 ../odigos
FROM alpine/helm:3.15.2 AS chart-builder
USER root
ARG CHART_VERSION=0.0.0
WORKDIR /workspace
COPY helm/ helm/
# Helm requires SemVer for chart.metadata.version. Non-SemVer tags used as image
# tags (e.g. e2e-tests, commit SHAs) are prefixed to 0.0.0-<tag>.
RUN set -e && \
    CHART_VER="${CHART_VERSION#v}" && \
    if ! echo "${CHART_VER}" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+([+-].*)?$'; then \
      CHART_VER="0.0.0-${CHART_VER}"; \
    fi && \
    echo "Packaging Helm charts for version ${CHART_VER}" && \
    for chart in helm/odigos helm/odigos-central; do \
      echo "Updating ${chart}/Chart.yaml with version ${CHART_VER}" && \
      sed -i -E 's/0.0.0/'"${CHART_VER}"'/' "${chart}/Chart.yaml"; \
    done && \
    mkdir -p /chart-artifacts && \
    helm package helm/odigos helm/odigos-central -d /chart-artifacts

FROM scratch
COPY --from=chart-builder /chart-artifacts/ /charts/
