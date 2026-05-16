# RolloutConcurrencyLimiter Tests

> 16 nodes · cohesion 0.18

## Key Concepts

- **config.go** (18 connections) — `config/config.go`
- **MergeConfigs()** (5 connections) — `config/config.go`
- **mergeTelemetry()** (5 connections) — `config/config.go`
- **mergeExtensions()** (2 connections) — `config/config.go`
- **mergeGenericMaps()** (2 connections) — `config/config.go`
- **mergeMetricsLevel()** (2 connections) — `config/config.go`
- **mergePipelines()** (2 connections) — `config/config.go`
- **mergeTelemetryReaders()** (2 connections) — `config/config.go`
- **mergeTelemetryResource()** (2 connections) — `config/config.go`
- **Config** (1 connections) — `config/config.go`
- **ExporterConfigurer** (1 connections) — `config/config.go`
- **GenericMap** (1 connections) — `config/config.go`
- **Pipeline** (1 connections) — `config/config.go`
- **ProcessorConfigurer** (1 connections) — `config/config.go`
- **SignalSpecific** (1 connections) — `config/config.go`
- **Telemetry** (1 connections) — `config/config.go`

## Relationships

- [[Self-Hosted Backend Docs]] (44 shared connections)
- [[Odiglet Main & Instrumentation]] (2 shared connections)
- [[Gateway Config Builder]] (1 shared connections)

## Source Files

- `config/config.go`

## Audit Trail

- EXTRACTED: 47 (100%)
- INFERRED: 0 (0%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*