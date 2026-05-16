# Odiglet Main & Instrumentation

> 24 nodes · cohesion 0.14

## Key Concepts

- **clusterCollectorMetrics** (11 connections) — `services/collector_metrics/cluster_collector.go`
- **cluster_collector.go** (11 connections) — `services/collector_metrics/cluster_collector.go`
- **.updateDestinationMetricsByExporter()** (8 connections) — `services/collector_metrics/cluster_collector.go`
- **.handleClusterCollectorMetrics()** (5 connections) — `services/collector_metrics/cluster_collector.go`
- **.newDestinationTrafficMetrics()** (5 connections) — `services/collector_metrics/cluster_collector.go`
- **averageSizeCalculator** (4 connections) — `services/collector_metrics/cluster_collector.go`
- **.updateAverageEstimates()** (4 connections) — `services/collector_metrics/cluster_collector.go`
- **.UpdateFromDataPoint()** (4 connections) — `services/collector_metrics/cluster_collector.go`
- **.lastCalculatedAvgLogSize()** (3 connections) — `services/collector_metrics/cluster_collector.go`
- **.lastCalculatedAvgMetricSize()** (3 connections) — `services/collector_metrics/cluster_collector.go`
- **newClusterCollectorMetrics()** (3 connections) — `services/collector_metrics/cluster_collector.go`
- **.newDestinationMetrics()** (3 connections) — `services/collector_metrics/cluster_collector.go`
- **.update()** (2 connections) — `services/collector_metrics/cluster_collector.go`
- **buildNodeID()** (2 connections) — `services/collector_metrics/cluster_collector.go`
- **copyClientServerStringAttrs()** (2 connections) — `services/collector_metrics/cluster_collector.go`
- **metricAttributesToDestinationID()** (2 connections) — `services/collector_metrics/cluster_collector.go`
- **newServiceGraph()** (2 connections) — `services/collector_metrics/cluster_collector.go`
- **.destinationsMetrics()** (2 connections) — `services/collector_metrics/cluster_collector.go`
- **ServiceGraph** (2 connections) — `services/collector_metrics/cluster_collector.go`
- **ServiceGraphEdge** (2 connections) — `services/collector_metrics/cluster_collector.go`
- **singleDestinationMetrics** (2 connections) — `services/collector_metrics/cluster_collector.go`
- **.removeClusterCollector()** (1 connections) — `services/collector_metrics/cluster_collector.go`
- **.removeDestination()** (1 connections) — `services/collector_metrics/cluster_collector.go`
- **destinationTrafficMetrics** (1 connections) — `services/collector_metrics/cluster_collector.go`

## Relationships

- [[Frontend Destination Connection Test]] (84 shared connections)
- [[gRPC Server Config (Collector)]] (1 shared connections)

## Source Files

- `services/collector_metrics/cluster_collector.go`

## Audit Trail

- EXTRACTED: 83 (98%)
- INFERRED: 2 (2%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*