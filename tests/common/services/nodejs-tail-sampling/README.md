# Node.js Tail Sampling Service

Express test app used by tail-sampling e2e tests. Serves error-focused routes
(`/ok`, `/error`, `/alternate`, `/hops`) and duration-focused routes
(`/duration`, `/duration/short|medium|long`) for gateway tail sampling dry-run
asserts.

## Endpoints

| Method | Path | Query params | Status | Description |
|--------|------|--------------|--------|-------------|
| GET | `/healthz` | — | 200 | Liveness check; returns `{ status: "healthy" }` |
| GET | `/ok` | — | 200 | Successful baseline request for cost-reduction tail sampling |
| GET | `/error` | — | 500 | Always returns HTTP 500 for error-focused tail sampling |
| GET | `/alternate` | — | 200 / 500 | Alternates HTTP 200 and 500 on each request (in-process toggle) |
| GET | `/hops` | `hops` (default `1`) | 200 (caller) / 500 (final hop) | Self HTTP hops; final hop always 500, callers always get 200 |
| GET | `/duration` | `ms` (default `0`), `error` | 200 / 500 | Response delayed by `?ms=`; `?error=true` forces 500 |
| GET | `/duration/short` | `error` | 200 / 500 | ~50ms delay; sampled through the 10% cost-reduction rule |
| GET | `/duration/medium` | `error` | 200 / 500 | ~750ms delay; sampled at least 50% |
| GET | `/duration/long` | `error` | 200 / 500 | ~1500ms delay; sampled at 100% |
| GET | `/http-server/templated/:id/users/:name/orders/:uuid` | — | 200 | Templated path; vary `id` / `name` / `uuid` per call so `http.route` is the template, not the concrete URL |
| GET | `/http-server/prefix/:id/items` | — | 200 | Templated prefix path for `routePrefix` sampling |
| GET | `/http-server/prefix/:id/items/:itemId` | — | 200 | Nested path under the templated prefix; still matches `routePrefix` |
| GET | `/http-server/static/leading-slash` | — | 200 | Static path; sampling rule `route` includes a leading `/` (`AllStatic`) |
| GET | `/http-server/static/no-leading-slash` | — | 200 | Static path; sampling rule `route` omits the leading `/` (`AllStatic`) |

Notes:

- `error=true` or `error=1` forces HTTP 500 where supported.
- `hops` must be an integer ≥ 1 (invalid values default to `1`).
- `ms` must be an integer ≥ 0 (invalid values default to `0`).

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
  -t us-central1-docker.pkg.dev/odigos-cloud/components/nodejs-tail-sampling:v0.0.3 \
  --push .
```

## Testing Locally

```bash
docker build -t nodejs-tail-sampling:v0.0.3 .
docker run -p 8080:8080 nodejs-tail-sampling:v0.0.3

curl http://localhost:8080/healthz
```

## Usage in Tests

This service is used in:
- `tests/e2e/tail-sampling-nodejs/` (pulls `us-central1-docker.pkg.dev/odigos-cloud/components/nodejs-tail-sampling:v0.0.3`)
