# Collector Factories

> 33 nodes · cohesion 0.12

## Key Concepts

- **.DeepCopy()** (26 connections) — `zz_generated.deepcopy.go`
- **.DeepCopyInto()** (26 connections) — `zz_generated.deepcopy.go`
- **sampling.go** (5 connections) — `consts/sampling.go`
- **ContainerCollectorConfig** (3 connections) — `api/instrumentationconfig_types.go`
- **configs.go** (3 connections) — `api/sampling/configs.go`
- **matchers_head.go** (3 connections) — `api/sampling/matchers_head.go`
- **matchers_tail.go** (3 connections) — `api/sampling/matchers_tail.go`
- **UrlTemplatizationConfig** (3 connections) — `api/instrumentationconfig_types.go`
- **MetricsSourceConfiguration** (3 connections) — `odigos_config.go`
- **MetricsSourceHostMetricsConfiguration** (3 connections) — `odigos_config.go`
- **MetricsSourceKubeletStatsConfiguration** (3 connections) — `odigos_config.go`
- **MetricsSourceOdigosOwnMetricsConfiguration** (3 connections) — `odigos_config.go`
- **MetricsSourceSpanMetricsConfiguration** (3 connections) — `odigos_config.go`
- **OtlpExporterConfiguration** (3 connections) — `odigos_config.go`
- **ProfilingConfiguration** (3 connections) — `odigos_config.go`
- **ResourceDetectorConfig** (3 connections) — `odigos_config.go`
- **ResourceDetectorsConfiguration** (3 connections) — `odigos_config.go`
- **RetryOnFailure** (3 connections) — `odigos_config.go`
- **CostReductionRule** (3 connections) — `api/sampling/sampling.go`
- **HeadSamplingHttpClientOperationMatcher** (3 connections) — `api/sampling/matchers_head.go`
- **HeadSamplingHttpServerOperationMatcher** (3 connections) — `api/sampling/matchers_head.go`
- **HeadSamplingOperationMatcher** (3 connections) — `api/sampling/matchers_head.go`
- **HighlyRelevantOperation** (3 connections) — `api/sampling/sampling.go`
- **NoisyOperation** (3 connections) — `api/sampling/sampling.go`
- **SpanSamplingAttributesConfiguration** (3 connections) — `api/sampling/configs.go`
- *... and 8 more nodes in this community*

## Relationships

- [[Common Test Helpers]] (120 shared connections)
- [[Instrumentation Rule Merging]] (12 shared connections)
- [[Community 296]] (10 shared connections)

## Source Files

- `api/instrumentationconfig_types.go`
- `api/sampling/configs.go`
- `api/sampling/matchers_head.go`
- `api/sampling/matchers_tail.go`
- `api/sampling/sampling.go`
- `consts/sampling.go`
- `odigos_config.go`
- `zz_generated.deepcopy.go`

## Audit Trail

- EXTRACTED: 142 (100%)
- INFERRED: 0 (0%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*