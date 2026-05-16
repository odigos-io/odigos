# Common Test Helpers

> 29 nodes · cohesion 0.07

## Key Concepts

- **InstrumentationLibraryConfig** (20 connections) — `odigos/v1alpha1/instrumentationconfig_types.go`
- **AgentTracesConfigApplyConfiguration** (10 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/agenttracesconfig.go`
- **SdkConfigApplyConfiguration** (10 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/sdkconfig.go`
- **AgentTracesConfig** (4 connections) — `odigos/v1alpha1/instrumentationconfig_types.go`
- **SdkConfig** (4 connections) — `odigos/v1alpha1/instrumentationconfig_types.go`
- **.WithCodeAttributes()** (3 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/agenttracesconfig.go`
- **.WithHeadersCollection()** (3 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/agenttracesconfig.go`
- **.WithCustomInstrumentations()** (3 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/sdkconfig.go`
- **.WithEbpfLogCapture()** (3 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/sdkconfig.go`
- **.WithLanguage()** (3 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/sdkconfig.go`
- **.WithSpanRenamer()** (2 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/actionspec.go`
- **.WithURLTemplatization()** (2 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/actionspec.go`
- **AgentLogsConfigApplyConfiguration** (2 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/agentlogsconfig.go`
- **.WithTraceVerbosity()** (2 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/agenttracesconfig.go`
- **.WithHealthy()** (2 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/instrumentationlibrarystatus.go`
- **.WithIdentifyingAttributes()** (2 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/instrumentationlibrarystatus.go`
- **.WithLastStatusTime()** (2 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/instrumentationlibrarystatus.go`
- **.WithMessage()** (2 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/instrumentationlibrarystatus.go`
- **.WithReason()** (2 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/instrumentationlibrarystatus.go`
- **.WithTraceConfig()** (2 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/instrumentationrulespec.go`
- **.WithHeadSampling()** (1 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/agenttracesconfig.go`
- **.WithIdGenerator()** (1 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/agenttracesconfig.go`
- **.WithEnabled()** (1 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/instrumentationlibraryconfigtraces.go`
- **.WithSpanKind()** (1 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/instrumentationlibraryid.go`
- **.WithDefaultCodeAttributes()** (1 connections) — `generated/odigos/applyconfiguration/odigos/v1alpha1/sdkconfig.go`
- *... and 4 more nodes in this community*

## Relationships

- [[Action Resolver Logic]] (61 shared connections)
- [[CRD DeepCopy Generated (api)]] (15 shared connections)
- [[K8s Workload GraphQL Schema]] (14 shared connections)
- [[Frontend GQL Conversion (Describe)]] (1 shared connections)
- [[Container Agent Config CRD]] (1 shared connections)

## Source Files

- `generated/odigos/applyconfiguration/odigos/v1alpha1/actionspec.go`
- `generated/odigos/applyconfiguration/odigos/v1alpha1/agentlogsconfig.go`
- `generated/odigos/applyconfiguration/odigos/v1alpha1/agenttracesconfig.go`
- `generated/odigos/applyconfiguration/odigos/v1alpha1/instrumentationlibraryconfigtraces.go`
- `generated/odigos/applyconfiguration/odigos/v1alpha1/instrumentationlibraryid.go`
- `generated/odigos/applyconfiguration/odigos/v1alpha1/instrumentationlibrarystatus.go`
- `generated/odigos/applyconfiguration/odigos/v1alpha1/instrumentationrulespec.go`
- `generated/odigos/applyconfiguration/odigos/v1alpha1/sdkconfig.go`
- `odigos/v1alpha1/instrumentationconfig_types.go`

## Audit Trail

- EXTRACTED: 92 (100%)
- INFERRED: 0 (0%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*