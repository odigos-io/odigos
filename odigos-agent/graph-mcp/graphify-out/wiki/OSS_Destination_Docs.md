# OSS Destination Docs

> 23 nodes · cohesion 0.11

## Key Concepts

- **utils.go** (48 connections) — `graph/utils.go`
- **GetDataCollectorContainerMetrics()** (6 connections) — `services/metrics/collector_metrics.go`
- **ConvertConditions()** (5 connections) — `services/utils.go`
- **ResourceAttributesToSourceID()** (4 connections) — `services/common/utils.go`
- **RESTART_WORKLOADS** (4 connections) — `webapp/graphql/mutations/source.ts`
- **TransformConditionStatus()** (4 connections) — `services/utils.go`
- **buildPodRegex()** (3 connections) — `services/metrics/utils.go`
- **RolloutRestartWorkload()** (3 connections) — `services/sources.go`
- **DerefK8sResourceKind()** (3 connections) — `services/utils.go`
- **K8sResourceKindPtrIfNotEmpty()** (3 connections) — `services/utils.go`
- **SamplingWorkloadLanguagePtrIfNotEmpty()** (3 connections) — `services/utils.go`
- **workloadKindFromMixedCaseString()** (2 connections) — `services/common/utils.go`
- **maxTime()** (2 connections) — `services/metrics/utils.go`
- **queryVector()** (2 connections) — `services/metrics/utils.go`
- **rateSumByPod()** (2 connections) — `services/metrics/utils.go`
- **regexpEscape()** (2 connections) — `services/metrics/utils.go`
- **ConvertSignals()** (2 connections) — `services/utils.go`
- **DerefProgrammingLanguage()** (2 connections) — `services/utils.go`
- **ProgrammingLanguagePtrIfNotEmpty()** (2 connections) — `services/utils.go`
- **WithGoroutine()** (2 connections) — `services/utils.go`
- **getKubeVersion()** (1 connections) — `services/utils.go`
- **GetPageLimit()** (1 connections) — `services/utils.go`
- **k8sLastTransitionTimeToGql()** (1 connections) — `services/utils.go`

## Relationships

- [[Config YAML Field Schema]] (59 shared connections)
- [[Service Graph Connector]] (7 shared connections)
- [[Frontend Layout & Providers]] (5 shared connections)
- [[Pod Webhook Env Injector]] (5 shared connections)
- [[CLI Centralized Install]] (4 shared connections)
- [[URL Template Processor Tests]] (3 shared connections)
- [[Sampling Rule Apply Configs (api)]] (3 shared connections)
- [[Action GraphQL Schema]] (3 shared connections)
- [[Community 230]] (2 shared connections)
- [[Frontend Destination Connection Test]] (2 shared connections)
- [[ServiceMap GraphQL]] (2 shared connections)
- [[Retry & OTLP Exporter Config]] (2 shared connections)

## Source Files

- `graph/utils.go`
- `services/common/utils.go`
- `services/metrics/collector_metrics.go`
- `services/metrics/utils.go`
- `services/sources.go`
- `services/utils.go`
- `webapp/graphql/mutations/source.ts`

## Audit Trail

- EXTRACTED: 77 (72%)
- INFERRED: 30 (28%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*