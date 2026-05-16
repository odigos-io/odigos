# Sampling Rule Types (GraphQL)

> 42 nodes · cohesion 0.13

## Key Concepts

- **.Start()** (32 connections) — `processors/odigoslogsresourceattrsprocessor/internal/kube/client.go`
- **connector_test.go** (26 connections) — `connectors/servicegraphconnector/connector_test.go`
- **.Shutdown()** (19 connections) — `connectors/servicegraphconnector/internal/metadata/generated_telemetry.go`
- **newConnector()** (14 connections) — `connectors/servicegraphconnector/connector.go`
- **ebpfReceiver** (13 connections) — `receivers/odigosebpfreceiver/metrics.go`
- **TestConnectorConsume()** (11 connections) — `connectors/servicegraphconnector/connector_test.go`
- **newMockMetricsExporter()** (10 connections) — `connectors/servicegraphconnector/connector_test.go`
- **TestExponentialHistogram()** (8 connections) — `connectors/servicegraphconnector/connector_test.go`
- **TestExtraDimensionsLabels()** (8 connections) — `connectors/servicegraphconnector/connector_test.go`
- **TestVirtualNodeClientLabels()** (8 connections) — `connectors/servicegraphconnector/connector_test.go`
- **TestVirtualNodeEmitsAllPeerAttributesOnMetrics()** (8 connections) — `connectors/servicegraphconnector/connector_test.go`
- **TestVirtualNodeServerLabels()** (8 connections) — `connectors/servicegraphconnector/connector_test.go`
- **TestMapsAreConsistentDuringCleanup()** (7 connections) — `connectors/servicegraphconnector/connector_test.go`
- **TestValidateOwnTelemetry()** (7 connections) — `connectors/servicegraphconnector/connector_test.go`
- **buildSampleTrace()** (6 connections) — `connectors/servicegraphconnector/connector_test.go`
- **TestStaleSeriesCleanup()** (6 connections) — `connectors/servicegraphconnector/connector_test.go`
- **mockMetricsExporter** (6 connections) — `connectors/servicegraphconnector/connector_test.go`
- **.GetMetrics()** (6 connections) — `connectors/servicegraphconnector/connector_test.go`
- **ptr()** (5 connections) — `connectors/servicegraphconnector/connector_test.go`
- **verifyHappyCaseMetricsWithDuration()** (5 connections) — `connectors/servicegraphconnector/connector_test.go`
- **.collectMetrics()** (4 connections) — `receivers/odigosebpfreceiver/metrics.go`
- **verifyCount()** (4 connections) — `connectors/servicegraphconnector/connector_test.go`
- **verifyHappyCaseLatencyMetrics()** (4 connections) — `connectors/servicegraphconnector/connector_test.go`
- **.parseResourceAttributes()** (3 connections) — `receivers/odigosebpfreceiver/metrics.go`
- **.processInnerMapMetrics()** (3 connections) — `receivers/odigosebpfreceiver/metrics.go`
- *... and 17 more nodes in this community*

## Relationships

- [[Cypress E2E Tests]] (11 shared connections)
- [[Sampling Matchers (Collector)]] (7 shared connections)
- [[CLI Uninstall & Logging]] (5 shared connections)
- [[Instrumentor Rollout Mocks]] (5 shared connections)
- [[Workload Resource Attrs (Collector)]] (3 shared connections)
- [[Community 205]] (2 shared connections)
- [[Collector Processors (Tail/Conditional)]] (2 shared connections)
- [[Collector Client gRPC Config]] (2 shared connections)
- [[Collector Settings CRD]] (1 shared connections)
- [[Connector & Receiver Tests]] (1 shared connections)
- [[Community 242]] (1 shared connections)
- [[Community 217]] (1 shared connections)

## Source Files

- `connectors/servicegraphconnector/connector.go`
- `connectors/servicegraphconnector/connector_test.go`
- `connectors/servicegraphconnector/internal/metadata/generated_telemetry.go`
- `processors/odigoslogsresourceattrsprocessor/internal/kube/client.go`
- `receivers/odigosebpfreceiver/metrics.go`
- `receivers/odigosebpfreceiver/odigosebpfreceiver.go`

## Audit Trail

- EXTRACTED: 233 (87%)
- INFERRED: 36 (13%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*