# Source Object Docs

> 19 nodes · cohesion 0.15

## Key Concepts

- **Client** (11 connections) — `pkg/kube/client.go`
- **KubeClientFromContext()** (7 connections) — `pkg/cmd_context/client.go`
- **.ApplyResource()** (5 connections) — `pkg/kube/client.go`
- **TypeMetaToDynamicResource()** (5 connections) — `pkg/kube/dynamic.go`
- **GetCLIClientOrExit()** (4 connections) — `pkg/kube/client.go`
- **PrintClientErrorAndExit()** (4 connections) — `pkg/kube/client.go`
- **.ApplyResourceIfAbsent()** (3 connections) — `pkg/kube/client.go`
- **.ExistsByObject()** (3 connections) — `pkg/kube/client.go`
- **ApplyResourceManagers()** (3 connections) — `cmd/resources/applyresources.go`
- **CreateClient()** (2 connections) — `pkg/kube/client.go`
- **objectKindToResourceName()** (2 connections) — `pkg/kube/dynamic.go`
- **dynamic.go** (2 connections) — `pkg/kube/dynamic.go`
- **SourceStatus** (2 connections) — `cmd/sources_utils/sources_utils.go`
- **ContextWithKubeClient()** (1 connections) — `pkg/cmd_context/client.go`
- **kubeClientContextKeyType** (1 connections) — `pkg/cmd_context/client.go`
- **sources_utils.go** (1 connections) — `cmd/sources_utils/sources_utils.go`
- **.DeleteOldOdigosSystemObjects()** (1 connections) — `pkg/kube/client.go`
- **Object** (1 connections) — `pkg/kube/client.go`
- **GetCurrentConfig()** (1 connections) — `cmd/resources/applyresources.go`

## Relationships

- [[InstrumentationRule CRUD]] (51 shared connections)
- [[CLI Component Resource Managers]] (2 shared connections)
- [[Enterprise Instrumentation Docs]] (1 shared connections)
- [[Destination CRD Docs (OTLP/Jaeger/GCS)]] (1 shared connections)
- [[Frontend Fetchers]] (1 shared connections)
- [[Service Graph Store]] (1 shared connections)
- [[Odiglet K8s Process Detector]] (1 shared connections)
- [[Effective Config Conversion (Frontend)]] (1 shared connections)

## Source Files

- `cmd/resources/applyresources.go`
- `cmd/sources_utils/sources_utils.go`
- `pkg/cmd_context/client.go`
- `pkg/kube/client.go`
- `pkg/kube/dynamic.go`

## Audit Trail

- EXTRACTED: 42 (71%)
- INFERRED: 17 (29%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*