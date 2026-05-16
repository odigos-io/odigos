# Filter Apply Configurations (api)

> 12 nodes · cohesion 0.39

## Key Concepts

- **fetchWorkloadManifests()** (13 connections) — `graph/loaders/fetchers.go`
- **.GetWorkloadManifest()** (12 connections) — `graph/loaders/getters.go`
- **workloadhealth.go** (8 connections) — `graph/status/workloadhealth.go`
- **IsDeploymentConfigAvailable()** (5 connections) — `kube/client.go`
- **timedAPICall()** (3 connections) — `graph/loaders/fetchers.go`
- **CalculateCronJobHealthStatus()** (3 connections) — `graph/status/workloadhealth.go`
- **CalculateDaemonSetHealthStatus()** (3 connections) — `graph/status/workloadhealth.go`
- **CalculateDeploymentConfigHealthStatus()** (3 connections) — `graph/status/workloadhealth.go`
- **CalculateDeploymentHealthStatus()** (3 connections) — `graph/status/workloadhealth.go`
- **CalculateRolloutHealthStatus()** (3 connections) — `graph/status/workloadhealth.go`
- **CalculateStatefulSetHealthStatus()** (3 connections) — `graph/status/workloadhealth.go`
- **CalculateStaticPodHealthStatus()** (3 connections) — `graph/status/workloadhealth.go`

## Relationships

- [[EventBatcher Receiver]] (52 shared connections)
- [[Odiglet Instance Status Reporter]] (4 shared connections)
- [[Autoscaler Manager Main]] (2 shared connections)
- [[Autoscaler ConfigMap Sync]] (1 shared connections)
- [[Frontend GraphQL Loaders]] (1 shared connections)
- [[Collector Factories]] (1 shared connections)
- [[Config YAML Field Schema]] (1 shared connections)

## Source Files

- `graph/loaders/fetchers.go`
- `graph/loaders/getters.go`
- `graph/status/workloadhealth.go`
- `kube/client.go`

## Audit Trail

- EXTRACTED: 27 (44%)
- INFERRED: 35 (56%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*