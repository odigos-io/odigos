# Frontend Collector Workload Helpers

> 58 nodes · cohesion 0.06

## Key Concepts

- **.InstallFromScratch()** (27 connections) — `cmd/resources/centralodigos/redis.go`
- **centralbackend.go** (10 connections) — `cmd/resources/centralodigos/centralbackend.go`
- **Central Proxy** (10 connections) — `cmd/resources/README.md`
- **ptrint32()** (7 connections) — `cmd/resources/centralodigos/centralproxy.go`
- **keycloak.go** (7 connections) — `cmd/resources/centralodigos/keycloak.go`
- **.Name()** (6 connections) — `cmd/resources/centralodigos/redis.go`
- **CreateCentralizedManagers()** (6 connections) — `cmd/resources/centralmanagers.go`
- **NewCentralBackendDeployment()** (5 connections) — `cmd/resources/centralodigos/centralbackend.go`
- **NewCentralBackendHPA()** (5 connections) — `cmd/resources/centralodigos/centralbackend.go`
- **GetImageName()** (5 connections) — `pkg/containers/name.go`
- **NewCentralProxyDeployment()** (4 connections) — `cmd/resources/centralodigos/centralproxy.go`
- **NewCentralUIDeployment()** (4 connections) — `cmd/resources/centralodigos/centralui.go`
- **keycloakResourceManager** (4 connections) — `cmd/resources/centralodigos/keycloak.go`
- **centralui.go** (4 connections) — `cmd/resources/centralodigos/centralui.go`
- **redis.go** (4 connections) — `cmd/resources/centralodigos/redis.go`
- **utils.go** (4 connections) — `cmd/resources/odigospro/utils.go`
- **intstrFromInt()** (3 connections) — `cmd/resources/centralodigos/centralbackend.go`
- **NewCentralBackendService()** (3 connections) — `cmd/resources/centralodigos/centralbackend.go`
- **centralBackendResourceManager** (3 connections) — `cmd/resources/centralodigos/centralbackend.go`
- **centralProxyResourceManager** (3 connections) — `cmd/resources/centralodigos/centralproxy.go`
- **NewCentralUIService()** (3 connections) — `cmd/resources/centralodigos/centralui.go`
- **centralUIResourceManager** (3 connections) — `cmd/resources/centralodigos/centralui.go`
- **NewKeycloakDeployment()** (3 connections) — `cmd/resources/centralodigos/keycloak.go`
- **.resolveAdminPassword()** (3 connections) — `cmd/resources/centralodigos/keycloak.go`
- **NewRedisDeployment()** (3 connections) — `cmd/resources/centralodigos/redis.go`
- *... and 33 more nodes in this community*

## Relationships

- [[CLI Install/Upgrade]] (2 shared connections)
- [[CLI Component Resource Managers]] (1 shared connections)
- [[Odiglet K8s Process Detector]] (1 shared connections)

## Source Files

- `cmd/resources/README.md`
- `cmd/resources/centralmanagers.go`
- `cmd/resources/centralodigos/centralbackend.go`
- `cmd/resources/centralodigos/centralproxy.go`
- `cmd/resources/centralodigos/centralui.go`
- `cmd/resources/centralodigos/keycloak.go`
- `cmd/resources/centralodigos/redis.go`
- `cmd/resources/odigospro/manager.go`
- `cmd/resources/odigospro/manifests.go`
- `cmd/resources/odigospro/utils.go`
- `pkg/containers/name.go`
- `pkg/containers/name_test.go`

## Audit Trail

- EXTRACTED: 163 (82%)
- INFERRED: 35 (18%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*