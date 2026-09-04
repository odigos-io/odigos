ARG ODIGLET_BASE_IMAGE=registry.odigos.io/odiglet-base:v1.20

FROM --platform=$BUILDPLATFORM busybox:1.36.1 AS dotnet-builder
WORKDIR /dotnet-instrumentation
ARG DOTNET_OTEL_VERSION=v1.16.0
ARG TARGETARCH
# SHA256 pins for the artifacts fetched in this stage. Verification runs between the
# download and the unpack, so nothing unverified is ever extracted. Refresh these
# together with DOTNET_OTEL_VERSION — a mismatched download fails the build.
RUN if [ "$TARGETARCH" = "arm64" ]; then \
    echo "arm64" > /tmp/arch_suffix; \
    else \
    echo "x64" > /tmp/arch_suffix; \
    fi

RUN ARCH_SUFFIX=$(cat /tmp/arch_suffix) && \
    wget https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation/releases/download/${DOTNET_OTEL_VERSION}/opentelemetry-dotnet-instrumentation-linux-glibc-${ARCH_SUFFIX}.zip && \
    case "$ARCH_SUFFIX" in \
      x64) echo "58b6a7e5282ffba6d2f8e09c69e412711050e49a36c82d31edfd84cad3afac49  opentelemetry-dotnet-instrumentation-linux-glibc-x64.zip" | sha256sum -c - ;; \
      arm64) echo "7600d3e1f57d06995d637c95650d9ee28b6e32cf2aa362fdb1a988267425191f  opentelemetry-dotnet-instrumentation-linux-glibc-arm64.zip" | sha256sum -c - ;; \
      *) echo "no checksum pinned for arch $ARCH_SUFFIX" && exit 1 ;; \
    esac && \
    unzip opentelemetry-dotnet-instrumentation-linux-glibc-${ARCH_SUFFIX}.zip && \
    rm opentelemetry-dotnet-instrumentation-linux-glibc-${ARCH_SUFFIX}.zip && \
    mv linux-$ARCH_SUFFIX linux-glibc

RUN ARCH_SUFFIX=$(cat /tmp/arch_suffix) && \
    wget https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation/releases/download/${DOTNET_OTEL_VERSION}/opentelemetry-dotnet-instrumentation-linux-musl-${ARCH_SUFFIX}.zip && \
    case "$ARCH_SUFFIX" in \
      x64) echo "c0bc22bac5b5aa8b345cb995e87c39e080025ed5923a01d45f9d61d49c60b301  opentelemetry-dotnet-instrumentation-linux-musl-x64.zip" | sha256sum -c - ;; \
      arm64) echo "a5494eaeeab99f298b3c875dfbac648c0e56c7f04ebc062800904cbb18251229  opentelemetry-dotnet-instrumentation-linux-musl-arm64.zip" | sha256sum -c - ;; \
      *) echo "no checksum pinned for arch $ARCH_SUFFIX" && exit 1 ;; \
    esac && \
    unzip -o opentelemetry-dotnet-instrumentation-linux-musl-${ARCH_SUFFIX}.zip && \
    rm opentelemetry-dotnet-instrumentation-linux-musl-${ARCH_SUFFIX}.zip && \
    mv linux-musl-$ARCH_SUFFIX linux-musl

# TODO(edenfed): Currently .NET Automatic instrumentation does not work on dotnet 6.0 with glibc,
# This is due to compilation of the .so file on a newer version of glibc than the one used by the dotnet runtime.
# The following override the .so file with our own which is compiled on the same glibc version as the dotnet runtime.
RUN ARCH_SUFFIX=$(cat /tmp/arch_suffix) && \
    wget https://github.com/odigos-io/opentelemetry-dotnet-instrumentation/releases/download/${DOTNET_OTEL_VERSION}/OpenTelemetry.AutoInstrumentation.Native-${ARCH_SUFFIX}.so && \
    case "$ARCH_SUFFIX" in \
      x64) echo "1a2964dd0ac92c29d5265ffd0093cb7d69605cdd3e98374e9cf25dbfb619855c  OpenTelemetry.AutoInstrumentation.Native-x64.so" | sha256sum -c - ;; \
      arm64) echo "06c6191aa282c1a605d919d3057b1f5ce3d1e5ada7b9804295b30bf2288976ab  OpenTelemetry.AutoInstrumentation.Native-arm64.so" | sha256sum -c - ;; \
      *) echo "no checksum pinned for arch $ARCH_SUFFIX" && exit 1 ;; \
    esac && \
    mv OpenTelemetry.AutoInstrumentation.Native-${ARCH_SUFFIX}.so linux-glibc/OpenTelemetry.AutoInstrumentation.Native.so


