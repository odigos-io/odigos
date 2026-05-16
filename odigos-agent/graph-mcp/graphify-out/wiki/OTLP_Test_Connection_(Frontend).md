# OTLP Test Connection (Frontend)

> 12 nodes · cohesion 0.33

## Key Concepts

- **ResolveDistroForContainer()** (7 connections) — `controllers/agentenabled/distroresolver/distroresolver.go`
- **distroresolver_test.go** (6 connections) — `controllers/agentenabled/distroresolver/distroresolver_test.go`
- **mustNewCommunityGetter()** (6 connections) — `controllers/agentenabled/distroresolver/distroresolver_test.go`
- **distroresolver.go** (4 connections) — `controllers/agentenabled/distroresolver/distroresolver.go`
- **CalculateDefaultDistroPerLanguage()** (4 connections) — `controllers/agentenabled/distroresolver/distroresolver.go`
- **TestCalculateDefaultDistroPerLanguage_mixedRuleWithOBIAndJavaStillMapsJava()** (3 connections) — `controllers/agentenabled/distroresolver/distroresolver_test.go`
- **TestCalculateDefaultDistroPerLanguage_skipsWildcardDistroFromRules()** (3 connections) — `controllers/agentenabled/distroresolver/distroresolver_test.go`
- **TestResolveDistroForContainer_nonWildcardEnforcesRuntimeSemver()** (3 connections) — `controllers/agentenabled/distroresolver/distroresolver_test.go`
- **TestResolveDistroForContainer_wildcardDistroSkipsRuntimeSemver()** (3 connections) — `controllers/agentenabled/distroresolver/distroresolver_test.go`
- **TestResolveDistroForContainer_wildcardOverrideAcceptsMismatchedContainerLanguage()** (3 connections) — `controllers/agentenabled/distroresolver/distroresolver_test.go`
- **resolveDistroByLanguage()** (2 connections) — `controllers/agentenabled/distroresolver/distroresolver.go`
- **resolveDistroByOverride()** (2 connections) — `controllers/agentenabled/distroresolver/distroresolver.go`

## Relationships

- [[Architecture Overview Docs]] (44 shared connections)
- [[Odiglet Runtime Inspection]] (2 shared connections)

## Source Files

- `controllers/agentenabled/distroresolver/distroresolver.go`
- `controllers/agentenabled/distroresolver/distroresolver_test.go`

## Audit Trail

- EXTRACTED: 34 (74%)
- INFERRED: 12 (26%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*