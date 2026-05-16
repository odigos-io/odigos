# Instrumentor Manager

> 25 nodes · cohesion 0.08

## Key Concepts

- **ContainerAgentConfigApplyConfiguration** (12 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/containeragentconfig.go`
- **RuntimeDetailsByContainerApplyConfiguration** (11 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/runtimedetailsbycontainer.go`
- **ContainerAgentConfig** (4 connections) — `odigos/v1alpha1/instrumentationconfig_types.go`
- **ContainerOverrideApplyConfiguration** (4 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/containeroverride.go`
- **.WithContainerName()** (4 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/containeroverride.go`
- **RuntimeDetailsByContainer** (4 connections) — `odigos/v1alpha1/instrumentationconfig_types.go`
- **.WithOtelDistroName()** (3 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/containeroverride.go`
- **.WithMetrics()** (2 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/collectorsgroupspec.go`
- **.WithPodManifestInjectionOptional()** (2 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/containeragentconfig.go`
- **.WithAgentEnabled()** (1 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/containeragentconfig.go`
- **.WithAgentEnabledMessage()** (1 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/containeragentconfig.go`
- **.WithAgentEnabledReason()** (1 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/containeragentconfig.go`
- **.WithDistroParams()** (1 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/containeragentconfig.go`
- **.WithEnvInjectionMethod()** (1 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/containeragentconfig.go`
- **.WithLogs()** (1 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/containeragentconfig.go`
- **.WithTraces()** (1 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/containeragentconfig.go`
- **.WithRuntimeInfo()** (1 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/containeroverride.go`
- **.WithCriErrorMessage()** (1 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/runtimedetailsbycontainer.go`
- **.WithEnvFromContainerRuntime()** (1 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/runtimedetailsbycontainer.go`
- **.WithEnvVars()** (1 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/runtimedetailsbycontainer.go`
- **.WithLibCType()** (1 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/runtimedetailsbycontainer.go`
- **.WithOtherAgent()** (1 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/runtimedetailsbycontainer.go`
- **.WithRuntimeUpdateState()** (1 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/runtimedetailsbycontainer.go`
- **.WithRuntimeVersion()** (1 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/runtimedetailsbycontainer.go`
- **.WithSecureExecutionMode()** (1 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/runtimedetailsbycontainer.go`

## Relationships

- [[Container Agent Config CRD]] (50 shared connections)
- [[K8s Workload GraphQL Schema]] (6 shared connections)
- [[CRD DeepCopy Generated (api)]] (4 shared connections)
- [[Workload Instrumentation Update]] (1 shared connections)
- [[Action Resolver Logic]] (1 shared connections)

## Source Files

- `generated/odigos/applyconfiguration/odigos/v1alpha1/collectorsgroupspec.go`
- `generated/odigos/applyconfiguration/odigos/v1alpha1/containeragentconfig.go`
- `generated/odigos/applyconfiguration/odigos/v1alpha1/containeroverride.go`
- `generated/odigos/applyconfiguration/odigos/v1alpha1/runtimedetailsbycontainer.go`
- `odigos/v1alpha1/instrumentationconfig_types.go`

## Audit Trail

- EXTRACTED: 62 (100%)
- INFERRED: 0 (0%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*