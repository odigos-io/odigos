# Compiles the OSS odiglet binary (cache-friendly layers like odigos/odiglet/Dockerfile).
# Bake context: odigos repo root.
# CSI / deviceplugin / grpc_health_probe live in Dockerfile.common-binaries.

ARG ODIGLET_BASE_IMAGE=registry.odigos.io/odiglet-base:v1.20

FROM --platform=$BUILDPLATFORM ${ODIGLET_BASE_IMAGE} AS odiglet-builder
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
# go-build cache speeds bpf2go / clang-driven generates on repeat builds.
RUN --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    --mount=type=cache,target=/go/pkg,sharing=locked \
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
RUN --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    --mount=type=cache,target=/go/pkg,sharing=locked \
    make go-mod-replace && \
    CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH LD_FLAGS="${LD_FLAGS}" make compile-odiglet

# File capabilities so the binary can use eBPF privileges when run as non-root.
# Setting this caps as the permitted set (=p) so that the binary can be executed with a sub set of these capabilities
# if the environment does not require all of them. A sub-set can be configured with helm values.
RUN setcap cap_bpf,cap_perfmon,cap_sys_admin,cap_sys_ptrace,cap_dac_read_search,cap_ipc_lock,cap_sys_resource,cap_setuid,cap_setgid=p ./odiglet
RUN if [ "$RHEL" = "true" ] ; then \
      make licenses ; \
    fi
