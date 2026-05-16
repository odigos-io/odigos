# Sources CLI Docs

> 23 nodes · cohesion 0.12

## Key Concepts

- **main()** (17 connections) — `main.go`
- **main.go** (8 connections) — `main.go`
- **Client** (6 connections) — `kube/client.go`
- **initKubernetesClient()** (5 connections) — `main.go`
- **startWatchers()** (5 connections) — `main.go`
- **serveClientFiles()** (3 connections) — `main.go`
- **Receiver** (3 connections) — `services/otlp/receiver.go`
- **IngestGate** (3 connections) — `services/profiles/ingest_gate.go`
- **receiver.go** (3 connections) — `services/otlp/receiver.go`
- **.Cancel()** (3 connections) — `kube/watchers/batcher.go`
- **parseFlags()** (2 connections) — `main.go`
- **CreateClient()** (2 connections) — `kube/client.go`
- **InitWorkloadKindsAvailability()** (2 connections) — `kube/client.go`
- **SetDefaultClient()** (2 connections) — `kube/client.go`
- **NewAPIFromURL()** (2 connections) — `services/metrics/client.go`
- **noExtensionsHost** (2 connections) — `services/otlp/receiver.go`
- **NewReceiver()** (2 connections) — `services/otlp/receiver.go`
- **.WaitAndShutdown()** (2 connections) — `services/otlp/receiver.go`
- **NewProfilesIngestGate()** (2 connections) — `services/profiles/ingest_gate.go`
- **Flags** (1 connections) — `main.go`
- **.GetExtensions()** (1 connections) — `services/otlp/receiver.go`
- **.IsEnabled()** (1 connections) — `services/profiles/ingest_gate.go`
- **.Set()** (1 connections) — `services/profiles/ingest_gate.go`

## Relationships

- [[Frontend GraphQL Loaders]] (47 shared connections)
- [[OTLP Test Connection (Frontend)]] (16 shared connections)
- [[Component Log Levels Config]] (2 shared connections)
- [[JVM Metrics Handler]] (2 shared connections)
- [[Frontend CSRF]] (2 shared connections)
- [[Pod Webhook Env Injector]] (1 shared connections)
- [[gRPC Server Config (Collector)]] (1 shared connections)
- [[CLI Endpoints Detection]] (1 shared connections)
- [[Community 230]] (1 shared connections)
- [[Retry & OTLP Exporter Config]] (1 shared connections)
- [[Community 229]] (1 shared connections)
- [[K8s Workload GraphQL Resolver]] (1 shared connections)

## Source Files

- `kube/client.go`
- `kube/watchers/batcher.go`
- `main.go`
- `services/metrics/client.go`
- `services/otlp/receiver.go`
- `services/profiles/ingest_gate.go`

## Audit Trail

- EXTRACTED: 51 (65%)
- INFERRED: 27 (35%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*