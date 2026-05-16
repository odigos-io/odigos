# Instrumentation Rule Docs

> 11 nodes · cohesion 0.27

## Key Concepts

- **configgrpc_benchmark_test.go** (8 connections) — `config/configgrpc/configgrpc_benchmark_test.go`
- **BenchmarkCompressors()** (5 connections) — `config/configgrpc/configgrpc_benchmark_test.go`
- **.marshal()** (4 connections) — `config/configgrpc/configgrpc_benchmark_test.go`
- **compress()** (2 connections) — `config/configgrpc/configgrpc_benchmark_test.go`
- **setupTestPayloads()** (2 connections) — `config/configgrpc/configgrpc_benchmark_test.go`
- **logMarshaler** (2 connections) — `config/configgrpc/configgrpc_benchmark_test.go`
- **marshaler** (2 connections) — `config/configgrpc/configgrpc_benchmark_test.go`
- **metricsMarshaler** (2 connections) — `config/configgrpc/configgrpc_benchmark_test.go`
- **traceMarshaler** (2 connections) — `config/configgrpc/configgrpc_benchmark_test.go`
- **NewMarshaler()** (1 connections) — `exporters/azureblobstorageexporter/marshaler.go`
- **testPayload** (1 connections) — `config/configgrpc/configgrpc_benchmark_test.go`

## Relationships

- [[Sampling Category Calculator]] (30 shared connections)
- [[Cypress E2E Tests]] (1 shared connections)

## Source Files

- `config/configgrpc/configgrpc_benchmark_test.go`
- `exporters/azureblobstorageexporter/marshaler.go`

## Audit Trail

- EXTRACTED: 30 (97%)
- INFERRED: 1 (3%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*