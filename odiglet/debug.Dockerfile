ARG ODIGLET_BASE_IMAGE=registry.odigos.io/odiglet-base:v1.11


######### python Native Community Agent #########

FROM python:3.11.9 AS python-builder
ARG ODIGOS_VERSION
WORKDIR /python-instrumentation
COPY agents/python ./agents/configurator
RUN pip install ./agents/configurator/  --target workspace
RUN echo "VERSION = \"$ODIGOS_VERSION\";" > /python-instrumentation/workspace/initializer/version.py

FROM --platform=$BUILDPLATFORM busybox:1.36.1 AS dotnet-builder
WORKDIR /dotnet-instrumentation
ARG DOTNET_OTEL_VERSION=v1.9.0
ARG TARGETARCH
RUN if [ "$TARGETARCH" = "arm64" ]; then \
    echo "arm64" > /tmp/arch_suffix; \
    else \
    echo "x64" > /tmp/arch_suffix; \
    fi

RUN ARCH_SUFFIX=$(cat /tmp/arch_suffix) && \
    wget https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation/releases/download/${DOTNET_OTEL_VERSION}/opentelemetry-dotnet-instrumentation-linux-glibc-${ARCH_SUFFIX}.zip && \
    unzip opentelemetry-dotnet-instrumentation-linux-glibc-${ARCH_SUFFIX}.zip && \
    rm opentelemetry-dotnet-instrumentation-linux-glibc-${ARCH_SUFFIX}.zip && \
    mv linux-$ARCH_SUFFIX linux-glibc
RUN ARCH_SUFFIX=$(cat /tmp/arch_suffix) && \
    wget https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation/releases/download/${DOTNET_OTEL_VERSION}/opentelemetry-dotnet-instrumentation-linux-musl-${ARCH_SUFFIX}.zip && \
    unzip -o opentelemetry-dotnet-instrumentation-linux-musl-${ARCH_SUFFIX}.zip && \
    rm opentelemetry-dotnet-instrumentation-linux-musl-${ARCH_SUFFIX}.zip && \
    mv linux-musl-$ARCH_SUFFIX linux-musl

# TODO(edenfed): Currently .NET Automatic instrumentation does not work on dotnet 6.0 with glibc,
# This is due to compilation of the .so file on a newer version of glibc than the one used by the dotnet runtime.
# The following override the .so file with our own which is compiled on the same glibc version as the dotnet runtime.
RUN ARCH_SUFFIX=$(cat /tmp/arch_suffix) && \
    wget https://github.com/odigos-io/opentelemetry-dotnet-instrumentation/releases/download/${DOTNET_OTEL_VERSION}/OpenTelemetry.AutoInstrumentation.Native-${ARCH_SUFFIX}.so && \
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

# Java
ARG JAVA_OTEL_VERSION=v2.10.0
ADD https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/download/$JAVA_OTEL_VERSION/opentelemetry-javaagent.jar /instrumentations/java/javaagent.jar
RUN chmod 644 /instrumentations/java/javaagent.jar

# Python
COPY --from=python-builder /python-instrumentation/workspace /instrumentations/python

# NodeJS
COPY --from=public.ecr.aws/odigos/agents/nodejs-community:v0.0.8 /instrumentations/opentelemetry-node /instrumentations/opentelemetry-node
COPY --from=public.ecr.aws/odigos/agents/nodejs-community:v0.0.8 /instrumentations/nodejs-community /instrumentations/nodejs-community

# .NET
COPY --from=dotnet-builder /dotnet-instrumentation /instrumentations/dotnet

# PHP
COPY --from=public.ecr.aws/odigos/agents/php-community:v0.2.6 /instrumentations/php /instrumentations/php

# Ruby
COPY --from=public.ecr.aws/odigos/agents/ruby-community:v0.0.8 /instrumentations/ruby /instrumentations/ruby

# loader
ARG ODIGOS_LOADER_VERSION=v0.0.6
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
