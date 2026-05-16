# Rollout Backoff Logic

> 29 nodes · cohesion 0.14

## Key Concepts

- **root_test.go** (14 connections) — `config/root_test.go`
- **.getConfigs()** (10 connections) — `config/qryn.go`
- **.GetID()** (8 connections) — `config/root_test.go`
- **.GetSignals()** (8 connections) — `config/root_test.go`
- **openTestData()** (7 connections) — `config/root_test.go`
- **DummyProcessor** (6 connections) — `config/root_test.go`
- **.GetType()** (6 connections) — `config/root_test.go`
- **mockDestination** (6 connections) — `config/otlphttp_test.go`
- **DummyDestination** (5 connections) — `config/root_test.go`
- **DummyTraceDestination** (5 connections) — `config/root_test.go`
- **mockGrpcDestination** (5 connections) — `config/genericotlp_test.go`
- **pyroscopeTestDest** (5 connections) — `config/pyroscope_test.go`
- **TestCalculateDataStreamAndDestinations()** (5 connections) — `config/root_test.go`
- **TestCalculateDataStreamMissingSources()** (5 connections) — `config/root_test.go`
- **TestServiceGraphOptions()** (4 connections) — `config/root_test.go`
- **QrynOSS** (3 connections) — `config/qryn_oss.go`
- **baseConfigWithSelfMetrics()** (3 connections) — `config/root_test.go`
- **TestCalculate()** (3 connections) — `config/root_test.go`
- **TestCalculateDataStreamMissingDestinatin()** (3 connections) — `config/root_test.go`
- **TestCalculateMinimal()** (3 connections) — `config/root_test.go`
- **TestCalculateWithBaseMinimal()** (3 connections) — `config/root_test.go`
- **genericotlp_test.go** (2 connections) — `config/genericotlp_test.go`
- **pyroscope_test.go** (2 connections) — `config/pyroscope_test.go`
- **qryn_oss.go** (2 connections) — `config/qryn_oss.go`
- **QrynOssDest** (2 connections) — `config/qryn_oss.go`
- *... and 4 more nodes in this community*

## Relationships

- [[Autoscaler Profiling OTLP Config]] (71 shared connections)
- [[Odigos Overview Docs]] (52 shared connections)
- [[Destination Configurations (common)]] (4 shared connections)
- [[Odiglet Main & Instrumentation]] (2 shared connections)
- [[Community 261]] (1 shared connections)

## Source Files

- `config/genericotlp_test.go`
- `config/otlphttp_test.go`
- `config/pyroscope_test.go`
- `config/qryn.go`
- `config/qryn_oss.go`
- `config/root_test.go`

## Audit Trail

- EXTRACTED: 121 (93%)
- INFERRED: 9 (7%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*