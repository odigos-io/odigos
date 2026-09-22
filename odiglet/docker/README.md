# OSS odiglet (split bake)

Split of `odiglet/Dockerfile` for Bake / Depot builds. The monolithic `odiglet/Dockerfile` remains the CI path until make targets are wired to this directory.

## Files

| File | Purpose |
| --- | --- |
| `Dockerfile.dotnet` | Download / verify .NET auto-instrumentation |
| `Dockerfile.agents-builder` | Assemble community agents under `/instrumentations` |
| `Dockerfile.agents` | BusyBox agents image (init container) |
| `Dockerfile.agents-rhel` | RHEL agents image |
| `Dockerfile.builder` | Compile odiglet (OBI → generate → compile layers) |
| `Dockerfile.builder-debug` | Add delve on top of `odiglet-builder` |
| `Dockerfile.common-binaries` | CSI driver, deviceplugin, grpc_health_probe |
| `Dockerfile` | Distroless odiglet (default runtime image) |
| `Dockerfile.debug` | Fedora-minimal odiglet + headless delve |
| `Dockerfile.rhel` | RHEL odiglet |
| `versions` | Auto-generated agent pins (tag + digest) via `make sync-agent-versions` |
| `docker-bake.hcl` | Bake targets (also covered by root `docker-bake.hcl`) |

## Bake graph

```
odiglet-dotnet
        │
        ▼
odiglet-agents-builder ──► agents
        │               └► agents-rhel
        │               └► odiglet
        │               └► odiglet-rhel
        │               └► odiglet-debug
        │
nodejs-community-local ──► odiglet-agents-builder-nodejs-dev ──► odiglet-nodejs-dev
        │
odiglet-builder ────────────┤ (odiglet / odiglet-nodejs-dev)
        └► odiglet-builder-debug ──► odiglet-debug
odiglet-builder-rhel ───────┤ (odiglet-rhel only; RHEL=true for licenses)
odiglet-common-binaries ────┘ (odiglet / odiglet-rhel / odiglet-nodejs-dev)
```

Final images take named contexts `builder`, `common-binaries`, and `agents-builder`. Intermediate targets use `output = [{ type = "cacheonly" }]` (no `platforms`) so Bake builds them once as contexts instead of dual-solving/exporting them.

## Manual bake (until make targets exist)

From the odigos repo root. Prefer the root bake file for full OSS builds (`docker buildx bake oss`).

```bash
make sync-agent-versions
export NODEJS_COMMUNITY_VERSION=$(sed -n 's/^nodejs-community=//p' odiglet/docker/versions)
export NODEJS_COMMUNITY_14_VERSION=$(sed -n 's/^nodejs-community-14=//p' odiglet/docker/versions)
export PHP_COMMUNITY_VERSION=$(sed -n 's/^php-community=//p' odiglet/docker/versions)
export RUBY_COMMUNITY_VERSION=$(sed -n 's/^ruby-community=//p' odiglet/docker/versions)

# Full OSS (root docker-bake.hcl — default lookup)
docker buildx bake oss

# Odigelet only
docker buildx bake odiglet
# or: docker buildx bake -f odiglet/docker/docker-bake.hcl odiglet
```

Agent images default to `public.ecr.aws/odigos/agents` (`AGENTS_REGISTRY`). `make sync-agent-versions` regenerates `odiglet/docker/versions` from `odiglet/Dockerfile`.
