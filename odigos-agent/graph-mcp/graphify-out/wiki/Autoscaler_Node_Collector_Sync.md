# Autoscaler Node Collector Sync

> 12 nodes · cohesion 0.29

## Key Concepts

- **PeerSources** (15 connections) — `graph/model/models_gen.go`
- **.marshalOResources2ᚖgithubᚗcomᚋodigosᚑioᚋodigosᚋfrontendᚋgraphᚋmodelᚐResources()** (9 connections) — `graph/generated.go`
- **SourcesScope** (6 connections) — `graph/model/models_gen.go`
- **.GetServiceMap()** (5 connections) — `graph/schema.resolvers.go`
- **EdgeToModel()** (5 connections) — `services/peer_sources.go`
- **BaseServiceName()** (4 connections) — `services/peer_sources.go`
- **mergeEdge()** (4 connections) — `services/peer_sources.go`
- **convertStringMapToNonIdentifyingAttributes()** (3 connections) — `services/peer_sources.go`
- **ServiceGraphNodeAttributesForServer()** (3 connections) — `services/peer_sources.go`
- **serviceGraphLabelsForPrefix()** (2 connections) — `services/peer_sources.go`
- **GET_PEER_SOURCES** (1 connections) — `webapp/graphql/queries/peer-sources.ts`
- **mapValues()** (1 connections) — `services/peer_sources.go`

## Relationships

- [[Community 208]] (35 shared connections)
- [[Frontend Hooks & Modals]] (5 shared connections)
- [[GraphQL Marshalers (Frontend)]] (4 shared connections)
- [[Odigos Collector Processor Catalog]] (4 shared connections)
- [[Managed Backend Destination Docs]] (3 shared connections)
- [[GraphQL Query Resolvers]] (2 shared connections)
- [[CLI Centralized Install]] (2 shared connections)
- [[Effective Collector Config Schema]] (1 shared connections)
- [[URL Template Processor]] (1 shared connections)
- [[Service Graph Connector]] (1 shared connections)

## Source Files

- `graph/generated.go`
- `graph/model/models_gen.go`
- `graph/schema.resolvers.go`
- `services/peer_sources.go`
- `webapp/graphql/queries/peer-sources.ts`

## Audit Trail

- EXTRACTED: 52 (90%)
- INFERRED: 6 (10%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*