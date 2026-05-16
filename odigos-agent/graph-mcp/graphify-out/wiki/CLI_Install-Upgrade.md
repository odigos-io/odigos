# CLI Install/Upgrade

> 28 nodes · cohesion 0.19

## Key Concepts

- **Execute()** (19 connections) — `cmd/root.go`
- **.String()** (18 connections) — `pkg/lifecycle/orchestrator.go`
- **.GetTransitionState()** (13 connections) — `pkg/lifecycle/postcheck.go`
- **Endpoints** (8 connections) — `cmd/resources/README.md`
- **.From()** (7 connections) — `pkg/lifecycle/postcheck.go`
- **.To()** (7 connections) — `pkg/lifecycle/postcheck.go`
- **.allInstrumentedPodsAreRunning()** (6 connections) — `pkg/lifecycle/instrumentation_ended.go`
- **.checkLanguageDetected()** (6 connections) — `pkg/lifecycle/waitforlangdetection.go`
- **DescribeSource()** (6 connections) — `pkg/remote/endpoints.go`
- **InstrumentationEnded** (5 connections) — `pkg/lifecycle/instrumentation_ended.go`
- **WorkloadKindFrombject()** (5 connections) — `pkg/lifecycle/orchestrator.go`
- **PreflightCheck** (5 connections) — `pkg/lifecycle/preflight.go`
- **WaitForLangDetection** (5 connections) — `pkg/lifecycle/waitforlangdetection.go`
- **InstrumentationStarted** (4 connections) — `pkg/lifecycle/instrumentation_started.go`
- **PostCheck** (4 connections) — `pkg/lifecycle/postcheck.go`
- **RequestLangDetection** (4 connections) — `pkg/lifecycle/requestlangdetection.go`
- **CreateSource()** (4 connections) — `pkg/remote/endpoints.go`
- **GetNumberOfDestinations()** (4 connections) — `pkg/remote/endpoints.go`
- **checks.go** (3 connections) — `pkg/preflight/checks.go`
- **isDestinationConfigured** (3 connections) — `pkg/preflight/checks.go`
- **isOdigosInstalled** (3 connections) — `pkg/preflight/checks.go`
- **isOdigosReady** (3 connections) — `pkg/preflight/checks.go`
- **.Description()** (3 connections) — `pkg/preflight/checks.go`
- **DescribeOdigos()** (3 connections) — `pkg/remote/endpoints.go`
- **DescribeOdigosEndpoint()** (3 connections) — `pkg/remote/endpoints.go`
- *... and 3 more nodes in this community*

## Relationships

- [[Enterprise Instrumentation Docs]] (5 shared connections)
- [[Community 232]] (3 shared connections)
- [[Destination CRD Docs (OTLP/Jaeger/GCS)]] (2 shared connections)
- [[CLI Component Resource Managers]] (2 shared connections)
- [[Frontend Collector Workload Helpers]] (2 shared connections)
- [[Service Graph Store]] (1 shared connections)
- [[Frontend Fetchers]] (1 shared connections)

## Source Files

- `cmd/resources/README.md`
- `cmd/root.go`
- `pkg/lifecycle/instrumentation_ended.go`
- `pkg/lifecycle/instrumentation_started.go`
- `pkg/lifecycle/orchestrator.go`
- `pkg/lifecycle/postcheck.go`
- `pkg/lifecycle/preflight.go`
- `pkg/lifecycle/requestlangdetection.go`
- `pkg/lifecycle/waitforlangdetection.go`
- `pkg/preflight/checks.go`
- `pkg/remote/endpoints.go`

## Audit Trail

- EXTRACTED: 111 (71%)
- INFERRED: 45 (29%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*