######### agents builder #########
# combine all agents into a single image
FROM --platform=$BUILDPLATFORM ${ODIGLET_BASE_IMAGE} AS agents-builder
WORKDIR /instrumentations

# java-community
ARG JAVA_OTEL_VERSION=v2.31.1
ADD https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/download/$JAVA_OTEL_VERSION/opentelemetry-javaagent.jar /instrumentations/java/javaagent.jar
RUN chmod 644 /instrumentations/java/javaagent.jar

# python-community/python-community3.8
COPY --from=public.ecr.aws/odigos/agents/python-community:v1.0.73-py3.8@sha256:661308cb08d293f4ff3e1823cc7ebc0537ba53dd1fff72f61b411637d62ec3d4 /instrumentations/python3.8 /instrumentations/python3.8
COPY --from=public.ecr.aws/odigos/agents/python-community:v1.0.90@sha256:26be5a681e6cde7585f0bff6714fd74dbc90515a239d4759f5b6225591a78185 /instrumentations/python /instrumentations/python

# nodejs-community
COPY --from=public.ecr.aws/odigos/agents/nodejs-community:v0.15.0@sha256:1535cef7abba747c1e1fa363ad907d485d547f8ab733d90df977f33712968078 /instrumentations/opentelemetry-node /instrumentations/opentelemetry-node
COPY --from=public.ecr.aws/odigos/agents/nodejs-community:v0.15.0@sha256:1535cef7abba747c1e1fa363ad907d485d547f8ab733d90df977f33712968078 /instrumentations/nodejs-community /instrumentations/nodejs-community
# nodejs-community-14
COPY --from=public.ecr.aws/odigos/agents/nodejs-community-14:v0.0.22@sha256:3198ee74c61539f39614ec96b4ef9c7069140079e121541231f11f8b13863283 /instrumentations/opentelemetry-node-14 /instrumentations/opentelemetry-node-14
COPY --from=public.ecr.aws/odigos/agents/nodejs-community-14:v0.0.22@sha256:3198ee74c61539f39614ec96b4ef9c7069140079e121541231f11f8b13863283 /instrumentations/nodejs-community-14 /instrumentations/nodejs-community-14

# nodejs-community
COPY --from=dotnet-builder /dotnet-instrumentation /instrumentations/dotnet

# php-community
COPY --from=public.ecr.aws/odigos/agents/php-community:v0.7.0@sha256:b3ed955a65f7c6f67d4143e99ab778a70126669352f14dc1d8e20687e7af9ca0 /instrumentations/php /instrumentations/php

# ruby-community
COPY --from=public.ecr.aws/odigos/agents/ruby-community:v0.0.9@sha256:5177737aba83e507ee968223fbd821d76e2df4e74bd1711e2d49e8d59f671d8c /instrumentations/ruby /instrumentations/ruby

# loader
ARG TARGETARCH
ARG ODIGOS_LOADER_VERSION=v0.0.8
RUN wget --directory-prefix=loader https://storage.googleapis.com/odigos-loader/$ODIGOS_LOADER_VERSION/$TARGETARCH/loader.so

######### Device Plugin builder #########
FROM --platform=$BUILDPLATFORM ${ODIGLET_BASE_IMAGE} AS deviceplugin-builder
ENV GOTOOLCHAIN=auto
WORKDIR /go/src/github.com/odigos-io/odigos
# Copy local modules required by deviceplugin
COPY common/ common/
WORKDIR /go/src/github.com/odigos-io/odigos/deviceplugin
COPY deviceplugin/ .
ARG TARGETARCH
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    GOOS=linux GOARCH=$TARGETARCH go build -o deviceplugin ./cmd

