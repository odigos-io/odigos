# Frontend Hooks & Modals

> 43 nodes · cohesion 0.07

## Key Concepts

- **GatewayDeploymentInfo** (15 connections) — `graph/model/models_gen.go`
- **CollectorDaemonSetInfo** (13 connections) — `graph/model/models_gen.go`
- **.fieldContext_CollectorDaemonSetInfo_nodes()** (8 connections) — `graph/generated.go`
- **.fieldContext_GatewayDeploymentInfo_hpa()** (8 connections) — `graph/generated.go`
- **.fieldContext_Query_gatewayDeploymentInfo()** (7 connections) — `graph/generated.go`
- **.fieldContext_Query_odigletDaemonSetInfo()** (7 connections) — `graph/generated.go`
- **.fieldContext_Resources_limits()** (7 connections) — `graph/generated.go`
- **.fieldContext_Resources_requests()** (7 connections) — `graph/generated.go`
- **NodesSummary** (7 connections) — `graph/model/models_gen.go`
- **Resources** (7 connections) — `graph/model/models_gen.go`
- **ResourceAmounts** (6 connections) — `graph/model/models_gen.go`
- **._CollectorDaemonSetInfo_configMapYAML()** (4 connections) — `graph/generated.go`
- **._CollectorDaemonSetInfo_manifestYAML()** (4 connections) — `graph/generated.go`
- **._CollectorDaemonSetInfo_rolloutInProgress()** (4 connections) — `graph/generated.go`
- **.fieldContext_GatewayDeploymentInfo_manifestYAML()** (4 connections) — `graph/generated.go`
- **.fieldContext_ResourceAmounts_cpu()** (4 connections) — `graph/generated.go`
- **.fieldContext_ResourceAmounts_memory()** (4 connections) — `graph/generated.go`
- **._GatewayDeploymentInfo_configMapYAML()** (4 connections) — `graph/generated.go`
- **._GatewayDeploymentInfo_manifestYAML()** (4 connections) — `graph/generated.go`
- **._GatewayDeploymentInfo_resources()** (4 connections) — `graph/generated.go`
- **._GatewayDeploymentInfo_rolloutInProgress()** (4 connections) — `graph/generated.go`
- **.marshalOResourceAmounts2ᚖgithubᚗcomᚋodigosᚑioᚋodigosᚋfrontendᚋgraphᚋmodelᚐResourceAmounts()** (4 connections) — `graph/generated.go`
- **._NodesSummary_ready()** (4 connections) — `graph/generated.go`
- **._Query_gatewayDeploymentInfo()** (4 connections) — `graph/generated.go`
- **._Query_odigletDaemonSetInfo()** (4 connections) — `graph/generated.go`
- *... and 18 more nodes in this community*

## Relationships

- [[Managed Backend Destination Docs]] (130 shared connections)
- [[GraphQL Marshalers (Frontend)]] (43 shared connections)
- [[Odigos Collector Processor Catalog]] (8 shared connections)
- [[CLI Centralized Install]] (5 shared connections)
- [[GraphQL Query Resolvers]] (4 shared connections)
- [[Effective Collector Config Schema]] (2 shared connections)
- [[GraphQL Mutation Schema]] (2 shared connections)
- [[URL Template Processor]] (2 shared connections)
- [[Frontend Sampling Rules]] (2 shared connections)
- [[Pod Webhook Env Injector]] (2 shared connections)
- [[Instrumentation Rule Schema (GraphQL)]] (1 shared connections)

## Source Files

- `graph/generated.go`
- `graph/model/models_gen.go`

## Audit Trail

- EXTRACTED: 203 (100%)
- INFERRED: 1 (0%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*