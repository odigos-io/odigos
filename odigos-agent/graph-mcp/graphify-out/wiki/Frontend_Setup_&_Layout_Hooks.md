# Frontend Setup & Layout Hooks

> 19 nodes · cohesion 0.17

## Key Concepts

- **Clientset** (10 connections) — `generated/odigos/clientset/versioned/fake/clientset_generated.go`
- **actions_client.go** (6 connections) — `generated/actions/clientset/versioned/typed/actions/v1alpha1/actions_client.go`
- **odigos_client.go** (6 connections) — `generated/odigos/clientset/versioned/typed/odigos/v1alpha1/odigos_client.go`
- **v1alpha1** (6 connections) — `config/crd/bases/actions.odigos.io_deleteattributes.yaml`
- **New()** (5 connections) — `generated/odigos/informers/externalversions/odigos/interface.go`
- **NewForConfig()** (5 connections) — `generated/odigos/clientset/versioned/clientset.go`
- **NewForConfigAndClient()** (5 connections) — `generated/odigos/clientset/versioned/clientset.go`
- **Interface** (4 connections) — `generated/odigos/informers/externalversions/odigos/interface.go`
- **setConfigDefaults()** (4 connections) — `generated/odigos/clientset/versioned/typed/odigos/v1alpha1/odigos_client.go`
- **clientset_generated.go** (3 connections) — `generated/odigos/clientset/versioned/fake/clientset_generated.go`
- **.OdigosV1alpha1()** (2 connections) — `generated/odigos/clientset/versioned/fake/clientset_generated.go`
- **group** (2 connections) — `generated/odigos/informers/externalversions/odigos/interface.go`
- **.ActionsV1alpha1()** (1 connections) — `generated/actions/clientset/versioned/fake/clientset_generated.go`
- **.Discovery()** (1 connections) — `generated/odigos/clientset/versioned/fake/clientset_generated.go`
- **NewClientset()** (1 connections) — `generated/odigos/clientset/versioned/fake/clientset_generated.go`
- **NewSimpleClientset()** (1 connections) — `generated/odigos/clientset/versioned/fake/clientset_generated.go`
- **.IsWatchListSemanticsUnSupported()** (1 connections) — `generated/odigos/clientset/versioned/fake/clientset_generated.go`
- **.Tracker()** (1 connections) — `generated/odigos/clientset/versioned/fake/clientset_generated.go`
- **ActionsV1alpha1Interface** (1 connections) — `generated/actions/clientset/versioned/typed/actions/v1alpha1/actions_client.go`

## Relationships

- [[Source CRD (api)]] (58 shared connections)
- [[Entity Analyze GraphQL]] (6 shared connections)
- [[InstrumentationConfig CRD]] (1 shared connections)

## Source Files

- `config/crd/bases/actions.odigos.io_deleteattributes.yaml`
- `generated/actions/clientset/versioned/fake/clientset_generated.go`
- `generated/actions/clientset/versioned/typed/actions/v1alpha1/actions_client.go`
- `generated/odigos/clientset/versioned/clientset.go`
- `generated/odigos/clientset/versioned/fake/clientset_generated.go`
- `generated/odigos/clientset/versioned/typed/odigos/v1alpha1/odigos_client.go`
- `generated/odigos/informers/externalversions/odigos/interface.go`

## Audit Trail

- EXTRACTED: 65 (100%)
- INFERRED: 0 (0%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*