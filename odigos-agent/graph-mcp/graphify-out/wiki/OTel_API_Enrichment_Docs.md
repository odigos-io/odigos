# OTel API Enrichment Docs

> 20 nodes · cohesion 0.21

## Key Concepts

- **ConvertEntityPropertyToGQL()** (11 connections) — `services/describe/utils/conversion.go`
- **source_describe.go** (9 connections) — `services/describe/source_describe/source_describe.go`
- **ConvertSourceAnalyzeToGQL()** (6 connections) — `services/describe/source_describe/source_describe.go`
- **convertOdigosToGQL()** (5 connections) — `services/describe/odigos_describe/odigos_describe.go`
- **odigos_describe.go** (5 connections) — `services/describe/odigos_describe/odigos_describe.go`
- **DESCRIBE_ODIGOS** (4 connections) — `webapp/graphql/queries/describe.ts`
- **DESCRIBE_SOURCE** (4 connections) — `webapp/graphql/queries/describe.ts`
- **convertPodContainersToGQL()** (4 connections) — `services/describe/source_describe/source_describe.go`
- **convertPodsToGQL()** (4 connections) — `services/describe/source_describe/source_describe.go`
- **convertClusterCollectorToGQL()** (3 connections) — `services/describe/odigos_describe/odigos_describe.go`
- **convertNodeCollectorToGQL()** (3 connections) — `services/describe/odigos_describe/odigos_describe.go`
- **GetOdigosDescription()** (3 connections) — `services/describe/odigos_describe/odigos_describe.go`
- **convertInstrumentationInstancesToGQL()** (3 connections) — `services/describe/source_describe/source_describe.go`
- **convertOtelAgentContainersToGQL()** (3 connections) — `services/describe/source_describe/source_describe.go`
- **convertRuntimeInfoContainersToGQL()** (3 connections) — `services/describe/source_describe/source_describe.go`
- **convertRuntimeInfoToGQL()** (3 connections) — `services/describe/source_describe/source_describe.go`
- **GetSourceDescription()** (3 connections) — `services/describe/source_describe/source_describe.go`
- **describe.go** (2 connections) — `services/describe.go`
- **describe.ts** (2 connections) — `webapp/graphql/queries/describe.ts`
- **OdigosService** (1 connections) — `services/describe/odigos_describe/odigos_describe.go`

## Relationships

- [[Odigos Collector Processor Catalog]] (2 shared connections)
- [[K8s Workload GraphQL Resolver]] (1 shared connections)

## Source Files

- `services/describe.go`
- `services/describe/odigos_describe/odigos_describe.go`
- `services/describe/source_describe/source_describe.go`
- `services/describe/utils/conversion.go`
- `webapp/graphql/queries/describe.ts`

## Audit Trail

- EXTRACTED: 59 (73%)
- INFERRED: 22 (27%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*