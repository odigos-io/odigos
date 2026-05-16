# Sampling Action Docs

> 20 nodes · cohesion 0.16

## Key Concepts

- **Loaders** (18 connections) — `graph/loaders/getters.go`
- **loader.go** (11 connections) — `graph/loaders/loader.go`
- **.loadWorkloadPods()** (9 connections) — `graph/loaders/loader.go`
- **.LoadWorkloadsWithFilter()** (6 connections) — `graph/loaders/loader.go`
- **.loadInstrumentationConfigs()** (4 connections) — `graph/loaders/loader.go`
- **.LoadConfig()** (3 connections) — `graph/loaders/loader.go`
- **.loadSources()** (3 connections) — `graph/loaders/loader.go`
- **.loadWorkloadManifests()** (3 connections) — `graph/loaders/loader.go`
- **getContainerStatus()** (2 connections) — `graph/loaders/loader.go`
- **getEnvValueFromManifest()** (2 connections) — `graph/loaders/loader.go`
- **getOdigosInstrumentationDeviceName()** (2 connections) — `graph/loaders/loader.go`
- **isDistroExpectingInstrumentationInstances()** (2 connections) — `graph/loaders/loader.go`
- **.EnsureHeavyWorkloadsLoaded()** (2 connections) — `graph/loaders/loader.go`
- **.SetWorkloadIdsDirect()** (2 connections) — `graph/loaders/loader.go`
- **.GetIgnoredContainers()** (1 connections) — `graph/loaders/getters.go`
- **.GetInstrumentationInstancesForWorkloadContainer()** (1 connections) — `graph/loaders/getters.go`
- **.GetWorkloadIds()** (1 connections) — `graph/loaders/loader.go`
- **loadersKeyType** (1 connections) — `graph/loaders/loader.go`
- **PodContainerId** (1 connections) — `graph/loaders/loader.go`
- **WorkloadContainerId** (1 connections) — `graph/loaders/loader.go`

## Relationships

- [[Autoscaler Manager Main]] (37 shared connections)
- [[Community 219]] (25 shared connections)
- [[Odiglet Instance Status Reporter]] (3 shared connections)
- [[Component Log Levels Config]] (2 shared connections)
- [[Collector Factories]] (2 shared connections)
- [[EventBatcher Receiver]] (2 shared connections)
- [[Autoscaler ConfigMap Sync]] (1 shared connections)
- [[Pod Webhook Env Injector]] (1 shared connections)
- [[Frontend Layout & Providers]] (1 shared connections)
- [[CLI Centralized Install]] (1 shared connections)

## Source Files

- `graph/loaders/getters.go`
- `graph/loaders/loader.go`

## Audit Trail

- EXTRACTED: 69 (92%)
- INFERRED: 6 (8%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*