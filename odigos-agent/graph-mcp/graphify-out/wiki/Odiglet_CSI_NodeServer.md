# Odiglet CSI NodeServer

> 12 nodes · cohesion 0.21

## Key Concepts

- **CalculateAgentInjectedStatus()** (10 connections) — `graph/status/agentInjected.go`
- **agentEnabledContainersToModel()** (7 connections) — `graph/utils.go`
- **.AgentEnabled()** (4 connections) — `graph/workload.resolvers.go`
- **agentInjected.go** (4 connections) — `graph/status/agentInjected.go`
- **.PodsAgentInjectionStatus()** (3 connections) — `graph/workload.resolvers.go`
- **agentInjectionEnabled.go** (2 connections) — `graph/status/agentInjectionEnabled.go`
- **distroParamsToModel()** (2 connections) — `graph/utils.go`
- **emptyStrToNil()** (2 connections) — `graph/utils.go`
- **CalculatePodAgentInjectedStatus()** (2 connections) — `graph/status/agentInjected.go`
- **agentEnabledStatusCondition()** (2 connections) — `graph/status/agentInjectionEnabled.go`
- **PodAgentInjectedReason** (2 connections) — `graph/status/agentInjected.go`
- **AgentInjectionReason** (1 connections) — `graph/status/agentInjected.go`

## Relationships

- [[Frontend Layout & Providers]] (30 shared connections)
- [[Autoscaler ConfigMap Sync]] (5 shared connections)
- [[Config YAML Field Schema]] (3 shared connections)
- [[ServiceMap GraphQL]] (2 shared connections)
- [[Community 219]] (1 shared connections)

## Source Files

- `graph/status/agentInjected.go`
- `graph/status/agentInjectionEnabled.go`
- `graph/utils.go`
- `graph/workload.resolvers.go`

## Audit Trail

- EXTRACTED: 25 (61%)
- INFERRED: 16 (39%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*