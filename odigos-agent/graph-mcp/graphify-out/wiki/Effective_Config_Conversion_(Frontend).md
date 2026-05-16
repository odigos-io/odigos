# Effective Config Conversion (Frontend)

> 26 nodes · cohesion 0.12

## Key Concepts

- **calculateCollectorConfigDomains()** (18 connections) — `controllers/nodecollector/configmap.go`
- **syncConfigMap()** (14 connections) — `controllers/clustercollector/configmap.go`
- **configmap.go** (13 connections) — `controllers/nodecollector/configmap.go`
- **config.go** (6 connections) — `k8sconfig/config.go`
- **addSelfTelemetryPipeline()** (4 connections) — `controllers/clustercollector/configmap.go`
- **calculateDataStreams()** (3 connections) — `controllers/clustercollector/configmap.go`
- **GetSpanMetricsConfig()** (3 connections) — `controllers/nodecollector/collectorconfig/spanmetrics.go`
- **ToProcessorConfigurerArray()** (3 connections) — `controllers/common/config.go`
- **isTracingLoadBalancingNeeded()** (3 connections) — `controllers/nodecollector/configmap.go`
- **createConfigMap()** (2 connections) — `controllers/clustercollector/configmap.go`
- **destinationExists()** (2 connections) — `controllers/clustercollector/configmap.go`
- **isOdigosTrafficMetricsProcessorRelevant()** (2 connections) — `controllers/clustercollector/configmap.go`
- **patchConfigMap()** (2 connections) — `controllers/clustercollector/configmap.go`
- **TestAddSelfTelemetryPipeline()** (2 connections) — `controllers/clustercollector/configmap_test.go`
- **NodeHasURLTemplateProcessor()** (2 connections) — `controllers/nodecollector/collectorconfig/odigos_config_extension.go`
- **NodeOdigosExtDomain()** (2 connections) — `controllers/nodecollector/collectorconfig/odigos_config_extension.go`
- **getSpanMetricsPipelineProcessors()** (2 connections) — `controllers/nodecollector/collectorconfig/spanmetrics.go`
- **ToExporterConfigurerArray()** (2 connections) — `controllers/common/config.go`
- **odigos_config_extension.go** (2 connections) — `controllers/nodecollector/collectorconfig/odigos_config_extension.go`
- **spanmetrics.go** (2 connections) — `controllers/nodecollector/collectorconfig/spanmetrics.go`
- **isSamplingActionsEnabled()** (2 connections) — `controllers/nodecollector/configmap.go`
- **ownMetricsTelemetryConfig()** (2 connections) — `controllers/nodecollector/configmap.go`
- **K8sExporterConfigurer** (1 connections) — `k8sconfig/config.go`
- **Rule** (1 connections) — `controllers/actions/sampling/config.go`
- **RuleType** (1 connections) — `controllers/actions/sampling/config.go`
- *... and 1 more nodes in this community*

## Relationships

- [[URL Templatization Rule GraphQL]] (78 shared connections)
- [[Autoscaler Resource Detection]] (4 shared connections)
- [[Instrumentor Assertions Helpers]] (3 shared connections)
- [[Pipeline Datastreams Docs]] (3 shared connections)
- [[Community 201]] (2 shared connections)
- [[Odiglet CSI NodeServer]] (2 shared connections)
- [[Community 274]] (1 shared connections)
- [[Community 293]] (1 shared connections)
- [[Community 294]] (1 shared connections)
- [[Community 275]] (1 shared connections)
- [[Frontend Setup & Layout Hooks]] (1 shared connections)

## Source Files

- `controllers/actions/sampling/config.go`
- `controllers/clustercollector/configmap.go`
- `controllers/clustercollector/configmap_test.go`
- `controllers/common/config.go`
- `controllers/nodecollector/collectorconfig/odigos_config_extension.go`
- `controllers/nodecollector/collectorconfig/spanmetrics.go`
- `controllers/nodecollector/configmap.go`
- `k8sconfig/config.go`

## Audit Trail

- EXTRACTED: 69 (71%)
- INFERRED: 28 (29%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*