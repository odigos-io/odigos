# Autoscaler Reconcilers

> 18 nodes · cohesion 0.19

## Key Concepts

- **ProfilingPipelineConfig()** (8 connections) — `controllers/nodecollector/collectorconfig/profiles.go`
- **MergeProfilingOtlpExporter()** (7 connections) — `controllers/common/profiling_otlp.go`
- **profiling_otlp.go** (6 connections) — `controllers/common/profiling_otlp.go`
- **ProfilingFilterProcessorConfig()** (5 connections) — `controllers/common/profiling_otlp.go`
- **profiling_otlp_test.go** (5 connections) — `controllers/common/profiling_otlp_test.go`
- **ProfilingProfileDropConditions()** (4 connections) — `controllers/common/profiling_otlp.go`
- **TestProfilingPipelineConfig_Enabled()** (3 connections) — `controllers/nodecollector/collectorconfig/profiles_test.go`
- **TestProfilingFilterProcessorConfig()** (3 connections) — `controllers/common/profiling_otlp_test.go`
- **TestProfilingPipelineConfig_Disabled()** (2 connections) — `controllers/nodecollector/collectorconfig/profiles_test.go`
- **cloneGenericMap()** (2 connections) — `controllers/common/profiling_otlp.go`
- **K8sAttributesProfilesProcessorConfig()** (2 connections) — `controllers/common/profiling_otlp.go`
- **ProfilingServiceNameTransformConfig()** (2 connections) — `controllers/common/profiling_otlp.go`
- **TestMergeProfilingOtlpExporter_NilOtlp()** (2 connections) — `controllers/common/profiling_otlp_test.go`
- **TestMergeProfilingOtlpExporter_TimeoutAndRetry()** (2 connections) — `controllers/common/profiling_otlp_test.go`
- **TestMergeProfilingOtlpExporter_WithOtlp_DoesNotMutateBase()** (2 connections) — `controllers/common/profiling_otlp_test.go`
- **TestProfilingProfileDropConditions()** (2 connections) — `controllers/common/profiling_otlp_test.go`
- **profiles_test.go** (2 connections) — `controllers/nodecollector/collectorconfig/profiles_test.go`
- **profiles.go** (1 connections) — `controllers/nodecollector/collectorconfig/profiles.go`

## Relationships

- [[Frontend Setup & Layout Hooks]] (58 shared connections)
- [[URL Templatization Rule GraphQL]] (1 shared connections)
- [[Community 274]] (1 shared connections)

## Source Files

- `controllers/common/profiling_otlp.go`
- `controllers/common/profiling_otlp_test.go`
- `controllers/nodecollector/collectorconfig/profiles.go`
- `controllers/nodecollector/collectorconfig/profiles_test.go`

## Audit Trail

- EXTRACTED: 32 (53%)
- INFERRED: 28 (47%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*