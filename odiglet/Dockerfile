FROM python:3.11 AS python-builder
WORKDIR /python-instrumentation
ADD odiglet/agents/python/requirements.txt .
RUN mkdir workspace && pip install --target workspace -r requirements.txt

FROM node:16 AS nodejs-builder
WORKDIR /nodejs-instrumentation
COPY odiglet/agents/nodejs .
RUN npm install

FROM busybox AS dotnet-builder
WORKDIR /dotnet-instrumentation
ARG DOTNET_OTEL_VERSION=v0.7.0
ADD https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation/releases/download/$DOTNET_OTEL_VERSION/opentelemetry-dotnet-instrumentation-linux-musl.zip .
RUN unzip opentelemetry-dotnet-instrumentation-linux-musl.zip && rm opentelemetry-dotnet-instrumentation-linux-musl.zip

FROM --platform=$BUILDPLATFORM keyval/odiglet-base:v1.3 as builder
WORKDIR /go/src/github.com/keyval-dev/odigos
# Copyy local modules required by the build
COPY api/ api/
COPY common/ common/
COPY procdiscovery/ procdiscovery/
WORKDIR /go/src/github.com/keyval-dev/odigos/odiglet
# Pre-copy/cache go.mod for pre-downloading dependencies and only redownloading
# them in subsequent builds if they change
COPY odiglet/go.mod odiglet/go.sum ./
RUN --mount=type=cache,target=/go/pkg \
    go mod download && go mod verify
# Copy rest of source code
COPY odiglet/ .

ARG TARGETARCH
# Go does not call go generate on dependencies, so we need to do it manually
RUN --mount=type=cache,target=/go/pkg \
    cd $(go list -m -f '{{.Dir}}' "go.opentelemetry.io/auto") && make generate
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    GOOS=linux GOARCH=$TARGETARCH go build -o odiglet cmd/main.go

WORKDIR /instrumentations

# Java
ARG JAVA_OTEL_VERSION=v1.29.0
ADD https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/download/$JAVA_OTEL_VERSION/opentelemetry-javaagent.jar /instrumentations/java/javaagent.jar
RUN chmod 644 /instrumentations/java/javaagent.jar

# Python
COPY --from=python-builder /python-instrumentation/workspace /instrumentations/python

# NodeJS
COPY --from=nodejs-builder /nodejs-instrumentation/build/workspace /instrumentations/nodejs

# .NET
COPY --from=dotnet-builder /dotnet-instrumentation /instrumentations/dotnet

FROM registry.fedoraproject.org/fedora-minimal:38
COPY --from=builder /go/src/github.com/keyval-dev/odigos/odiglet/odiglet /root/odiglet
WORKDIR /instrumentations/
COPY --from=builder /instrumentations/ .
CMD ["/root/odiglet"]
