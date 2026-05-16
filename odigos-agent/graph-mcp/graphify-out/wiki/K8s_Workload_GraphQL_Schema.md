# K8s Workload GraphQL Schema

> 139 nodes · cohesion 0.03

## Key Concepts

- **.defaultInformer()** (38 connections) — `generated/odigos/informers/externalversions/odigos/v1alpha1/action.go`
- **.Lister()** (37 connections) — `generated/odigos/informers/externalversions/odigos/v1alpha1/action.go`
- **SpanAttributeSampler** (29 connections) — `actions/v1alpha1/spanattributesampler_types.go`
- **Destination** (28 connections) — `odigos/v1alpha1/destination_types.go`
- **AddClusterInfo** (25 connections) — `actions/v1alpha1/addclusterinfo_types.go`
- **Processor** (25 connections) — `odigos/v1alpha1/processor_types.go`
- **ProbabilisticSampler** (24 connections)
- **InstrumentationRule** (24 connections) — `odigos/v1alpha1/instrumentationrule_type.go`
- **.DeepCopyObject()** (23 connections) — `odigos/v1alpha1/zz_generated.deepcopy.go`
- **LatencySampler** (23 connections) — `actions/v1alpha1/latencysampler_types.go`
- **ServiceNameSampler** (23 connections) — `actions/v1alpha1/servicenamesampler_types.go`
- **PiiMasking** (22 connections)
- **.Informer()** (22 connections) — `generated/odigos/informers/externalversions/odigos/v1alpha1/action.go`
- **ErrorSampler** (21 connections)
- **version** (19 connections) — `generated/odigos/informers/externalversions/odigos/v1alpha1/interface.go`
- **K8sAttributesResolver** (18 connections)
- **Action** (18 connections) — `odigos/v1alpha1/action_types.go`
- **FakeActionsV1alpha1** (12 connections) — `generated/actions/clientset/versioned/typed/actions/v1alpha1/fake/fake_actions_client.go`
- **ActionsV1alpha1Client** (12 connections) — `generated/actions/clientset/versioned/typed/actions/v1alpha1/actions_client.go`
- **action.go** (11 connections) — `generated/odigos/listers/odigos/v1alpha1/action.go`
- **FakeOdigosV1alpha1** (10 connections) — `generated/odigos/clientset/versioned/typed/odigos/v1alpha1/fake/fake_odigos_client.go`
- **OdigosV1alpha1Client** (10 connections) — `generated/odigos/clientset/versioned/typed/odigos/v1alpha1/odigos_client.go`
- **ProcessorList** (7 connections) — `odigos/v1alpha1/processor_types.go`
- **processors** (7 connections) — `generated/odigos/clientset/versioned/typed/odigos/v1alpha1/processor.go`
- **ActionList** (6 connections) — `odigos/v1alpha1/action_types.go`
- *... and 114 more nodes in this community*

## Relationships

- [[Entity Analyze GraphQL]] (531 shared connections)
- [[CRD DeepCopy Generated (api)]] (62 shared connections)
- [[InstrumentationConfig CRD]] (26 shared connections)
- [[Collector Buffer Cache]] (13 shared connections)
- [[Workload Instrumentation Update]] (9 shared connections)
- [[Source CRD (api)]] (6 shared connections)
- [[Autoscaler Reconcilers]] (2 shared connections)
- [[RolloutConcurrencyLimiter Tests]] (2 shared connections)

## Source Files

- `actions/v1alpha1/addclusterinfo_types.go`
- `actions/v1alpha1/latencysampler_types.go`
- `actions/v1alpha1/servicenamesampler_types.go`
- `actions/v1alpha1/spanattributesampler_types.go`
- `config/crd/bases/actions.odigos.io_piimaskings.yaml`
- `config/crd/bases/odigos.io_destinations.yaml`
- `config/crd/bases/odigos.io_processors.yaml`
- `generated/actions/applyconfiguration/actions/v1alpha1/piimaskingstatus.go`
- `generated/actions/clientset/versioned/typed/actions/v1alpha1/actions_client.go`
- `generated/actions/clientset/versioned/typed/actions/v1alpha1/addclusterinfo.go`
- `generated/actions/clientset/versioned/typed/actions/v1alpha1/errorsampler.go`
- `generated/actions/clientset/versioned/typed/actions/v1alpha1/fake/fake_actions_client.go`
- `generated/actions/clientset/versioned/typed/actions/v1alpha1/fake/fake_addclusterinfo.go`
- `generated/actions/clientset/versioned/typed/actions/v1alpha1/fake/fake_errorsampler.go`
- `generated/actions/clientset/versioned/typed/actions/v1alpha1/fake/fake_k8sattributesresolver.go`
- `generated/actions/clientset/versioned/typed/actions/v1alpha1/fake/fake_latencysampler.go`
- `generated/actions/clientset/versioned/typed/actions/v1alpha1/fake/fake_piimasking.go`
- `generated/actions/clientset/versioned/typed/actions/v1alpha1/fake/fake_probabilisticsampler.go`
- `generated/actions/clientset/versioned/typed/actions/v1alpha1/fake/fake_servicenamesampler.go`
- `generated/actions/clientset/versioned/typed/actions/v1alpha1/fake/fake_spanattributesampler.go`

## Audit Trail

- EXTRACTED: 649 (89%)
- INFERRED: 77 (11%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*