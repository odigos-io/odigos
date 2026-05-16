# Community 210

> 9 nodes · cohesion 0.42

## Key Concepts

- **recoverFromRollback()** (10 connections) — `controllers/agentenabled/rollout/rollout.go`
- **newRecoveryIC()** (8 connections) — `controllers/agentenabled/rollout/recover_from_rollback_test.go`
- **Test_recoverFromRollback_FirstRecovery_ClearsRollbackAndSetsAnnotation()** (2 connections) — `controllers/agentenabled/rollout/recover_from_rollback_test.go`
- **Test_recoverFromRollback_MalformedProcessedAnnotation_OverwritesAndRecovers()** (2 connections) — `controllers/agentenabled/rollout/recover_from_rollback_test.go`
- **Test_recoverFromRollback_NilAnnotationsMap_InitializesAndSets()** (2 connections) — `controllers/agentenabled/rollout/recover_from_rollback_test.go`
- **Test_recoverFromRollback_NoRecoveryAnnotation_NoChange()** (2 connections) — `controllers/agentenabled/rollout/recover_from_rollback_test.go`
- **Test_recoverFromRollback_ProcessedMatchesDesired_NoChange()** (2 connections) — `controllers/agentenabled/rollout/recover_from_rollback_test.go`
- **Test_recoverFromRollback_RollbackNotOccurred_NoChange()** (2 connections) — `controllers/agentenabled/rollout/recover_from_rollback_test.go`
- **Test_recoverFromRollback_SecondRecovery_UpdatesAnnotation()** (2 connections) — `controllers/agentenabled/rollout/recover_from_rollback_test.go`

## Relationships

- [[Community 214]] (30 shared connections)
- [[Rollout Backoff Logic]] (1 shared connections)
- [[Frontend Generated Models]] (1 shared connections)

## Source Files

- `controllers/agentenabled/rollout/recover_from_rollback_test.go`
- `controllers/agentenabled/rollout/rollout.go`

## Audit Trail

- EXTRACTED: 18 (56%)
- INFERRED: 14 (44%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*