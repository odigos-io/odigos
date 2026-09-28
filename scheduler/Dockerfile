# Compatibility image for odigos-scheduler.
#
# Scheduler controllers now run inside the instrumentor. This image is still
# published under the odigos-scheduler tags so existing pull/mirror scripts
# keep working. It is a pause container and does not run any Odigos logic.
# The helm chart keeps a replicas: 0 Deployment that references this image so
# Kubernetes GC does not delete CollectorsGroups still owned by that Deployment.
#
# Default target is pause. The rhel stage is only used when publishing
# odigos-scheduler-rhel-certified.
FROM registry.k8s.io/pause:3.10 AS pause

FROM registry.access.redhat.com/ubi9/ubi-micro:latest AS rhel
ARG VERSION
ARG RELEASE
ARG SUMMARY
ARG DESCRIPTION
LABEL "name"="scheduler"
LABEL "vendor"="Odigos"
LABEL "maintainer"="Odigos"
LABEL "version"=$VERSION
LABEL "release"=$RELEASE
LABEL "summary"=$SUMMARY
LABEL "description"=$DESCRIPTION
COPY --from=pause /pause /pause
COPY LICENSE /licenses/LICENSE
USER 65532:65532
ENTRYPOINT ["/pause"]

FROM registry.k8s.io/pause:3.10
