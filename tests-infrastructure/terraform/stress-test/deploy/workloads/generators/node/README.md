# Node.js Span Generator

A Node.js application that generates OpenTelemetry spans for load testing purposes.

## Features

- Configurable span generation rate (spans per second)
- Configurable span payload size
- OpenTelemetry integration
- Proper logging and monitoring
- Lightweight and efficient

## Environment Variables

- `SPANS_PER_SEC`: Number of spans to generate per second (default: 6000)
- `SPAN_BYTES`: Size of the payload in bytes (default: 4000)
- `OTEL_SERVICE_NAME`: Service name for OpenTelemetry (default: node-span-generator)
- `OTEL_RESOURCE_ATTRIBUTES`: Resource attributes for OpenTelemetry

## Building

### Local Build
```bash
make build
# or
npm install
```

### Docker Build
```bash
make docker-build
# or
docker build -t node-span-gen .
```

## Running

### Local Run
```bash
make run
# or
node app.js
```

### Docker Run
```bash
make docker-run
# or
docker run -e SPANS_PER_SEC=3000 -e SPAN_BYTES=2000 node-span-gen
```

## ECR Deployment

### Build and Push to ECR
```bash
make ecr-build-push
# or
./build-and-push.sh
```

### Deploy to Kubernetes
```bash
kubectl apply -f deployment.yaml
```

## Usage in Kubernetes

This application is designed to be deployed in Kubernetes with Odigos instrumentation. The deployment will automatically generate spans that can be collected and processed by your observability pipeline.

## Dependencies

- Node.js 20+
- @opentelemetry/api 1.8.0

## Architecture

The application uses `setInterval` to generate spans at the specified rate:
1. Generates the specified number of spans every second
2. Each span includes configurable attributes
3. Logs progress every second with total span count
4. Maintains consistent timing using Node.js event loop
