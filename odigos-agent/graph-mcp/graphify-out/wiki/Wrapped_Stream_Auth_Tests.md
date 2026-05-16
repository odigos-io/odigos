# Wrapped Stream Auth Tests

> 12 nodes · cohesion 0.26

## Key Concepts

- **helpers.go** (9 connections) — `pkg/helm/helpers.go`
- **PrepareChartAndValues()** (6 connections) — `pkg/helm/helpers.go`
- **runCentralInstallOrUpgradeWithLegacyCheck()** (5 connections) — `cmd/pro.go`
- **ValidateCentralHelmInstallPreconditions()** (4 connections) — `pkg/helm/helpers.go`
- **IsLegacyCentralInstallation()** (3 connections) — `pkg/helm/helpers.go`
- **PrepareCentralChartAndValues()** (3 connections) — `pkg/helm/helpers.go`
- **LoadEmbeddedChart()** (2 connections) — `pkg/helm/embedded.go`
- **ensureHelmRepo()** (2 connections) — `pkg/helm/helpers.go`
- **isHelmOwnedByRelease()** (2 connections) — `pkg/helm/helpers.go`
- **legacyCentralLeftoverErr()** (2 connections) — `pkg/helm/helpers.go`
- **refreshHelmRepo()** (2 connections) — `pkg/helm/helpers.go`
- **embedded.go** (1 connections) — `pkg/helm/embedded.go`

## Relationships

- [[Frontend Fetchers]] (34 shared connections)
- [[Destination CRD Docs (OTLP/Jaeger/GCS)]] (5 shared connections)
- [[InstrumentationRule CRUD]] (1 shared connections)
- [[CLI Install/Upgrade]] (1 shared connections)

## Source Files

- `cmd/pro.go`
- `pkg/helm/embedded.go`
- `pkg/helm/helpers.go`

## Audit Trail

- EXTRACTED: 31 (76%)
- INFERRED: 10 (24%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*