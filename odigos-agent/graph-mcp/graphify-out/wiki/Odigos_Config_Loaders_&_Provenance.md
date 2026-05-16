# Odigos Config Loaders & Provenance

> 42 nodes · cohesion 0.14

## Key Concepts

- **MutationResolver** (41 connections) — `graph/generated.go`
- **SamplingRules** (30 connections) — `graph/model/models_gen.go`
- **sampling.ts** (25 connections) — `webapp/types/sampling.ts`
- **TailSamplingOperationMatcher** (15 connections) — `graph/model/models_gen.go`
- **HeadSamplingOperationMatcher** (11 connections) — `graph/model/models_gen.go`
- **updateSamplingCR()** (10 connections) — `services/sampling/sampling_rules.go`
- **DerefString()** (10 connections) — `services/utils.go`
- **UPDATE_COST_REDUCTION_RULE** (8 connections) — `webapp/graphql/mutations/sampling.ts`
- **UPDATE_HIGHLY_RELEVANT_OPERATION_RULE** (8 connections) — `webapp/graphql/mutations/sampling.ts`
- **UPDATE_NOISY_OPERATION_RULE** (8 connections) — `webapp/graphql/mutations/sampling.ts`
- **getSamplingCRByID()** (8 connections) — `services/sampling/sampling_rules.go`
- **StringPtrIfNotEmpty()** (8 connections) — `services/utils.go`
- **.fieldContext_Sampling_rules()** (7 connections) — `graph/generated.go`
- **CREATE_COST_REDUCTION_RULE** (7 connections) — `webapp/graphql/mutations/sampling.ts`
- **CREATE_HIGHLY_RELEVANT_OPERATION_RULE** (7 connections) — `webapp/graphql/mutations/sampling.ts`
- **CREATE_NOISY_OPERATION_RULE** (7 connections) — `webapp/graphql/mutations/sampling.ts`
- **convertCostReductionRuleToModel()** (7 connections) — `services/sampling/conversions.go`
- **convertHighlyRelevantOperationToModel()** (7 connections) — `services/sampling/conversions.go`
- **convertNoisyOperationToModel()** (7 connections) — `services/sampling/conversions.go`
- **costReductionRuleFromInput()** (7 connections) — `services/sampling/conversions.go`
- **highlyRelevantOperationFromInput()** (7 connections) — `services/sampling/conversions.go`
- **noisyOperationFromInput()** (7 connections) — `services/sampling/conversions.go`
- **sourcesScopeCRDToModel()** (7 connections) — `services/sampling/conversions.go`
- **sourcesScopeInputToCRD()** (7 connections) — `services/sampling/conversions.go`
- **DELETE_COST_REDUCTION_RULE** (6 connections) — `webapp/graphql/mutations/sampling.ts`
- *... and 17 more nodes in this community*

## Relationships

- [[Service Graph Connector]] (239 shared connections)
- [[Frontend Hooks & Modals]] (18 shared connections)
- [[Quickstart & Sources Docs]] (11 shared connections)
- [[Odigos Collector Processor Catalog]] (8 shared connections)
- [[Config YAML Field Schema]] (8 shared connections)
- [[GraphQL Marshalers (Frontend)]] (6 shared connections)
- [[JVM Metrics Handler]] (5 shared connections)
- [[Sampling Rule Apply Configs (api)]] (5 shared connections)
- [[URL Template Processor Tests]] (5 shared connections)
- [[CLI Centralized Install]] (5 shared connections)
- [[Frontend Utils & SourceID]] (4 shared connections)
- [[Action GraphQL Schema]] (3 shared connections)

## Source Files

- `graph/generated.go`
- `graph/model/models_gen.go`
- `graph/schema.resolvers.go`
- `services/proxy.go`
- `services/sampling/conversions.go`
- `services/sampling/sampling_rules.go`
- `services/utils.go`
- `webapp/graphql/mutations/sampling.ts`
- `webapp/graphql/queries/sampling.ts`
- `webapp/types/sampling.ts`

## Audit Trail

- EXTRACTED: 254 (78%)
- INFERRED: 72 (22%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*