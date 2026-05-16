# Instrumentor Workload Sync

> 21 nodes · cohesion 0.15

## Key Concepts

- **For()** (23 connections) — `graph/loaders/loader.go`
- **K8sWorkloadResolver** (19 connections) — `graph/generated.go`
- **._DataStream_name()** (8 connections) — `graph/generated.go`
- **aggregateConditionsBySeverity()** (6 connections) — `graph/utils.go`
- **runtimeDetailsContainersToModel()** (6 connections) — `graph/utils.go`
- **.Containers()** (5 connections) — `graph/workload.resolvers.go`
- **.RuntimeInfo()** (5 connections) — `graph/workload.resolvers.go`
- **.Pods()** (4 connections) — `graph/workload.resolvers.go`
- **.PodsHealthStatus()** (4 connections) — `graph/workload.resolvers.go`
- **containerAgentConfigToAgentConfigModel()** (3 connections) — `graph/conversions.go`
- **.MarkedForInstrumentation()** (3 connections) — `graph/workload.resolvers.go`
- **CalculatePodContainerHealthStatus()** (3 connections) — `graph/status/podhealth.go`
- **WorkloadHealthStatusReason** (3 connections) — `graph/status/workloadhealth.go`
- **.NumberOfInstances()** (2 connections) — `graph/workload.resolvers.go`
- **.RollbackOccurred()** (2 connections) — `graph/workload.resolvers.go`
- **.ServiceName()** (2 connections) — `graph/workload.resolvers.go`
- **.WorkloadsByIds()** (2 connections) — `graph/workload.resolvers.go`
- **podhealth.go** (2 connections) — `graph/status/podhealth.go`
- **envVarsToModel()** (2 connections) — `graph/utils.go`
- **.TelemetryMetrics()** (1 connections) — `graph/workload.resolvers.go`
- **PodContainerHealthReason** (1 connections) — `graph/status/podhealth.go`

## Relationships

- [[Autoscaler ConfigMap Sync]] (54 shared connections)
- [[Frontend Layout & Providers]] (18 shared connections)
- [[ServiceMap GraphQL]] (13 shared connections)
- [[Config YAML Field Schema]] (5 shared connections)
- [[Odigos Collector Processor Catalog]] (4 shared connections)
- [[GraphQL Marshalers (Frontend)]] (2 shared connections)
- [[Quickstart & Sources Docs]] (1 shared connections)
- [[GraphQL Query Resolvers]] (1 shared connections)
- [[Sampling Rule Apply Configs (api)]] (1 shared connections)
- [[Collector Factories]] (1 shared connections)
- [[Odigos CRD Informers (api)]] (1 shared connections)
- [[CLI Centralized Install]] (1 shared connections)

## Source Files

- `graph/conversions.go`
- `graph/generated.go`
- `graph/loaders/loader.go`
- `graph/status/podhealth.go`
- `graph/status/workloadhealth.go`
- `graph/utils.go`
- `graph/workload.resolvers.go`

## Audit Trail

- EXTRACTED: 50 (47%)
- INFERRED: 56 (53%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*