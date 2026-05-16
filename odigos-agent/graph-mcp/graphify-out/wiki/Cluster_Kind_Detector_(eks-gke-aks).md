# Cluster Kind Detector (eks/gke/aks)

> 22 nodes · cohesion 0.16

## Key Concepts

- **common.go** (26 connections) — `controllers/instrumentationconfig/common.go`
- **syncWorkload()** (14 connections) — `controllers/podsinjectionstatus/sync.go`
- **conditions.go** (6 connections) — `controllers/agentenabled/rollout/conditions.go`
- **codeattributes.go** (5 connections) — `controllers/agentenabled/dynamicconfig/traces/codeattributes.go`
- **mergeCodeAttributesRules()** (5 connections) — `controllers/agentenabled/dynamicconfig/traces/codeattributes.go`
- **CalculateCodeAttributesConfig()** (4 connections) — `controllers/agentenabled/dynamicconfig/traces/codeattributes.go`
- **merge2RuleBooleans()** (4 connections) — `controllers/agentenabled/dynamicconfig/traces/codeattributes.go`
- **populateRuntimeMetricsInSdkConfigs()** (3 connections) — `controllers/instrumentationconfig/common.go`
- **syncNamespaceWorkloads()** (3 connections) — `controllers/sourceinstrumentation/common.go`
- **syncRegexSourceWorkloads()** (3 connections) — `controllers/sourceinstrumentation/common.go`
- **convertJavaRuntimeMetricsConfig()** (2 connections) — `controllers/instrumentationconfig/common.go`
- **calculateDesiredServiceName()** (2 connections) — `controllers/sourceinstrumentation/common.go`
- **createInstrumentationConfigForWorkload()** (2 connections) — `controllers/sourceinstrumentation/common.go`
- **deleteWorkloadInstrumentationConfig()** (2 connections) — `controllers/sourceinstrumentation/common.go`
- **updateContainerOverride()** (2 connections) — `controllers/sourceinstrumentation/common.go`
- **updateDatastreamLabels()** (2 connections) — `controllers/sourceinstrumentation/common.go`
- **updateRecoveredFromRollbackAt()** (2 connections) — `controllers/sourceinstrumentation/common.go`
- **updateServiceName()** (2 connections) — `controllers/sourceinstrumentation/common.go`
- **initiateAgentEnabledConditionIfMissing()** (2 connections) — `controllers/sourceinstrumentation/conditions.go`
- **initiateRuntimeDetailsConditionIfMissing()** (2 connections) — `controllers/sourceinstrumentation/conditions.go`
- **sortIcConditionsByLogicalOrder()** (2 connections) — `controllers/sourceinstrumentation/conditions.go`
- **DistroSupportsTracesCodeAttributes()** (2 connections) — `controllers/agentenabled/dynamicconfig/traces/codeattributes.go`

## Relationships

- [[Autoscaler Collector Config Domains]] (59 shared connections)
- [[Community 273]] (16 shared connections)
- [[Sources CLI Docs]] (14 shared connections)
- [[Rollout Backoff Logic]] (3 shared connections)
- [[Destination & Processor CRDs]] (2 shared connections)
- [[Community 292]] (1 shared connections)
- [[Odiglet Runtime Inspection]] (1 shared connections)
- [[Docs Generator Functions]] (1 shared connections)

## Source Files

- `controllers/agentenabled/dynamicconfig/traces/codeattributes.go`
- `controllers/agentenabled/rollout/conditions.go`
- `controllers/instrumentationconfig/common.go`
- `controllers/podsinjectionstatus/sync.go`
- `controllers/sourceinstrumentation/common.go`
- `controllers/sourceinstrumentation/conditions.go`

## Audit Trail

- EXTRACTED: 88 (91%)
- INFERRED: 9 (9%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*