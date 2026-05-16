# Autoscaler ConfigMap Sync

> 17 nodes · cohesion 0.15

## Key Concepts

- **.Reconcile()** (10 connections) — `pkg/kube/loglevel/reconciler.go`
- **InstrumentationConfigReconciler** (6 connections) — `pkg/kube/runtime_details/instrumentationconfigs_controller.go`
- **instrumentationConfigPredicate** (5 connections) — `pkg/kube/runtime_details/instrumentationconfigs_controller.go`
- **.sendInstrumentationRequest()** (4 connections) — `pkg/kube/instrumentation_ebpf/instrumentationconfig.go`
- **instrumentationconfig.go** (4 connections) — `pkg/kube/instrumentation_ebpf/instrumentationconfig.go`
- **consumerBusyError** (2 connections) — `pkg/kube/instrumentation_ebpf/instrumentationconfig.go`
- **resolveErrReconcileResult()** (2 connections) — `pkg/kube/instrumentation_ebpf/instrumentationconfig.go`
- **.sendConfigUpdates()** (2 connections) — `pkg/kube/instrumentation_ebpf/instrumentationconfig.go`
- **.sendUnInstrumentationRequest()** (2 connections) — `pkg/kube/instrumentation_ebpf/instrumentationconfig.go`
- **instrumentationconfigs_controller.go** (2 connections) — `pkg/kube/runtime_details/instrumentationconfigs_controller.go`
- **pods_controller.go** (2 connections) — `pkg/kube/runtime_details/pods_controller.go`
- **PodsReconciler** (2 connections) — `pkg/kube/runtime_details/pods_controller.go`
- **requestType** (1 connections) — `pkg/kube/instrumentation_ebpf/instrumentationconfig.go`
- **.Delete()** (1 connections) — `pkg/kube/runtime_details/instrumentationconfigs_controller.go`
- **.Generic()** (1 connections) — `pkg/kube/runtime_details/instrumentationconfigs_controller.go`
- **.Update()** (1 connections) — `pkg/kube/runtime_details/instrumentationconfigs_controller.go`
- **InstrumentationConfigContainsUnknownLanguage()** (1 connections) — `pkg/kube/runtime_details/pods_controller.go`

## Relationships

- [[VM Agent Docs]] (40 shared connections)
- [[OSS Destination Docs]] (3 shared connections)
- [[Source Object Docs]] (2 shared connections)
- [[Instrumentation Flow Docs]] (1 shared connections)
- [[Community 236]] (1 shared connections)
- [[Instrumentor Manager]] (1 shared connections)

## Source Files

- `pkg/kube/instrumentation_ebpf/instrumentationconfig.go`
- `pkg/kube/loglevel/reconciler.go`
- `pkg/kube/runtime_details/instrumentationconfigs_controller.go`
- `pkg/kube/runtime_details/pods_controller.go`

## Audit Trail

- EXTRACTED: 43 (90%)
- INFERRED: 5 (10%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*