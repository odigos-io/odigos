# CLI Uninstall & Logging

> 29 nodes · cohesion 0.09

## Key Concepts

- **uninstall-dep.go** (18 connections) — `cmd/uninstall-dep.go`
- **Print()** (7 connections) — `pkg/log/logger.go`
- **CreateKubeResourceWithLogging()** (5 connections) — `pkg/cmdutil/kube_logging.go`
- **logLine** (5 connections) — `pkg/log/logger.go`
- **.addSpaces()** (5 connections) — `pkg/log/logger.go`
- **logger.go** (5 connections) — `pkg/log/logger.go`
- **removeAllSources()** (2 connections) — `cmd/uninstall-dep.go`
- **UninstallClusterResources()** (2 connections) — `cmd/uninstall-dep.go`
- **UninstallOdigosResources()** (2 connections) — `cmd/uninstall-dep.go`
- **waitForNamespaceDeletion()** (2 connections) — `cmd/uninstall-dep.go`
- **waitForPodsToRolloutWithoutInstrumentation()** (2 connections) — `cmd/uninstall-dep.go`
- **.Error()** (2 connections) — `pkg/log/logger.go`
- **.Success()** (2 connections) — `pkg/log/logger.go`
- **.Warn()** (2 connections) — `pkg/log/logger.go`
- **kube_logging.go** (2 connections) — `pkg/cmdutil/kube_logging.go`
- **cleanupNodeOdigosLabels()** (1 connections) — `cmd/uninstall-dep.go`
- **uninstallConfigMaps()** (1 connections) — `cmd/uninstall-dep.go`
- **uninstallCRDs()** (1 connections) — `cmd/uninstall-dep.go`
- **uninstallDaemonSets()** (1 connections) — `cmd/uninstall-dep.go`
- **uninstallDeployments()** (1 connections) — `cmd/uninstall-dep.go`
- **uninstallMutatingWebhookConfigs()** (1 connections) — `cmd/uninstall-dep.go`
- **uninstallNamespace()** (1 connections) — `cmd/uninstall-dep.go`
- **uninstallRBAC()** (1 connections) — `cmd/uninstall-dep.go`
- **uninstallSecrets()** (1 connections) — `cmd/uninstall-dep.go`
- **uninstallServices()** (1 connections) — `cmd/uninstall-dep.go`
- *... and 4 more nodes in this community*

## Relationships

- [[Effective Config Conversion (Frontend)]] (66 shared connections)
- [[Destination CRD Docs (OTLP/Jaeger/GCS)]] (8 shared connections)
- [[Odiglet K8s Process Detector]] (2 shared connections)
- [[InstrumentationRule CRUD]] (1 shared connections)

## Source Files

- `cmd/uninstall-dep.go`
- `pkg/cmdutil/kube_logging.go`
- `pkg/helm/logger.go`
- `pkg/log/logger.go`

## Audit Trail

- EXTRACTED: 63 (82%)
- INFERRED: 14 (18%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*