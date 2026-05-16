# InstrumentationConfig CRD

> 48 nodes · cohesion 0.07

## Key Concepts

- **Service Graph Connector** (34 connections) — `./connectors/servicegraphconnector/README.md`
- **connector.go** (16 connections) — `connectors/servicegraphconnector/connector.go`
- **.aggregateMetricsForEdge()** (9 connections) — `connectors/servicegraphconnector/connector.go`
- **.aggregateMetrics()** (8 connections) — `connectors/servicegraphconnector/connector.go`
- **.buildMetricKeyFromEdge()** (5 connections) — `connectors/servicegraphconnector/connector.go`
- **.collectClientLatencyMetrics()** (5 connections) — `connectors/servicegraphconnector/connector.go`
- **.collectCountMetrics()** (5 connections) — `connectors/servicegraphconnector/connector.go`
- **.collectServerLatencyMetrics()** (5 connections) — `connectors/servicegraphconnector/connector.go`
- **buildDimensions()** (4 connections) — `connectors/servicegraphconnector/connector.go`
- **mapDurationsToFloat()** (4 connections) — `connectors/servicegraphconnector/connector.go`
- **sortedMapKeys()** (4 connections) — `connectors/servicegraphconnector/connector.go`
- **.buildMetrics()** (4 connections) — `connectors/servicegraphconnector/connector.go`
- **.collectLatencyMetrics()** (4 connections) — `connectors/servicegraphconnector/connector.go`
- **.dimensionsForSeries()** (4 connections) — `connectors/servicegraphconnector/connector.go`
- **.flushMetrics()** (4 connections) — `connectors/servicegraphconnector/connector.go`
- **.nowWithOffset()** (4 connections) — `connectors/servicegraphconnector/connector.go`
- **.onExpire()** (4 connections) — `connectors/servicegraphconnector/connector.go`
- **.updateDurationMetrics()** (4 connections) — `connectors/servicegraphconnector/connector.go`
- **.cacheLoop()** (3 connections) — `connectors/servicegraphconnector/connector.go`
- **.metricFlushLoop()** (3 connections) — `connectors/servicegraphconnector/connector.go`
- **.onComplete()** (3 connections) — `connectors/servicegraphconnector/connector.go`
- **.updateClientDurationMetrics()** (3 connections) — `connectors/servicegraphconnector/connector.go`
- **.updateServerDurationMetrics()** (3 connections) — `connectors/servicegraphconnector/connector.go`
- **getFirstMatchingValue()** (3 connections) — `connectors/servicegraphconnector/util.go`
- **util.go** (2 connections) — `connectors/servicegraphconnector/util.go`
- *... and 23 more nodes in this community*

## Relationships

- [[Sampling Matchers (Collector)]] (160 shared connections)
- [[Cypress E2E Tests]] (10 shared connections)
- [[Sampling Rule Types (GraphQL)]] (7 shared connections)
- [[CLI Uninstall & Logging]] (5 shared connections)
- [[RenameAttribute CRD]] (1 shared connections)
- [[GraphQL Introspection]] (1 shared connections)
- [[Collector Settings CRD]] (1 shared connections)

## Source Files

- `./connectors/servicegraphconnector/README.md`
- `./connectors/servicegraphconnector/documentation.md`
- `connectors/odigosrouterconnector/connector.go`
- `connectors/servicegraphconnector/connector.go`
- `connectors/servicegraphconnector/util.go`
- `connectors/servicegraphconnector/util_test.go`
- `receivers/odigosebpfreceiver/config.go`

## Audit Trail

- EXTRACTED: 167 (90%)
- INFERRED: 18 (10%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*