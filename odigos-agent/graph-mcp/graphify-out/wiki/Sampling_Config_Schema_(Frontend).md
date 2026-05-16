# Sampling Config Schema (Frontend)

> 30 nodes · cohesion 0.12

## Key Concepts

- **SamplingConfig** (15 connections) — `graph/model/models_gen.go`
- **.fieldContext_Sampling_configs()** (13 connections) — `graph/generated.go`
- **.fieldContext_SamplingConfig_k8sHealthProbesSampling()** (8 connections) — `graph/generated.go`
- **.fieldContext_SamplingConfig_spanSamplingAttributes()** (8 connections) — `graph/generated.go`
- **K8sHealthProbesSamplingConfig** (8 connections) — `graph/model/models_gen.go`
- **SpanSamplingAttributesConfig** (8 connections) — `graph/model/models_gen.go`
- **.marshalOSamplingConfig2ᚖgithubᚗcomᚋodigosᚑioᚋodigosᚋfrontendᚋgraphᚋmodelᚐSamplingConfig()** (7 connections) — `graph/generated.go`
- **TailSamplingConfig** (7 connections) — `graph/model/models_gen.go`
- **.fieldContext_SamplingConfigs_localUiConfig()** (5 connections) — `graph/generated.go`
- **.fieldContext_SamplingConfigs_remoteConfigFromCentral()** (5 connections) — `graph/generated.go`
- **.fieldContext_SpanSamplingAttributesConfig_disabled()** (4 connections) — `graph/generated.go`
- **._K8sHealthProbesSamplingConfig_keepPercentage()** (4 connections) — `graph/generated.go`
- **._SamplingConfig_k8sHealthProbesSampling()** (4 connections) — `graph/generated.go`
- **._SamplingConfig_spanSamplingAttributes()** (4 connections) — `graph/generated.go`
- **._SamplingConfig_tailSampling()** (4 connections) — `graph/generated.go`
- **._SamplingConfigs_effective()** (4 connections) — `graph/generated.go`
- **._SamplingConfigs_helmDeployment()** (4 connections) — `graph/generated.go`
- **._SamplingConfigs_localUiConfig()** (4 connections) — `graph/generated.go`
- **._SamplingConfigs_remoteConfigFromCentral()** (4 connections) — `graph/generated.go`
- **._SpanSamplingAttributesConfig_samplingCategoryDisabled()** (4 connections) — `graph/generated.go`
- **._SpanSamplingAttributesConfig_spanDecisionAttributesDisabled()** (4 connections) — `graph/generated.go`
- **.fieldContext_K8sHealthProbesSamplingConfig_enabled()** (3 connections) — `graph/generated.go`
- **.fieldContext_K8sHealthProbesSamplingConfig_keepPercentage()** (3 connections) — `graph/generated.go`
- **.fieldContext_SpanSamplingAttributesConfig_samplingCategoryDisabled()** (3 connections) — `graph/generated.go`
- **.fieldContext_TailSamplingConfig_disabled()** (3 connections) — `graph/generated.go`
- *... and 5 more nodes in this community*

## Relationships

- [[GraphQL Marshalers (Frontend)]] (30 shared connections)
- [[Odigos Collector Processor Catalog]] (7 shared connections)
- [[Odigos CRD Informers (api)]] (6 shared connections)
- [[GraphQL Mutation Schema]] (4 shared connections)
- [[CLI Centralized Install]] (4 shared connections)
- [[MetricsSource Config Schema]] (1 shared connections)
- [[Service Graph Connector]] (1 shared connections)

## Source Files

- `graph/generated.go`
- `graph/model/models_gen.go`

## Audit Trail

- EXTRACTED: 155 (100%)
- INFERRED: 0 (0%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*