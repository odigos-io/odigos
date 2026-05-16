# Connector & Receiver Tests

> 42 nodes · cohesion 0.09

## Key Concepts

- **.ExtractJVMMetricsFromInnerMap()** (19 connections) — `receivers/odigosebpfreceiver/internal/metrics/jvm/handler.go`
- **JVMMetricsHandler** (16 connections) — `receivers/odigosebpfreceiver/internal/metrics/jvm/handler.go`
- **types.go** (13 connections) — `receivers/odigosebpfreceiver/types.go`
- **.emitGaugeMetric()** (7 connections) — `receivers/odigosebpfreceiver/internal/metrics/jvm/handler.go`
- **.String()** (6 connections) — `receivers/odigosebpfreceiver/internal/metrics/jvm/types.go`
- **setMemoryAttributes()** (5 connections) — `receivers/odigosebpfreceiver/internal/metrics/jvm/handler.go`
- **handler.go** (5 connections) — `receivers/odigosebpfreceiver/internal/metrics/jvm/handler.go`
- **.addGCHistogramMetric()** (4 connections) — `receivers/odigosebpfreceiver/internal/metrics/jvm/handler.go`
- **.addMemoryCommittedMetric()** (4 connections) — `receivers/odigosebpfreceiver/internal/metrics/jvm/handler.go`
- **.addMemoryLimitMetric()** (4 connections) — `receivers/odigosebpfreceiver/internal/metrics/jvm/handler.go`
- **.addMemoryUsedMetric()** (4 connections) — `receivers/odigosebpfreceiver/internal/metrics/jvm/handler.go`
- **MetricKey** (4 connections) — `receivers/odigosebpfreceiver/internal/metrics/jvm/types.go`
- **MetricValue** (4 connections) — `receivers/odigosebpfreceiver/internal/metrics/jvm/types.go`
- **GCAction** (3 connections) — `receivers/odigosebpfreceiver/internal/metrics/jvm/types.go`
- **GCName** (3 connections) — `receivers/odigosebpfreceiver/internal/metrics/jvm/types.go`
- **.addClassCountMetric()** (3 connections) — `receivers/odigosebpfreceiver/internal/metrics/jvm/handler.go`
- **.addClassLoadedMetric()** (3 connections) — `receivers/odigosebpfreceiver/internal/metrics/jvm/handler.go`
- **.addClassUnloadedMetric()** (3 connections) — `receivers/odigosebpfreceiver/internal/metrics/jvm/handler.go`
- **.addMemoryUsedAfterGCMetric()** (3 connections) — `receivers/odigosebpfreceiver/internal/metrics/jvm/handler.go`
- **.addThreadCountMetric()** (3 connections) — `receivers/odigosebpfreceiver/internal/metrics/jvm/handler.go`
- **MemoryPoolName** (3 connections) — `receivers/odigosebpfreceiver/internal/metrics/jvm/types.go`
- **MemoryType** (3 connections) — `receivers/odigosebpfreceiver/internal/metrics/jvm/types.go`
- **MetricType** (3 connections) — `receivers/odigosebpfreceiver/internal/metrics/jvm/types.go`
- **ThreadDaemon** (3 connections) — `receivers/odigosebpfreceiver/internal/metrics/jvm/types.go`
- **ThreadState** (3 connections) — `receivers/odigosebpfreceiver/internal/metrics/jvm/types.go`
- *... and 17 more nodes in this community*

## Relationships

- [[Sampling Rule Types (GraphQL)]] (1 shared connections)

## Source Files

- `receivers/odigosebpfreceiver/internal/metrics/jvm/handler.go`
- `receivers/odigosebpfreceiver/internal/metrics/jvm/types.go`
- `receivers/odigosebpfreceiver/types.go`

## Audit Trail

- EXTRACTED: 142 (92%)
- INFERRED: 13 (8%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*