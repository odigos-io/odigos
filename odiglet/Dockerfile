FROM python:3.11 AS python-builder
WORKDIR /python-instrumentation
ADD odiglet/agents/python/requirements.txt .
RUN mkdir workspace && pip install --target workspace -r requirements.txt

FROM node:18 AS nodejs-builder
ARG ODIGOS_VERSION
WORKDIR /nodejs-instrumentation
COPY agents/nodejs/package.json agents/nodejs/yarn.lock ./
# prepare the production node_modules content in a separate directory
RUN yarn --production --frozen-lockfile
RUN mv node_modules ./prod_node_modules
# install all dependencies including dev so we can yarn compile
RUN yarn --frozen-lockfile
COPY agents/nodejs .
# inject the actual version into the agent code
RUN echo "export const VERSION = \"$ODIGOS_VERSION\";" > ./src/version.ts
RUN yarn compile

FROM busybox AS dotnet-builder
WORKDIR /dotnet-instrumentation
ARG DOTNET_OTEL_VERSION=v0.7.0
ADD https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation/releases/download/$DOTNET_OTEL_VERSION/opentelemetry-dotnet-instrumentation-linux-musl.zip .
RUN unzip opentelemetry-dotnet-instrumentation-linux-musl.zip && rm opentelemetry-dotnet-instrumentation-linux-musl.zip

FROM --platform=$BUILDPLATFORM keyval/odiglet-base:v1.4 as builder
WORKDIR /go/src/github.com/odigos-io/odigos
# Copyy local modules required by the build
COPY api/ api/
COPY common/ common/
COPY k8sutils/ k8sutils/
COPY procdiscovery/ procdiscovery/
COPY opampserver/ opampserver/
WORKDIR /go/src/github.com/odigos-io/odigos/odiglet
COPY odiglet/ .

ARG TARGETARCH
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    GOOS=linux GOARCH=$TARGETARCH make build-odiglet

WORKDIR /instrumentations

# Java
ARG JAVA_OTEL_VERSION=v2.3.0
ADD https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/download/$JAVA_OTEL_VERSION/opentelemetry-javaagent.jar /instrumentations/java/javaagent.jar
RUN chmod 644 /instrumentations/java/javaagent.jar

# Python
COPY --from=python-builder /python-instrumentation/workspace /instrumentations/python

# NodeJS
COPY --from=nodejs-builder /nodejs-instrumentation/build/src /instrumentations/nodejs
COPY --from=nodejs-builder /nodejs-instrumentation/prod_node_modules /instrumentations/nodejs/node_modules

# .NET
COPY --from=dotnet-builder /dotnet-instrumentation /instrumentations/dotnet

FROM registry.fedoraproject.org/fedora-minimal:38
COPY --from=builder /go/src/github.com/odigos-io/odigos/odiglet/odiglet /root/odiglet
WORKDIR /instrumentations/
COPY --from=builder /instrumentations/ .
CMD ["/root/odiglet"]
