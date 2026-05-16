# Autoscaler Sampler Handlers

> 19 nodes · cohesion 0.16

## Key Concepts

- **mocks.go** (20 connections) — `internal/testutil/mocks.go`
- **assertNoStatusChange()** (14 connections) — `controllers/agentenabled/rollout/helpers_test.go`
- **Test_NoRollout_JobOrCronjobNoIC()** (9 connections) — `controllers/agentenabled/rollout/rollout_per_workload_test.go`
- **rollout_per_workload_test.go** (7 connections) — `controllers/agentenabled/rollout/rollout_per_workload_test.go`
- **Test_Rollout_ICNil_HasAgents_RestartsUsing_rolloutRestartWorkload()** (7 connections) — `controllers/agentenabled/rollout/rollout_per_workload_test.go`
- **Test_NoRollout_StaticPodNoIC()** (6 connections) — `controllers/agentenabled/rollout/rollout_per_workload_test.go`
- **Test_Rollout_ICNil_HasAgents_RestartsArgoRollout()** (6 connections) — `controllers/agentenabled/rollout/rollout_per_workload_test.go`
- **Test_Rollout_ICNil_HasAgents_RestartsDaemonSet()** (6 connections) — `controllers/agentenabled/rollout/rollout_per_workload_test.go`
- **Test_Rollout_ICNil_HasAgents_RestartsStatefulSet()** (6 connections) — `controllers/agentenabled/rollout/rollout_per_workload_test.go`
- **NewMockTestStaticPod()** (4 connections) — `internal/testutil/mocks.go`
- **NewMockTestJob()** (3 connections) — `internal/testutil/mocks.go`
- **NewMockTestCronJob()** (2 connections) — `internal/testutil/mocks.go`
- **NewMockTestDaemonSet()** (2 connections) — `internal/testutil/mocks.go`
- **NewMockTestStatefulSet()** (2 connections) — `internal/testutil/mocks.go`
- **NewMockDataCollection()** (1 connections) — `internal/testutil/mocks.go`
- **NewMockOdigosConfig()** (1 connections) — `internal/testutil/mocks.go`
- **NewMockRegexSource()** (1 connections) — `internal/testutil/mocks.go`
- **NewMockSource()** (1 connections) — `internal/testutil/mocks.go`
- **NewOdigosSystemNamespace()** (1 connections) — `internal/testutil/mocks.go`

## Relationships

- [[Frontend Generated Models]] (64 shared connections)
- [[Auto-Instrumentation Docs]] (32 shared connections)
- [[Autoscaler Deployment Sync]] (2 shared connections)
- [[CLI Kube Client]] (1 shared connections)

## Source Files

- `controllers/agentenabled/rollout/helpers_test.go`
- `controllers/agentenabled/rollout/rollout_per_workload_test.go`
- `internal/testutil/mocks.go`

## Audit Trail

- EXTRACTED: 44 (44%)
- INFERRED: 55 (56%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*