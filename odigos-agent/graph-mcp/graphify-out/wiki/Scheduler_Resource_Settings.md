# Scheduler Resource Settings

> 28 nodes · cohesion 0.13

## Key Concepts

- **common.go** (14 connections) — `controllers/nodecollectorsgroup/common.go`
- **sync()** (8 connections) — `controllers/nodecollectorsgroup/common.go`
- **getResourceSettings()** (6 connections) — `controllers/nodecollectorsgroup/common.go`
- **updateMetricsSettingsForDestination()** (6 connections) — `controllers/nodecollectorsgroup/common.go`
- **getGatewayResourceSettings()** (5 connections) — `controllers/clustercollectorsgroup/resource_config.go`
- **common_test.go** (5 connections) — `controllers/nodecollectorsgroup/common_test.go`
- **newNodeCollectorGroup()** (5 connections) — `controllers/nodecollectorsgroup/common.go`
- **calculateMemoryLimiterHardLimitMiB()** (4 connections) — `controllers/nodecollectorsgroup/common.go`
- **checkInt()** (4 connections) — `controllers/nodecollectorsgroup/common_test.go`
- **resource_config_test.go** (3 connections) — `controllers/clustercollectorsgroup/resource_config_test.go`
- **TestGetResourceSettings_NodeCollector_Defaults()** (3 connections) — `controllers/nodecollectorsgroup/common_test.go`
- **TestGetResourceSettings_NodeCollector_Sizes()** (3 connections) — `controllers/nodecollectorsgroup/common_test.go`
- **TestGetResourceSettings_NodeCollector_UserOverrides()** (3 connections) — `controllers/nodecollectorsgroup/common_test.go`
- **getOwnMetricsConfig()** (2 connections) — `controllers/clustercollectorsgroup/common.go`
- **isTailSamplingEnabled()** (2 connections) — `controllers/clustercollectorsgroup/common.go`
- **newClusterCollectorGroup()** (2 connections) — `controllers/clustercollectorsgroup/common.go`
- **resolveTailSamplingConfig()** (2 connections) — `controllers/clustercollectorsgroup/common.go`
- **TestGetGatewayResourceSettings_Defaults()** (2 connections) — `controllers/clustercollectorsgroup/resource_config_test.go`
- **TestGetGatewayResourceSettings_SmallOverride()** (2 connections) — `controllers/clustercollectorsgroup/resource_config_test.go`
- **resource_config.go** (2 connections) — `controllers/clustercollectorsgroup/resource_config.go`
- **calculateSpanMetricsEnabled()** (2 connections) — `controllers/nodecollectorsgroup/common.go`
- **getHostMetricsConfiguration()** (2 connections) — `controllers/nodecollectorsgroup/common.go`
- **getKubeletStatsConfiguration()** (2 connections) — `controllers/nodecollectorsgroup/common.go`
- **getOwnMetricsSettings()** (2 connections) — `controllers/nodecollectorsgroup/common.go`
- **getSpanMetricsConfiguration()** (2 connections) — `controllers/nodecollectorsgroup/common.go`
- *... and 3 more nodes in this community*

## Relationships

- [[Frontend Config Schema]] (98 shared connections)

## Source Files

- `controllers/clustercollectorsgroup/common.go`
- `controllers/clustercollectorsgroup/resource_config.go`
- `controllers/clustercollectorsgroup/resource_config_test.go`
- `controllers/nodecollectorsgroup/common.go`
- `controllers/nodecollectorsgroup/common_test.go`
- `utils/ownerreference.go`

## Audit Trail

- EXTRACTED: 84 (86%)
- INFERRED: 14 (14%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*