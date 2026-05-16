# SharedInformerFactory (api)

> 18 nodes · cohesion 0.13

## Key Concepts

- **delete.go** (11 connections) — `pkg/kube/delete.go`
- **pro-dep.go** (7 connections) — `cmd/pro-dep.go`
- **installCentralBackendAndUIDep()** (6 connections) — `cmd/pro-dep.go`
- **createOdigosCentralSecretDep()** (2 connections) — `cmd/pro-dep.go`
- **deleteCentralTokenSecretAdapterDep()** (2 connections) — `cmd/pro-dep.go`
- **GetImageReferencesDep()** (2 connections) — `cmd/pro-dep.go`
- **DeleteCentralTokenSecret()** (2 connections) — `pkg/kube/delete.go`
- **DeleteClusterRolesByLabel()** (2 connections) — `pkg/kube/delete.go`
- **NamespaceHasLabel()** (2 connections) — `pkg/kube/delete.go`
- **createNamespaceDep()** (1 connections) — `cmd/pro-dep.go`
- **DeleteConfigMapsByLabel()** (1 connections) — `pkg/kube/delete.go`
- **DeleteDeploymentsByLabel()** (1 connections) — `pkg/kube/delete.go`
- **DeleteHPAsByLabel()** (1 connections) — `pkg/kube/delete.go`
- **DeleteRoleBindingsByLabel()** (1 connections) — `pkg/kube/delete.go`
- **DeleteRolesByLabel()** (1 connections) — `pkg/kube/delete.go`
- **DeleteSecretsByLabel()** (1 connections) — `pkg/kube/delete.go`
- **DeleteServiceAccountsByLabel()** (1 connections) — `pkg/kube/delete.go`
- **DeleteServicesByLabel()** (1 connections) — `pkg/kube/delete.go`

## Relationships

- [[Odiglet K8s Process Detector]] (40 shared connections)
- [[Effective Config Conversion (Frontend)]] (2 shared connections)
- [[Destination CRD Docs (OTLP/Jaeger/GCS)]] (1 shared connections)
- [[Frontend Collector Workload Helpers]] (1 shared connections)
- [[InstrumentationRule CRUD]] (1 shared connections)

## Source Files

- `cmd/pro-dep.go`
- `pkg/kube/delete.go`

## Audit Trail

- EXTRACTED: 40 (89%)
- INFERRED: 5 (11%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*