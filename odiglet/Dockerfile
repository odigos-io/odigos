
######### python Native Community Agent #########

FROM python:3.11 AS python-builder
ARG ODIGOS_VERSION
WORKDIR /python-instrumentation
COPY ../agents/python ./agents/configurator
RUN pip install ./agents/configurator/  --target workspace
RUN echo "VERSION = \"$ODIGOS_VERSION\";" > /python-instrumentation/workspace/initializer/version.py

######### Node.js Native Community Agent #########
#
# The Node.js agent is built in multiple stages so it can be built with either upstream
# @odigos/opentelemetry-node or with a local clone to test changes during development.
# The implemntation is based on the following blog post:
# https://www.docker.com/blog/dockerfiles-now-support-multiple-build-contexts/

# The first build stage 'nodejs-agent-clone' clones the agent sources from github main branch.
FROM alpine AS nodejs-agent-clone
RUN apk add git
WORKDIR /src
ARG NODEJS_AGENT_VERSION=main
RUN git clone https://github.com/odigos-io/opentelemetry-node.git && cd opentelemetry-node && git checkout $NODEJS_AGENT_VERSION

# The second build stage 'nodejs-agent-src' prepares the actual code we are going to compile and embed in odiglet.
# By default, it uses the previous 'nodejs-agent-src' stage, but one can override it by setting the 
# --build-context nodejs-agent-src=../opentelemetry-node flag in the docker build command.
# This allows us to nobe the agent sources and test changes during development.
# The output of this stage is the resolved source code to be used in the next stage.
FROM scratch AS nodejs-agent-src
COPY --from=nodejs-agent-clone /src/opentelemetry-node /

# The third step 'nodejs-agent-build' compiles the agent sources and prepares it for 
# being dependency of the native-community agent.
FROM node:18 AS nodejs-agent-build
ARG ODIGOS_VERSION
WORKDIR /opentelemetry-node
COPY --from=nodejs-agent-src package.json yarn.lock ./
# install dependencies with dev so we can build the agent
RUN yarn --frozen-lockfile
COPY --from=nodejs-agent-src / .
RUN echo "export const VERSION = \"$ODIGOS_VERSION\";" > ./src/version.ts
RUN yarn compile


# The fourth step 'nodejs-agent-native-community-src' prepares the agent sources for the native-community agent.
# it COPY the nodejs agent source from 'nodejs-agent-build' stage and then build the agent in the 'agents/nodejs-native-community' directory.
# The output of this stage is the compiled agent code in:
#    - package source code in '/nodejs-instrumentation/build/src' directory.
#    - all required dependencies in '/nodejs-instrumentation/prod_node_modules' directory.
# These artifacts are later copied into the odiglet final image to be mounted into auto-instrumented pods at runtime.
FROM node:18 AS nodejs-agent-native-community-builder
ARG ODIGOS_VERSION
WORKDIR /repos
COPY ./agents/nodejs-native-community ./odigos/agents/nodejs-native-community
COPY --from=nodejs-agent-build /opentelemetry-node opentelemetry-node
# prepare the production node_modules content in a separate directory
RUN yarn --cwd ./odigos/agents/nodejs-native-community --production --frozen-lockfile



FROM --platform=$BUILDPLATFORM busybox:1.36.1 AS dotnet-builder
WORKDIR /dotnet-instrumentation
ARG DOTNET_OTEL_VERSION=v1.8.0
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

FROM --platform=$BUILDPLATFORM keyval/odiglet-base:v1.5 AS builder
WORKDIR /go/src/github.com/odigos-io/odigos
# Copy local modules required by the build
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
ARG JAVA_OTEL_VERSION=v2.9.0
ADD https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/download/$JAVA_OTEL_VERSION/opentelemetry-javaagent.jar /instrumentations/java/javaagent.jar
RUN chmod 644 /instrumentations/java/javaagent.jar

# Python
COPY --from=python-builder /python-instrumentation/workspace /instrumentations/python

# NodeJS
COPY --from=nodejs-agent-native-community-builder /repos/odigos/agents/nodejs-native-community /instrumentations/nodejs

# .NET
COPY --from=dotnet-builder /dotnet-instrumentation /instrumentations/dotnet

FROM registry.fedoraproject.org/fedora-minimal:38
COPY --from=builder /go/src/github.com/odigos-io/odigos/odiglet/odiglet /root/odiglet
WORKDIR /instrumentations/
COPY --from=builder /instrumentations/ .
CMD ["/root/odiglet"]
