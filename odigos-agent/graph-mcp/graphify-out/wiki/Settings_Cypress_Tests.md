# Settings Cypress Tests

> 14 nodes · cohesion 0.22

## Key Concepts

- **startHTTPServer()** (11 connections) — `main.go`
- **workload.go** (11 connections) — `services/workload.go`
- **DescribeWorkloadWithFilters()** (5 connections) — `services/workload.go`
- **DescribeWorkload()** (4 connections) — `services/workload.go`
- **getFilterAndVerbosityFromContext()** (4 connections) — `services/workload.go`
- **populateNamespaceWorkloadLightFields()** (3 connections) — `graph/workload_populate.go`
- **NewLoaders()** (3 connections) — `graph/loaders/loader.go`
- **WithLoaders()** (3 connections) — `graph/loaders/loader.go`
- **NewExecutableSchema()** (2 connections) — `graph/generated.go`
- **getParamOrQuery()** (2 connections) — `services/workload.go`
- **getQueryForVerbosity()** (2 connections) — `services/workload.go`
- **senatizeKind()** (2 connections) — `services/workload.go`
- **CachedWorkloadManifest** (1 connections) — `graph/computed/workload.go`
- **workload_populate.go** (1 connections) — `graph/workload_populate.go`

## Relationships

- [[Component Log Levels Config]] (40 shared connections)
- [[Odigos Collector Processor Catalog]] (3 shared connections)
- [[Frontend GraphQL Loaders]] (2 shared connections)
- [[Collector Client Tests]] (2 shared connections)
- [[Community 219]] (2 shared connections)
- [[Retry & OTLP Exporter Config]] (1 shared connections)
- [[Community 207]] (1 shared connections)
- [[Community 229]] (1 shared connections)
- [[Collector Factories]] (1 shared connections)
- [[Autoscaler ConfigMap Sync]] (1 shared connections)

## Source Files

- `graph/computed/workload.go`
- `graph/generated.go`
- `graph/loaders/loader.go`
- `graph/workload_populate.go`
- `main.go`
- `services/workload.go`

## Audit Trail

- EXTRACTED: 33 (61%)
- INFERRED: 21 (39%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*