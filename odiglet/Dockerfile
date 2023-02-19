FROM golang:1.19 AS builder
WORKDIR /go/src/github.com/keyval-dev/odigos/odiglet
COPY . .
RUN apt-get update && apt-get install -y nasm
RUN nasm pkg/allocator/payload/mmap.asm
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o app cmd/main.go

ARG OTEL_VERSIN=1.22.1
WORKDIR /instrumentations
ADD https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/download/v$OTEL_VERSIN/opentelemetry-javaagent.jar /instrumentations/javaagent.jar

FROM gcr.io/distroless/static-debian10
WORKDIR /root/
COPY --from=builder /go/src/github.com/keyval-dev/odigos/odiglet/app .
WORKDIR /instrumentations/
COPY --from=builder /instrumentations/ .
CMD ["/root/app"]
