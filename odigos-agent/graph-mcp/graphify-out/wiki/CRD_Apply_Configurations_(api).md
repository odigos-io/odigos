# CRD Apply Configurations (api)

> 112 nodes · cohesion 0.04

## Key Concepts

- **.DeepCopy()** (110 connections) — `odigos/v1alpha1/zz_generated.deepcopy.go`
- **.DeepCopyInto()** (109 connections) — `odigos/v1alpha1/zz_generated.deepcopy.go`
- **CollectorsGroup** (30 connections) — `odigos/v1alpha1/zz_generated.deepcopy.go`
- **init()** (19 connections) — `generated/odigos/clientset/versioned/fake/register.go`
- **Source** (17 connections) — `odigos/v1alpha1/zz_generated.deepcopy.go`
- **CollectorRole** (11 connections) — `k8sconsts/collectors.go`
- **.OrderHint()** (10 connections) — `odigos/v1alpha1/actions/spanrenamer.go`
- **.ProcessorType()** (10 connections) — `odigos/v1alpha1/actions/spanrenamer.go`
- **source_types.go** (9 connections) — `odigos/v1alpha1/source_types.go`
- **SpanRenamerConfig** (8 connections) — `odigos/v1alpha1/actions/spanrenamer.go`
- **samplers_types.go** (8 connections) — `actions/v1alpha1/samplers_types.go`
- **ExtractAttributeConfig** (7 connections) — `odigos/v1alpha1/actions/extractattribute.go`
- **URLTemplatizationConfig** (7 connections) — `odigos/v1alpha1/actions/urltemplatization.go`
- **PayloadCollection** (7 connections) — `odigos/v1alpha1/instrumentationrules/payloadcollection.go`
- **SourceSpec** (7 connections) — `odigos/v1alpha1/zz_generated.deepcopy.go`
- **action_types.go** (6 connections) — `odigos/v1alpha1/action_types.go`
- **destination_types.go** (6 connections) — `odigos/v1alpha1/destination_types.go`
- **ContainerOverride** (6 connections) — `odigos/v1alpha1/zz_generated.deepcopy.go`
- **DefaultSamplerConfig** (6 connections) — `actions/v1alpha1/samplers_types.go`
- **DeleteAttributeConfig** (6 connections) — `actions/v1alpha1/deleteattribute_types.go`
- **K8sAttributesConfig** (6 connections) — `actions/v1alpha1/k8sattributes_types.go`
- **PiiMaskingConfig** (6 connections) — `actions/v1alpha1/piimasking_types.go`
- **RenameAttributeConfig** (6 connections) — `actions/v1alpha1/renameattribute_types.go`
- **SourceList** (6 connections) — `odigos/v1alpha1/zz_generated.deepcopy.go`
- **SourceStatus** (6 connections) — `odigos/v1alpha1/zz_generated.deepcopy.go`
- *... and 87 more nodes in this community*

## Relationships

- [[K8s Workload GraphQL Schema]] (464 shared connections)
- [[Entity Analyze GraphQL]] (69 shared connections)
- [[Workload Instrumentation Update]] (53 shared connections)
- [[CRD DeepCopy Generated (api)]] (25 shared connections)
- [[Collector Buffer Cache]] (20 shared connections)
- [[Action Resolver Logic]] (11 shared connections)
- [[InstrumentationConfig CRD]] (8 shared connections)
- [[Frontend GQL Conversion (Describe)]] (8 shared connections)
- [[Container Agent Config CRD]] (5 shared connections)
- [[RolloutConcurrencyLimiter Tests]] (5 shared connections)
- [[Community 300]] (1 shared connections)
- [[Community 240]] (1 shared connections)

## Source Files

- `actions/v1alpha1/addclusterinfo_types.go`
- `actions/v1alpha1/deleteattribute_types.go`
- `actions/v1alpha1/errorsampler_types.go`
- `actions/v1alpha1/k8sattributes_types.go`
- `actions/v1alpha1/latencysampler_types.go`
- `actions/v1alpha1/piimasking_types.go`
- `actions/v1alpha1/renameattribute_types.go`
- `actions/v1alpha1/samplers_types.go`
- `actions/v1alpha1/spanattributesampler_types.go`
- `config/crd/bases/odigos.io_sources.yaml`
- `generated/actions/applyconfiguration/actions/v1alpha1/otelattributewithvalue.go`
- `generated/odigos/applyconfiguration/odigos/v1alpha1/actionstatus.go`
- `generated/odigos/applyconfiguration/odigos/v1alpha1/agentlogsconfig.go`
- `generated/odigos/applyconfiguration/odigos/v1alpha1/destinationstatus.go`
- `generated/odigos/applyconfiguration/odigos/v1alpha1/otheragent.go`
- `generated/odigos/applyconfiguration/odigos/v1alpha1/sourceselector.go`
- `generated/odigos/applyconfiguration/odigos/v1alpha1/sourcestatus.go`
- `generated/odigos/applyconfiguration/odigos/v1alpha1/spanrenamerconfig.go`
- `generated/odigos/clientset/versioned/fake/register.go`
- `generated/odigos/clientset/versioned/typed/odigos/v1alpha1/collectorsgroup.go`

## Audit Trail

- EXTRACTED: 662 (99%)
- INFERRED: 8 (1%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*