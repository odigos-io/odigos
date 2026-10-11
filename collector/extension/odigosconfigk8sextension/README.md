# Odigos Config K8s Extension

This extension runs a Kubernetes dynamic informer for `InstrumentationConfig` resources and maintains a cache of per-workload collector configuration (e.g. sampling, URL templatization) keyed by workload (namespace, kind, name, container). Other collector components (processors, connectors) can use it to look up configuration for each resource at runtime.

## Interface

The extension implements the Odigos config extension interface defined in `github.com/odigos-io/odigos/common/collector`:

```go
type OdigosConfigExtension interface {
    // Given a resource, return the container collector config for that workload if it exists.
    GetFromResource(res pcommon.Resource) (*commonapi.ContainerCollectorConfig, bool)
}
```

`ContainerCollectorConfig` is defined in `github.com/odigos-io/odigos/common/api` and includes fields such as `TailSampling`, `UrlTemplatization`, and `ContainerName`.

## How other processors (or connectors) can use it

### 1. Ensure the extension is in the collector config

The pipeline config must register this extension by name (e.g. `odigos_config_k8s`) and add it to `service.extensions`. When using Odigos gateway config generation, set `GatewayConfigOptions.OdigosConfigExtensionName` to the extension’s component type so the generated config includes it.

### 2. Resolve the extension in `Start`

In your processor’s (or connector’s) `Start` method, get the extension from the host. The extension’s component type is available as `odigosconfigk8sextension.Type` (e.g. `odigos_config_k8s`). The instance name in the config may be the type with no name, so look up by type or by the same ID used in the config.

```go
import (
    "go.opentelemetry.io/collector/component"
    "github.com/odigos-io/odigos/common/collector"
    odigosconfigk8sextension "github.com/odigos-io/odigos/collector/extension/odigosconfigk8sextension"
)

type myProcessor struct {
    logger   *zap.Logger
    odigosConfig collector.OdigosConfigExtension // may be nil if extension not configured
}

func (p *myProcessor) Start(ctx context.Context, host component.Host) error {
    // Extension ID must match how it is declared in the collector config (e.g. "odigos_config_k8s" or "odigos_config_k8s/myinstance").
    extID := component.NewID(odigosconfigk8sextension.Type)
    exts := host.GetExtensions()
    if exts == nil {
        return nil
    }
    ext, ok := exts[extID]
    if !ok {
        // Try without name in case config uses type only
        for id, e := range exts {
            if id.Type() == odigosconfigk8sextension.Type {
                ext = e
                break
            }
        }
    }
    if ext != nil {
        if odigosExt, ok := ext.(collector.OdigosConfigExtension); ok {
            p.odigosConfig = odigosExt
        }
    }
    return nil
}
```

If your processor config explicitly stores the extension name/ID from the pipeline, use that to look up `exts[thatID]` instead.

### 3. Wait for the informer cache to sync (optional but recommended)

The extension starts the Kubernetes informer without blocking on initial list/watch sync. Until the cache has synced, `GetFromResource` may miss existing `InstrumentationConfig` resources. To avoid using an empty cache on the first batches, dependents can call `WaitForCacheSync` on the extension’s informer.

`WaitForCacheSync` is not part of the `OdigosConfigExtension` interface; it is on the concrete type `*odigosconfigk8sextension.OdigosWorkloadConfig`. After resolving the extension as in step 2, type-assert to that type and call `WaitForCacheSync` with a context (e.g. a short timeout or the collector’s lifetime context). It returns `true` if the cache synced successfully, and `false` if the context was cancelled or the extension is not running in-cluster.

Example: wait for sync in a goroutine at `Start` so the collector does not block, then set a “ready” flag for your processor:

```go
import (
    "context"
    "sync/atomic"

    "go.opentelemetry.io/collector/component"
    "github.com/odigos-io/odigos/common/collector"
    odigosconfigk8sextension "github.com/odigos-io/odigos/collector/extension/odigosconfigk8sextension"
)

type myProcessor struct {
    logger       *zap.Logger
    odigosConfig collector.OdigosConfigExtension
    cacheReady   atomic.Bool
}

func (p *myProcessor) Start(ctx context.Context, host component.Host) error {
    // ... resolve ext and set p.odigosConfig as in step 2 ...

    if k8sExt, ok := p.odigosConfig.(*odigosconfigk8sextension.OdigosWorkloadConfig); ok {
        go func() {
            if k8sExt.WaitForCacheSync(ctx) {
                p.cacheReady.Store(true)
            }
        }()
    } else {
        p.cacheReady.Store(true) // no K8s informer, treat as ready
    }
    return nil
}
```

Alternatively, call `WaitForCacheSync` once (e.g. with a timeout) before the first time you rely on `GetFromResource` in a critical path.

### 4. Use the config when processing telemetry

For each resource (e.g. in a trace or metric batch), call `GetFromResource` with that resource. Use the returned config to drive per-workload behavior (e.g. sampling rules, URL templatization rules).

```go
import (
    "go.opentelemetry.io/collector/pdata/pcommon"
    commonapi "github.com/odigos-io/odigos/common/api"
)

func (p *myProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
    rss := td.ResourceSpans()
    for i := 0; i < rss.Len(); i++ {
        rs := rss.At(i)
        res := rs.Resource()

        var workloadConfig *commonapi.ContainerCollectorConfig
        if p.odigosConfig != nil {
            if cfg, ok := p.odigosConfig.GetFromResource(res); ok {
                workloadConfig = cfg
            }
        }

        // Use workloadConfig (e.g. workloadConfig.TailSampling, workloadConfig.UrlTemplatization)
        // to apply per-workload behavior when processing rs...
    }
    return td, nil
}
```

### 5. Handle missing or optional extension

- If the extension is not in the config, `host.GetExtensions()` may not contain it; keep `p.odigosConfig` as `nil` and skip per-workload lookups.
- `GetFromResource` returns `(nil, false)` when the resource does not identify a known workload or when there is no config for that workload; processors should fall back to their default or static config in that case.

## Extension type and config

- **Type:** `odigos_config_k8s` (from `internal/metadata`).
- **Config:** The extension accepts an empty config; it discovers `InstrumentationConfig` resources via the in-cluster Kubernetes client.

## Resource attributes used for lookup

The K8s implementation builds a cache key from resource attributes such as:

- `k8s.namespace.name`
- `k8s.deployment.name` / `k8s.statefulset.name` / `k8s.daemonset.name` / `k8s.job.name` / `k8s.cronjob.name` / `k8s.argoproj.rollout.name`
- `k8s.container.name`

If your telemetry does not carry these attributes (e.g. non-K8s environment), `GetFromResource` will return `(nil, false)`.
