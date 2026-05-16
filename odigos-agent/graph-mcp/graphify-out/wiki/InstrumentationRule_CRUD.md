# InstrumentationRule CRUD

> 25 nodes · cohesion 0.13

## Key Concepts

- **RolloutConcurrencyLimiter** (28 connections) — `controllers/agentenabled/rollout/rollout_concurrency_limiter.go`
- **setConfigConcurrentRolloutLimit()** (14 connections) — `controllers/agentenabled/rollout/helpers_test.go`
- **assertWorkloadRestarted()** (9 connections) — `controllers/agentenabled/rollout/helpers_test.go`
- **newRolloutConcurrencyLimiterExhausted()** (9 connections) — `controllers/agentenabled/rollout/helpers_test.go`
- **Test_Instrumentation_RateLimited_WaitingInQueue()** (9 connections) — `controllers/agentenabled/rollout/rollout_concurrency_limiting_test.go`
- **Test_Deinstrumentation_AllowedByRateLimiter()** (8 connections) — `controllers/agentenabled/rollout/rollout_concurrency_limiting_test.go`
- **Test_Deinstrumentation_RateLimited_NoRequeue()** (8 connections) — `controllers/agentenabled/rollout/rollout_concurrency_limiting_test.go`
- **Test_RateLimiting_JobsAndCronjobs_NotAffected()** (8 connections) — `controllers/agentenabled/rollout/rollout_concurrency_limiting_test.go`
- **Test_RateLimiting_WorkloadNotRequiringRollout_NotAffected()** (8 connections) — `controllers/agentenabled/rollout/rollout_concurrency_limiting_test.go`
- **Test_RateLimiting_StaticPods_NotAffected()** (7 connections) — `controllers/agentenabled/rollout/rollout_concurrency_limiting_test.go`
- **WorkloadKey()** (6 connections) — `controllers/agentenabled/rollout/rollout.go`
- **Test_CustomValues()** (2 connections) — `controllers/agentenabled/rollout/rollout_concurrencylimiter_test.go`
- **Test_RateLimitingDisabled_ZeroValue()** (2 connections) — `controllers/agentenabled/rollout/rollout_concurrencylimiter_test.go`
- **Test_RateLimitingEnabled()** (2 connections) — `controllers/agentenabled/rollout/rollout_concurrencylimiter_test.go`
- **Test_InFlightCount()** (1 connections) — `controllers/agentenabled/rollout/rollout_concurrencylimiter_test.go`
- **Test_NilReceiver_Release()** (1 connections) — `controllers/agentenabled/rollout/rollout_concurrencylimiter_test.go`
- **Test_NilReceiver_TryAcquire()** (1 connections) — `controllers/agentenabled/rollout/rollout_concurrencylimiter_test.go`
- **Test_Release_MultipleSlots()** (1 connections) — `controllers/agentenabled/rollout/rollout_concurrencylimiter_test.go`
- **Test_Release_NonexistentWorkload_DoesNotAffectOtherSlots()** (1 connections) — `controllers/agentenabled/rollout/rollout_concurrencylimiter_test.go`
- **Test_Release_ReturnsSlot()** (1 connections) — `controllers/agentenabled/rollout/rollout_concurrencylimiter_test.go`
- **Test_SameWorkloadCanReacquire()** (1 connections) — `controllers/agentenabled/rollout/rollout_concurrencylimiter_test.go`
- **Test_SingleConcurrentRollout()** (1 connections) — `controllers/agentenabled/rollout/rollout_concurrencylimiter_test.go`
- **.InFlightCount()** (1 connections) — `controllers/agentenabled/rollout/rollout_concurrency_limiter.go`
- **.ReleaseWorkloadRolloutSlot()** (1 connections) — `controllers/agentenabled/rollout/rollout_concurrency_limiter.go`
- **.TryAcquire()** (1 connections) — `controllers/agentenabled/rollout/rollout_concurrency_limiter.go`

## Relationships

- [[Frontend Generated Models]] (80 shared connections)
- [[Pro Central Install Docs]] (42 shared connections)
- [[Auto-Instrumentation Docs]] (5 shared connections)
- [[Autoscaler Deployment Sync]] (3 shared connections)
- [[Rollout Backoff Logic]] (1 shared connections)

## Source Files

- `controllers/agentenabled/rollout/helpers_test.go`
- `controllers/agentenabled/rollout/rollout.go`
- `controllers/agentenabled/rollout/rollout_concurrency_limiter.go`
- `controllers/agentenabled/rollout/rollout_concurrency_limiting_test.go`
- `controllers/agentenabled/rollout/rollout_concurrencylimiter_test.go`

## Audit Trail

- EXTRACTED: 61 (47%)
- INFERRED: 70 (53%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*