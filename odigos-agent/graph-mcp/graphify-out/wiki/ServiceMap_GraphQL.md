# ServiceMap GraphQL

> 22 nodes · cohesion 0.11

## Key Concepts

- **OdigosMetricsConsumer** (13 connections) — `services/collector_metrics/collector_metrics.go`
- **collector_metrics.go** (8 connections) — `services/metrics/collector_metrics.go`
- **NewOdigosMetrics()** (4 connections) — `services/collector_metrics/collector_metrics.go`
- **trafficMetrics** (4 connections) — `services/collector_metrics/collector_metrics.go`
- **getSenderPod()** (3 connections) — `services/collector_metrics/collector_metrics.go`
- **newSourcesMetrics()** (3 connections) — `services/collector_metrics/node_collector.go`
- **.ConsumeMetrics()** (3 connections) — `services/collector_metrics/collector_metrics.go`
- **.RunDeleteWatcherAndNotifications()** (3 connections) — `services/collector_metrics/collector_metrics.go`
- **node_collector.go** (3 connections) — `services/collector_metrics/node_collector.go`
- **collectorRoleFromResource()** (2 connections) — `services/collector_metrics/collector_metrics.go`
- **.runNotificationsLoop()** (2 connections) — `services/collector_metrics/collector_metrics.go`
- **.Capabilities()** (1 connections) — `services/collector_metrics/collector_metrics.go`
- **.GetDestinationsMetrics()** (1 connections) — `services/collector_metrics/collector_metrics.go`
- **.GetServiceGraphEdges()** (1 connections) — `services/collector_metrics/collector_metrics.go`
- **.GetSingleDestinationMetrics()** (1 connections) — `services/collector_metrics/collector_metrics.go`
- **.GetSingleSourceMetrics()** (1 connections) — `services/collector_metrics/collector_metrics.go`
- **.NotifyDestinationDeleted()** (1 connections) — `services/collector_metrics/collector_metrics.go`
- **.NotifySourceAdded()** (1 connections) — `services/collector_metrics/collector_metrics.go`
- **.NotifySourceDeleted()** (1 connections) — `services/collector_metrics/collector_metrics.go`
- **.TotalDataSent()** (1 connections) — `services/collector_metrics/collector_metrics.go`
- **.TotalThroughput()** (1 connections) — `services/collector_metrics/collector_metrics.go`
- **PodRates** (1 connections) — `services/metrics/collector_metrics.go`

## Relationships

- [[gRPC Server Config (Collector)]] (50 shared connections)
- [[Frontend Destination Connection Test]] (3 shared connections)
- [[CLI Centralized Install]] (2 shared connections)
- [[Frontend GraphQL Loaders]] (1 shared connections)
- [[Community 245]] (1 shared connections)
- [[Frontend Sampling Rules]] (1 shared connections)
- [[Config YAML Field Schema]] (1 shared connections)

## Source Files

- `services/collector_metrics/collector_metrics.go`
- `services/collector_metrics/node_collector.go`
- `services/metrics/collector_metrics.go`

## Audit Trail

- EXTRACTED: 54 (92%)
- INFERRED: 5 (8%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*