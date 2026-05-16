# Common Logger

> 14 nodes · cohesion 0.21

## Key Concepts

- **root.go** (10 connections) — `config/root.go`
- **CrdProcessorToConfig()** (5 connections) — `config/processor.go`
- **isSignalExists()** (5 connections) — `config/root.go`
- **isLoggingEnabled()** (4 connections) — `config/root.go`
- **isMetricsEnabled()** (4 connections) — `config/root.go`
- **isTracingEnabled()** (4 connections) — `config/root.go`
- **isProfilingEnabled()** (3 connections) — `config/root.go`
- **processor.go** (2 connections) — `config/processor.go`
- **addProfilesPipeline()** (2 connections) — `config/root.go`
- **addProtocol()** (2 connections) — `config/root.go`
- **LoadConfigers()** (2 connections) — `config/root.go`
- **Configer** (1 connections) — `config/root.go`
- **CrdProcessorResults** (1 connections) — `config/processor.go`
- **ResourceStatuses** (1 connections) — `config/root.go`

## Relationships

- [[Common CRD-to-Config Root]] (38 shared connections)
- [[Destination Configurations (common)]] (6 shared connections)
- [[Odigos Overview Docs]] (2 shared connections)

## Source Files

- `config/processor.go`
- `config/root.go`

## Audit Trail

- EXTRACTED: 32 (70%)
- INFERRED: 14 (30%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*