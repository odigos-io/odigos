# Community 211

> 9 nodes · cohesion 0.28

## Key Concepts

- **syncCollectorGroup()** (5 connections) — `controllers/nodecollector/daemonset.go`
- **.RunSyncDaemonSetWithDelayAndSkipNewCalls()** (4 connections) — `controllers/nodecollector/daemonset.go`
- **UpdateCollectorGroupReceiverSignals()** (3 connections) — `controllers/common/signals.go`
- **DelayManager** (3 connections) — `controllers/nodecollector/daemonset.go`
- **daemonset.go** (2 connections) — `controllers/nodecollector/daemonset.go`
- **getConfigMap()** (2 connections) — `controllers/nodecollector/configmap.go`
- **getSignalsFromOtelcolConfig()** (2 connections) — `controllers/nodecollector/configmap.go`
- **.finishProgress()** (2 connections) — `controllers/nodecollector/daemonset.go`
- **signals.go** (1 connections) — `controllers/common/signals.go`

## Relationships

- [[Instrumentor Assertions Helpers]] (21 shared connections)
- [[URL Templatization Rule GraphQL]] (2 shared connections)
- [[Autoscaler Sampler Handlers]] (1 shared connections)

## Source Files

- `controllers/common/signals.go`
- `controllers/nodecollector/configmap.go`
- `controllers/nodecollector/daemonset.go`

## Audit Trail

- EXTRACTED: 16 (67%)
- INFERRED: 8 (33%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*