# Collector Workload Info GraphQL

> 49 nodes · cohesion 0.04

## Key Concepts

- **InstrumentationConfig** (57 connections)
- **fakeInstrumentationInstances** (7 connections) — `generated/odigos/clientset/versioned/typed/odigos/v1alpha1/fake/fake_instrumentationinstance.go`
- **Attribute** (4 connections) — `odigos/v1alpha1/zz_generated.deepcopy.go`
- **HeadersCollectionConfig** (4 connections) — `odigos/v1alpha1/instrumentationconfig_types.go`
- **IdGeneratorConfig** (4 connections) — `odigos/v1alpha1/instrumentationconfig_types.go`
- **IdGeneratorTimedWallConfig** (4 connections) — `odigos/v1alpha1/instrumentationconfig_types.go`
- **PodsManifestInjectionStatus** (4 connections) — `odigos/v1alpha1/instrumentationconfig_types.go`
- **PodsManifestInjectionStatusApplyConfiguration** (4 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/podsmanifestinjectionstatus.go`
- **SpanRenamerScopeRules** (4 connections) — `odigos/v1alpha1/instrumentationconfig_types.go`
- **AgentDisabledInfo** (3 connections) — `odigos/v1alpha1/instrumentationconfig_types.go`
- **AttributeApplyConfiguration** (3 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/attribute.go`
- **EnvVarApplyConfiguration** (3 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/envvar.go`
- **IdGeneratorConfigApplyConfiguration** (3 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/idgeneratorconfig.go`
- **IdGeneratorRandomConfig** (3 connections) — `odigos/v1alpha1/instrumentationconfig_types.go`
- **NewFilteredInstrumentationConfigInformer()** (3 connections) — `generated/odigos/informers/externalversions/odigos/v1alpha1/instrumentationconfig.go`
- **NewFilteredInstrumentationInstanceInformer()** (3 connections) — `generated/odigos/informers/externalversions/odigos/v1alpha1/instrumentationinstance.go`
- **SpanRenamerScopeConfig** (3 connections) — `odigos/v1alpha1/instrumentationconfig_types.go`
- **SpanRenamerScopeRulesApplyConfiguration** (3 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/spanrenamerscoperules.go`
- **envvar.go** (2 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/envvar.go`
- **.WithValue()** (2 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/envvar.go`
- **HeadersCollectionConfigApplyConfiguration** (2 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/headerscollectionconfig.go`
- **IdGeneratorTimedWallConfigApplyConfiguration** (2 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/idgeneratortimedwallconfig.go`
- **AgentEnabledReason enum** (1 connections)
- **fakeInstrumentationConfigs** (1 connections) — `generated/odigos/clientset/versioned/typed/odigos/v1alpha1/fake/fake_instrumentationconfig.go`
- **fakeInstrumentationRules** (1 connections) — `generated/odigos/clientset/versioned/typed/odigos/v1alpha1/fake/fake_instrumentationrule.go`
- *... and 24 more nodes in this community*

## Relationships

- [[K8s Workload GraphQL Schema]] (116 shared connections)
- [[Entity Analyze GraphQL]] (24 shared connections)
- [[CRD DeepCopy Generated (api)]] (6 shared connections)
- [[Action Resolver Logic]] (3 shared connections)
- [[Container Agent Config CRD]] (2 shared connections)
- [[Frontend GQL Conversion (Describe)]] (2 shared connections)
- [[InstrumentationConfig CRD]] (1 shared connections)

## Source Files

- `generated/odigos/applyconfiguration/odigos/v1alpha1/attribute.go`
- `generated/odigos/applyconfiguration/odigos/v1alpha1/envvar.go`
- `generated/odigos/applyconfiguration/odigos/v1alpha1/headerscollectionconfig.go`
- `generated/odigos/applyconfiguration/odigos/v1alpha1/idgeneratorconfig.go`
- `generated/odigos/applyconfiguration/odigos/v1alpha1/idgeneratortimedwallconfig.go`
- `generated/odigos/applyconfiguration/odigos/v1alpha1/podsmanifestinjectionstatus.go`
- `generated/odigos/applyconfiguration/odigos/v1alpha1/spanrenamerscoperules.go`
- `generated/odigos/clientset/versioned/typed/odigos/v1alpha1/fake/fake_instrumentationconfig.go`
- `generated/odigos/clientset/versioned/typed/odigos/v1alpha1/fake/fake_instrumentationinstance.go`
- `generated/odigos/clientset/versioned/typed/odigos/v1alpha1/fake/fake_instrumentationrule.go`
- `generated/odigos/clientset/versioned/typed/odigos/v1alpha1/instrumentationconfig.go`
- `generated/odigos/informers/externalversions/odigos/v1alpha1/instrumentationconfig.go`
- `generated/odigos/informers/externalversions/odigos/v1alpha1/instrumentationinstance.go`
- `odigos/v1alpha1/instrumentationconfig_types.go`
- `odigos/v1alpha1/zz_generated.deepcopy.go`

## Audit Trail

- EXTRACTED: 150 (97%)
- INFERRED: 4 (3%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*