# Common Config Merging

> 16 nodes · cohesion 0.22

## Key Concepts

- **Condition** (17 connections) — `graph/model/models_gen.go`
- **.populateWorkloadFields()** (13 connections) — `graph/workload_populate.go`
- **WorkloadOdigosHealthStatusReason** (10 connections) — `graph/status/workloadodigoshealth.go`
- **aggregateProcessesHealthForWorkload()** (6 connections) — `graph/utils.go`
- **CalculateRolloutStatus()** (6 connections) — `graph/status/rollout.go`
- **CalculateRuntimeInspectionStatus()** (6 connections) — `graph/status/runtimeinspection.go`
- **getContainerNamesWithOptionalPodManifestInjection()** (5 connections) — `graph/utils.go`
- **ProcessHealthStatusReason** (5 connections) — `graph/status/instrumentedprocess.go`
- **CalculateExpectingTelemetryStatus()** (4 connections) — `graph/status/expectingtelemetry.go`
- **.Rollout()** (3 connections) — `graph/workload.resolvers.go`
- **ExpectingTelemetryReason** (3 connections) — `graph/status/expectingtelemetry.go`
- **rollout.go** (2 connections) — `graph/status/rollout.go`
- **runtimeinspection.go** (2 connections) — `graph/status/runtimeinspection.go`
- **workloadRolloutStatusCondition()** (2 connections) — `graph/status/rollout.go`
- **runtimeDetectionStatusCondition()** (2 connections) — `graph/status/runtimeinspection.go`
- **workloadodigoshealth.go** (1 connections) — `graph/status/workloadodigoshealth.go`

## Relationships

- [[ServiceMap GraphQL]] (48 shared connections)
- [[Autoscaler ConfigMap Sync]] (12 shared connections)
- [[Frontend Layout & Providers]] (12 shared connections)
- [[Instrumentation Rule Schema (GraphQL)]] (4 shared connections)
- [[Config YAML Field Schema]] (3 shared connections)
- [[Odigos Collector Processor Catalog]] (2 shared connections)
- [[CLI Centralized Install]] (2 shared connections)
- [[Collector Factories]] (1 shared connections)
- [[GraphQL Marshalers (Frontend)]] (1 shared connections)
- [[GraphQL Query Resolvers]] (1 shared connections)
- [[Odigos CRD Informers (api)]] (1 shared connections)

## Source Files

- `graph/model/models_gen.go`
- `graph/status/expectingtelemetry.go`
- `graph/status/instrumentedprocess.go`
- `graph/status/rollout.go`
- `graph/status/runtimeinspection.go`
- `graph/status/workloadodigoshealth.go`
- `graph/utils.go`
- `graph/workload.resolvers.go`
- `graph/workload_populate.go`

## Audit Trail

- EXTRACTED: 32 (37%)
- INFERRED: 55 (63%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*