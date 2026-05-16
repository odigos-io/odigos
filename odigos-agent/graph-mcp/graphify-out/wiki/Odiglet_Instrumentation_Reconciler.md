# Odiglet Instrumentation Reconciler

> 18 nodes · cohesion 0.16

## Key Concepts

- **New()** (13 connections) — `odiglet.go`
- **common.go** (6 connections) — `pkg/ebpf/common.go`
- **K8sProcessDetails** (5 connections) — `pkg/ebpf/process_details.go`
- **MatchingPodsForWorkloadOnNode()** (4 connections) — `pkg/kube/common/common.go`
- **WorkloadPodsOnCurrentNode()** (4 connections) — `pkg/kube/common/common.go`
- **DefaultK8sDetectorOptions()** (4 connections) — `pkg/detector/detector.go`
- **NewManager()** (4 connections) — `pkg/ebpf/common.go`
- **process_details.go** (3 connections) — `pkg/ebpf/process_details.go`
- **IsPodInCurrentNode()** (2 connections) — `pkg/kube/common/common.go`
- **relevantEnvVars()** (2 connections) — `pkg/detector/detector.go`
- **newHandler()** (2 connections) — `pkg/ebpf/common.go`
- **.ConfigGroup()** (2 connections) — `pkg/ebpf/process_details.go`
- **.Distribution()** (2 connections) — `pkg/ebpf/process_details.go`
- **.ProcessGroup()** (2 connections) — `pkg/ebpf/process_details.go`
- **detector.go** (2 connections) — `pkg/detector/detector.go`
- **InstrumentationManagerOptions** (1 connections) — `pkg/ebpf/common.go`
- **K8sConfigGroup** (1 connections) — `pkg/ebpf/process_details.go`
- **K8sProcessGroup** (1 connections) — `pkg/ebpf/process_details.go`

## Relationships

- [[Source Object Docs]] (48 shared connections)
- [[VM Agent Docs]] (2 shared connections)
- [[Odiglet File Copy]] (2 shared connections)
- [[Odiglet Pod Manager]] (2 shared connections)
- [[OSS Destination Docs]] (1 shared connections)
- [[Community 225]] (1 shared connections)
- [[Community 236]] (1 shared connections)
- [[Prometheus-compatible Backend Docs]] (1 shared connections)
- [[Instrumentor Manager]] (1 shared connections)
- [[Odiglet Agent Filesystem Setup]] (1 shared connections)

## Source Files

- `odiglet.go`
- `pkg/detector/detector.go`
- `pkg/ebpf/common.go`
- `pkg/ebpf/process_details.go`
- `pkg/kube/common/common.go`

## Audit Trail

- EXTRACTED: 38 (63%)
- INFERRED: 22 (37%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*