# Retry & OTLP Exporter Config

> 15 nodes · cohesion 0.17

## Key Concepts

- **watchers.go** (6 connections) — `services/collector_metrics/watchers.go`
- **.Close()** (5 connections) — `services/db/sqlite.go`
- **diagnose.go** (5 connections) — `services/diagnose.go`
- **runWatcher()** (4 connections) — `services/collector_metrics/watchers.go`
- **DiagnoseGraphQL()** (4 connections) — `services/diagnose.go`
- **runWatcherLoop()** (3 connections) — `services/collector_metrics/watchers.go`
- **NewSQLiteDB()** (3 connections) — `services/db/sqlite.go`
- **newCollectorWatcher()** (2 connections) — `services/collector_metrics/watchers.go`
- **sqlite.go** (2 connections) — `services/db/sqlite.go`
- **countFilesAndSize()** (2 connections) — `services/diagnose.go`
- **DiagnoseDownload()** (2 connections) — `services/diagnose.go`
- **writeTarGzToWriter()** (2 connections) — `services/diagnose.go`
- **deletedObject** (1 connections) — `services/collector_metrics/watchers.go`
- **deleteWatcher** (1 connections) — `services/collector_metrics/watchers.go`
- **notification** (1 connections) — `services/collector_metrics/watchers.go`

## Relationships

- [[Frontend Diagnose SSE]] (21 shared connections)
- [[Community 245]] (17 shared connections)
- [[Odigos Collector Processor Catalog]] (3 shared connections)
- [[gRPC Server Config (Collector)]] (1 shared connections)
- [[ServiceMap GraphQL]] (1 shared connections)

## Source Files

- `services/collector_metrics/watchers.go`
- `services/db/sqlite.go`
- `services/diagnose.go`

## Audit Trail

- EXTRACTED: 33 (77%)
- INFERRED: 10 (23%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*