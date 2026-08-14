ARG ODIGLET_BASE_IMAGE=registry.odigos.io/odiglet-base:v1.18


######### python Native Community Agent #########

FROM --platform=$BUILDPLATFORM busybox:1.36.1 AS dotnet-builder
WORKDIR /dotnet-instrumentation
ARG DOTNET_OTEL_VERSION=v1.9.0
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


######### ODIGLET #########
FROM --platform=$BUILDPLATFORM ${ODIGLET_BASE_IMAGE} AS builder
WORKDIR /go/src/github.com/odigos-io/odigos
# Copy local modules required by the build
COPY api/ api/
COPY common/ common/
COPY k8sutils/ k8sutils/
COPY procdiscovery/ procdiscovery/
COPY opampserver/ opampserver/
COPY instrumentation/ instrumentation/
COPY distros/ distros/
WORKDIR /go/src/github.com/odigos-io/odigos/odiglet
COPY odiglet/ .

ARG TARGETARCH
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    GOOS=linux GOARCH=$TARGETARCH make build-odiglet

# Install delve
RUN go install github.com/go-delve/delve/cmd/dlv@latest

WORKDIR /instrumentations

# java-community
ARG JAVA_OTEL_VERSION=v2.10.0
ADD https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/download/$JAVA_OTEL_VERSION/opentelemetry-javaagent.jar /instrumentations/java/javaagent.jar
RUN chmod 644 /instrumentations/java/javaagent.jar

# python-community/python-community3.8
COPY --from=public.ecr.aws/odigos/agents/python-community:v1.0.73-py3.8@sha256:661308cb08d293f4ff3e1823cc7ebc0537ba53dd1fff72f61b411637d62ec3d4 /instrumentations/python3.8 /instrumentations/python3.8
COPY --from=public.ecr.aws/odigos/agents/python-community:v1.0.90@sha256:26be5a681e6cde7585f0bff6714fd74dbc90515a239d4759f5b6225591a78185 /instrumentations/python /instrumentations/python

# nodejs-community
COPY --from=public.ecr.aws/odigos/agents/nodejs-community:v0.13.0@sha256:f908dd3a2ae4d9213d923521685b10f138d6739c902ec1f9a20c0ce5b7b435c8 /instrumentations/opentelemetry-node /instrumentations/opentelemetry-node
COPY --from=public.ecr.aws/odigos/agents/nodejs-community:v0.13.0@sha256:f908dd3a2ae4d9213d923521685b10f138d6739c902ec1f9a20c0ce5b7b435c8 /instrumentations/nodejs-community /instrumentations/nodejs-community
# nodejs-community-14
COPY --from=public.ecr.aws/odigos/agents/nodejs-community-14:v0.0.19@sha256:c000e785fd525eb4fd5056cf922b7e2c3f12d845d36fb9661c1d505f8a3f3d53 /instrumentations/opentelemetry-node-14 /instrumentations/opentelemetry-node-14
COPY --from=public.ecr.aws/odigos/agents/nodejs-community-14:v0.0.19@sha256:c000e785fd525eb4fd5056cf922b7e2c3f12d845d36fb9661c1d505f8a3f3d53 /instrumentations/nodejs-community-14 /instrumentations/nodejs-community-14

# dotnet-community
COPY --from=dotnet-builder /dotnet-instrumentation /instrumentations/dotnet

# php-community
COPY --from=public.ecr.aws/odigos/agents/php-community:v0.5.0@sha256:2742c2e003ddece79010fd47b4da021b54e3ce046dafd850f1b4a772d38aa08b /instrumentations/php /instrumentations/php

# ruby-community
COPY --from=public.ecr.aws/odigos/agents/ruby-community:v0.0.8@sha256:a9df41f7684f550a3b7e1693456f7c4eb4ce432aba82d4b9e22d2a73687473b1 /instrumentations/ruby /instrumentations/ruby

# loader
ARG ODIGOS_LOADER_VERSION=v0.0.8
RUN wget --directory-prefix=loader https://storage.googleapis.com/odigos-loader/$ODIGOS_LOADER_VERSION/$TARGETARCH/loader.so

FROM ${ODIGLET_BASE_IMAGE} AS rsync-base

FROM registry.fedoraproject.org/fedora-minimal:38
COPY --from=builder /go/src/github.com/odigos-io/odigos/odiglet/odiglet /root/odiglet
COPY --from=builder /go/bin/dlv /root/dlv
# Copy statically compiled rsync (no shared libraries needed)
COPY --from=rsync-base /usr/bin/rsync /usr/bin/rsync
WORKDIR /instrumentations/
COPY --from=builder /instrumentations/ .

EXPOSE 2345
ENTRYPOINT ["/root/dlv" ,"--listen=:2345", "--headless=true", "--api-version=2", "--accept-multiclient", "exec", "/root/odiglet"]
