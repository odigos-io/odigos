# CLI Endpoints Detection

> 28 nodes · cohesion 0.11

## Key Concepts

- **init()** (19 connections) — `cmd/ui.go`
- **Actions** (9 connections) — `cmd/resources/README.md`
- **runInstallOrUpgrade()** (7 connections) — `cmd/helm-install.go`
- **runCentralInstallOrUpgrade()** (7 connections) — `cmd/pro.go`
- **root.go** (5 connections) — `cmd/root.go`
- **InstallOrUpgrade()** (5 connections) — `pkg/helm/actions.go`
- **PrintSummary()** (5 connections) — `pkg/helm/logger.go`
- **runInstallOrUpgradeWithLegacyCheck()** (4 connections) — `cmd/helm-install.go`
- **runHelmUninstall()** (4 connections) — `cmd/helm-uninstall.go`
- **pro.go** (4 connections) — `cmd/pro.go`
- **runCentralHelmUninstall()** (4 connections) — `cmd/pro.go`
- **helm-install.go** (3 connections) — `cmd/helm-install.go`
- **FormatInstallOrUpgradeMessage()** (3 connections) — `pkg/helm/actions.go`
- **RunUninstall()** (3 connections) — `pkg/helm/actions.go`
- **IsLegacyInstallation()** (3 connections) — `pkg/helm/helpers.go`
- **helm-uninstall.go** (2 connections) — `cmd/helm-uninstall.go`
- **version.go** (2 connections) — `cmd/version.go`
- **RunInstall()** (2 connections) — `pkg/helm/actions.go`
- **RunUpgrade()** (2 connections) — `pkg/helm/actions.go`
- **cleanup.go** (1 connections) — `cmd/cleanup.go`
- **profile.go** (1 connections) — `cmd/profile.go`
- **enableVerbosity()** (1 connections) — `cmd/root.go`
- **RootCmd()** (1 connections) — `cmd/root.go`
- **ui.go** (1 connections) — `cmd/ui.go`
- **ReleaseExists()** (1 connections) — `pkg/helm/actions.go`
- *... and 3 more nodes in this community*

## Relationships

- [[Destination CRD Docs (OTLP/Jaeger/GCS)]] (84 shared connections)
- [[Frontend Fetchers]] (5 shared connections)
- [[Enterprise Instrumentation Docs]] (4 shared connections)
- [[CLI Component Resource Managers]] (3 shared connections)
- [[CLI Install/Upgrade]] (2 shared connections)
- [[InstrumentationRule CRUD]] (1 shared connections)
- [[Odiglet K8s Process Detector]] (1 shared connections)
- [[Effective Config Conversion (Frontend)]] (1 shared connections)
- [[Service Graph Store]] (1 shared connections)

## Source Files

- `cmd/cleanup.go`
- `cmd/helm-install.go`
- `cmd/helm-uninstall.go`
- `cmd/pro.go`
- `cmd/profile.go`
- `cmd/resources/README.md`
- `cmd/root.go`
- `cmd/ui.go`
- `cmd/version.go`
- `pkg/helm/actions.go`
- `pkg/helm/helpers.go`
- `pkg/helm/logger.go`
- `pkg/preflight/root.go`

## Audit Trail

- EXTRACTED: 76 (75%)
- INFERRED: 26 (25%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*