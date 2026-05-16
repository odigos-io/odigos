# CLI Component Resource Managers

> 27 nodes · cohesion 0.11

## Key Concepts

- **calculateTracesConfig()** (16 connections) — `controllers/agentenabled/dynamicconfig/config.go`
- **CalculateMetricsConfig()** (7 connections) — `controllers/agentenabled/dynamicconfig/metrics/spanmetrics.go`
- **spanmetrics.go** (6 connections) — `controllers/agentenabled/dynamicconfig/metrics/spanmetrics.go`
- **CalculateDynamicContainerConfig()** (5 connections) — `controllers/agentenabled/dynamicconfig/config.go`
- **config.go** (4 connections) — `controllers/agentenabled/dynamicconfig/config.go`
- **CalculateAgentRuntimeMetricsConfig()** (3 connections) — `controllers/agentenabled/dynamicconfig/metrics/runtimemetrics.go`
- **AgentSpanMetricsEnabled()** (3 connections) — `controllers/agentenabled/dynamicconfig/metrics/spanmetrics.go`
- **CalculateHeaderCollectionConfig()** (3 connections) — `controllers/agentenabled/dynamicconfig/traces/headercollection.go`
- **CalculateIdGeneratorConfig()** (3 connections) — `controllers/agentenabled/dynamicconfig/traces/idgenerator.go`
- **CalculateSpanRenamerConfig()** (3 connections) — `controllers/agentenabled/dynamicconfig/traces/spanrenamer.go`
- **runtimemetrics.go** (2 connections) — `controllers/agentenabled/dynamicconfig/metrics/runtimemetrics.go`
- **headercollection.go** (2 connections) — `controllers/agentenabled/dynamicconfig/traces/headercollection.go`
- **idgenerator.go** (2 connections) — `controllers/agentenabled/dynamicconfig/traces/idgenerator.go`
- **spanrenamer.go** (2 connections) — `controllers/agentenabled/dynamicconfig/traces/spanrenamer.go`
- **urltemplatization.go** (2 connections) — `controllers/agentenabled/dynamicconfig/traces/urltemplatization.go`
- **DistroSupportsAgentRuntimeMetrics()** (2 connections) — `controllers/agentenabled/dynamicconfig/metrics/runtimemetrics.go`
- **CalculateAgentSpanMetricsConfig()** (2 connections) — `controllers/agentenabled/dynamicconfig/metrics/spanmetrics.go`
- **CalculateDryRun()** (2 connections) — `controllers/agentenabled/dynamicconfig/metrics/spanmetrics.go`
- **CalculateSpanMetricsMode()** (2 connections) — `controllers/agentenabled/dynamicconfig/metrics/spanmetrics.go`
- **DistroSupportsAgentSpanMetrics()** (2 connections) — `controllers/agentenabled/dynamicconfig/metrics/spanmetrics.go`
- **DistroSupportsTracesHeadersCollection()** (2 connections) — `controllers/agentenabled/dynamicconfig/traces/headercollection.go`
- **TimedWallEnabled()** (2 connections) — `controllers/agentenabled/dynamicconfig/traces/idgenerator.go`
- **DistroSupportsHeadSampling()** (2 connections) — `controllers/agentenabled/dynamicconfig/traces/sampling.go`
- **DistroSupportsTracesSpanRenamer()** (2 connections) — `controllers/agentenabled/dynamicconfig/traces/spanrenamer.go`
- **CalculateUrlTemplatizationConfig()** (2 connections) — `controllers/agentenabled/dynamicconfig/traces/urltemplatization.go`
- *... and 2 more nodes in this community*

## Relationships

- [[Docs Generator Functions]] (78 shared connections)
- [[Pro-Dep CLI Page Docs]] (2 shared connections)
- [[Odiglet Runtime Inspection]] (1 shared connections)
- [[Autoscaler Collector Config Domains]] (1 shared connections)
- [[Sources CLI Docs]] (1 shared connections)
- [[Community 273]] (1 shared connections)
- [[Community 253]] (1 shared connections)
- [[Community 292]] (1 shared connections)

## Source Files

- `controllers/agentenabled/dynamicconfig/config.go`
- `controllers/agentenabled/dynamicconfig/metrics/runtimemetrics.go`
- `controllers/agentenabled/dynamicconfig/metrics/spanmetrics.go`
- `controllers/agentenabled/dynamicconfig/traces/headercollection.go`
- `controllers/agentenabled/dynamicconfig/traces/idgenerator.go`
- `controllers/agentenabled/dynamicconfig/traces/sampling.go`
- `controllers/agentenabled/dynamicconfig/traces/spanrenamer.go`
- `controllers/agentenabled/dynamicconfig/traces/urltemplatization.go`

## Audit Trail

- EXTRACTED: 53 (62%)
- INFERRED: 33 (38%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*