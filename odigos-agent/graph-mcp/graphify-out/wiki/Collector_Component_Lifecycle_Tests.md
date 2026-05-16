# Collector Component Lifecycle Tests

> 10 nodes · cohesion 0.47

## Key Concepts

- **configmap_test.go** (10 connections) — `controllers/nodecollector/configmap_test.go`
- **TestCalculateConfigMapData()** (8 connections) — `controllers/nodecollector/configmap_test.go`
- **TestCalculateConfigMapDataTracesOnlyNoLoadBalancing()** (8 connections) — `controllers/nodecollector/configmap_test.go`
- **NewMockInstrumentationConfig()** (3 connections) — `controllers/nodecollector/configmap_test.go`
- **NewMockNamespace()** (3 connections) — `controllers/nodecollector/configmap_test.go`
- **NewMockTestDaemonSet()** (3 connections) — `controllers/nodecollector/configmap_test.go`
- **NewMockTestDeployment()** (3 connections) — `controllers/nodecollector/configmap_test.go`
- **NewMockTestStatefulSet()** (3 connections) — `controllers/nodecollector/configmap_test.go`
- **openTestData()** (3 connections) — `controllers/nodecollector/configmap_test.go`
- **NewMockDestinationList()** (1 connections) — `controllers/nodecollector/configmap_test.go`

## Relationships

- [[Pipeline Datastreams Docs]] (42 shared connections)
- [[URL Templatization Rule GraphQL]] (3 shared connections)

## Source Files

- `controllers/nodecollector/configmap_test.go`

## Audit Trail

- EXTRACTED: 43 (96%)
- INFERRED: 2 (4%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*