# Instrumentor Reconcilers

> 27 nodes · cohesion 0.12

## Key Concepts

- **enableClusterSourceCmd()** (12 connections) — `cmd/sources.go`
- **sources_cluster.go** (10 connections) — `cmd/sources_cluster.go`
- **instrumentCluster()** (7 connections) — `cmd/sources_cluster.go`
- **portforward.go** (7 connections) — `pkg/kube/portforward.go`
- **startDiagnose()** (5 connections) — `cmd/diagnose.go`
- **PortForwardWithContext()** (5 connections) — `pkg/kube/portforward.go`
- **instrumentNamespace()** (4 connections) — `cmd/sources_cluster.go`
- **.Close()** (4 connections) — `pkg/remote/portforward.go`
- **diagnose.go** (3 connections) — `cmd/diagnose.go`
- **instrumentApp()** (3 connections) — `cmd/sources_cluster.go`
- **FindPodWithAppLabel()** (3 connections) — `pkg/kube/portforward.go`
- **StartResilientPortForward()** (3 connections) — `pkg/kube/portforward.go`
- **UIClientViaPortForward** (3 connections) — `pkg/remote/portforward.go`
- **createTarGz()** (2 connections) — `cmd/diagnose.go`
- **isAppExcluded()** (2 connections) — `cmd/sources_cluster.go`
- **isFatalError()** (2 connections) — `cmd/sources_cluster.go`
- **isNamespaceExcluded()** (2 connections) — `cmd/sources_cluster.go`
- **readLinesFromFile()** (2 connections) — `cmd/sources_cluster.go`
- **runPreflightChecks()** (2 connections) — `cmd/sources_cluster.go`
- **sliceToMap()** (2 connections) — `cmd/sources_cluster.go`
- **createDialer()** (2 connections) — `pkg/kube/portforward.go`
- **GetCoolOff()** (2 connections) — `pkg/lifecycle/cooloff.go`
- **SetCoolOff()** (2 connections) — `pkg/lifecycle/cooloff.go`
- **cooloff.go** (2 connections) — `pkg/lifecycle/cooloff.go`
- **NewUIClient()** (2 connections) — `pkg/remote/portforward.go`
- *... and 2 more nodes in this community*

## Relationships

- [[Service Graph Store]] (77 shared connections)
- [[Enterprise Instrumentation Docs]] (10 shared connections)
- [[Destination CRD Docs (OTLP/Jaeger/GCS)]] (2 shared connections)
- [[CLI Install/Upgrade]] (2 shared connections)
- [[Community 232]] (2 shared connections)
- [[CLI Component Resource Managers]] (1 shared connections)
- [[InstrumentationRule CRUD]] (1 shared connections)

## Source Files

- `cmd/diagnose.go`
- `cmd/sources.go`
- `cmd/sources_cluster.go`
- `pkg/kube/portforward.go`
- `pkg/lifecycle/cooloff.go`
- `pkg/remote/portforward.go`

## Audit Trail

- EXTRACTED: 75 (79%)
- INFERRED: 20 (21%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*