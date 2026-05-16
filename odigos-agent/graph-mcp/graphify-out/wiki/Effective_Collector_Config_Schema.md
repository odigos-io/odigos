# Effective Collector Config Schema

> 122 nodes · cohesion 0.03

## Key Concepts

- **.marshalOBoolean2bool()** (86 connections) — `graph/generated.go`
- **.fieldContext_K8sWorkload_id()** (46 connections) — `graph/generated.go`
- **K8sWorkload** (34 connections) — `graph/model/models_gen.go`
- **K8sWorkloadContainer** (26 connections) — `graph/model/models_gen.go`
- **.marshalNDesiredConditionStatus2githubᚗcomᚋodigosᚑioᚋodigosᚋfrontendᚋgraphᚋmodelᚐDesiredConditionStatus()** (18 connections) — `graph/generated.go`
- **K8sWorkloadAgentEnabled** (15 connections) — `graph/model/models_gen.go`
- **K8sWorkloadRuntimeInfo** (15 connections) — `graph/model/models_gen.go`
- **.fieldContext_K8sWorkload_runtimeInfo()** (14 connections) — `graph/generated.go`
- **.fieldContext_Query_workloads()** (14 connections) — `graph/generated.go`
- **.fieldContext_K8sNamespace_workloads()** (13 connections) — `graph/generated.go`
- **K8sWorkloadTelemetryMetrics** (13 connections) — `graph/model/models_gen.go`
- **.fieldContext_DesiredConditionStatus_name()** (11 connections) — `graph/generated.go`
- **K8sWorkloadConditions** (11 connections) — `graph/model/models_gen.go`
- **K8sWorkloadPodContainerProcess** (11 connections) — `graph/model/models_gen.go`
- **.fieldContext_DesiredConditionStatus_reasonEnum()** (10 connections) — `graph/generated.go`
- **DesiredConditionStatus** (9 connections) — `graph/model/models_gen.go`
- **K8sWorkloadTelemetryMetricsExpectingTelemetryStatus** (9 connections) — `graph/model/models_gen.go`
- **K8sWorkloadMarkedForInstrumentation** (8 connections) — `graph/model/models_gen.go`
- **K8sWorkloadPodContainerProcessInstrumentation** (8 connections) — `graph/model/models_gen.go`
- **K8sWorkloadContainerAgentConfigTraces** (7 connections) — `graph/model/models_gen.go`
- **K8sWorkloadContainerOverrides** (7 connections) — `graph/model/models_gen.go`
- **K8sWorkloadRollout** (7 connections) — `graph/model/models_gen.go`
- **.fieldContext_K8sWorkload_podsAgentInjectionStatus()** (6 connections) — `graph/generated.go`
- **.fieldContext_K8sWorkload_telemetryMetrics()** (6 connections) — `graph/generated.go`
- **.fieldContext_K8sWorkload_workloadOdigosHealthStatus()** (6 connections) — `graph/generated.go`
- *... and 97 more nodes in this community*

## Relationships

- [[Odigos CRD Informers (api)]] (486 shared connections)
- [[GraphQL Marshalers (Frontend)]] (139 shared connections)
- [[Odigos Collector Processor Catalog]] (26 shared connections)
- [[GraphQL Query Resolvers]] (19 shared connections)
- [[GraphQL Mutation Schema]] (19 shared connections)
- [[CLI Centralized Install]] (17 shared connections)
- [[Instrumentation Rule Schema (GraphQL)]] (9 shared connections)
- [[Sampling Config Schema (Frontend)]] (6 shared connections)
- [[Frontend Source CRUD]] (6 shared connections)
- [[Span Rule Engine (Collector)]] (5 shared connections)
- [[Common Sampling Config Types]] (5 shared connections)
- [[MetricsSource Config Schema]] (3 shared connections)

## Source Files

- `graph/generated.go`
- `graph/model/models_gen.go`

## Audit Trail

- EXTRACTED: 752 (100%)
- INFERRED: 0 (0%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*