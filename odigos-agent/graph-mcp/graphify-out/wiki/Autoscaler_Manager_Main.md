# Autoscaler Manager Main

> 17 nodes · cohesion 0.26

## Key Concepts

- **.ConsumeTraces()** (10 connections) — `exporters/azureblobstorageexporter/exporter.go`
- **routerConnector** (8 connections) — `connectors/odigosrouterconnector/connector.go`
- **.ConsumeLogs()** (7 connections) — `exporters/azureblobstorageexporter/exporter.go`
- **MockDestinationExporter** (7 connections) — `exporters/mockdestinationexporter/exporter.go`
- **.Capabilities()** (6 connections) — `exporters/azureblobstorageexporter/exporter.go`
- **exporter.go** (6 connections) — `exporters/mockdestinationexporter/exporter.go`
- **.ConsumeMetrics()** (6 connections) — `connectors/odigosrouterconnector/connector.go`
- **.resolveDataStreams()** (5 connections) — `connectors/odigosrouterconnector/connector.go`
- **ABSExporter** (4 connections) — `exporters/azureblobstorageexporter/exporter.go`
- **GCSExporter** (4 connections) — `exporters/googlecloudstorageexporter/exporter.go`
- **NewMockDestinationExporter()** (4 connections) — `exporters/mockdestinationexporter/exporter.go`
- **createLogsExporter()** (4 connections) — `exporters/mockdestinationexporter/factory.go`
- **createTracesExporter()** (4 connections) — `exporters/mockdestinationexporter/factory.go`
- **.mockExport()** (4 connections) — `exporters/mockdestinationexporter/exporter.go`
- **NewAzureBlobExporter()** (3 connections) — `exporters/azureblobstorageexporter/exporter.go`
- **NewGCSExporter()** (3 connections) — `exporters/googlecloudstorageexporter/exporter.go`
- **createMetricsExporter()** (2 connections) — `exporters/mockdestinationexporter/factory.go`

## Relationships

- [[CLI Uninstall & Logging]] (73 shared connections)
- [[Sampling Matchers (Collector)]] (5 shared connections)
- [[Sampling Rule Types (GraphQL)]] (4 shared connections)
- [[Cypress E2E Tests]] (4 shared connections)
- [[GraphQL Introspection]] (1 shared connections)

## Source Files

- `connectors/odigosrouterconnector/connector.go`
- `exporters/azureblobstorageexporter/exporter.go`
- `exporters/googlecloudstorageexporter/exporter.go`
- `exporters/mockdestinationexporter/exporter.go`
- `exporters/mockdestinationexporter/factory.go`

## Audit Trail

- EXTRACTED: 68 (78%)
- INFERRED: 19 (22%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*