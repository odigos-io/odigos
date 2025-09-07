# Go Span Generator

A simple Go application that generates OpenTelemetry spans for load testing purposes.

## Features

- Configurable span generation rate (spans per second)
- Configurable span payload size
- OpenTelemetry integration
- Lightweight and efficient

## Environment Variables

- `SPANS_PER_SEC`: Number of spans to generate per second (default: 5000)
- `SPAN_BYTES`: Size of the payload in bytes (default: 2000)
- `OTEL_SERVICE_NAME`: Service name for OpenTelemetry (default: go-span-gen)
- `OTEL_RESOURCE_ATTRIBUTES`: Resource attributes for OpenTelemetry

## Building

### Local Build
```bash
go mod tidy
go build -o go-span-gen .
```

### Docker Build
```bash
docker build -t go-span-gen .
```

## Running

### Local Run
```bash
./go-span-gen
```

### Docker Run
```bash
docker run -e SPANS_PER_SEC=5000 -e SPAN_BYTES=2000 go-span-gen
```

## Usage in Kubernetes

This application is designed to be deployed in Kubernetes with Odigos instrumentation. The deployment will automatically generate spans that can be collected and processed by your observability pipeline.

## Performance

- **Default Rate**: 5,000 spans/second
- **Payload Size**: 2,000 bytes per span
- **Resource Usage**: 500m-2000m CPU, 256Mi-2Gi memory
- **Replicas**: 5 pods (25,000 total spans/sec)

## Dependencies

- Go 1.22+
- go.opentelemetry.io/otel v1.37.0
