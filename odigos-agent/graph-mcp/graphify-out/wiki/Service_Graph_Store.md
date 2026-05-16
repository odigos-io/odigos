# Service Graph Store

> 23 nodes · cohesion 0.18

## Key Concepts

- **updateInstrumentationConfigForWorkload()** (20 connections) — `controllers/instrumentationconfig/common.go`
- **boolPtr()** (12 connections) — `controllers/agentenabled/dynamicconfig/traces/codeattributes.go`
- **common_test.go** (8 connections) — `controllers/instrumentationconfig/common_test.go`
- **payloadcollection.go** (7 connections) — `controllers/agentenabled/dynamicconfig/traces/payloadcollection.go`
- **mergeDbPayloadCollectionRules()** (6 connections) — `controllers/agentenabled/dynamicconfig/traces/payloadcollection.go`
- **mergeHttpPayloadCollectionRules()** (5 connections) — `controllers/agentenabled/dynamicconfig/traces/payloadcollection.go`
- **mergeMessagingPayloadCollectionRules()** (5 connections) — `controllers/agentenabled/dynamicconfig/traces/payloadcollection.go`
- **mergePayloadCollectionConfigs()** (5 connections) — `controllers/agentenabled/dynamicconfig/traces/payloadcollection.go`
- **Int64Ptr()** (4 connections) — `controllers/instrumentationconfig/common_test.go`
- **TestUpdateInstrumentationConfigForWorkload_MultipleDefaultRules()** (4 connections) — `controllers/instrumentationconfig/common_test.go`
- **TestUpdateInstrumentationConfigForWorkload_SingleMatchingRule()** (4 connections) — `controllers/instrumentationconfig/common_test.go`
- **CalculatePayloadCollectionConfig()** (4 connections) — `controllers/agentenabled/dynamicconfig/traces/payloadcollection.go`
- **mergeDefaultTracingConfig()** (3 connections) — `controllers/instrumentationconfig/common.go`
- **mergeTracingConfig()** (3 connections) — `controllers/instrumentationconfig/common.go`
- **TestMergeHttpPayloadCollectionRules()** (3 connections) — `controllers/instrumentationconfig/common_test.go`
- **createDefaultSdkConfig()** (2 connections) — `controllers/instrumentationconfig/common.go`
- **findOrCreateSdkLibraryConfig()** (2 connections) — `controllers/instrumentationconfig/common.go`
- **mergeHttpHeadersCollectionrules()** (2 connections) — `controllers/instrumentationconfig/common.go`
- **TestUpdateInstrumentationConfigForWorkload_LibraryRuleOtherLanguage()** (2 connections) — `controllers/instrumentationconfig/common_test.go`
- **TestUpdateInstrumentationConfigForWorkload_NoLanguages()** (2 connections) — `controllers/instrumentationconfig/common_test.go`
- **TestUpdateInstrumentationConfigForWorkload_RuleFoOverrideContainer()** (2 connections) — `controllers/instrumentationconfig/common_test.go`
- **DistroSupportsTracesPayloadCollection()** (2 connections) — `controllers/agentenabled/dynamicconfig/traces/payloadcollection.go`
- **mergeDbQuerySanitizationPolicy()** (2 connections) — `controllers/agentenabled/dynamicconfig/traces/payloadcollection.go`

## Relationships

- [[Sources CLI Docs]] (90 shared connections)
- [[Autoscaler Collector Config Domains]] (12 shared connections)
- [[Community 273]] (3 shared connections)
- [[Destination & Processor CRDs]] (1 shared connections)
- [[Community 292]] (1 shared connections)
- [[Auto-Instrumentation Docs]] (1 shared connections)
- [[Docs Generator Functions]] (1 shared connections)

## Source Files

- `controllers/agentenabled/dynamicconfig/traces/codeattributes.go`
- `controllers/agentenabled/dynamicconfig/traces/payloadcollection.go`
- `controllers/instrumentationconfig/common.go`
- `controllers/instrumentationconfig/common_test.go`

## Audit Trail

- EXTRACTED: 96 (88%)
- INFERRED: 13 (12%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*