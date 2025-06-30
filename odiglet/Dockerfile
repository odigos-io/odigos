######### python Native Community Agent #########

FROM python:3.11.9 AS python-builder
ARG ODIGOS_VERSION
WORKDIR /python-instrumentation
COPY agents/python ./agents/configurator
RUN pip install ./agents/configurator/  --target workspace
RUN echo "VERSION = \"$ODIGOS_VERSION\";" > /python-instrumentation/workspace/initializer/version.py

######### Node.js Native Community Agent #########
#
# The Node.js agent is built in multiple stages so it can be built with either upstream
# @odigos/opentelemetry-node or with a local clone to test changes during development.
# The implementation is based on the following blog post:
# https://www.docker.com/blog/dockerfiles-now-support-multiple-build-contexts/

# The first build stage 'nodejs-agent-clone' clones the agent sources from github main branch.
FROM alpine AS nodejs-agent-clone
RUN apk add git
WORKDIR /src
ARG NODEJS_AGENT_VERSION=main
RUN git clone https://github.com/odigos-io/opentelemetry-node.git && cd opentelemetry-node && git checkout $NODEJS_AGENT_VERSION

# The second build stage 'nodejs-agent-src' prepares the actual code we are going to compile and embed in odiglet.
# By default, it uses the previous 'nodejs-agent-clone' stage, but one can override it by setting the 
# --build-context nodejs-agent-src=../opentelemetry-node flag in the docker build command.
# This allows us to use the agent sources and test changes during development.
# The output of this stage is the resolved source code to be used in the next stage.
FROM scratch AS nodejs-agent-src
COPY --from=nodejs-agent-clone /src/opentelemetry-node /

# The third step 'nodejs-agent-build' compiles the agent sources and prepares it for 
# being dependency of the native-community agent.
FROM node:18 AS nodejs-agent-build
# Run yarn install to generate the production node_modules directory
WORKDIR /opentelemetry-node-prod
COPY --from=nodejs-agent-src package.json yarn.lock ./
RUN yarn install --frozen-lockfile --production
# Build the agent from typescript sources
ARG ODIGOS_VERSION
WORKDIR /opentelemetry-node
COPY --from=nodejs-agent-src package.json yarn.lock ./
RUN yarn --frozen-lockfile
COPY --from=nodejs-agent-src / .
RUN echo "export const VERSION = \"$ODIGOS_VERSION\";" > ./src/version.ts
RUN yarn compile

FROM --platform=$BUILDPLATFORM busybox:1.36.1 AS dotnet-builder
WORKDIR /dotnet-instrumentation
ARG DOTNET_OTEL_VERSION=v1.11.0
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

# PHP
FROM --platform=$BUILDPLATFORM maniator/gh AS php-agents
WORKDIR /php-agents
ARG TARGETARCH
ARG PHP_AGENT_VERSION="v0.1.21"
ARG PHP_VERSIONS="8.0 8.1 8.2 8.3 8.4"
ENV PHP_VERSIONS=${PHP_VERSIONS}
# Clone agents repo (contains pre-compiled binaries, and pre-installed dependencies for each PHP version)
RUN git clone https://github.com/odigos-io/opentelemetry-php \
    && cd opentelemetry-php \
    && git checkout tags/${PHP_AGENT_VERSION}
