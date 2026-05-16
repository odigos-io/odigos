# Clientset (api)

> 19 nodes · cohesion 0.18

## Key Concepts

- **CalculateGatewayConfig()** (17 connections) — `pipelinegen/config_builder.go`
- **config_builder.go** (11 connections) — `pipelinegen/config_builder.go`
- **GetGatewayConfig()** (5 connections) — `pipelinegen/config_builder.go`
- **GetTelemetryRootPipelineName()** (5 connections) — `pipelinegen/datastreams.go`
- **insertServiceGraphPipeline()** (4 connections) — `pipelinegen/config_builder.go`
- **applyRootPipelineForSignal()** (3 connections) — `pipelinegen/config_builder.go`
- **GetBasicConfig()** (3 connections) — `pipelinegen/config_builder.go`
- **insertClusterMetricsResources()** (3 connections) — `pipelinegen/config_builder.go`
- **insertRootPipelinesToConfig()** (3 connections) — `pipelinegen/config_builder.go`
- **DataStreams** (3 connections) — `pipelinegen/datastreams.go`
- **GetSignalsRootPipelineNames()** (3 connections) — `pipelinegen/datastreams.go`
- **TestCalculateWithBaseNoOTLP()** (2 connections) — `config/root_test.go`
- **AddServiceGraphScrapeConfig()** (2 connections) — `pipelinegen/config_builder.go`
- **filterSmallBatchesProcessor()** (2 connections) — `pipelinegen/config_builder.go`
- **getTailSamplingProcessors()** (2 connections) — `pipelinegen/config_builder.go`
- **buildDataStreamPipelines()** (2 connections) — `pipelinegen/pipeline_builder.go`
- **Destination** (1 connections) — `pipelinegen/datastreams.go`
- **GatewayConfigOptions** (1 connections) — `pipelinegen/config_builder.go`
- **pipeline_builder.go** (1 connections) — `pipelinegen/pipeline_builder.go`

## Relationships

- [[Odigos Overview Docs]] (71 shared connections)
- [[Common CRD-to-Config Root]] (2 shared connections)

## Source Files

- `config/root_test.go`
- `pipelinegen/config_builder.go`
- `pipelinegen/datastreams.go`
- `pipelinegen/pipeline_builder.go`

## Audit Trail

- EXTRACTED: 51 (70%)
- INFERRED: 22 (30%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*