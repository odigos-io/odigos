# Profiling feature: PR split plan

This document breaks the continuous profiling work into **small, reviewable PRs**. Use it as a checklist when opening PRs against `main` (or your release branch).

**Merge order (dependencies):**

```
PR1 → PR2 → PR3
         ↘ PR4 (Helm; can open after PR2–3 land or in parallel if coordinated)
PR5 (UI backend) depends on PR3–4 for a full E2E install; can merge after PR2 if only testing APIs locally.
PR6 (webapp) depends on PR5.
PR7 optional last (docs + heavy fixtures).
```

---

## PR 1 — Collector: ebpf-profiler receiver + `odigosotelcol` build

**Goal:** The collector image includes the **profiling** receiver from `go.opentelemetry.io/ebpf-profiler`; CI builds succeed.

**Paths (typical):**

- `collector/builder-config.yaml`
- `collector/Dockerfile`
- `collector/README.md`, `collector/receivers/README.md` (pattern vs `odigosebpfreceiver`)
- `collector/odigosotelcol/components.go`
- `collector/odigosotelcol/go.mod`
- `collector/odigosotelcol/go.sum`
- `collector/odigosotelcol/main.go`
- `collector/odigosotelcol/main_windows.go`
- `collector/docs/profiling-pr-split.md` (this roadmap; optional to merge with PR1 or a meta-PR)

**Review focus:** OCB module pins, import path `.../ebpf-profiler/collector`, Dockerfile `GOMAXPROCS`.

**How to test:** `make -C collector build-odigoscol` (or `go build` in `collector/odigosotelcol`).

**Branch name (suggested):** `profiling/pr-1-collector-ebpf-receiver`

---

## PR 2 — Autoscaler: node collector profiles pipeline + ConfigMap overrides

**Goal:** Autoscaler emits a **`profiles`** pipeline on the node collector (receivers, processors, exporter) and supports optional CM **`odigos-node-collector-profiles-config`**.

**Paths (typical):**

- `api/k8sconsts/nodecollector.go`
- `autoscaler/controllers/nodecollector/collectorconfig/common.go`
- `autoscaler/controllers/nodecollector/collectorconfig/profiles.go`
- `autoscaler/controllers/nodecollector/configmap.go`
- `autoscaler/controllers/nodecollector/configmap_test.go`
- `autoscaler/controllers/nodecollector/testdata/logs_included.yaml`
- `autoscaler/controllers/nodecollector/testdata/traces_only_no_loadbalancing.yaml`

**Review focus:** Processor order (`memory_limiter` → `k8sattributes/profiles` → `resource/profiles-service-name`), no `batch` on profiles, CM override keys `profiles` / `profilingReceiver`.

**How to test:** `go test ./controllers/nodecollector/...` from `autoscaler/` module.

**Branch name (suggested):** `profiling/pr-2-autoscaler-node-profiles`

**Depends on:** PR1 (collector binary must define `profiling` receiver for config to be valid at runtime).

---

## PR 3 — Autoscaler: gateway profiles pipeline

**Goal:** Cluster gateway accepts and forwards **profiles** when env vars are set; feature gate enabled.

**Paths (typical):**

- `autoscaler/controllers/clustercollector/configmap.go`
- `autoscaler/controllers/clustercollector/deployment.go`

**Review focus:** Empty processor list or minimal pipeline, no batch on profiles, `PROFILES_EXPORTER_*` / endpoint envs, `service.profilesSupport` on gateway.

**Depends on:** PR2 (conceptually); can be same release train as PR2.

**Branch name (suggested):** `profiling/pr-3-autoscaler-gateway-profiles`

---

## PR 4 — Helm: templates, values, example overlays

**Goal:** Installable path: odiglet + gateway + UI ports/env; defaults safe (profiling off until values enable).

**Paths (typical):**

- `helm/odigos/templates/autoscaler/deployment.yaml`
- `helm/odigos/templates/nodecollector-profiles-config.yaml`
- `helm/odigos/templates/odiglet/daemonset.yaml`
- `helm/odigos/templates/ui/deployment.yaml`
- `helm/odigos/templates/ui/service.yaml`
- `helm/odigos/values.yaml`
- `helm/odigos/values.schema.json`
- `helm/odigos/values-collector-demo.yaml`
- `helm/odigos/values-profiles-deploy.yaml`
- `helm/odigos/values-profiles-override.yaml`
- `scripts/build-and-push-profiling-images.sh`

