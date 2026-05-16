# Community 248

> 7 nodes · cohesion 0.29

## Key Concepts

- **InstrumentedProcess** (4 connections) — `services/db/models.go`
- **startDatabase()** (3 connections) — `main.go`
- **.Processes()** (3 connections) — `graph/workload.resolvers.go`
- **InitializeDatabaseSchema()** (2 connections) — `services/db/models.go`
- **models.go** (2 connections) — `services/db/models.go`
- **CalculateProcessHealthStatus()** (2 connections) — `graph/status/instrumentedprocess.go`
- **.TableName()** (1 connections) — `services/db/models.go`

## Relationships

- [[ServiceMap GraphQL]] (13 shared connections)
- [[Frontend GraphQL Loaders]] (1 shared connections)
- [[Frontend Diagnose SSE]] (1 shared connections)
- [[Odigos CRD Informers (api)]] (1 shared connections)
- [[Autoscaler ConfigMap Sync]] (1 shared connections)

## Source Files

- `graph/status/instrumentedprocess.go`
- `graph/workload.resolvers.go`
- `main.go`
- `services/db/models.go`

## Audit Trail

- EXTRACTED: 11 (65%)
- INFERRED: 6 (35%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*