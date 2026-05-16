# Prometheus-compatible Backend Docs

> 11 nodes · cohesion 0.25

## Key Concepts

- **EventBatcher** (6 connections) — `kube/watchers/batcher.go`
- **sse.go** (5 connections) — `services/sse/sse.go`
- **SendMessageToClient()** (5 connections) — `services/sse/sse.go`
- **.flushLocked()** (5 connections) — `kube/watchers/batcher.go`
- **HandleSSEConnections()** (3 connections) — `services/sse/sse.go`
- **.AddEvent()** (2 connections) — `kube/watchers/batcher.go`
- **.prepareBatchMessage()** (2 connections) — `kube/watchers/batcher.go`
- **.sendBatch()** (2 connections) — `kube/watchers/batcher.go`
- **MessageEvent** (1 connections) — `services/sse/sse.go`
- **MessageType** (1 connections) — `services/sse/sse.go`
- **SSEMessage** (1 connections) — `services/sse/sse.go`

## Relationships

- [[Frontend CSRF]] (16 shared connections)
- [[Frontend Diagnose SSE]] (15 shared connections)
- [[Community 229]] (1 shared connections)
- [[OTLP Test Connection (Frontend)]] (1 shared connections)

## Source Files

- `kube/watchers/batcher.go`
- `services/sse/sse.go`

## Audit Trail

- EXTRACTED: 28 (85%)
- INFERRED: 5 (15%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*