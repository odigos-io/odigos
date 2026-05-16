# Workload Instrumentation Update

> 23 nodes · cohesion 0.23

## Key Concepts

- **InstrumentationRule** (38 connections) — `graph/model/models_gen.go`
- **.CreateInstrumentationRule()** (15 connections) — `graph/schema.resolvers.go`
- **.UpdateInstrumentationRule()** (15 connections) — `graph/schema.resolvers.go`
- **ProgrammingLanguage** (15 connections) — `graph/model/models_gen.go`
- **convertSourcesScope()** (9 connections) — `services/instrumentationrule.go`
- **GetInstrumentationRule()** (9 connections) — `services/instrumentationrule.go`
- **SpanKind** (7 connections) — `graph/model/models_gen.go`
- **convertPayloadCollection()** (7 connections) — `services/instrumentationrule.go`
- **convertInstrumentationLibraries()** (6 connections) — `services/instrumentationrule.go`
- **convertCustomInstrumentations()** (4 connections) — `services/instrumentationrule.go`
- **convertHeadersCollection()** (4 connections) — `services/instrumentationrule.go`
- **deriveTypeFromRule()** (4 connections) — `services/instrumentationrule.go`
- **handleNotFoundError()** (4 connections) — `services/instrumentationrule.go`
- **DerefSamplingWorkloadLanguage()** (4 connections) — `services/utils.go`
- **.DeleteInstrumentationRule()** (3 connections) — `graph/schema.resolvers.go`
- **getCodeAttributesInput()** (3 connections) — `services/instrumentationrule.go`
- **getCustomInstrumentationsInput()** (3 connections) — `services/instrumentationrule.go`
- **getHeadersCollectionInput()** (3 connections) — `services/instrumentationrule.go`
- **getPayloadCollectionInput()** (3 connections) — `services/instrumentationrule.go`
- **.marshalNInstrumentationRuleType2githubᚗcomᚋodigosᚑioᚋodigosᚋfrontendᚋgraphᚋmodelᚐInstrumentationRuleType()** (2 connections) — `graph/generated.go`
- **toDbQueryPayload()** (2 connections) — `services/instrumentationrule.go`
- **toHTTPPayload()** (2 connections) — `services/instrumentationrule.go`
- **toMessagingPayload()** (2 connections) — `services/instrumentationrule.go`

## Relationships

- [[Frontend Utils & SourceID]] (112 shared connections)
- [[CLI Centralized Install]] (14 shared connections)
- [[Config YAML Field Schema]] (11 shared connections)
- [[GraphQL Marshalers (Frontend)]] (5 shared connections)
- [[Service Graph Connector]] (5 shared connections)
- [[Instrumentation Rule Schema (GraphQL)]] (4 shared connections)
- [[Frontend Layout & Providers]] (3 shared connections)
- [[Odigos Collector Processor Catalog]] (2 shared connections)
- [[Frontend Hooks & Modals]] (2 shared connections)
- [[Sampling Rule Apply Configs (api)]] (1 shared connections)
- [[Effective Collector Config Schema]] (1 shared connections)
- [[Odigos CRD Informers (api)]] (1 shared connections)

## Source Files

- `graph/generated.go`
- `graph/model/models_gen.go`
- `graph/schema.resolvers.go`
- `services/instrumentationrule.go`
- `services/utils.go`

## Audit Trail

- EXTRACTED: 132 (80%)
- INFERRED: 32 (20%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*