# Java Head Sampling Service

Spring Boot test app used by head-sampling e2e tests. Serves health probes, sampling percentage/route endpoints, HTTP route-matching paths, and optional periodic outbound HTTP client traffic.

## Currently built & pushed manually.

```bash
# Navigate to the service directory
cd tests/common/services/java-head-sampling

# Authenticate (if needed)
gcloud auth configure-docker us-central1-docker.pkg.dev
# If push fails with "docker-credential-gcloud: executable file not found" (Homebrew gcloud):
# ln -sf /opt/homebrew/share/google-cloud-sdk/bin/docker-credential-gcloud /opt/homebrew/bin/docker-credential-gcloud

# Build and push multi-arch (amd64 + arm64) to Artifact Registry
docker buildx build --platform linux/amd64,linux/arm64 \
  -t us-central1-docker.pkg.dev/odigos-cloud/components/java-head-sampling:v0.0.1 \
  --push .
```

## Testing Locally

```bash
docker build -t java-head-sampling:v0.0.1 .
docker run -p 8080:8080 java-head-sampling:v0.0.1

curl http://localhost:8080/healthz
```

## Usage in Tests

This service is used in:
- `tests/e2e/head-sampling-http-java/` (pulls `us-central1-docker.pkg.dev/odigos-cloud/components/java-head-sampling:v0.0.1`)
