# Crash Demo Service

A test service designed to simulate application issues with instrumentation for testing Odigos auto-rollback functionality.

## Behavior

1. **Initial State**: Starts successfully, serves HTTP requests on port 3000 (Node.js 20.19.2)
2. **Startup Check**: Checks for OTEL_SERVICE_NAME environment variable at startup
3. **Immediate Crash**: If instrumentation detected at startup → crashes immediately (before serving any requests)
4. **Rollback Testing**: Designed to trigger Odigos automatic rollback after grace period

## Currently built & pushed manually.

```bash
# Navigate to the service directory
cd tests/common/services/crash-demo

# Build for AMD64 (GitHub Actions compatibility)
docker build --platform linux/amd64 -t crash-demo:v2.0.0 .

# Alternative: Use buildx for multi-platform build
# docker buildx build --platform linux/amd64,linux/arm64 -t crash-demo:v2.0.0 .

# Tag for ghcr.io
docker tag crash-demo:v2.0.0 ghcr.io/odigos-io/simple-demo/odigos-demo-crash:v2.0.0

# Push to GitHub Container Registry
docker push ghcr.io/odigos-io/simple-demo/odigos-demo-crash:v2.0.0
```

## Testing Locally

```bash
# Run without instrumentation (should work fine)
docker run -p 3000:3000 crash-demo:v2.0.0

# Run with simulated instrumentation (should crash immediately)
docker run -p 3000:3000 -e OTEL_SERVICE_NAME=crash-demo crash-demo:v2.0.0

# Test endpoints (only works without instrumentation)
curl http://localhost:3000
```

## Usage in Tests

This service is used in:
- `tests/e2e/instrumentation-rollback/`
- `tests/e2e/instrumentation-rollback-stability-window/`

The service provides realistic crash behavior for testing Odigos rollback scenarios.
