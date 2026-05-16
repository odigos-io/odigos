# Odiglet eBPF Metrics Collector

> 10 nodes · cohesion 0.36

## Key Concepts

- **sampling.go** (10 connections) — `controllers/agentenabled/dynamicconfig/traces/sampling.go`
- **CalculateSamplingCategoryRulesForContainer()** (6 connections) — `controllers/agentenabled/dynamicconfig/traces/sampling.go`
- **calculateKubeletHealthProbesSamplingRules()** (4 connections) — `controllers/agentenabled/dynamicconfig/traces/sampling.go`
- **calculateKubeletHttpGetProbePaths()** (3 connections) — `controllers/agentenabled/dynamicconfig/traces/sampling.go`
- **addProbePathAndName()** (2 connections) — `controllers/agentenabled/dynamicconfig/traces/sampling.go`
- **calculateK8sHealthProbeSamplingPercentage()** (2 connections) — `controllers/agentenabled/dynamicconfig/traces/sampling.go`
- **getPercentageOrZero()** (2 connections) — `controllers/agentenabled/dynamicconfig/traces/sampling.go`
- **isK8sHealthProbesSamplingEnabled()** (2 connections) — `controllers/agentenabled/dynamicconfig/traces/sampling.go`
- **isServiceInRuleScope()** (2 connections) — `controllers/agentenabled/dynamicconfig/traces/sampling.go`
- **kubeletProbePathAndName** (1 connections) — `controllers/agentenabled/dynamicconfig/traces/sampling.go`

## Relationships

- [[Pro-Dep CLI Page Docs]] (32 shared connections)
- [[Docs Generator Functions]] (2 shared connections)

## Source Files

- `controllers/agentenabled/dynamicconfig/traces/sampling.go`

## Audit Trail

- EXTRACTED: 33 (97%)
- INFERRED: 1 (3%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*