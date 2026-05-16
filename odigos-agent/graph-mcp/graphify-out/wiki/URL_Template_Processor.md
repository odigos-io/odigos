# URL Template Processor

> 21 nodes · cohesion 0.10

## Key Concepts

- **.Reconcile()** (25 connections) — `controllers/actions/action_controller.go`
- **ActionReconciler** (4 connections) — `controllers/actions/action_controller.go`
- **InstrumentationConfigReconciler** (3 connections) — `controllers/nodecollector/instrumentationconfig_controller.go`
- **.reportReconciledToProcessor()** (2 connections) — `controllers/actions/action_controller.go`
- **SharedURLTemplatizationProcessorReconciler** (2 connections) — `controllers/actions/sharedprocessor_controller.go`
- **ClusterCollectorDeploymentReconciler** (2 connections) — `controllers/clustercollector/gatewaydeployment_controller.go`
- **DestinationReconciler** (2 connections) — `controllers/clustercollector/destination_controller.go`
- **SecretReconciler** (2 connections) — `controllers/clustercollector/secret_controller.go`
- **CAUpdaterReconciler** (2 connections) — `controllers/metricshandler/cacync_controller.go`
- **AutoscalerDeploymentReconciler** (2 connections) — `controllers/nodecollector/autoscaler_deployment_controller.go`
- **CollectorsGroupReconciler** (2 connections) — `controllers/nodecollector/collectorsgroup_controller.go`
- **.reportProcessorNotRequired()** (1 connections) — `controllers/actions/action_controller.go`
- **sharedprocessor_controller.go** (1 connections) — `controllers/actions/sharedprocessor_controller.go`
- **destination_controller.go** (1 connections) — `controllers/clustercollector/destination_controller.go`
- **gatewaydeployment_controller.go** (1 connections) — `controllers/clustercollector/gatewaydeployment_controller.go`
- **secret_controller.go** (1 connections) — `controllers/clustercollector/secret_controller.go`
- **source_controller.go** (1 connections) — `controllers/clustercollector/source_controller.go`
- **cacync_controller.go** (1 connections) — `controllers/metricshandler/cacync_controller.go`
- **autoscaler_deployment_controller.go** (1 connections) — `controllers/nodecollector/autoscaler_deployment_controller.go`
- **collectorsgroup_controller.go** (1 connections) — `controllers/nodecollector/collectorsgroup_controller.go`
- **instrumentationconfig_controller.go** (1 connections) — `controllers/nodecollector/instrumentationconfig_controller.go`

## Relationships

- [[Destination CR Docs]] (39 shared connections)
- [[Autoscaler Collector Group Sync]] (8 shared connections)
- [[Community 320]] (5 shared connections)
- [[Enterprise Installation Docs]] (2 shared connections)
- [[Instrumentor Assertions Helpers]] (1 shared connections)
- [[Community 200]] (1 shared connections)
- [[Community 202]] (1 shared connections)
- [[Community 295]] (1 shared connections)

## Source Files

- `controllers/actions/action_controller.go`
- `controllers/actions/sharedprocessor_controller.go`
- `controllers/clustercollector/destination_controller.go`
- `controllers/clustercollector/gatewaydeployment_controller.go`
- `controllers/clustercollector/secret_controller.go`
- `controllers/clustercollector/source_controller.go`
- `controllers/metricshandler/cacync_controller.go`
- `controllers/nodecollector/autoscaler_deployment_controller.go`
- `controllers/nodecollector/collectorsgroup_controller.go`
- `controllers/nodecollector/instrumentationconfig_controller.go`

## Audit Trail

- EXTRACTED: 55 (95%)
- INFERRED: 3 (5%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*