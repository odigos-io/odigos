FROM python:3.10-alpine AS python-builder
WORKDIR /python-instrumentation
ADD agents/python/requirements.txt .
RUN mkdir workspace && pip install --target workspace -r requirements.txt

FROM node:16 AS nodejs-builder
WORKDIR /nodejs-instrumentation
COPY agents/nodejs .
RUN npm install

FROM busybox AS dotnet-builder
WORKDIR /dotnet-instrumentation
ADD https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation/releases/download/v0.0.1/otel-dotnet-autoinstrumentation-0.0.1-musl.tar.gz .
RUN tar -xvf otel-dotnet-autoinstrumentation-0.0.1-musl.tar.gz && rm otel-dotnet-autoinstrumentation-0.0.1-musl.tar.gz

FROM golang:1.19 AS builder
WORKDIR /go/src/github.com/keyval-dev/odigos/odiglet
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o odiglet cmd/main.go

WORKDIR /instrumentations

# Java
ARG JAVA_OTEL_VERSIN=1.22.1
ADD https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/download/v$JAVA_OTEL_VERSIN/opentelemetry-javaagent.jar /instrumentations/java/javaagent.jar

# Python
COPY --from=python-builder /python-instrumentation/workspace /instrumentations/python

# NodeJS
COPY --from=nodejs-builder /nodejs-instrumentation/build/workspace /instrumentations/nodejs

# .NET
COPY --from=dotnet-builder /dotnet-instrumentation /instrumentations/dotnet
RUN chmod -R 777 /instrumentations/dotnet

FROM gcr.io/distroless/static-debian10
WORKDIR /root/
COPY --from=builder /go/src/github.com/keyval-dev/odigos/odiglet/odiglet .
WORKDIR /instrumentations/
COPY --from=builder /instrumentations/ .
CMD ["/root/odiglet"]
