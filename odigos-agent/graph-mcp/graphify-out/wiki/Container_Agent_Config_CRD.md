# Container Agent Config CRD

> 25 nodes · cohesion 0.13

## Key Concepts

- **OdigosWorkloadConfig** (13 connections) — `extension/odigosconfigk8sextension/odigosconfig.go`
- **.handleInstrumentationConfig()** (7 connections) — `extension/odigosconfigk8sextension/informer.go`
- **workloadKeyFromResourceAttributes()** (7 connections) — `extension/odigosconfigk8sextension/resourceattrs.go`
- **resourceattrs.go** (5 connections) — `extension/odigosconfigk8sextension/resourceattrs.go`
- **workloadContainerKeyFromResourceAttributes()** (5 connections) — `extension/odigosconfigk8sextension/resourceattrs.go`
- **informer.go** (4 connections) — `extension/odigosconfigk8sextension/informer.go`
- **workloadKeyFromObject()** (4 connections) — `extension/odigosconfigk8sextension/informer.go`
- **.syncWorkloadToDesiredState()** (4 connections) — `extension/odigosconfigk8sextension/informer.go`
- **.parseWorkloadCollectorConfig()** (3 connections) — `extension/odigosconfigk8sextension/informer.go`
- **getKindAndName()** (3 connections) — `extension/odigosconfigk8sextension/resourceattrs.go`
- **getNamespace()** (3 connections) — `extension/odigosconfigk8sextension/resourceattrs.go`
- **k8sSourceKey()** (3 connections) — `extension/odigosconfigk8sextension/sourcekey.go`
- **WorkloadKeyString()** (3 connections) — `extension/odigosconfigk8sextension/sourcekey.go`
- **sourcekey.go** (2 connections) — `extension/odigosconfigk8sextension/sourcekey.go`
- **extractDataStreamLabels()** (2 connections) — `extension/odigosconfigk8sextension/informer.go`
- **kindFromInstrumentationConfigName()** (2 connections) — `extension/odigosconfigk8sextension/informer.go`
- **.GetDataStreamsForWorkload()** (2 connections) — `extension/odigosconfigk8sextension/odigosconfig.go`
- **.GetFromResource()** (2 connections) — `extension/odigosconfigk8sextension/odigosconfig.go`
- **.GetWorkloadCacheKey()** (2 connections) — `extension/odigosconfigk8sextension/odigosconfig.go`
- **.startInformer()** (2 connections) — `extension/odigosconfigk8sextension/informer.go`
- **.WaitForCacheSync()** (2 connections) — `extension/odigosconfigk8sextension/informer.go`
- **getContainerName()** (2 connections) — `extension/odigosconfigk8sextension/resourceattrs.go`
- **containerEntry** (1 connections) — `extension/odigosconfigk8sextension/informer.go`
- **.RegisterWorkloadConfigCacheCallback()** (1 connections) — `extension/odigosconfigk8sextension/odigosconfig.go`
- **.UnregisterWorkloadConfigCacheCallback()** (1 connections) — `extension/odigosconfigk8sextension/odigosconfig.go`

## Relationships

- [[Workload Resource Attrs (Collector)]] (78 shared connections)
- [[Cypress E2E Tests]] (3 shared connections)
- [[Sampling Rule Types (GraphQL)]] (3 shared connections)
- [[CLI Uninstall & Logging]] (1 shared connections)

## Source Files

- `extension/odigosconfigk8sextension/informer.go`
- `extension/odigosconfigk8sextension/odigosconfig.go`
- `extension/odigosconfigk8sextension/resourceattrs.go`
- `extension/odigosconfigk8sextension/sourcekey.go`

## Audit Trail

- EXTRACTED: 67 (79%)
- INFERRED: 18 (21%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*