**Review focus:** Privileged / `requirePrivileged`, Service port **4318**, autoscaler env wiring.

**Depends on:** PR2–3 behavior (templates should match emitted config).

**Branch name (suggested):** `profiling/pr-4-helm-profiling`

---

## PR 5 — Frontend (Go): OTLP profiles receiver + store + HTTP API

**Goal:** UI pod ingests OTLP profiles and serves `/api/.../profiling` without requiring the Next app.

**Paths (typical):**

- `frontend/main.go`
- `frontend/middlewares/csrf.go`
- `frontend/go.mod`
- `frontend/go.sum`
- `frontend/services/collector_profiles/**` (exclude or defer heavy `profile-dumps/` and huge `testdata/` to PR7 if desired)
- `frontend/scripts/debug-ui-endpoints.sh` (optional)

**Review focus:** `SourceKeyFromResource`, buffer limits, port **4318**, CSRF exceptions for profiling routes.

**How to test:** `go test ./services/collector_profiles/...` from `frontend/` module.

**Branch name (suggested):** `profiling/pr-5-ui-backend-profiles`

**Depends on:** PR3–4 for cluster E2E; can be tested with mocked OTLP before that.

---

## PR 6 — Frontend (Next): Profiling page + flame graph

**Goal:** Product UI for profiling under Sources.

**Paths (typical):**

- `frontend/webapp/**` (profiling page, components, hooks, utils, `next.config.ts`, routes, `yarn.lock`, `.env.example`)

**Review focus:** Polling, CSRF headers, dev rewrites to backend.

**Depends on:** PR5 APIs.

**Branch name (suggested):** `profiling/pr-6-webapp-profiling-ui`

---

## PR 7 — Docs + large fixtures (optional)

**Goal:** Keep code PRs small; land documentation and heavy JSON separately.

**Paths (typical):**

- `frontend/services/collector_profiles/*.md`
- `frontend/services/collector_profiles/ASCII_ARCHITECTURE.txt`
- `frontend/services/collector_profiles/profile-dumps/*.json`
- `frontend/services/collector_profiles/testdata/accounting-merged.json` (if extracted from PR5)

**Branch name (suggested):** `profiling/pr-7-docs-and-profilers-testdata`

---

## Progress checklist

Use this when merging to `main`:

- [x] PR1 Collector (branch `profiling/pr-1-collector-ebpf-receiver`)
- [ ] PR2 Autoscaler node + tests
- [ ] PR3 Autoscaler gateway
- [ ] PR4 Helm
- [ ] PR5 UI backend
- [ ] PR6 Webapp
- [ ] PR7 Docs/fixtures (optional)

---

## Git tips (extracting from a single branch)

If all work lives on one branch (e.g. `pr-profiler-testing1`):

1. **PR1 only:** From `main`, create `profiling/pr-1-...`, then bring in paths with  
   `git checkout <source-branch> -- collector/`  
   (or `git add` only those paths after a partial cherry-pick).

2. After each PR merges, **rebase** the next branch onto updated `main` to reduce conflicts.

3. Keep a **stash** or **backup branch** of the full feature before splitting commits.

---

## Workspace notes

- **PR split plan** lives in this file; update checkboxes as PRs merge.
- **PR1** (`profiling/pr-1-collector-ebpf-receiver`): adds `go.opentelemetry.io/ebpf-profiler` to `builder-config.yaml`, Dockerfile `GOMAXPROCS`, and regenerates `collector/odigosotelcol/` via `make genodigoscol`. Documents the **same receiver wiring pattern** as `odigosebpfreceiver` (see `collector/receivers/README.md`).  
  - **Upstream version:** pin the **latest git tag** from [open-telemetry/opentelemetry-ebpf-profiler](https://github.com/open-telemetry/opentelemetry-ebpf-profiler/releases) (CalVer `v0.0.YYYYMMDD`). As of this doc, **`v0.0.202610`** (verify with `git ls-remote --tags` before release).  
  - On **macOS**, `make genodigoscol` may fail at the final compile step (`unix.Prlimit` in ebpf-profiler). **Linux** (CI or `docker build -f collector/Dockerfile`) is the source of truth for a green binary build.
- If you **stashed** the full feature branch before splitting: `git stash list` then `git stash pop` on your integration branch (e.g. `pr-profiler-testing1`) to restore remaining work after PR1 is committed/pushed.

---

*Last updated: PR plan file + PR1 branch started (`profiling/pr-1-collector-ebpf-receiver`).*
