# Action GraphQL Schema

> 56 nodes · cohesion 0.06

## Key Concepts

- **recover_from_rollback.go** (11 connections) — `services/recover_from_rollback.go`
- **GetGatewayDeploymentInfo()** (9 connections) — `services/collectors/gateway.go`
- **pods.go** (9 connections) — `services/collectors/pods.go`
- **GetOdigletDaemonSetInfo()** (8 connections) — `services/collectors/odiglet.go`
- **pod.go** (7 connections) — `services/pod.go`
- **newScheme()** (7 connections) — `services/recover_from_rollback_test.go`
- **StringPtr()** (7 connections) — `services/utils.go`
- **source.ts** (7 connections) — `webapp/graphql/queries/source.ts`
- **podToInfo()** (6 connections) — `services/collectors/pods.go`
- **extractImageVersionForContainer()** (5 connections) — `services/collectors/helpers.go`
- **buildCollectorContainerOverview()** (5 connections) — `services/collectors/pods.go`
- **GetCollectorPodDetails()** (5 connections) — `services/collectors/pods.go`
- **GetPodDetails()** (5 connections) — `services/pod.go`
- **newFakeClient()** (5 connections) — `services/recover_from_rollback_test.go`
- **computeGatewayHPA()** (4 connections) — `services/collectors/gateway.go`
- **extractResourcesForContainer()** (4 connections) — `services/collectors/helpers.go`
- **GetPodsBySelector()** (4 connections) — `services/collectors/pods.go`
- **gateway.go** (4 connections) — `services/collectors/gateway.go`
- **buildContainerResources()** (4 connections) — `services/pod.go`
- **buildContainersOverview()** (4 connections) — `services/pod.go`
- **mapPodPhase()** (4 connections) — `services/pod.go`
- **newWorkloadSource()** (4 connections) — `services/recover_from_rollback_test.go`
- **TestRecoverFromRollback_AlreadySet()** (4 connections) — `services/recover_from_rollback_test.go`
- **TestRecoverFromRollback_Success()** (4 connections) — `services/recover_from_rollback_test.go`
- **findLastRolloutTime()** (3 connections) — `services/collectors/gateway.go`
- *... and 31 more nodes in this community*

## Relationships

- [[Pod Webhook Env Injector]] (188 shared connections)
- [[Config YAML Field Schema]] (5 shared connections)
- [[Odigos Collector Processor Catalog]] (5 shared connections)
- [[Managed Backend Destination Docs]] (2 shared connections)
- [[Frontend GraphQL Loaders]] (2 shared connections)
- [[Service Graph Connector]] (2 shared connections)
- [[Collector Factories]] (2 shared connections)
- [[CLI Centralized Install]] (1 shared connections)
- [[Autoscaler Manager Main]] (1 shared connections)

## Source Files

- `graph/collectors.resolvers.go`
- `graph/computed/pod.go`
- `graph/pod.resolvers.go`
- `graph/schema.resolvers.go`
- `kube/cache.go`
- `services/collectors/gateway.go`
- `services/collectors/helpers.go`
- `services/collectors/odiglet.go`
- `services/collectors/pods.go`
- `services/collectors/resources.go`
- `services/manifest.go`
- `services/pod.go`
- `services/recover_from_rollback.go`
- `services/recover_from_rollback_test.go`
- `services/utils.go`
- `webapp/graphql/mutations/source.ts`
- `webapp/graphql/queries/source.ts`

## Audit Trail

- EXTRACTED: 145 (70%)
- INFERRED: 63 (30%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*