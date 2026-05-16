# Frontend Agent Injection Status

> 10 nodes · cohesion 0.36

## Key Concepts

- **OwnMetricsConfigUi()** (6 connections) — `controllers/nodecollector/collectorconfig/ownmetrics-ui.go`
- **ownmetrics-ui.go** (6 connections) — `controllers/nodecollector/collectorconfig/ownmetrics-ui.go`
- **addOwnMetricsPipeline()** (5 connections) — `controllers/clustercollector/ownmetrics.go`
- **receiversConfigForOwnMetricsUi()** (4 connections) — `controllers/nodecollector/collectorconfig/ownmetrics-ui.go`
- **ownmetrics.go** (4 connections) — `controllers/clustercollector/ownmetrics.go`
- **gatewayPrometheusReceiverConfig()** (2 connections) — `controllers/clustercollector/ownmetrics.go`
- **victoriaMetricsExporter()** (2 connections) — `controllers/clustercollector/ownmetrics.go`
- **ownMetricsExportersUi()** (2 connections) — `controllers/nodecollector/collectorconfig/ownmetrics-ui.go`
- **ownMetricsPipelinesUi()** (2 connections) — `controllers/nodecollector/collectorconfig/ownmetrics-ui.go`
- **serviceTelemetryConfigForOwnMetricsUi()** (2 connections) — `controllers/nodecollector/collectorconfig/ownmetrics-ui.go`

## Relationships

- [[Community 201]] (32 shared connections)
- [[URL Templatization Rule GraphQL]] (2 shared connections)
- [[Enterprise Installation Docs]] (1 shared connections)

## Source Files

- `controllers/clustercollector/ownmetrics.go`
- `controllers/nodecollector/collectorconfig/ownmetrics-ui.go`

## Audit Trail

- EXTRACTED: 33 (94%)
- INFERRED: 2 (6%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*