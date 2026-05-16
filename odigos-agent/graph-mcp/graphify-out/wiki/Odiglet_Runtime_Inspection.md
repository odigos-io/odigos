# Odiglet Runtime Inspection

> 20 nodes · cohesion 0.33

## Key Concepts

- **NewMockTestDeployment()** (46 connections) — `internal/testutil/mocks.go`
- **NewMockNamespace()** (13 connections) — `internal/testutil/mocks.go`
- **PodWorkloadFromDeployment()** (10 connections) — `internal/testutil/mocks.go`
- **instrumentationrules_test.go** (9 connections) — `controllers/utils/instrumentationrules_test.go`
- **NewMockInstrumentationRuleWithSourcesScope()** (9 connections) — `internal/testutil/mocks.go`
- **IsInstrumentationConfigParticipatingInRule()** (8 connections) — `controllers/utils/instrumentationrules.go`
- **Test_IsWorkloadParticipatingInRule_SourcesScopeOnly()** (8 connections) — `controllers/utils/instrumentationrules_test.go`
- **Test_IsInstrumentationConfigParticipatingInRule_AllContainersMiss()** (7 connections) — `controllers/utils/instrumentationrules_test.go`
- **Test_IsInstrumentationConfigParticipatingInRule_NoContainerOverrides_Fallback()** (7 connections) — `controllers/utils/instrumentationrules_test.go`
- **Test_IsInstrumentationConfigParticipatingInRule_NoOverrides_StatusRuntime_ContainerScopeMiss()** (7 connections) — `controllers/utils/instrumentationrules_test.go`
- **IsWorkloadParticipatingInRule()** (6 connections) — `controllers/utils/instrumentationrules.go`
- **Test_IsInstrumentationConfigParticipatingInRule_AnyContainerMatches()** (6 connections) — `controllers/utils/instrumentationrules_test.go`
- **Test_IsInstrumentationConfigParticipatingInRule_ContainerNameInScope()** (6 connections) — `controllers/utils/instrumentationrules_test.go`
- **Test_IsWorkloadParticipatingInRule_BothFields_ScopeMatch_IgnoresWorkloads()** (6 connections) — `controllers/utils/instrumentationrules_test.go`
- **Test_IsWorkloadParticipatingInRule_SourcesScope_PartialNamespace_Miss()** (6 connections) — `controllers/utils/instrumentationrules_test.go`
- **Test_IsWorkloadParticipatingInRule_WorkloadsOnly()** (6 connections) — `controllers/utils/instrumentationrules_test.go`
- **NewMockEmptyInstrumentationRule()** (5 connections) — `internal/testutil/mocks.go`
- **NewMockInstrumentationRuleAllWorkloads()** (4 connections) — `internal/testutil/mocks.go`
- **NewMockInstrumentationRuleWithSourcesScopeAndWorkloads()** (4 connections) — `internal/testutil/mocks.go`
- **NewMockInstrumentationRuleDisabled()** (3 connections) — `internal/testutil/mocks.go`

## Relationships

- [[Auto-Instrumentation Docs]] (132 shared connections)
- [[Frontend Generated Models]] (33 shared connections)
- [[Rollout Backoff Logic]] (4 shared connections)
- [[Autoscaler Deployment Sync]] (3 shared connections)
- [[Instrumentor CRUD Predicates]] (2 shared connections)
- [[CLI Kube Client]] (1 shared connections)
- [[Sources CLI Docs]] (1 shared connections)

## Source Files

- `controllers/utils/instrumentationrules.go`
- `controllers/utils/instrumentationrules_test.go`
- `internal/testutil/mocks.go`

## Audit Trail

- EXTRACTED: 39 (22%)
- INFERRED: 137 (78%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*