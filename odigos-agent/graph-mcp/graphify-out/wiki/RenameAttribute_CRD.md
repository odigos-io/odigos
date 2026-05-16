# RenameAttribute CRD

> 39 nodes · cohesion 0.09

## Key Concepts

- **manager.go** (11 connections) — `controllers/manager.go`
- **New()** (9 connections) — `instrumentor.go`
- **.Create()** (7 connections) — `controllers/utils/predicates/runtime_details_changed.go`
- **.Delete()** (7 connections) — `controllers/utils/predicates/runtime_details_changed.go`
- **.Update()** (7 connections) — `controllers/utils/predicates/runtime_details_changed.go`
- **.Generic()** (6 connections) — `controllers/utils/predicates/runtime_details_changed.go`
- **InstrumentationConfigPodsInjectionPredicate** (5 connections) — `controllers/podsinjectionstatus/manager.go`
- **AgentInjectionEnabledActionsPredicate** (5 connections) — `controllers/utils/predicates/actions.go`
- **AgentInjectionRelevantRulesPredicate** (5 connections) — `controllers/utils/predicates/instrumentation_rule.go`
- **ContainerOverridesChangedPredicate** (5 connections) — `controllers/utils/predicates/runtime_details_changed.go`
- **RecoveredFromRollbackAtChangedPredicate** (5 connections) — `controllers/utils/predicates/runtime_details_changed.go`
- **RuntimeDetailsChangedPredicate** (5 connections) — `controllers/utils/predicates/runtime_details_changed.go`
- **instrumentationrules.go** (4 connections) — `controllers/utils/instrumentationrules.go`
- **.Run()** (4 connections) — `instrumentor.go`
- **isRuleRelevantForAgentInjection()** (4 connections) — `controllers/utils/predicates/instrumentation_rule.go`
- **events.go** (4 connections) — `report/events.go`
- **Start()** (4 connections) — `report/events.go`
- **main()** (3 connections) — `cmd/main.go`
- **CreateManager()** (3 connections) — `controllers/manager.go`
- **SetupWithManager()** (3 connections) — `controllers/manager.go`
- **runtime_details_changed.go** (3 connections) — `controllers/utils/predicates/runtime_details_changed.go`
- **instrumentor.go** (3 connections) — `instrumentor.go`
- **PodsTracker** (3 connections) — `controllers/podsinjectionstatus/podstracker.go`
- **reportEvent()** (3 connections) — `report/events.go`
- **generateUUIDNamespace()** (3 connections) — `internal/testutil/mocks.go`
- *... and 14 more nodes in this community*

## Relationships

- [[Instrumentor CRUD Predicates]] (67 shared connections)
- [[CLI Kube Client]] (65 shared connections)
- [[Auto-Instrumentation Docs]] (4 shared connections)
- [[Pod Details GraphQL]] (2 shared connections)
- [[Rollout Backoff Logic]] (1 shared connections)
- [[Autoscaler Deployment Sync]] (1 shared connections)
- [[Odiglet Runtime Inspection]] (1 shared connections)

## Source Files

- `cmd/main.go`
- `controllers/manager.go`
- `controllers/podsinjectionstatus/manager.go`
- `controllers/podsinjectionstatus/podstracker.go`
- `controllers/sourceinstrumentation/manager.go`
- `controllers/utils/instrumentationrules.go`
- `controllers/utils/predicates/actions.go`
- `controllers/utils/predicates/instrumentation_rule.go`
- `controllers/utils/predicates/runtime_details_changed.go`
- `instrumentor.go`
- `internal/testutil/mocks.go`
- `report/events.go`

## Audit Trail

- EXTRACTED: 121 (86%)
- INFERRED: 20 (14%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*