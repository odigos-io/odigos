# Instrumentor Rule Mocks

> 29 nodes · cohesion 0.10

## Key Concepts

- **rollout.go** (13 connections) — `controllers/agentenabled/rollout/rollout.go`
- **WorkloadHasNonInstrumentedPodInBackoff()** (9 connections) — `controllers/agentenabled/rollout/rollout.go`
- **rollout_backoff_test.go** (5 connections) — `controllers/agentenabled/rollout/rollout_backoff_test.go`
- **podBackOffDuration()** (5 connections) — `controllers/agentenabled/rollout/rollout_backoff_helpers.go`
- **TestWorkloadHasPodInBackoff_CrashLoopBackOff_WithOdigosLabel()** (5 connections) — `controllers/agentenabled/rollout/rollout_backoff_test.go`
- **TestWorkloadHasPodInBackoff_InitContainerBackoff()** (5 connections) — `controllers/agentenabled/rollout/rollout_backoff_test.go`
- **TestWorkloadHasPodInBackoff_NoPods()** (5 connections) — `controllers/agentenabled/rollout/rollout_backoff_test.go`
- **triggerRollback()** (5 connections) — `controllers/agentenabled/rollout/rollout.go`
- **rollout_backoff_helpers.go** (4 connections) — `controllers/agentenabled/rollout/rollout_backoff_helpers.go`
- **TestWorkloadHasPodInBackoff_HealthyPods()** (4 connections) — `controllers/agentenabled/rollout/rollout_backoff_test.go`
- **TestWorkloadHasPodInBackoff_StaticPod()** (4 connections) — `controllers/agentenabled/rollout/rollout_backoff_test.go`
- **instrumentedPodsSelector()** (4 connections) — `controllers/agentenabled/rollout/rollout.go`
- **rolloutRestartWorkload()** (4 connections) — `controllers/agentenabled/rollout/rollout.go`
- **podHasBackOff()** (3 connections) — `controllers/agentenabled/rollout/rollout_backoff_helpers.go`
- **notInstrumentedWorkloadPodsSelector()** (3 connections) — `controllers/agentenabled/rollout/rollout.go`
- **rolloutCondition()** (3 connections) — `controllers/agentenabled/rollout/rollout.go`
- **shouldTriggerRollback()** (3 connections) — `controllers/agentenabled/rollout/rollout.go`
- **workloadHasOdigosAgents()** (3 connections) — `controllers/agentenabled/rollout/rollout.go`
- **workloadLabelSelector()** (3 connections) — `controllers/agentenabled/rollout/rollout.go`
- **newConditionAgentDisabledDueToBackoff()** (2 connections) — `controllers/agentenabled/rollout/conditions.go`
- **newConditionFailedToPatch()** (2 connections) — `controllers/agentenabled/rollout/conditions.go`
- **newConditionTriggeredWithMessage()** (2 connections) — `controllers/agentenabled/rollout/conditions.go`
- **newCrashLoopBackOffPod()** (2 connections) — `controllers/agentenabled/rollout/helpers_test.go`
- **newCrashLoopBackOffStaticPod()** (2 connections) — `controllers/agentenabled/rollout/helpers_test.go`
- **newImagePullBackOffPod()** (2 connections) — `controllers/agentenabled/rollout/helpers_test.go`
- *... and 4 more nodes in this community*

## Relationships

- [[Rollout Backoff Logic]] (82 shared connections)
- [[Frontend Generated Models]] (11 shared connections)
- [[Autoscaler Deployment Sync]] (5 shared connections)
- [[Auto-Instrumentation Docs]] (4 shared connections)
- [[Autoscaler Collector Config Domains]] (3 shared connections)
- [[Pro Central Install Docs]] (1 shared connections)
- [[Community 214]] (1 shared connections)
- [[CLI Kube Client]] (1 shared connections)

## Source Files

- `controllers/agentenabled/rollout/conditions.go`
- `controllers/agentenabled/rollout/helpers_test.go`
- `controllers/agentenabled/rollout/rollout.go`
- `controllers/agentenabled/rollout/rollout_backoff_helpers.go`
- `controllers/agentenabled/rollout/rollout_backoff_test.go`

## Audit Trail

- EXTRACTED: 67 (62%)
- INFERRED: 41 (38%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*