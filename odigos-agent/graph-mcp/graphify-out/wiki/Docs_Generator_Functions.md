# Docs Generator Functions

> 27 nodes · cohesion 0.10

## Key Concepts

- **factory.go** (16 connections) — `receivers/odigosebpfreceiver/factory.go`
- **createTracesProcessor()** (8 connections) — `processors/odigostrafficmetrics/factory.go`
- **provider** (7 connections) — `providers/odigosk8scmprovider/provider.go`
- **newThroughputMeasurementProcessor()** (5 connections) — `processors/odigostrafficmetrics/processor.go`
- **.Retrieve()** (4 connections) — `providers/odigosk8scmprovider/provider.go`
- **calculateUniqueNewAttributes()** (3 connections) — `processors/odigosconditionalattributes/factory.go`
- **NewOdigosConfig()** (3 connections) — `extension/odigosconfigk8sextension/odigosconfig.go`
- **.runInformer()** (3 connections) — `providers/odigosk8scmprovider/provider.go`
- **newSamplingProcessor()** (3 connections) — `processors/odigossamplingprocessor/processor.go`
- **normalizeLanguages()** (3 connections) — `processors/odigossqldboperationprocessor/factory.go`
- **createLogsProcessor()** (3 connections) — `processors/odigostrafficmetrics/factory.go`
- **createMetricsProcessor()** (3 connections) — `processors/odigostrafficmetrics/factory.go`
- **provider.go** (3 connections) — `providers/odigosk8scmprovider/provider.go`
- **odigosconfig.go** (2 connections) — `extension/odigosconfigk8sextension/odigosconfig.go`
- **newCache()** (2 connections) — `extension/odigosconfigk8sextension/cache.go`
- **create()** (2 connections) — `extension/odigosconfigk8sextension/factory.go`
- **NewFactory()** (2 connections) — `receivers/odigosebpfreceiver/factory.go`
- **.debouncedNotify()** (2 connections) — `providers/odigosk8scmprovider/provider.go`
- **.waitForConfigMap()** (2 connections) — `providers/odigosk8scmprovider/provider.go`
- **newTailSamplingProcessor()** (2 connections) — `processors/odigostailsamplingprocessor/processor.go`
- **createTracesToMetricsConnector()** (2 connections) — `connectors/servicegraphconnector/factory.go`
- **createDefaultConfig()** (1 connections) — `receivers/odigosebpfreceiver/factory.go`
- **createLogsReceiver()** (1 connections) — `receivers/odigosebpfreceiver/factory.go`
- **createMetricsReceiver()** (1 connections) — `receivers/odigosebpfreceiver/factory.go`
- **createTracesReceiver()** (1 connections) — `receivers/odigosebpfreceiver/factory.go`
- *... and 2 more nodes in this community*

## Relationships

- [[CLI Uninstall & Logging]] (33 shared connections)
- [[Community 217]] (23 shared connections)
- [[Instrumentor Rollout Mocks]] (20 shared connections)
- [[Sampling Rule Types (GraphQL)]] (2 shared connections)
- [[Cypress E2E Tests]] (2 shared connections)
- [[Frontend Destination CRUD]] (2 shared connections)
- [[Workload Resource Attrs (Collector)]] (1 shared connections)
- [[Collector Client gRPC Config]] (1 shared connections)
- [[Pyroscope Profiling Conversion]] (1 shared connections)
- [[Community 242]] (1 shared connections)

## Source Files

- `connectors/servicegraphconnector/factory.go`
- `extension/odigosconfigk8sextension/cache.go`
- `extension/odigosconfigk8sextension/factory.go`
- `extension/odigosconfigk8sextension/odigosconfig.go`
- `processors/odigosconditionalattributes/factory.go`
- `processors/odigossamplingprocessor/processor.go`
- `processors/odigossqldboperationprocessor/factory.go`
- `processors/odigostailsamplingprocessor/processor.go`
- `processors/odigostrafficmetrics/factory.go`
- `processors/odigostrafficmetrics/processor.go`
- `providers/odigosk8scmprovider/provider.go`
- `receivers/odigosebpfreceiver/factory.go`

## Audit Trail

- EXTRACTED: 64 (74%)
- INFERRED: 22 (26%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*