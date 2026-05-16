# VM Agent Docs

> 17 nodes · cohesion 0.15

## Key Concepts

- **manager.go** (6 connections) — `controllers/manager.go`
- **init()** (5 connections) — `cmd/main.go`
- **main()** (5 connections) — `cmd/main.go`
- **SetupWithManager()** (4 connections) — `controllers/actions/root.go`
- **root.go** (4 connections) — `k8sconfig/root.go`
- **main.go** (3 connections) — `cmd/main.go`
- **CreateManager()** (3 connections) — `controllers/manager.go`
- **ProcessorReconciler** (3 connections) — `controllers/nodecollector/processor_controller.go`
- **RegisterWebhooks()** (2 connections) — `controllers/actions/root.go`
- **isRunningOnGKE()** (2 connections) — `cmd/main.go`
- **reconciler.go** (2 connections) — `controllers/loglevel/reconciler.go`
- **durationPointer()** (2 connections) — `controllers/manager.go`
- **LoadK8sConfigers()** (2 connections) — `k8sconfig/root.go`
- **LogLevelReconciler** (2 connections) — `controllers/loglevel/reconciler.go`
- **KubeManagerOptions** (1 connections) — `controllers/manager.go`
- **processor_controller.go** (1 connections) — `controllers/nodecollector/processor_controller.go`
- **K8sConfiger** (1 connections) — `k8sconfig/root.go`

## Relationships

- [[Enterprise Installation Docs]] (42 shared connections)
- [[Destination CR Docs]] (2 shared connections)
- [[Community 201]] (1 shared connections)
- [[Autoscaler Resource Detection]] (1 shared connections)
- [[Community 200]] (1 shared connections)
- [[Autoscaler Sampler Handlers]] (1 shared connections)

## Source Files

- `cmd/main.go`
- `controllers/actions/root.go`
- `controllers/loglevel/reconciler.go`
- `controllers/manager.go`
- `controllers/nodecollector/processor_controller.go`
- `k8sconfig/root.go`

## Audit Trail

- EXTRACTED: 44 (92%)
- INFERRED: 4 (8%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*