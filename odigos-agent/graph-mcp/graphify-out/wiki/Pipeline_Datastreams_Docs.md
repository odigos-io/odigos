# Pipeline Datastreams Docs

> 10 nodes · cohesion 0.20

## Key Concepts

- **.createMigratedAction()** (8 connections) — `controllers/actions/piimasking_controller.go`
- **DeleteAttributeReconciler** (3 connections) — `controllers/actions/deleteattribute_controller.go`
- **K8sAttributesResolverReconciler** (3 connections) — `controllers/actions/k8sattributesresolver_controller.go`
- **PiiMaskingReconciler** (3 connections) — `controllers/actions/piimasking_controller.go`
- **ProbabilisticSamplerReconciler** (3 connections) — `controllers/actions/probabilisticsampler_controller.go`
- **deleteAttributeConfig()** (2 connections) — `controllers/actions/deleteattribute_controller.go`
- **deleteattribute_controller.go** (2 connections) — `controllers/actions/deleteattribute_controller.go`
- **piimasking_controller.go** (2 connections) — `controllers/actions/piimasking_controller.go`
- **PiiMaskingConfig** (1 connections) — `controllers/actions/piimasking_controller.go`
- **ProbabilisticSamplerConfig** (1 connections) — `controllers/actions/probabilisticsampler_controller.go`

## Relationships

- [[Destination CR Docs]] (23 shared connections)
- [[Autoscaler Collector Group Sync]] (4 shared connections)
- [[Instrumentation Rule Docs]] (1 shared connections)

## Source Files

- `controllers/actions/deleteattribute_controller.go`
- `controllers/actions/k8sattributesresolver_controller.go`
- `controllers/actions/piimasking_controller.go`
- `controllers/actions/probabilisticsampler_controller.go`

## Audit Trail

- EXTRACTED: 27 (96%)
- INFERRED: 1 (4%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*