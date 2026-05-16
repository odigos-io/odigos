# MetricsSource Config Schema

> 55 nodes · cohesion 0.07

## Key Concepts

- **.injectOdigos()** (13 connections) — `controllers/agentenabled/pods_webhook.go`
- **env.go** (13 connections) — `controllers/agentenabled/podswebhook/env.go`
- **.injectOdigosToContainer()** (12 connections) — `controllers/agentenabled/pods_webhook.go`
- **webhook_env_injector.go** (10 connections) — `internal/webhook_env_injector/webhook_env_injector.go`
- **InjectConstEnvVarToPodContainer()** (9 connections) — `controllers/agentenabled/podswebhook/env.go`
- **InjectOdigosAgentEnvVars()** (8 connections) — `internal/webhook_env_injector/webhook_env_injector.go`
- **PodsWebhook** (7 connections) — `controllers/agentenabled/pods_webhook.go`
- **otelresource.go** (6 connections) — `controllers/agentenabled/podswebhook/otelresource.go`
- **pods_webhook.go** (5 connections) — `controllers/agentenabled/pods_webhook.go`
- **mount.go** (5 connections) — `controllers/agentenabled/podswebhook/mount.go`
- **InjectOtelResourceAndServiceNameEnvVars()** (5 connections) — `controllers/agentenabled/podswebhook/otelresource.go`
- **handleManifestEnvVar()** (5 connections) — `internal/webhook_env_injector/webhook_env_injector.go`
- **.injectOdigosInstrumentation()** (4 connections) — `controllers/agentenabled/pods_webhook.go`
- **injectNodeIpEnvVar()** (4 connections) — `controllers/agentenabled/podswebhook/env.go`
- **InjectOdigosK8sEnvVars()** (4 connections) — `controllers/agentenabled/podswebhook/env.go`
- **InjectOpampServerEnvVar()** (4 connections) — `controllers/agentenabled/podswebhook/env.go`
- **InjectOtlpHttpEndpointEnvVar()** (4 connections) — `controllers/agentenabled/podswebhook/env.go`
- **InjectSignalsAsStaticOtelEnvVars()** (4 connections) — `controllers/agentenabled/podswebhook/env.go`
- **InjectStaticEnvVarsToPodContainer()** (4 connections) — `controllers/agentenabled/podswebhook/env.go`
- **InjectUserEnvForLang()** (4 connections) — `controllers/agentenabled/podswebhook/env.go`
- **mountPodVolumeIfNotExists()** (4 connections) — `controllers/agentenabled/podswebhook/mount.go`
- **getResourceAttributes()** (4 connections) — `controllers/agentenabled/podswebhook/otelresource.go`
- **getContainerEnvVarPointer()** (4 connections) — `internal/webhook_env_injector/webhook_env_injector.go`
- **injectEnvVarsFromRuntime()** (4 connections) — `internal/webhook_env_injector/webhook_env_injector.go`
- **createInitContainer()** (3 connections) — `controllers/agentenabled/pods_webhook.go`
- *... and 30 more nodes in this community*

## Relationships

- [[Pod Details GraphQL]] (210 shared connections)
- [[CLI Kube Client]] (2 shared connections)
- [[Community 235]] (1 shared connections)

## Source Files

- `controllers/agentenabled/pods_webhook.go`
- `controllers/agentenabled/podswebhook/device.go`
- `controllers/agentenabled/podswebhook/env.go`
- `controllers/agentenabled/podswebhook/mount.go`
- `controllers/agentenabled/podswebhook/otelresource.go`
- `internal/pod/pod.go`
- `internal/webhook_env_injector/webhook_env_injector.go`

## Audit Trail

- EXTRACTED: 175 (82%)
- INFERRED: 38 (18%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*