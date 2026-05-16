# Source CRD (api)

> 19 nodes · cohesion 0.13

## Key Concepts

- **Service** (8 connections) — `config/config.go`
- **logs/in** (5 connections) — `config/testdata/debugexporter.yaml`
- **logs/debug-d1** (4 connections) — `config/testdata/debugexporter.yaml`
- **logs/debug-dummy** (4 connections) — `config/testdata/destnosources.yaml`
- **logs/dummy-group** (4 connections) — `config/testdata/destnosources.yaml`
- **metrics/otelcol** (4 connections) — `config/testdata/sourcesnodest.yaml`
- **forward/logs/debug-d1** (3 connections) — `config/testdata/debugexporter.yaml`
- **batch/generic-batch-processor** (3 connections) — `config/testdata/minimal.yaml`
- **odigosrouterconnector/logs** (2 connections) — `config/testdata/debugexporter.yaml`
- **debug/d1** (1 connections) — `config/testdata/debugexporter.yaml`
- **debug/dummy** (1 connections) — `config/testdata/destnosources.yaml`
- **resource/dummy-processor** (1 connections) — `config/testdata/destnosources.yaml`
- **health_check** (1 connections) — `config/testdata/minimal.yaml`
- **otlp** (1 connections) — `config/testdata/minimal.yaml`
- **pprof** (1 connections) — `config/testdata/minimal.yaml`
- **resource/odigos-version** (1 connections) — `config/testdata/minimal.yaml`
- **otlp_grpc/odigos-own-telemetry-ui** (1 connections) — `config/testdata/sourcesnodest.yaml`
- **prometheus/self-metrics** (1 connections) — `config/testdata/sourcesnodest.yaml`
- **resource/pod-name** (1 connections) — `config/testdata/sourcesnodest.yaml`

## Relationships

- [[Gateway Config Builder]] (46 shared connections)
- [[Self-Hosted Backend Docs]] (1 shared connections)

## Source Files

- `config/config.go`
- `config/testdata/debugexporter.yaml`
- `config/testdata/destnosources.yaml`
- `config/testdata/minimal.yaml`
- `config/testdata/sourcesnodest.yaml`

## Audit Trail

- EXTRACTED: 47 (100%)
- INFERRED: 0 (0%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*