######### CSI Driver builder #########
FROM --platform=$BUILDPLATFORM ${ODIGLET_BASE_IMAGE} AS csi-driver-builder
ENV GOTOOLCHAIN=auto
WORKDIR /go/src/github.com/odigos-io/odigos
# Copy local modules required by the build
COPY api/ api/
COPY common/ common/
COPY k8sutils/ k8sutils/
COPY procdiscovery/ procdiscovery/
COPY opampserver/ opampserver/
COPY instrumentation/ instrumentation/
COPY distros/ distros/
# Build CSI driver binary from odiglet module
WORKDIR /go/src/github.com/odigos-io/odigos/odiglet
COPY odiglet/ .
ARG TARGETARCH
ARG LD_FLAGS
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    GOOS=linux GOARCH=$TARGETARCH go build -ldflags="${LD_FLAGS}" -o odigos-csi-driver ./cmd/csi-driver

######### Odiglet builder #########
FROM --platform=$BUILDPLATFORM ${ODIGLET_BASE_IMAGE} AS builder
ENV GOTOOLCHAIN=auto

# 1) Materialize OBI under .obi/ — only obi.mk (OBI_COMMIT / OBI_VERSION). Cache independently of Makefile / go.mod / sources.
WORKDIR /go/src/github.com/odigos-io/odigos/odiglet
COPY odiglet/obi.mk .
RUN make -f obi.mk materialize-obi

# 2) Generate — local replace go.mod/sum only (not full sources). Cache until module metadata changes.
WORKDIR /go/src/github.com/odigos-io/odigos
COPY api/go.mod api/go.sum ./api/
COPY common/go.mod common/go.sum ./common/
COPY k8sutils/go.mod k8sutils/go.sum ./k8sutils/
COPY procdiscovery/go.mod procdiscovery/go.sum ./procdiscovery/
COPY opampserver/go.mod opampserver/go.sum ./opampserver/
COPY instrumentation/go.mod instrumentation/go.sum ./instrumentation/
COPY distros/go.mod distros/go.sum ./distros/
COPY odiglet/go.mod odiglet/go.sum ./odiglet/
COPY odiglet/pkg/ebpf/sdks/obi/go.mod odiglet/pkg/ebpf/sdks/obi/go.sum ./odiglet/pkg/ebpf/sdks/obi/
COPY odiglet/Makefile ./odiglet/
WORKDIR /go/src/github.com/odigos-io/odigos/odiglet
# Same /go/pkg cache as compile so generated bpf objects are visible to the build step.
RUN --mount=type=cache,target=/go/pkg \
    make -f Makefile generate

# 3) Compile — full sources; re-apply OBI replace (COPY overwrites go.mod) then build only.
# No re-download: modules (+ generate artifacts) live in the shared /go/pkg cache from the generate layer.
WORKDIR /go/src/github.com/odigos-io/odigos
COPY api/ api/
COPY common/ common/
COPY k8sutils/ k8sutils/
COPY procdiscovery/ procdiscovery/
COPY opampserver/ opampserver/
COPY instrumentation/ instrumentation/
COPY distros/ distros/
COPY odiglet/ odiglet/

ARG TARGETARCH
ARG LD_FLAGS
ARG RHEL=false
WORKDIR /go/src/github.com/odigos-io/odigos/odiglet
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    make go-mod-replace && \
    CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH LD_FLAGS="${LD_FLAGS}" make compile-odiglet

# File capabilities so the binary can use eBPF privileges when run as non-root.
# Setting this caps as the permitted set (=p) so that the binary can be executed with a sub set of these capabilities
# if the environment does not require all of them. A sub-set can be configured with helm values.
RUN setcap cap_bpf,cap_perfmon,cap_sys_admin,cap_sys_ptrace,cap_dac_read_search,cap_ipc_lock,cap_sys_resource,cap_setuid,cap_setgid=p ./odiglet
RUN if [ "$RHEL" = "true" ] ; then \
      make licenses ; \
    fi

# Download grpc_health_probe this is used by the deviceplugin container to check if the deviceplugin is healthy
ARG GRPC_HEALTH_PROBE_VERSION=v0.4.55
RUN if [ "$TARGETARCH" = "arm64" ]; then \
        GRPC_ARCH="arm64"; \
    else \
        GRPC_ARCH="amd64"; \
    fi && \
    wget -O /tmp/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-${GRPC_ARCH} && \
    chmod +x /tmp/grpc_health_probe

