# Managed Backend Destination Docs

> 44 nodes · cohesion 0.07

## Key Concepts

- **generated_telemetry_test.go** (25 connections) — `receivers/odigosebpfreceiver/internal/metadata/generated_telemetry_test.go`
- **TestSetupTelemetry()** (18 connections) — `connectors/servicegraphconnector/internal/metadatatest/generated_telemetrytest_test.go`
- **LogsBuilder** (5 connections) — `receivers/odigosebpfreceiver/internal/metadata/generated_logs.go`
- **generated_logs.go** (5 connections) — `receivers/odigosebpfreceiver/internal/metadata/generated_logs.go`
- **generated_telemetry.go** (5 connections) — `receivers/odigosebpfreceiver/internal/metadata/generated_telemetry.go`
- **.Meter()** (4 connections) — `connectors/servicegraphconnector/internal/metadata/generated_telemetry_test.go`
- **.apply()** (4 connections) — `receivers/odigosebpfreceiver/internal/metadata/generated_logs.go`
- **TestLogsBuilderAppendLogRecord()** (3 connections) — `receivers/odigosebpfreceiver/internal/metadata/generated_logs_test.go`
- **WithLogsResource()** (3 connections) — `receivers/odigosebpfreceiver/internal/metadata/generated_logs.go`
- **NewTelemetryBuilder()** (3 connections) — `connectors/servicegraphconnector/internal/metadata/generated_telemetry.go`
- **TestProviders()** (3 connections) — `connectors/servicegraphconnector/internal/metadata/generated_telemetry_test.go`
- **.EmitForResource()** (3 connections) — `receivers/odigosebpfreceiver/internal/metadata/generated_logs.go`
- **.Tracer()** (3 connections) — `connectors/servicegraphconnector/internal/metadata/generated_telemetry_test.go`
- **resourceLogsOptionFunc** (3 connections) — `receivers/odigosebpfreceiver/internal/metadata/generated_logs.go`
- **AssertEqualConnectorServicegraphTotalEdges()** (3 connections) — `connectors/servicegraphconnector/internal/metadatatest/generated_telemetrytest.go`
- **.Emit()** (2 connections) — `receivers/odigosebpfreceiver/internal/metadata/generated_logs.go`
- **mockMeterProvider** (2 connections) — `connectors/servicegraphconnector/internal/metadata/generated_telemetry_test.go`
- **mockTracerProvider** (2 connections) — `connectors/servicegraphconnector/internal/metadata/generated_telemetry_test.go`
- **TelemetryBuilder** (2 connections) — `connectors/servicegraphconnector/internal/metadata/generated_telemetry.go`
- **TelemetryBuilderOption** (2 connections) — `connectors/servicegraphconnector/internal/metadata/generated_telemetry.go`
- **AssertEqualConnectorServicegraphDroppedSpans()** (2 connections) — `connectors/servicegraphconnector/internal/metadatatest/generated_telemetrytest.go`
- **AssertEqualConnectorServicegraphExpiredEdges()** (2 connections) — `connectors/servicegraphconnector/internal/metadatatest/generated_telemetrytest.go`
- **AssertEqualEbpfLogsAttrCacheSize()** (2 connections) — `receivers/odigosebpfreceiver/internal/metadatatest/generated_telemetrytest.go`
- **AssertEqualEbpfLostSamples()** (2 connections) — `receivers/odigosebpfreceiver/internal/metadatatest/generated_telemetrytest.go`
- **AssertEqualEbpfMemoryPressureWaitTimeTotal()** (2 connections) — `receivers/odigosebpfreceiver/internal/metadatatest/generated_telemetrytest.go`
- *... and 19 more nodes in this community*

## Relationships

- [[Collector Processors (Tail/Conditional)]] (138 shared connections)
- [[Sampling Rule Types (GraphQL)]] (2 shared connections)

## Source Files

- `connectors/servicegraphconnector/internal/metadata/generated_telemetry.go`
- `connectors/servicegraphconnector/internal/metadata/generated_telemetry_test.go`
- `connectors/servicegraphconnector/internal/metadatatest/generated_telemetrytest.go`
- `connectors/servicegraphconnector/internal/metadatatest/generated_telemetrytest_test.go`
- `processors/odigostailsamplingprocessor/internal/metadatatest/generated_telemetrytest.go`
- `processors/odigostrafficmetrics/internal/metadatatest/generated_telemetrytest.go`
- `receivers/odigosebpfreceiver/internal/metadata/generated_logs.go`
- `receivers/odigosebpfreceiver/internal/metadata/generated_logs_test.go`
- `receivers/odigosebpfreceiver/internal/metadata/generated_telemetry.go`
- `receivers/odigosebpfreceiver/internal/metadata/generated_telemetry_test.go`
- `receivers/odigosebpfreceiver/internal/metadatatest/generated_telemetrytest.go`

## Audit Trail

- EXTRACTED: 101 (72%)
- INFERRED: 39 (28%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*