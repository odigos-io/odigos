# gRPC Server Middleware Tests

> 12 nodes · cohesion 0.26

## Key Concepts

- **fetchers.go** (13 connections) — `graph/loaders/fetchers.go`
- **fetchSources()** (6 connections) — `graph/loaders/fetchers.go`
- **collectWorkloadPods()** (4 connections) — `graph/loaders/fetchers.go`
- **fetchWorkloadPods()** (4 connections) — `graph/loaders/fetchers.go`
- **fetchInstrumentationConfigs()** (3 connections) — `graph/loaders/fetchers.go`
- **fetchWorkloadPodsWithSelector()** (3 connections) — `graph/loaders/fetchers.go`
- **fetchAllSources()** (2 connections) — `graph/loaders/fetchers.go`
- **fetchNamespaces()** (2 connections) — `graph/loaders/fetchers.go`
- **fetchSourcesForNamespace()** (2 connections) — `graph/loaders/fetchers.go`
- **fetchSourcesForWorkload()** (2 connections) — `graph/loaders/fetchers.go`
- **formatOperationMessage()** (2 connections) — `graph/loaders/fetchers.go`
- **.loadNamespaces()** (2 connections) — `graph/loaders/loader.go`

## Relationships

- [[Odiglet Instance Status Reporter]] (33 shared connections)
- [[EventBatcher Receiver]] (4 shared connections)
- [[Autoscaler Manager Main]] (4 shared connections)
- [[CLI Centralized Install]] (3 shared connections)
- [[Community 219]] (1 shared connections)

## Source Files

- `graph/loaders/fetchers.go`
- `graph/loaders/loader.go`

## Audit Trail

- EXTRACTED: 37 (82%)
- INFERRED: 8 (18%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*