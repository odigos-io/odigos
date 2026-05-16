# Architecture Overview Docs

> 12 nodes · cohesion 0.33

## Key Concepts

- **ComponentLogLevelsConfig** (11 connections) — `graph/model/models_gen.go`
- **.marshalOOdigosLogLevel2ᚖgithubᚗcomᚋodigosᚑioᚋodigosᚋfrontendᚋgraphᚋmodelᚐOdigosLogLevel()** (8 connections) — `graph/generated.go`
- **.fieldContext_ComponentLogLevelsConfig_ui()** (7 connections) — `graph/generated.go`
- **.fieldContext_EffectiveConfig_componentLogLevels()** (5 connections) — `graph/generated.go`
- **._ComponentLogLevelsConfig_autoscaler()** (4 connections) — `graph/generated.go`
- **._ComponentLogLevelsConfig_collector()** (4 connections) — `graph/generated.go`
- **._ComponentLogLevelsConfig_deviceplugin()** (4 connections) — `graph/generated.go`
- **._ComponentLogLevelsConfig_instrumentor()** (4 connections) — `graph/generated.go`
- **._ComponentLogLevelsConfig_scheduler()** (4 connections) — `graph/generated.go`
- **._EffectiveConfig_componentLogLevels()** (4 connections) — `graph/generated.go`
- **.fieldContext_ComponentLogLevelsConfig_deviceplugin()** (3 connections) — `graph/generated.go`
- **.marshalOComponentLogLevelsConfig2ᚖgithubᚗcomᚋodigosᚑioᚋodigosᚋfrontendᚋgraphᚋmodelᚐComponentLogLevelsConfig()** (3 connections) — `graph/generated.go`

## Relationships

- [[Frontend Source CRUD]] (45 shared connections)
- [[GraphQL Marshalers (Frontend)]] (12 shared connections)
- [[GraphQL Mutation Schema]] (2 shared connections)
- [[Odigos Collector Processor Catalog]] (1 shared connections)
- [[CLI Centralized Install]] (1 shared connections)

## Source Files

- `graph/generated.go`
- `graph/model/models_gen.go`

## Audit Trail

- EXTRACTED: 61 (100%)
- INFERRED: 0 (0%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*