# Move the pre-compiled binaries to the correct directories
RUN for v in ${PHP_VERSIONS}; do \
    mv opentelemetry-php/$v/bin/${TARGETARCH}/* opentelemetry-php/$v/; \
    rm -rf opentelemetry-php/$v/bin; \
    done


# Ruby
FROM --platform=$BUILDPLATFORM maniator/gh AS ruby-agents
WORKDIR /ruby-agents
ARG TARGETARCH
ARG RUBY_AGENT_VERSION="v0.0.4"
ARG RUBY_VERSIONS="3.1 3.2 3.3 3.4"
ENV RUBY_VERSIONS=${RUBY_VERSIONS}
# Clone agents repo (contains pre-compiled binaries, and pre-installed dependencies for each Ruby version)
RUN git clone https://github.com/odigos-io/opentelemetry-ruby \
    && cd opentelemetry-ruby \
    && git checkout tags/${RUBY_AGENT_VERSION}
# Move the gems & binaries to the correct directories
RUN for v in ${RUBY_VERSIONS}; do \
    mv opentelemetry-ruby/$v/${TARGETARCH}/* opentelemetry-ruby/$v/; \
    cp opentelemetry-ruby/Gemfile opentelemetry-ruby/$v/Gemfile; \
    cp opentelemetry-ruby/index.rb opentelemetry-ruby/$v/index.rb; \
    rm -rf opentelemetry-ruby/$v/amd64; \
    rm -rf opentelemetry-ruby/$v/arm64; \
    done


######### ODIGLET #########
FROM --platform=$BUILDPLATFORM registry.odigos.io/odiglet-base:v1.8 AS builder
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
ARG LD_FLAGS
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    GOOS=linux GOARCH=$TARGETARCH LD_FLAGS="${LD_FLAGS}" make build-odiglet

WORKDIR /instrumentations

# Java
ARG JAVA_OTEL_VERSION=v2.17.0
ADD https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/download/$JAVA_OTEL_VERSION/opentelemetry-javaagent.jar /instrumentations/java/javaagent.jar
RUN chmod 644 /instrumentations/java/javaagent.jar

# Python
COPY --from=python-builder /python-instrumentation/workspace /instrumentations/python

# NodeJS
COPY --from=nodejs-agent-build /opentelemetry-node/package.json /instrumentations/opentelemetry-node/package.json
COPY --from=nodejs-agent-build /opentelemetry-node/LICENSE /instrumentations/opentelemetry-node/LICENSE
COPY --from=nodejs-agent-build /opentelemetry-node/build /instrumentations/opentelemetry-node/build
COPY --from=nodejs-agent-build /opentelemetry-node-prod/node_modules /instrumentations/opentelemetry-node/node_modules

# nodejs-community
COPY ./agents/nodejs-community/autoinstrumentation.js /instrumentations/nodejs-community/autoinstrumentation.js

# .NET
COPY --from=dotnet-builder /dotnet-instrumentation /instrumentations/dotnet

# PHP
COPY --from=php-agents /php-agents/opentelemetry-php/8.0 /instrumentations/php/8.0
COPY --from=php-agents /php-agents/opentelemetry-php/8.1 /instrumentations/php/8.1
COPY --from=php-agents /php-agents/opentelemetry-php/8.2 /instrumentations/php/8.2
COPY --from=php-agents /php-agents/opentelemetry-php/8.3 /instrumentations/php/8.3
COPY --from=php-agents /php-agents/opentelemetry-php/8.4 /instrumentations/php/8.4

# Ruby
COPY --from=ruby-agents /ruby-agents/opentelemetry-ruby/3.1 /instrumentations/ruby/3.1
COPY --from=ruby-agents /ruby-agents/opentelemetry-ruby/3.2 /instrumentations/ruby/3.2
COPY --from=ruby-agents /ruby-agents/opentelemetry-ruby/3.3 /instrumentations/ruby/3.3
COPY --from=ruby-agents /ruby-agents/opentelemetry-ruby/3.4 /instrumentations/ruby/3.4

# loader
ARG ODIGOS_LOADER_VERSION=v0.0.3
RUN wget --directory-prefix=loader https://storage.googleapis.com/odigos-loader/$ODIGOS_LOADER_VERSION/$TARGETARCH/loader.so

FROM gcr.io/distroless/base:latest
COPY --from=builder /go/src/github.com/odigos-io/odigos/odiglet/odiglet /root/odiglet
WORKDIR /instrumentations/
COPY --from=builder /instrumentations/ .
CMD ["/root/odiglet"]
