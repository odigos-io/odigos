# Collector Client gRPC Config

> 39 nodes · cohesion 0.08

## Key Concepts

- **local_ui_config.go** (12 connections) — `services/local_ui_config.go`
- **RemoteConfig** (9 connections) — `graph/model/models_gen.go`
- **odigos_config.go** (8 connections) — `services/odigos_config.go`
- **ComputeProvenance()** (7 connections) — `services/provenance.go`
- **SamplingConfigsResolver** (6 connections) — `graph/generated.go`
- **applyLocalUiConfigInput()** (6 connections) — `services/local_ui_config.go`
- **getOdigosConfigFromConfigMap()** (6 connections) — `services/odigos_config.go`
- **UPDATE_LOCAL_UI_CONFIG** (5 connections) — `webapp/graphql/mutations/config.ts`
- **GET_EFFECTIVE_CONFIG** (5 connections) — `webapp/graphql/queries/config.ts`
- **GetRemoteConfig()** (5 connections) — `services/odigos_config.go`
- **ResolveProfilingFromEffectiveConfig()** (5 connections) — `services/profiling.go`
- **provenance.go** (5 connections) — `services/provenance.go`
- **StartProfilingConfigWatcher()** (5 connections) — `kube/watchers/profiling_config_watcher.go`
- **SetComponentLogLevel()** (4 connections) — `services/log_level.go`
- **GetHelmDeploymentConfig()** (4 connections) — `services/odigos_config.go`
- **detectProfileProvenance()** (4 connections) — `services/provenance.go`
- **.Effective()** (3 connections) — `graph/sampling.resolvers.go`
- **.HelmDeployment()** (3 connections) — `graph/sampling.resolvers.go`
- **.RemoteConfigFromCentral()** (3 connections) — `graph/sampling.resolvers.go`
- **RESET_LOCAL_UI_CONFIG_TO_FACTORY_DEFAULTS** (3 connections) — `webapp/graphql/mutations/config.ts`
- **StoreLimitsFromEnv()** (3 connections) — `services/profiles/env.go`
- **applyComponentLogLevelsInput()** (3 connections) — `services/local_ui_config.go`
- **createLocalUiConfigMap()** (3 connections) — `services/local_ui_config.go`
- **OdigosConfigurationFromConfigMap()** (3 connections) — `services/odigos_config.go`
- **PersistUiLocalSamplingConfig()** (3 connections) — `services/odigos_config.go`
- *... and 14 more nodes in this community*

## Relationships

- [[JVM Metrics Handler]] (125 shared connections)
- [[Odigos Collector Processor Catalog]] (6 shared connections)
- [[Quickstart & Sources Docs]] (5 shared connections)
- [[Service Graph Connector]] (5 shared connections)
- [[GraphQL Mutation Schema]] (3 shared connections)
- [[CLI Centralized Install]] (3 shared connections)
- [[Collector Generated Telemetry]] (2 shared connections)
- [[Frontend GraphQL Loaders]] (2 shared connections)
- [[GraphQL Marshalers (Frontend)]] (1 shared connections)

## Source Files

- `graph/configs.resolvers.go`
- `graph/conversions.go`
- `graph/generated.go`
- `graph/model/models_gen.go`
- `graph/sampling.resolvers.go`
- `kube/watchers/profiling_config_watcher.go`
- `services/local_ui_config.go`
- `services/log_level.go`
- `services/odigos_config.go`
- `services/profiles/env.go`
- `services/profiling.go`
- `services/provenance.go`
- `webapp/graphql/mutations/config.ts`
- `webapp/graphql/queries/config.ts`

## Audit Trail

- EXTRACTED: 116 (76%)
- INFERRED: 36 (24%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*