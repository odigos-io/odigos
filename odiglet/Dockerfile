FROM python:3.10-alpine AS python-builder
WORKDIR /python-instrumentation
ADD agents/python/requirements.txt .
RUN mkdir workspace && pip install --target workspace -r requirements.txt

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

FROM gcr.io/distroless/static-debian10
WORKDIR /root/
COPY --from=builder /go/src/github.com/keyval-dev/odigos/odiglet/odiglet .
WORKDIR /instrumentations/
COPY --from=builder /instrumentations/ .
CMD ["/root/odiglet"]
