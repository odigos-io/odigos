# URL Templatization Rule GraphQL

> 26 nodes · cohesion 0.10

## Key Concepts

- **reconcileAll()** (23 connections) — `controllers/agentenabled/sync.go`
- **workload_controllers.go** (6 connections) — `controllers/sourceinstrumentation/workload_controllers.go`
- **InstrumentationRuleReconciler** (3 connections) — `controllers/agentenabled/instrumentationrule_controller.go`
- **ActionReconciler** (2 connections) — `controllers/agentenabled/action_controller.go`
- **CollectorsGroupReconciler** (2 connections) — `controllers/agentenabled/collectorsgroup_controller.go`
- **EffectiveConfigReconciler** (2 connections) — `controllers/agentenabled/effectiveconfig_controller.go`
- **InstrumentationConfigReconciler** (2 connections) — `controllers/agentenabled/instrumentationconfig_controller.go`
- **.reportRuleValidationStatus()** (2 connections) — `controllers/instrumentationconfig/instrumentationrule_controller.go`
- **InstrumentationConfigController** (2 connections) — `controllers/podsinjectionstatus/instrumentationconfig_controller.go`
- **PodsController** (2 connections) — `controllers/podsinjectionstatus/pods_controller.go`
- **.handleDeletedPod()** (2 connections) — `controllers/podsinjectionstatus/pods_controller.go`
- **CronJobReconciler** (2 connections) — `controllers/sourceinstrumentation/workload_controllers.go`
- **DaemonSetReconciler** (2 connections) — `controllers/sourceinstrumentation/workload_controllers.go`
- **DeploymentConfigReconciler** (2 connections) — `controllers/sourceinstrumentation/workload_controllers.go`
- **DeploymentReconciler** (2 connections) — `controllers/sourceinstrumentation/workload_controllers.go`
- **NamespaceReconciler** (2 connections) — `controllers/sourceinstrumentation/namespace_controller.go`
- **RolloutReconciler** (2 connections) — `controllers/sourceinstrumentation/workload_controllers.go`
- **SourceReconciler** (2 connections) — `controllers/sourceinstrumentation/source_controller.go`
- **StatefulSetReconciler** (2 connections) — `controllers/sourceinstrumentation/workload_controllers.go`
- **SamplingController** (1 connections) — `controllers/agentenabled/sampling_controller.go`
- **action_controller.go** (1 connections) — `controllers/agentenabled/action_controller.go`
- **collectorsgroup_controller.go** (1 connections) — `controllers/agentenabled/collectorsgroup_controller.go`
- **effectiveconfig_controller.go** (1 connections) — `controllers/agentenabled/effectiveconfig_controller.go`
- **instrumentationrule_controller.go** (1 connections) — `controllers/agentenabled/instrumentationrule_controller.go`
- **namespace_controller.go** (1 connections) — `controllers/sourceinstrumentation/namespace_controller.go`
- *... and 1 more nodes in this community*

## Relationships

- [[Destination & Processor CRDs]] (66 shared connections)
- [[Odiglet Runtime Inspection]] (2 shared connections)
- [[Autoscaler Collector Config Domains]] (2 shared connections)
- [[Sources CLI Docs]] (1 shared connections)

## Source Files

- `controllers/agentenabled/action_controller.go`
- `controllers/agentenabled/collectorsgroup_controller.go`
- `controllers/agentenabled/effectiveconfig_controller.go`
- `controllers/agentenabled/instrumentationconfig_controller.go`
- `controllers/agentenabled/instrumentationrule_controller.go`
- `controllers/agentenabled/sampling_controller.go`
- `controllers/agentenabled/sync.go`
- `controllers/instrumentationconfig/instrumentationrule_controller.go`
- `controllers/podsinjectionstatus/instrumentationconfig_controller.go`
- `controllers/podsinjectionstatus/pods_controller.go`
- `controllers/sourceinstrumentation/namespace_controller.go`
- `controllers/sourceinstrumentation/source_controller.go`
- `controllers/sourceinstrumentation/workload_controllers.go`

## Audit Trail

- EXTRACTED: 67 (94%)
- INFERRED: 4 (6%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*