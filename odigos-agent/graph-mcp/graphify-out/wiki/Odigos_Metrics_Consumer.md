# Odigos Metrics Consumer

> 22 nodes · cohesion 0.11

## Key Concepts

- **Metrics** (13 connections) — `services/otlp/metrics.go`
- **sourcesMetrics** (8 connections) — `services/collector_metrics/node_collector.go`
- **.updateSourceMetrics()** (5 connections) — `services/collector_metrics/node_collector.go`
- **calculateThroughput()** (4 connections) — `services/collector_metrics/utils.go`
- **GetOdigletPodsWithMetrics()** (4 connections) — `services/collectors/metrics.go`
- **Profiles** (4 connections) — `services/otlp/profiles.go`
- **.handleNodeCollectorMetrics()** (3 connections) — `services/collector_metrics/node_collector.go`
- **.metricsByID()** (3 connections) — `services/collector_metrics/node_collector.go`
- **.Start()** (3 connections) — `services/otlp/metrics.go`
- **metrics.go** (3 connections) — `services/otlp/metrics.go`
- **newTrafficMetrics()** (2 connections) — `services/collector_metrics/collector_metrics.go`
- **TestCalculateThroughput()** (2 connections) — `services/collector_metrics/utils_test.go`
- **.OdigletPods()** (2 connections) — `graph/collectors.resolvers.go`
- **NewMetricsPipeline()** (2 connections) — `services/otlp/metrics.go`
- **.Register()** (2 connections) — `services/otlp/metrics.go`
- **.Shutdown()** (2 connections) — `services/otlp/metrics.go`
- **NewProfilesPipeline()** (2 connections) — `services/otlp/profiles.go`
- **profiles.go** (2 connections) — `services/otlp/profiles.go`
- **.addSource()** (1 connections) — `services/collector_metrics/node_collector.go`
- **.removeNodeCollector()** (1 connections) — `services/collector_metrics/node_collector.go`
- **.removeSource()** (1 connections) — `services/collector_metrics/node_collector.go`
- **GET_METRICS** (1 connections) — `webapp/graphql/queries/metrics.ts`

## Relationships

- [[Frontend Destination Connection Test]] (38 shared connections)
- [[OTLP Test Connection (Frontend)]] (14 shared connections)
- [[Frontend GraphQL Loaders]] (9 shared connections)
- [[Config YAML Field Schema]] (3 shared connections)
- [[gRPC Server Config (Collector)]] (2 shared connections)
- [[K8s Workload GraphQL Resolver]] (1 shared connections)
- [[Pod Webhook Env Injector]] (1 shared connections)
- [[Odigos Collector Processor Catalog]] (1 shared connections)
- [[Frontend Sampling Rules]] (1 shared connections)

## Source Files

- `graph/collectors.resolvers.go`
- `services/collector_metrics/collector_metrics.go`
- `services/collector_metrics/node_collector.go`
- `services/collector_metrics/utils.go`
- `services/collector_metrics/utils_test.go`
- `services/collectors/metrics.go`
- `services/otlp/metrics.go`
- `services/otlp/profiles.go`
- `webapp/graphql/queries/metrics.ts`

## Audit Trail

- EXTRACTED: 56 (80%)
- INFERRED: 14 (20%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*