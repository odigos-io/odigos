# Python Span Generator

A Python application that generates OpenTelemetry spans for load testing purposes.

## Features

- Configurable span generation rate (spans per second)
- Configurable span payload size
- OpenTelemetry integration
- Proper logging and monitoring
- Lightweight and efficient

## Environment Variables

- `SPANS_PER_SEC`: Number of spans to generate per second (default: 2000)
- `SPAN_BYTES`: Size of the payload in bytes (default: 2000)
- `OTEL_SERVICE_NAME`: Service name for OpenTelemetry (default: python-span-generator)
- `OTEL_RESOURCE_ATTRIBUTES`: Resource attributes for OpenTelemetry

## Building

### Local Build
```bash
make build
# or
pip install -r requirements.txt
```

### Docker Build
```bash
make docker-build
# or
docker build -t python-span-gen .
```

## Running

### Local Run
```bash
make run
# or
python app.py
```

### Docker Run
```bash
make docker-run
# or
docker run -e SPANS_PER_SEC=2000 -e SPAN_BYTES=2000 python-span-gen
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

- Python 3.12+
- opentelemetry-api==1.26.0

## Architecture

The application uses a time-based loop to generate spans at the specified rate:
1. Records start time
2. Generates the specified number of spans
3. Calculates elapsed time
4. Sleeps for the remaining time to maintain 1-second intervals
5. Logs progress every second

This ensures consistent span generation rates regardless of processing time.
