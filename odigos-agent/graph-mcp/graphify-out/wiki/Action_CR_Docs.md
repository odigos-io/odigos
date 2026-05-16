# Action CR Docs

> 24 nodes · cohesion 0.12

## Key Concepts

- **store_test.go** (6 connections) — `connectors/servicegraphconnector/internal/store/store_test.go`
- **NewKey()** (6 connections) — `connectors/servicegraphconnector/internal/store/store.go`
- **NewStore()** (6 connections) — `connectors/servicegraphconnector/internal/store/store.go`
- **store.go** (5 connections) — `connectors/servicegraphconnector/internal/store/store.go`
- **Store** (5 connections) — `connectors/servicegraphconnector/internal/store/store.go`
- **edge.go** (4 connections) — `connectors/servicegraphconnector/internal/store/edge.go`
- **countingCallback()** (4 connections) — `connectors/servicegraphconnector/internal/store/store_test.go`
- **TestStoreExpire()** (4 connections) — `connectors/servicegraphconnector/internal/store/store_test.go`
- **TestStoreUpsertEdge()** (4 connections) — `connectors/servicegraphconnector/internal/store/store_test.go`
- **TestStoreUpsertEdge_errTooManyItems()** (4 connections) — `connectors/servicegraphconnector/internal/store/store_test.go`
- **Edge** (3 connections) — `connectors/servicegraphconnector/internal/store/edge.go`
- **TestStoreConcurrency()** (3 connections) — `connectors/servicegraphconnector/internal/store/store_test.go`
- **.UpsertEdge()** (3 connections) — `connectors/servicegraphconnector/internal/store/store.go`
- **newEdge()** (2 connections) — `connectors/servicegraphconnector/internal/store/edge.go`
- **Key** (2 connections) — `connectors/servicegraphconnector/internal/store/store.go`
- **.Expire()** (2 connections) — `connectors/servicegraphconnector/internal/store/store.go`
- **.tryEvictHead()** (2 connections) — `connectors/servicegraphconnector/internal/store/store.go`
- **Callback** (1 connections) — `connectors/servicegraphconnector/internal/store/store.go`
- **ConnectionType** (1 connections) — `connectors/servicegraphconnector/internal/store/edge.go`
- **.isComplete()** (1 connections) — `connectors/servicegraphconnector/internal/store/edge.go`
- **.isExpired()** (1 connections) — `connectors/servicegraphconnector/internal/store/edge.go`
- **.SpanIDIsEmpty()** (1 connections) — `connectors/servicegraphconnector/internal/store/store.go`
- **noopCallback()** (1 connections) — `connectors/servicegraphconnector/internal/store/store_test.go`
- **VirtualNodeLabel** (1 connections) — `connectors/servicegraphconnector/internal/store/edge.go`

## Relationships

- [[Collector Settings CRD]] (68 shared connections)
- [[Cypress E2E Tests]] (2 shared connections)
- [[Sampling Matchers (Collector)]] (1 shared connections)
- [[Sampling Rule Types (GraphQL)]] (1 shared connections)

## Source Files

- `connectors/servicegraphconnector/internal/store/edge.go`
- `connectors/servicegraphconnector/internal/store/store.go`
- `connectors/servicegraphconnector/internal/store/store_test.go`

## Audit Trail

- EXTRACTED: 52 (72%)
- INFERRED: 20 (28%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*