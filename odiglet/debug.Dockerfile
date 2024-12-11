FROM python:3.11 AS python-builder
ARG ODIGOS_VERSION
WORKDIR /python-instrumentation
COPY ../agents/python ./agents/python
RUN echo "VERSION = \"$ODIGOS_VERSION\";" > ./agents/python/configurator/version.py
RUN mkdir workspace && pip install ./agents/python/  --target workspace


######### Node.js Native Community Agent #########
#
# The Node.js agent is built in multiple stages so it can be built with either upstream
# @odigos/opentelemetry-node or with a local clone to test changes during development.
# The implemntation is based on the following blog post:
# https://www.docker.com/blog/dockerfiles-now-support-multiple-build-contexts/

# The first build stage 'nodejs-agent-native-community-clone' clones the agent sources from github main branch.
FROM alpine AS nodejs-agent-native-community-clone
RUN apk add git
WORKDIR /src
ARG NODEJS_AGENT_VERSION=main
RUN git clone https://github.com/odigos-io/opentelemetry-node.git && cd opentelemetry-node && git checkout $NODEJS_AGENT_VERSION

# The second build stage 'nodejs-agent-native-community-src' prepares the actual code we are going to compile and embed in odiglet.
# By default, it uses the previous 'nodejs-agent-native-community-src' stage, but one can override it by setting the
# --build-context nodejs-agent-native-community-src=../opentelemetry-node flag in the docker build command.
# This allows us to nobe the agent sources and test changes during development.
# The output of this stage is the resolved source code to be used in the next stage.
FROM scratch AS nodejs-agent-native-community-src
COPY --from=nodejs-agent-native-community-clone /src/opentelemetry-node /

# The third build stage 'nodejs-agent-native-community-builder' compiles the agent sources and prepares the final output.
# it COPY from the previous 'nodejs-agent-native-community-src' stage, so it can be used with either the upstream or local sources.
# The output of this stage is the compiled agent code in:
#    - package source code in '/nodejs-instrumentation/build/src' directory.
#    - all required dependencies in '/nodejs-instrumentation/prod_node_modules' directory.
# These artifacts are later copied into the odiglet final image to be mounted into auto-instrumented pods at runtime.
FROM node:18 AS nodejs-agent-native-community-builder
ARG ODIGOS_VERSION
WORKDIR /nodejs-instrumentation
# TODO: change YARN -> NPM in "odigos-io/opentelemetry-node" repository, then change "yarn.lock" to "package-lock.json", and change "npm i" to "npm ci".
COPY --from=nodejs-agent-native-community-src /package.json /yarn.lock ./
# prepare the production node_modules content in a separate directory
RUN npm i --only=production
RUN mv node_modules ./prod_node_modules
# install all dependencies including dev so we can run "compile"
RUN npm i
COPY --from=nodejs-agent-native-community-src / ./
# inject the actual version into the agent code
RUN echo "export const VERSION = \"$ODIGOS_VERSION\";" > ./src/version.ts
RUN npm run compile

FROM busybox:1.36.1 AS dotnet-builder
WORKDIR /dotnet-instrumentation
ARG DOTNET_OTEL_VERSION=v1.7.0
ARG TARGETARCH
RUN if [ "$TARGETARCH" = "arm64" ]; then \
        echo "arm64" > /tmp/arch_suffix; \
    else \
        echo "x64" > /tmp/arch_suffix; \
    fi

RUN ARCH_SUFFIX=$(cat /tmp/arch_suffix) && \
    wget https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation/releases/download/${DOTNET_OTEL_VERSION}/opentelemetry-dotnet-instrumentation-linux-glibc-${ARCH_SUFFIX}.zip && \
    unzip opentelemetry-dotnet-instrumentation-linux-glibc-${ARCH_SUFFIX}.zip && \
    rm opentelemetry-dotnet-instrumentation-linux-glibc-${ARCH_SUFFIX}.zip

FROM --platform=$BUILDPLATFORM keyval/odiglet-base:v1.7 AS builder
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
    GOOS=linux GOARCH=$TARGETARCH make debug-build-odiglet

# Install delve
RUN go install github.com/go-delve/delve/cmd/dlv@latest

WORKDIR /instrumentations

# Java
ARG JAVA_OTEL_VERSION=v2.6.0
ADD https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/download/$JAVA_OTEL_VERSION/opentelemetry-javaagent.jar /instrumentations/java/javaagent.jar
RUN chmod 644 /instrumentations/java/javaagent.jar

# Python
COPY --from=python-builder /python-instrumentation/workspace /instrumentations/python

# NodeJS
COPY --from=nodejs-agent-native-community-builder /nodejs-instrumentation/build/src /instrumentations/nodejs
COPY --from=nodejs-agent-native-community-builder /nodejs-instrumentation/prod_node_modules /instrumentations/nodejs/node_modules


# .NET
COPY --from=dotnet-builder /dotnet-instrumentation /instrumentations/dotnet

FROM registry.fedoraproject.org/fedora-minimal:38
COPY --from=builder /go/src/github.com/odigos-io/odigos/odiglet/odiglet /root/odiglet
COPY --from=builder /go/bin/dlv /root/dlv
WORKDIR /instrumentations/
COPY --from=builder /instrumentations/ .

EXPOSE 2345
ENTRYPOINT ["/root/dlv" ,"--listen=:2345", "--headless=true", "--api-version=2", "--accept-multiclient", "exec", "/root/odiglet"]
