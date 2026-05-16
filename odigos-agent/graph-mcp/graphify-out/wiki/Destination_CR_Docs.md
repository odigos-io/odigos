# Destination CR Docs

> 27 nodes · cohesion 0.12

## Key Concepts

- **Query** (35 connections) — `graph/model/models_gen.go`
- **.marshalNPodInfo2ᚕᚖgithubᚗcomᚋodigosᚑioᚋodigosᚋfrontendᚋgraphᚋmodelᚐPodInfoᚄ()** (15 connections) — `graph/generated.go`
- **PodInfo** (12 connections) — `graph/model/models_gen.go`
- **.fieldContext_Query_gatewayPods()** (9 connections) — `graph/generated.go`
- **.fieldContext_Query_odigletPods()** (9 connections) — `graph/generated.go`
- **SQLiteDB** (6 connections) — `services/db/sqlite.go`
- **.fieldContext_PodInfo_name()** (5 connections) — `graph/generated.go`
- **.Exec()** (4 connections) — `services/db/sqlite.go`
- **.fieldContext_PodInfo_creationTimestamp()** (4 connections) — `graph/generated.go`
- **.fieldContext_PodInfo_image()** (4 connections) — `graph/generated.go`
- **.fieldContext_PodInfo_namespace()** (4 connections) — `graph/generated.go`
- **.fieldContext_PodInfo_restartsCount()** (4 connections) — `graph/generated.go`
- **.fieldContext_PodInfo_status()** (4 connections) — `graph/generated.go`
- **.fieldContext_Query_sampling()** (4 connections) — `graph/generated.go`
- **._PodInfo_creationTimestamp()** (4 connections) — `graph/generated.go`
- **._PodInfo_image()** (4 connections) — `graph/generated.go`
- **._PodInfo_name()** (4 connections) — `graph/generated.go`
- **._PodInfo_namespace()** (4 connections) — `graph/generated.go`
- **._PodInfo_nodeName()** (4 connections) — `graph/generated.go`
- **._PodInfo_restartsCount()** (4 connections) — `graph/generated.go`
- **._PodInfo_status()** (4 connections) — `graph/generated.go`
- **._Query_gatewayPods()** (4 connections) — `graph/generated.go`
- **._Query_odigletPods()** (4 connections) — `graph/generated.go`
- **._Query_pod()** (4 connections) — `graph/generated.go`
- **._Query_sampling()** (4 connections) — `graph/generated.go`
- *... and 2 more nodes in this community*

## Relationships

- [[MetricsSource Config Schema]] (64 shared connections)
- [[Odigos Collector Processor Catalog]] (35 shared connections)
- [[GraphQL Marshalers (Frontend)]] (25 shared connections)
- [[GraphQL Query Resolvers]] (8 shared connections)
- [[Frontend API Tokens & Metrics]] (4 shared connections)
- [[Effective Collector Config Schema]] (4 shared connections)
- [[URL Template Processor]] (3 shared connections)
- [[CLI Centralized Install]] (3 shared connections)
- [[Frontend Sampling Rules]] (3 shared connections)
- [[Frontend Diagnose SSE]] (2 shared connections)
- [[CRD Apply Configurations (api)]] (2 shared connections)
- [[Managed Backend Destination Docs]] (2 shared connections)

## Source Files

- `graph/generated.go`
- `graph/model/models_gen.go`
- `services/db/sqlite.go`

## Audit Trail

- EXTRACTED: 165 (100%)
- INFERRED: 0 (0%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*