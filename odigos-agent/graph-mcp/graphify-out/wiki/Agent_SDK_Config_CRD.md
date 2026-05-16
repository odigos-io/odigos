# Agent SDK Config CRD

> 32 nodes · cohesion 0.11

## Key Concepts

- **SourceID** (42 connections) — `services/common/source.go`
- **GetWorkloadsInNamespace()** (9 connections) — `services/sources.go`
- **EnsureSourceCRD()** (8 connections) — `services/sources.go`
- **UPDATE_K8S_ACTUAL_SOURCE** (7 connections) — `webapp/graphql/mutations/source.ts`
- **ExtractDataStreamsFromSource()** (7 connections) — `services/data_stream.go`
- **deleteSourceCRD()** (7 connections) — `services/sources.go`
- **.GetInstrumentationConfig()** (5 connections) — `graph/loaders/getters.go`
- **ToggleSourceCRD()** (5 connections) — `services/sources.go`
- **toggleSourceWithAPI()** (5 connections) — `services/sources.go`
- **GetSourceCRD()** (4 connections) — `services/sources.go`
- **UpdateSourceCRDLabel()** (4 connections) — `services/sources.go`
- **UpdateSourceCRDSpec()** (4 connections) — `services/sources.go`
- **mapConditionsToConditionArray()** (3 connections) — `webapp/utils/functions/sources.ts`
- **GetOtherConditionsForSources()** (3 connections) — `services/sources.go`
- **mapDesiredStatusToConditionStatus()** (2 connections) — `webapp/utils/functions/sources.ts`
- **mapWorkloadToSource()** (2 connections) — `webapp/utils/functions/sources.ts`
- **CreateSourceWithAPI()** (2 connections) — `services/sources.go`
- **DeleteSourceWithAPI()** (2 connections) — `services/sources.go`
- **getCronJobs()** (2 connections) — `services/sources.go`
- **getDaemonSets()** (2 connections) — `services/sources.go`
- **getDeploymentConfigs()** (2 connections) — `services/sources.go`
- **getDeployments()** (2 connections) — `services/sources.go`
- **getRollouts()** (2 connections) — `services/sources.go`
- **getStatefulSets()** (2 connections) — `services/sources.go`
- **stringToWorkloadKind()** (2 connections) — `services/sources.go`
- *... and 7 more nodes in this community*

## Relationships

- [[Collector Factories]] (115 shared connections)
- [[Sampling Rule Apply Configs (api)]] (5 shared connections)
- [[CLI Centralized Install]] (3 shared connections)
- [[Odigos Collector Processor Catalog]] (2 shared connections)
- [[Pod Webhook Env Injector]] (2 shared connections)
- [[Config YAML Field Schema]] (2 shared connections)
- [[Autoscaler Manager Main]] (2 shared connections)
- [[Service Graph Connector]] (2 shared connections)
- [[Collector Generated Telemetry]] (1 shared connections)
- [[Quickstart & Sources Docs]] (1 shared connections)
- [[GraphQL Query Resolvers]] (1 shared connections)
- [[Autoscaler ConfigMap Sync]] (1 shared connections)

## Source Files

- `graph/loaders/getters.go`
- `services/common/source.go`
- `services/data_stream.go`
- `services/sources.go`
- `webapp/graphql/mutations/source.ts`
- `webapp/types/sources.ts`
- `webapp/utils/functions/sources.ts`

## Audit Trail

- EXTRACTED: 114 (80%)
- INFERRED: 29 (20%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*