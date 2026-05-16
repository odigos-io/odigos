# Sampling Matchers (Collector)

> 49 nodes · cohesion 0.19

## Key Concepts

- **Do()** (56 connections) — `controllers/agentenabled/rollout/rollout.go`
- **newTestSetup()** (53 connections) — `controllers/agentenabled/rollout/helpers_test.go`
- **newRolloutConcurrencyLimiter()** (41 connections) — `controllers/agentenabled/rollout/helpers_test.go`
- **NewMockInstrumentationConfig()** (41 connections) — `internal/testutil/mocks.go`
- **mockICRolloutRequiredDistro()** (20 connections) — `controllers/agentenabled/rollout/helpers_test.go`
- **assertTriggeredRolloutNoRequeue()** (16 connections) — `controllers/agentenabled/rollout/helpers_test.go`
- **mockICMidRollout()** (13 connections) — `controllers/agentenabled/rollout/helpers_test.go`
- **newMockDeploymentMidRollout()** (12 connections) — `controllers/agentenabled/rollout/helpers_test.go`
- **Test_Rollback_BypassesRateLimiter_WhenOtherDeploymentsWaitingInQueue()** (12 connections) — `controllers/agentenabled/rollout/rollout_rollback_test.go`
- **Test_Instrumentation_FirstWorkload_AllowedByRateLimiter()** (10 connections) — `controllers/agentenabled/rollout/rollout_concurrency_limiting_test.go`
- **Test_TriggeredRollout_PodInMidRollout_RollbackRestartAnnotation()** (10 connections) — `controllers/agentenabled/rollout/rollout_rollback_test.go`
- **rollout_rollback_test.go** (9 connections) — `controllers/agentenabled/rollout/rollout_rollback_test.go`
- **Test_Instrumentation_MultipleWorkloads_HigherLimitAllowsMore()** (9 connections) — `controllers/agentenabled/rollout/rollout_concurrency_limiting_test.go`
- **Test_Instrumentation_SecondWorkload_RateLimited()** (9 connections) — `controllers/agentenabled/rollout/rollout_concurrency_limiting_test.go`
- **Test_NoRateLimit_AllWorkloadsProcessedImmediately()** (9 connections) — `controllers/agentenabled/rollout/rollout_concurrency_limiting_test.go`
- **Test_NoRollout_PodInMidRollout_ClientUpdateError()** (9 connections) — `controllers/agentenabled/rollout/rollout_rollback_test.go`
- **Test_NoRollout_PodInMidRollout_WithRollbackDisabled()** (9 connections) — `controllers/agentenabled/rollout/rollout_rollback_test.go`
- **newMockCrashingPod()** (8 connections) — `controllers/agentenabled/rollout/helpers_test.go`
- **Test_RateLimiting_NilRateLimiter_FailsOpen()** (8 connections) — `controllers/agentenabled/rollout/rollout_concurrency_limiting_test.go`
- **Test_RateLimiting_PreviousRolloutOngoing_RateLimiterNotConsumed()** (8 connections) — `controllers/agentenabled/rollout/rollout_concurrency_limiting_test.go`
- **Test_NoRollout_PodInMidRollout_BackoffDurationLessThanGraceTime()** (8 connections) — `controllers/agentenabled/rollout/rollout_rollback_test.go`
- **Test_NoRollout_PodInMidRollout_FailedToGetBackoffInfo()** (8 connections) — `controllers/agentenabled/rollout/rollout_rollback_test.go`
- **Test_Rollback_WebhookInstrumentedPodCrashloops_WhileWorkloadRolloutNotStarted()** (8 connections) — `controllers/agentenabled/rollout/rollout_rollback_test.go`
- **Test_Rollback_WebhookInstrumentedPodCrashloops_WhileWorkloadWaitingInQueue()** (8 connections) — `controllers/agentenabled/rollout/rollout_rollback_test.go`
- **Test_NoRollout_PodInMidRollout_AlreadyComplete()** (8 connections) — `controllers/agentenabled/rollout/rollout_rollout_in_progress_test.go`
- *... and 24 more nodes in this community*

## Relationships

- [[Frontend Generated Models]] (499 shared connections)
- [[Auto-Instrumentation Docs]] (29 shared connections)
- [[Autoscaler Deployment Sync]] (13 shared connections)
- [[Rollout Backoff Logic]] (11 shared connections)
- [[Pro Central Install Docs]] (8 shared connections)
- [[Odiglet Runtime Inspection]] (1 shared connections)
- [[Community 317]] (1 shared connections)
- [[Community 214]] (1 shared connections)

## Source Files

- `controllers/agentenabled/rollout/helpers_test.go`
- `controllers/agentenabled/rollout/rollout.go`
- `controllers/agentenabled/rollout/rollout_concurrency_limiting_test.go`
- `controllers/agentenabled/rollout/rollout_crashloop_recovery_test.go`
- `controllers/agentenabled/rollout/rollout_per_workload_test.go`
- `controllers/agentenabled/rollout/rollout_rollback_test.go`
- `controllers/agentenabled/rollout/rollout_rollout_in_progress_test.go`
- `controllers/agentenabled/rollout/rollout_rolloutdisabled_test.go`
- `controllers/agentenabled/rollout/rollout_test.go`
- `internal/testutil/mocks.go`

## Audit Trail

- EXTRACTED: 82 (15%)
- INFERRED: 481 (85%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*