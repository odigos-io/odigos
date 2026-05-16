# Instrumentor Rollout Mocks

> 63 nodes · cohesion 0.05

## Key Concepts

- **.marshalNInt2int()** (29 connections) — `graph/generated.go`
- **ProfilingSlots** (13 connections) — `graph/model/models_gen.go`
- **APIToken** (12 connections) — `graph/model/models_gen.go`
- **SingleSourceMetricsResponse** (11 connections) — `graph/model/models_gen.go`
- **EnableProfilingResult** (10 connections) — `graph/model/models_gen.go`
- **DisableProfilingResult** (8 connections) — `graph/model/models_gen.go`
- **SingleDestinationMetricsResponse** (8 connections) — `graph/model/models_gen.go`
- **.fieldContext_ComputePlatform_apiTokens()** (7 connections) — `graph/generated.go`
- **.fieldContext_EnableProfilingResult_maxSlots()** (7 connections) — `graph/generated.go`
- **.fieldContext_Query_profilingSlots()** (7 connections) — `graph/generated.go`
- **.marshalNEnableProfilingResult2githubᚗcomᚋodigosᚑioᚋodigosᚋfrontendᚋgraphᚋmodelᚐEnableProfilingResult()** (7 connections) — `graph/generated.go`
- **OverviewMetricsResponse** (7 connections) — `graph/model/models_gen.go`
- **SourceProfilingResult** (7 connections) — `graph/model/models_gen.go`
- **.fieldContext_Mutation_enableSourceProfiling()** (6 connections) — `graph/generated.go`
- **.fieldContext_OverviewMetricsResponse_sources()** (6 connections) — `graph/generated.go`
- **.fieldContext_OverviewMetricsResponse_destinations()** (5 connections) — `graph/generated.go`
- **._ApiToken_expiresAt()** (4 connections) — `graph/generated.go`
- **._ApiToken_issuedAt()** (4 connections) — `graph/generated.go`
- **._ApiToken_name()** (4 connections) — `graph/generated.go`
- **._ApiToken_token()** (4 connections) — `graph/generated.go`
- **._DisableProfilingResult_activeSlots()** (4 connections) — `graph/generated.go`
- **._DisableProfilingResult_sourceKey()** (4 connections) — `graph/generated.go`
- **._EnableProfilingResult_activeSlots()** (4 connections) — `graph/generated.go`
- **._EnableProfilingResult_maxSlots()** (4 connections) — `graph/generated.go`
- **._EnableProfilingResult_sourceKey()** (4 connections) — `graph/generated.go`
- *... and 38 more nodes in this community*

## Relationships

- [[Frontend Sampling Rules]] (165 shared connections)
- [[GraphQL Marshalers (Frontend)]] (63 shared connections)
- [[Odigos Collector Processor Catalog]] (50 shared connections)
- [[GraphQL Query Resolvers]] (12 shared connections)
- [[CLI Centralized Install]] (8 shared connections)
- [[Effective Collector Config Schema]] (5 shared connections)
- [[CRD Apply Configurations (api)]] (3 shared connections)
- [[Collector Generated Telemetry]] (3 shared connections)
- [[Odiglet Instrumentation Reconciler]] (2 shared connections)
- [[Managed Backend Destination Docs]] (2 shared connections)
- [[GraphQL Mutation Schema]] (1 shared connections)
- [[Span Rule Engine (Collector)]] (1 shared connections)

## Source Files

- `graph/generated.go`
- `graph/model/models_gen.go`

## Audit Trail

- EXTRACTED: 320 (99%)
- INFERRED: 2 (1%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*