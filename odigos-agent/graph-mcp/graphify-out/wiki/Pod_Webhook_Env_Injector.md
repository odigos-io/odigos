# Pod Webhook Env Injector

> 57 nodes · cohesion 0.07

## Key Concepts

- **NewTrace()** (25 connections) — `processors/odigossamplingprocessor/internal/sampling/testutil/tracefactory.go`
- **NewRuleEngine()** (11 connections) — `processors/odigossamplingprocessor/rule_engine.go`
- **WithAttribute()** (9 connections) — `processors/odigossamplingprocessor/internal/sampling/testutil/tracefactory.go`
- **WithLatency()** (9 connections) — `processors/odigossamplingprocessor/internal/sampling/testutil/tracefactory.go`
- **rule_engine_test.go** (8 connections) — `processors/odigossamplingprocessor/rule_engine_test.go`
- **buildTrace()** (8 connections) — `processors/odigossamplingprocessor/internal/sampling/spanattribute_test.go`
- **servicename.go** (7 connections) — `processors/odigossamplingprocessor/internal/sampling/servicename.go`
- **spanattribute_test.go** (7 connections) — `processors/odigossamplingprocessor/internal/sampling/spanattribute_test.go`
- **tracefactory.go** (7 connections) — `processors/odigossamplingprocessor/internal/sampling/testutil/tracefactory.go`
- **error_test.go** (6 connections) — `processors/odigossamplingprocessor/internal/sampling/error_test.go`
- **TestRuleEngine_EndpointOverridesGlobal()** (5 connections) — `processors/odigossamplingprocessor/rule_engine_test.go`
- **TestRuleEngine_LatencyRuleSatisfied()** (5 connections) — `processors/odigossamplingprocessor/rule_engine_test.go`
- **TestRuleEngine_MultipleEndpointRules_OneSatisfied()** (5 connections) — `processors/odigossamplingprocessor/rule_engine_test.go`
- **RuleEngine** (5 connections) — `processors/odigossamplingprocessor/rule_engine.go`
- **latency_test.go** (5 connections) — `processors/odigossamplingprocessor/internal/sampling/latency_test.go`
- **TestRuleEngine_PreferHigherLevelSatisfied()** (4 connections) — `processors/odigossamplingprocessor/rule_engine_test.go`
- **TestHttpRouteLatencyRule_Evaluate()** (4 connections) — `processors/odigossamplingprocessor/internal/sampling/latency_test.go`
- **TestHttpRouteLatencyRule_Evaluate_EndpointMismatch()** (4 connections) — `processors/odigossamplingprocessor/internal/sampling/latency_test.go`
- **TestHttpRouteLatencyRule_Evaluate_LatencyEqualsThreshold()** (4 connections) — `processors/odigossamplingprocessor/internal/sampling/latency_test.go`
- **TestHttpRouteLatencyRule_Evaluate_NoMatchingServiceOrEndpoint()** (4 connections) — `processors/odigossamplingprocessor/internal/sampling/latency_test.go`
- **TestHttpRouteLatencyRule_Evaluate_ServiceMismatch()** (4 connections) — `processors/odigossamplingprocessor/internal/sampling/latency_test.go`
- **TraceBuilder** (4 connections) — `processors/odigossamplingprocessor/internal/sampling/testutil/tracefactory.go`
- **WithStatus()** (4 connections) — `processors/odigossamplingprocessor/internal/sampling/testutil/tracefactory.go`
- **TestRuleEngine_EmptyRules()** (3 connections) — `processors/odigossamplingprocessor/rule_engine_test.go`
- **TestRuleEngine_ErrorRuleFallbackOnly()** (3 connections) — `processors/odigossamplingprocessor/rule_engine_test.go`
- *... and 32 more nodes in this community*

## Relationships

- [[Pyroscope Profiling Conversion]] (218 shared connections)
- [[Instrumentor Rollout Mocks]] (1 shared connections)
- [[RenameAttribute CRD]] (1 shared connections)
- [[Cypress E2E Tests]] (1 shared connections)

## Source Files

- `processors/odigossamplingprocessor/internal/sampling/error_test.go`
- `processors/odigossamplingprocessor/internal/sampling/latency_test.go`
- `processors/odigossamplingprocessor/internal/sampling/servicename.go`
- `processors/odigossamplingprocessor/internal/sampling/servicename_test.go`
- `processors/odigossamplingprocessor/internal/sampling/spanattribute_test.go`
- `processors/odigossamplingprocessor/internal/sampling/testutil/tracefactory.go`
- `processors/odigossamplingprocessor/rule_engine.go`
- `processors/odigossamplingprocessor/rule_engine_test.go`

## Audit Trail

- EXTRACTED: 118 (53%)
- INFERRED: 103 (47%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*