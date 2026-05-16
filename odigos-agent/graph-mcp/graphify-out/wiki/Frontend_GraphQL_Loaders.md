# Frontend GraphQL Loaders

> 20 nodes · cohesion 0.13

## Key Concepts

- **nodeCollectorBaseReconciler** (7 connections) — `controllers/nodecollector/service.go`
- **syncGateway()** (6 connections) — `controllers/clustercollector/sync.go`
- **reconcileClusterCollector()** (5 connections) — `controllers/clustercollector/sync.go`
- **service** (5 connections) — `controllers/nodecollector/testdata/logs_included.yaml`
- **sync.go** (4 connections) — `controllers/nodecollector/sync.go`
- **syncService()** (3 connections) — `controllers/clustercollector/service.go`
- **GetCollectorsGroupDeployedConditionsPatch()** (3 connections) — `controllers/common/deployedcondition.go`
- **.persistCollectorConfig()** (3 connections) — `controllers/nodecollector/configmap.go`
- **.reconcileNodeCollector()** (3 connections) — `controllers/nodecollector/sync.go`
- **deletePreviousServices()** (2 connections) — `controllers/clustercollector/service.go`
- **updateGatewaySvc()** (2 connections) — `controllers/clustercollector/service.go`
- **FilterAndSortProcessorsByOrderHint()** (2 connections) — `controllers/common/processors.go`
- **GetGenericBatchProcessor()** (2 connections) — `controllers/common/processors.go`
- **processors.go** (2 connections) — `controllers/common/processors.go`
- **.logConfigMapDataIfChanged()** (2 connections) — `controllers/nodecollector/configmap.go`
- **.syncDataCollection()** (2 connections) — `controllers/nodecollector/sync.go`
- **sortInstrumentationConfigsByNameNamespace()** (2 connections) — `controllers/nodecollector/sync.go`
- **deployedcondition.go** (1 connections) — `controllers/common/deployedcondition.go`
- **health_check** (1 connections) — `controllers/nodecollector/testdata/logs_included.yaml`
- **pprof** (1 connections) — `controllers/nodecollector/testdata/logs_included.yaml`

## Relationships

- [[Odiglet CSI NodeServer]] (33 shared connections)
- [[Instrumentor Assertions Helpers]] (14 shared connections)
- [[Autoscaler Sampler Handlers]] (7 shared connections)
- [[URL Templatization Rule GraphQL]] (3 shared connections)
- [[Destination CR Docs]] (1 shared connections)

## Source Files

- `controllers/clustercollector/service.go`
- `controllers/clustercollector/sync.go`
- `controllers/common/deployedcondition.go`
- `controllers/common/processors.go`
- `controllers/nodecollector/configmap.go`
- `controllers/nodecollector/service.go`
- `controllers/nodecollector/sync.go`
- `controllers/nodecollector/testdata/logs_included.yaml`

## Audit Trail

- EXTRACTED: 46 (79%)
- INFERRED: 12 (21%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*