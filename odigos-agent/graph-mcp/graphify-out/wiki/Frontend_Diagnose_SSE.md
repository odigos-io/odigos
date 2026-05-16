# Frontend Diagnose SSE

> 13 nodes · cohesion 0.22

## Key Concepts

- **common.go** (10 connections) — `controllers/nodecollector/collectorconfig/common.go`
- **resourcedetection** (6 connections) — `controllers/nodecollector/testdata/logs_included.yaml`
- **CommonApplicationTelemetryConfig()** (4 connections) — `controllers/nodecollector/collectorconfig/common.go`
- **commonProcessors()** (4 connections) — `controllers/nodecollector/collectorconfig/common.go`
- **BuildResourceDetectors()** (3 connections) — `controllers/nodecollector/collectorconfig/common.go`
- **getCommonExporters()** (3 connections) — `controllers/nodecollector/collectorconfig/common.go`
- **buildBaseExporterConfig()** (2 connections) — `controllers/nodecollector/collectorconfig/common.go`
- **CommonConfig()** (2 connections) — `controllers/nodecollector/collectorconfig/common.go`
- **isDetectorEnabled()** (2 connections) — `controllers/nodecollector/collectorconfig/common.go`
- **CommonSignalConfig** (2 connections) — `controllers/nodecollector/collectorconfig/common.go`
- **GetMemoryLimiterConfig()** (2 connections) — `controllers/common/memorylimiter.go`
- **.WithProcessors()** (1 connections) — `controllers/nodecollector/collectorconfig/common.go`
- **memorylimiter.go** (1 connections) — `controllers/common/memorylimiter.go`

## Relationships

- [[Autoscaler Resource Detection]] (34 shared connections)
- [[URL Templatization Rule GraphQL]] (4 shared connections)
- [[CLI Pro-Dep Central Backend]] (3 shared connections)
- [[Enterprise Installation Docs]] (1 shared connections)

## Source Files

- `controllers/common/memorylimiter.go`
- `controllers/nodecollector/collectorconfig/common.go`
- `controllers/nodecollector/testdata/logs_included.yaml`

## Audit Trail

- EXTRACTED: 36 (86%)
- INFERRED: 6 (14%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*