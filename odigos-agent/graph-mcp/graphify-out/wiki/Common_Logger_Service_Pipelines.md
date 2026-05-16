# Common Logger Service Pipelines

> 19 nodes · cohesion 0.16

## Key Concepts

- **helpers_test.go** (25 connections) — `controllers/agentenabled/helpers_test.go`
- **hasUninstrumentedPodsWithBackoff()** (7 connections) — `controllers/agentenabled/sync.go`
- **newSyncTestSetup()** (6 connections) — `controllers/agentenabled/helpers_test.go`
- **TestHasUninstrumentedPodsWithBackoff_CrashLoopBackOff()** (6 connections) — `controllers/agentenabled/sync_test.go`
- **TestHasUninstrumentedPodsWithBackoff_Job_Skipped()** (6 connections) — `controllers/agentenabled/sync_test.go`
- **TestHasUninstrumentedPodsWithBackoff_NoPods()** (5 connections) — `controllers/agentenabled/sync_test.go`
- **sync_test.go** (4 connections) — `controllers/agentenabled/sync_test.go`
- **testSetup** (4 connections) — `controllers/agentenabled/rollout/helpers_test.go`
- **TestHasUninstrumentedPodsWithBackoff_WorkloadNotFound()** (3 connections) — `controllers/agentenabled/sync_test.go`
- **newMockArgoRollout()** (3 connections) — `controllers/agentenabled/rollout/helpers_test.go`
- **newMockCronJob()** (2 connections) — `controllers/agentenabled/helpers_test.go`
- **newMockJob()** (2 connections) — `controllers/agentenabled/helpers_test.go`
- **syncTestSetup** (2 connections) — `controllers/agentenabled/helpers_test.go`
- **assertTriggeredRollback()** (2 connections) — `controllers/agentenabled/rollout/helpers_test.go`
- **newCrashLoopBackOffPodWithoutOdigosLabel()** (2 connections) — `controllers/agentenabled/rollout/helpers_test.go`
- **.newFakeClient()** (2 connections) — `controllers/agentenabled/rollout/helpers_test.go`
- **.newFakeClientWithICUpdateError()** (2 connections) — `controllers/agentenabled/rollout/helpers_test.go`
- **newHealthyPod()** (1 connections) — `controllers/agentenabled/rollout/helpers_test.go`
- **.newFakeClientWithStatus()** (1 connections) — `controllers/agentenabled/rollout/helpers_test.go`

## Relationships

- [[Autoscaler Deployment Sync]] (56 shared connections)
- [[Frontend Generated Models]] (18 shared connections)
- [[Rollout Backoff Logic]] (5 shared connections)
- [[Auto-Instrumentation Docs]] (3 shared connections)
- [[Odiglet Runtime Inspection]] (2 shared connections)
- [[CLI Kube Client]] (1 shared connections)

## Source Files

- `controllers/agentenabled/helpers_test.go`
- `controllers/agentenabled/rollout/helpers_test.go`
- `controllers/agentenabled/sync.go`
- `controllers/agentenabled/sync_test.go`

## Audit Trail

- EXTRACTED: 52 (61%)
- INFERRED: 33 (39%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*