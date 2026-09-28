# Compatibility image for odigos-scheduler.
#
# Scheduler controllers now run inside the instrumentor. This is a pause
# container with no Odigos logic. It is still published so that if someone
# scales the odigos-scheduler Deployment above zero, the pod starts and does
# nothing instead of competing with the instrumentor.
# The helm chart keeps that Deployment at replicas: 0 so Kubernetes GC does
# not delete CollectorsGroups still owned by it. Users should not pull or
# mirror this image.
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
