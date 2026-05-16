# Service Graph Connector

> 51 nodes · cohesion 0.06

## Key Concepts

- **HighlyRelevantOperationRule** (13 connections) — `graph/model/models_gen.go`
- **CostReductionRule** (12 connections) — `graph/model/models_gen.go`
- **NoisyOperationRule** (12 connections) — `graph/model/models_gen.go`
- **.fieldContext_HighlyRelevantOperationRule_name()** (11 connections) — `graph/generated.go`
- **.fieldContext_HeadSamplingOperationMatcher_httpServer()** (9 connections) — `graph/generated.go`
- **.fieldContext_CostReductionRule_name()** (8 connections) — `graph/generated.go`
- **.marshalNID2string()** (8 connections) — `graph/generated.go`
- **.marshalOHeadSamplingOperationMatcher2ᚖgithubᚗcomᚋodigosᚑioᚋodigosᚋfrontendᚋgraphᚋmodelᚐHeadSamplingOperationMatcher()** (8 connections) — `graph/generated.go`
- **HeadSamplingHTTPClientMatcher** (8 connections) — `graph/model/models_gen.go`
- **.fieldContext_SamplingRules_costReductionRules()** (6 connections) — `graph/generated.go`
- **.fieldContext_SamplingRules_noisyOperations()** (6 connections) — `graph/generated.go`
- **TailSamplingHTTPServerMatcher** (6 connections) — `graph/model/models_gen.go`
- **TailSamplingKafkaMatcher** (6 connections) — `graph/model/models_gen.go`
- **.fieldContext_CostReductionRule_sourceScopes()** (5 connections) — `graph/generated.go`
- **.fieldContext_NoisyOperationRule_name()** (5 connections) — `graph/generated.go`
- **.fieldContext_NoisyOperationRule_sourceScopes()** (5 connections) — `graph/generated.go`
- **.fieldContext_SourcesScope_workloadName()** (5 connections) — `graph/generated.go`
- **.fieldContext_TailSamplingOperationMatcher_kafkaConsumer()** (5 connections) — `graph/generated.go`
- **.fieldContext_TailSamplingOperationMatcher_kafkaProducer()** (5 connections) — `graph/generated.go`
- **.marshalNNoisyOperationRule2githubᚗcomᚋodigosᚑioᚋodigosᚋfrontendᚋgraphᚋmodelᚐNoisyOperationRule()** (5 connections) — `graph/generated.go`
- **.marshalOSourcesScope2ᚕᚖgithubᚗcomᚋodigosᚑioᚋodigosᚋfrontendᚋgraphᚋmodelᚐSourcesScopeᚄ()** (5 connections) — `graph/generated.go`
- **._SourcesScope_workloadName()** (5 connections) — `graph/generated.go`
- **SamplingRulesResolver** (5 connections) — `graph/generated.go`
- **._ComputePlatform_instrumentationRules()** (4 connections) — `graph/generated.go`
- **._CostReductionRule_operation()** (4 connections) — `graph/generated.go`
- *... and 26 more nodes in this community*

## Relationships

- [[Frontend Hooks & Modals]] (128 shared connections)
- [[GraphQL Marshalers (Frontend)]] (49 shared connections)
- [[Effective Collector Config Schema]] (11 shared connections)
- [[Odigos Collector Processor Catalog]] (9 shared connections)
- [[JVM Metrics Handler]] (7 shared connections)
- [[CLI Centralized Install]] (6 shared connections)
- [[MetricsSource Config Schema]] (2 shared connections)
- [[Frontend Utils & SourceID]] (2 shared connections)
- [[Sampling Evaluation Logic]] (2 shared connections)
- [[GraphQL Mutation Schema]] (1 shared connections)
- [[Span Rule Engine (Collector)]] (1 shared connections)
- [[Sampling Rule Apply Configs (api)]] (1 shared connections)

## Source Files

- `graph/generated.go`
- `graph/model/models_gen.go`
- `graph/sampling.resolvers.go`

## Audit Trail

- EXTRACTED: 264 (100%)
- INFERRED: 1 (0%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*