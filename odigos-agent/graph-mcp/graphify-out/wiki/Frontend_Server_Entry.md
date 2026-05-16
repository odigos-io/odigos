# Frontend Server Entry

> 23 nodes · cohesion 0.15

## Key Concepts

- **inspection.go** (14 connections) — `pkg/kube/runtime_details/inspection.go`
- **persistRuntimeDetailsToInstrumentationConfig()** (6 connections) — `pkg/kube/runtime_details/inspection.go`
- **.scan()** (6 connections) — `pkg/kube/runtime_details/startup_scan.go`
- **GroupByPodContainer()** (5 connections) — `pkg/process/process_linux.go`
- **inspectContainerProcesses()** (5 connections) — `pkg/kube/runtime_details/inspection.go`
- **runtimeInspection()** (4 connections) — `pkg/kube/runtime_details/inspection.go`
- **runtimeInspectionFromGroupedPIDs()** (4 connections) — `pkg/kube/runtime_details/inspection.go`
- **updateRuntimeDetailsWithContainerRuntimeEnvs()** (4 connections) — `pkg/kube/runtime_details/inspection.go`
- **process_linux.go** (3 connections) — `pkg/process/process_linux.go`
- **addConditions()** (3 connections) — `pkg/kube/runtime_details/inspection.go`
- **fetchAndSetEnvFromContainerRuntime()** (3 connections) — `pkg/kube/runtime_details/inspection.go`
- **mergeRuntimeDetails()** (3 connections) — `pkg/kube/runtime_details/inspection.go`
- **startupRuntimeDetection** (3 connections) — `pkg/kube/runtime_details/startup_scan.go`
- **isInPodContainersBatchPredicate()** (2 connections) — `pkg/process/process_linux.go`
- **checkEnvVarsInContainerManifest()** (2 connections) — `pkg/kube/runtime_details/inspection.go`
- **getContainerID()** (2 connections) — `pkg/kube/runtime_details/inspection.go`
- **mergeLdPreloadEnvVars()** (2 connections) — `pkg/kube/runtime_details/inspection.go`
- **relevantProcessesDetailsInContainer()** (2 connections) — `pkg/kube/runtime_details/inspection.go`
- **InspectionResults** (2 connections) — `pkg/kube/runtime_details/inspection.go`
- **.runtimeDetailsList()** (2 connections) — `pkg/kube/runtime_details/inspection.go`
- **.Start()** (2 connections) — `pkg/kube/runtime_details/startup_scan.go`
- **startup_scan.go** (1 connections) — `pkg/kube/runtime_details/startup_scan.go`
- **PodContainerUID** (1 connections) — `pkg/process/process_linux.go`

## Relationships

- [[OSS Destination Docs]] (74 shared connections)
- [[VM Agent Docs]] (3 shared connections)
- [[Community 278]] (2 shared connections)
- [[Community 225]] (1 shared connections)
- [[Source Object Docs]] (1 shared connections)

## Source Files

- `pkg/kube/runtime_details/inspection.go`
- `pkg/kube/runtime_details/startup_scan.go`
- `pkg/process/process_linux.go`

## Audit Trail

- EXTRACTED: 68 (84%)
- INFERRED: 13 (16%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*