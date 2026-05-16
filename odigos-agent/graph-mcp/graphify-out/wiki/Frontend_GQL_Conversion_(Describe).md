# Frontend GQL Conversion (Describe)

> 20 nodes · cohesion 0.15

## Key Concepts

- **sync.go** (15 connections) — `controllers/agentenabled/sync.go`
- **updateInstrumentationConfigSpec()** (13 connections) — `controllers/agentenabled/sync.go`
- **calculateContainerAgentConfig()** (4 connections) — `controllers/agentenabled/sync.go`
- **isReadyForInstrumentation()** (4 connections) — `controllers/agentenabled/sync.go`
- **reconcileWorkload()** (4 connections) — `controllers/agentenabled/sync.go`
- **HashForContainersConfig()** (4 connections) — `controllers/agentenabled/rollout/hash.go`
- **getEnvInjectionDecision()** (3 connections) — `controllers/agentenabled/sync.go`
- **getEnvVarFromList()** (3 connections) — `controllers/agentenabled/sync.go`
- **isLoaderInjectionSupportedByRuntimeDetails()** (3 connections) — `controllers/agentenabled/sync.go`
- **containerConfigToStatusCondition()** (2 connections) — `controllers/agentenabled/sync.go`
- **gotReadySignals()** (2 connections) — `controllers/agentenabled/sync.go`
- **isNodeCollectorReady()** (2 connections) — `controllers/agentenabled/sync.go`
- **updateInstrumentationConfigAgentsMetaHash()** (2 connections) — `controllers/agentenabled/sync.go`
- **signals.go** (2 connections) — `controllers/agentenabled/signals/signals.go`
- **TestHashForContainersConfig()** (2 connections) — `controllers/agentenabled/rollout/hash_test.go`
- **GetEnabledSignalsForContainer()** (2 connections) — `controllers/agentenabled/signals/signals.go`
- **agentInjectedStatusCondition** (1 connections) — `controllers/agentenabled/sync.go`
- **hash.go** (1 connections) — `controllers/agentenabled/rollout/hash.go`
- **hash_test.go** (1 connections) — `controllers/agentenabled/rollout/hash_test.go`
- **EnabledSignals** (1 connections) — `controllers/agentenabled/signals/signals.go`

## Relationships

- [[Odiglet Runtime Inspection]] (58 shared connections)
- [[Community 235]] (2 shared connections)
- [[Destination & Processor CRDs]] (2 shared connections)
- [[Autoscaler Deployment Sync]] (2 shared connections)
- [[Architecture Overview Docs]] (2 shared connections)
- [[Frontend Generated Models]] (1 shared connections)
- [[Community 252]] (1 shared connections)
- [[Docs Generator Functions]] (1 shared connections)
- [[Autoscaler Collector Config Domains]] (1 shared connections)
- [[CLI Kube Client]] (1 shared connections)

## Source Files

- `controllers/agentenabled/rollout/hash.go`
- `controllers/agentenabled/rollout/hash_test.go`
- `controllers/agentenabled/signals/signals.go`
- `controllers/agentenabled/sync.go`

## Audit Trail

- EXTRACTED: 56 (79%)
- INFERRED: 15 (21%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*