# Node.js Tail Sampling Service

Express test app used by tail-sampling e2e tests. Serves error-focused routes
(`/ok`, `/error`, `/alternate`, `/hops`) and duration-focused routes
(`/duration`, `/duration/short|medium|long`) for gateway tail sampling dry-run
asserts.

## Currently built & pushed manually.

```bash
# Navigate to the service directory
cd tests/common/services/nodejs-tail-sampling

# Authenticate (if needed)
gcloud auth configure-docker us-central1-docker.pkg.dev
# If push fails with "docker-credential-gcloud: executable file not found" (Homebrew gcloud):
# ln -sf /opt/homebrew/share/google-cloud-sdk/bin/docker-credential-gcloud /opt/homebrew/bin/docker-credential-gcloud

# Build and push multi-arch (amd64 + arm64) to Artifact Registry
docker buildx build --platform linux/amd64,linux/arm64 \
  -t us-central1-docker.pkg.dev/odigos-cloud/components/nodejs-tail-sampling:v0.0.1 \
  --push .
```

## Testing Locally

```bash
docker build -t nodejs-tail-sampling:v0.0.1 .
docker run -p 8080:8080 nodejs-tail-sampling:v0.0.1

curl http://localhost:8080/healthz
```

## Usage in Tests

This service is used in:
- `tests/e2e/tail-sampling-nodejs/` (pulls `us-central1-docker.pkg.dev/odigos-cloud/components/nodejs-tail-sampling:v0.0.1`)
