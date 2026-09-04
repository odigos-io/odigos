# Default (distroless) OSS odiglet image.
# Expects named bake contexts: builder, common-binaries, agents-builder.
# COPY order: heavy/stable layers first, odiglet binary last (changes most often).

ARG ODIGLET_BASE_IMAGE=registry.odigos.io/odiglet-base:v1.20

# Note that we do not pass --platform=$BUILDPLATFORM here because we want to have the base image for all target architecture - each contains the corresponding rsync binary for the target arch.
FROM ${ODIGLET_BASE_IMAGE} AS rsync-base

FROM gcr.io/distroless/base@sha256:f4a335ca209e1d2ee873102c17c389ad0142e3d5b21aee2817e9cc9c01d87d20 AS odiglet
COPY --from=rsync-base /usr/bin/rsync /usr/bin/rsync
# This one is heavy (takes a while to copy since it's many hundreds of MB) so do it first.
COPY --from=agents-builder /instrumentations/ /instrumentations/
COPY --from=common-binaries /tmp/grpc_health_probe /root/grpc_health_probe
COPY --from=common-binaries /go/src/github.com/odigos-io/odigos/deviceplugin/deviceplugin /root/deviceplugin
COPY --from=common-binaries /go/src/github.com/odigos-io/odigos/odiglet/odigos-csi-driver /root/odigos-csi-driver
# This one changes most often so do it last.
COPY --from=builder /go/src/github.com/odigos-io/odigos/odiglet/odiglet /root/odiglet
# make sure we run as non root user
# this is required when this container is added to a pod that has runAsNonRoot: true
USER 65532:65532
CMD ["/root/odiglet"]
