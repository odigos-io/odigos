# Java Span Generator

A Java application that generates OpenTelemetry spans for load testing purposes.

## Features

- Configurable span generation rate (spans per second)
- Configurable span payload size
- OpenTelemetry integration
- Proper logging and monitoring
- Lightweight and efficient
- Multi-threaded span generation

## Environment Variables

- `SPANS_PER_SEC`: Number of spans to generate per second (default: 3000)
- `SPAN_BYTES`: Size of the payload in bytes (default: 3000)
- `OTEL_SERVICE_NAME`: Service name for OpenTelemetry (default: java-span-generator)
- `OTEL_RESOURCE_ATTRIBUTES`: Resource attributes for OpenTelemetry

## Building

### Local Build
```bash
make build
# or
mvn clean compile
```

### Package JAR
```bash
make package
# or
mvn clean package -DskipTests
```

### Docker Build
```bash
make docker-build
# or
docker build -t java-span-gen .
```

## Running

### Local Run
```bash
make run
# or
mvn exec:java -Dexec.mainClass="com.example.JavaSpanGenerator"
```

### Docker Run
```bash
make docker-run
# or
docker run -e SPANS_PER_SEC=2000 -e SPAN_BYTES=2000 java-span-gen
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

- Java 17+
- Maven 3.9+
- OpenTelemetry API 1.35.0

## Architecture

The application uses a scheduled executor to generate spans at the specified rate:
1. Creates a single-threaded executor for consistent timing
2. Schedules span generation every second
3. Generates the specified number of spans within each second
4. Includes configurable attributes on each span
5. Logs progress every second with total span count

## Maven Configuration

The project uses:
- **Maven Shade Plugin**: Creates a fat JAR with all dependencies
- **Java 17**: Modern Java features and performance
- **OpenTelemetry BOM**: Manages dependency versions
