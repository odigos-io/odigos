# Collector gRPC Benchmarks

> 10 nodes · cohesion 0.27

## Key Concepts

- **custom_metrics_handler.go** (7 connections) — `controllers/metricshandler/custom_metrics_handler.go`
- **MetricHandler()** (4 connections) — `controllers/metricshandler/custom_metrics_handler.go`
- **RegisterCustomMetricsAPI()** (4 connections) — `controllers/metricshandler/custom_metrics_handler.go`
- **IsOwnedByOdigos()** (4 connections) — `controllers/metricshandler/helpers.go`
- **isPodOOMKilled()** (2 connections) — `controllers/metricshandler/custom_metrics_handler.go`
- **scrapeGatewayMetric()** (2 connections) — `controllers/metricshandler/custom_metrics_handler.go`
- **helpers.go** (1 connections) — `controllers/metricshandler/helpers.go`
- **DiscoveryHandler()** (1 connections) — `controllers/metricshandler/custom_metrics_handler.go`
- **MetricValue** (1 connections) — `controllers/metricshandler/custom_metrics_handler.go`
- **MetricValueList** (1 connections) — `controllers/metricshandler/custom_metrics_handler.go`

## Relationships

- [[Community 200]] (24 shared connections)
- [[Enterprise Installation Docs]] (1 shared connections)
- [[Autoscaler Sampler Handlers]] (1 shared connections)
- [[Destination CR Docs]] (1 shared connections)

## Source Files

- `controllers/metricshandler/custom_metrics_handler.go`
- `controllers/metricshandler/helpers.go`

## Audit Trail

- EXTRACTED: 22 (81%)
- INFERRED: 5 (19%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*