# Note that we do not pass --platform=$BUILDPLATFORM here because we want to have the base image for all target architecture - each contains the corresponding rsync binary for the target arch.
FROM ${ODIGLET_BASE_IMAGE} AS rsync-base

######### agents final image #########
# Use BusyBox for the final image instead of distroless.
# Rationale:
# - Provides built-in `sh`, `cp`, and other essential Unix tools.
# - Allows `sh -c` command chaining (`cp dir1 dst1 && cp dir2 dst2`) in init containers.
# - Very small (~1MB), making it good for init containers.
FROM busybox:1.36.1 AS agents
WORKDIR /instrumentations/

# Copy all instrumentation assets from the build stage
COPY --from=agents-builder /instrumentations/ .

# make sure we run as non root user
# this is required when this container is added to a pod that has runAsNonRoot: true
USER 1000:1000

######### RHEL agents final image #########
FROM registry.access.redhat.com/ubi9/ubi-micro:latest AS agents-rhel
ARG VERSION
ARG RELEASE
ARG SUMMARY
ARG DESCRIPTION
LABEL "name"="Odigos-Init-Container"
LABEL "vendor"="Odigos"
LABEL "maintainer"="Odigos"
LABEL "version"=$VERSION
LABEL "release"=$RELEASE
LABEL "summary"=$SUMMARY
LABEL "description"=$DESCRIPTION

WORKDIR /instrumentations/
COPY --from=agents-builder /instrumentations/ .

# make sure we run as non root user
# this is required when this container is added to a pod that has runAsNonRoot: true
USER 65532:65532

######### RHEL Odiglet #########
FROM registry.access.redhat.com/ubi9/ubi-micro:latest AS rhel
ARG VERSION
ARG RELEASE
ARG SUMMARY
ARG DESCRIPTION
LABEL "name"="Odiglet"
LABEL "vendor"="Odigos"
LABEL "maintainer"="Odigos"
LABEL "version"=$VERSION
LABEL "release"=$RELEASE
LABEL "summary"=$SUMMARY
LABEL "description"=$DESCRIPTION
# Copy statically compiled rsync (no shared libraries needed)
COPY --from=rsync-base /usr/bin/rsync /usr/bin/rsync
COPY --from=builder /tmp/grpc_health_probe /root/grpc_health_probe
COPY --from=agents-builder /instrumentations/ /instrumentations/
COPY --from=deviceplugin-builder /go/src/github.com/odigos-io/odigos/deviceplugin/deviceplugin /root/deviceplugin
COPY --from=csi-driver-builder /go/src/github.com/odigos-io/odigos/odiglet/odigos-csi-driver /root/odigos-csi-driver

COPY --from=builder /go/src/github.com/odigos-io/odigos/odiglet/odiglet /root/odiglet
COPY --from=builder /go/src/github.com/odigos-io/odigos/odiglet/licenses /licenses
COPY --from=builder /go/src/github.com/odigos-io/odigos/odiglet/LICENSE /licenses/.
# make sure we run as non root user
# this is required when this container is added to a pod that has runAsNonRoot: true
USER 65532:65532
CMD ["/root/odiglet"]

######### Default Odiglet #########
FROM gcr.io/distroless/base@sha256:f4a335ca209e1d2ee873102c17c389ad0142e3d5b21aee2817e9cc9c01d87d20
COPY --from=rsync-base /usr/bin/rsync /usr/bin/rsync
COPY --from=builder /tmp/grpc_health_probe /root/grpc_health_probe
COPY --from=agents-builder /instrumentations/ /instrumentations/
COPY --from=deviceplugin-builder /go/src/github.com/odigos-io/odigos/deviceplugin/deviceplugin /root/deviceplugin
COPY --from=csi-driver-builder /go/src/github.com/odigos-io/odigos/odiglet/odigos-csi-driver /root/odigos-csi-driver

COPY --from=builder /go/src/github.com/odigos-io/odigos/odiglet/odiglet /root/odiglet
# make sure we run as non root user
# this is required when this container is added to a pod that has runAsNonRoot: true
USER 65532:65532
CMD ["/root/odiglet"]
