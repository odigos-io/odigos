# Odigos Workload Config Extension

An OpenTelemetry Collector extension that watches Kubernetes InstrumentationConfig custom resources and exposes a cache of workload-level sampling configuration. Other collector components (processors, connectors, etc.) can use this extension to look up config by workload identity derived from resource attributes.

## Overview

- **Starts a dynamic informer** for `InstrumentationConfig` resources (odigos.io/v1alpha1) when running in-cluster.
- **Maintains a cache** of workload sampling config keyed by `WorkloadKey` (namespace, kind, name). The extension derives the key from each InstrumentationConfig’s metadata (object name format: `<kind>-<name>`, e.g. `deployment-myapp`).
- **Exposes a read API** so processors and other components can look up config for the workload associated with incoming telemetry (e.g. using resource attributes like `k8s.namespace.name`, `k8s.deployment.name`).

When not running in a cluster (e.g. local dev), the extension still starts; the informer is skipped and the cache remains empty.

## Configuration

The extension has no configuration options. Add it under `extensions` and reference it from your pipeline:

```yaml
extensions:
  odigos_workload_config:

service:
  extensions: [odigos_workload_config]
  pipelines:
    traces:
      receivers: [otlp]
      processors: [your_processor]
      exporters: [otlp]
```

## Using the extension from other components

Processors, connectors, and other components that receive a `component.Host` can obtain the extension and use it to look up workload config for the current resource.

### 1. Declare a dependency on the extension

In your component’s config or factory, document that the component expects an extension of type `odigos_workload_config`

### 2. Get the extension from the host

In `Start` (or when processing), get the extension from the host and type-assert to the extension’s type:

```go
import (
    "go.opentelemetry.io/collector/component"
    odigosworkloadconfigextension "github.com/odigos-io/odigos/collector/extension/odigosworkloadconfigextension"
)

// In your struct, store the extension after Start:
//   workloadConfig *odigosworkloadconfigextension.OdigosWorkloadConfig

func (p *myProcessor) Start(ctx context.Context, host component.Host) error {
    extID := component.NewID(odigosworkloadconfigextension.Type)
    ext, ok := host.GetExtensions()[extID]
    if !ok {
        return fmt.Errorf("extension %q not found", extID)
    }
    p.workloadConfig, ok = ext.(*odigosworkloadconfigextension.OdigosWorkloadConfig)
    if !ok {
        return fmt.Errorf("extension %q is not OdigosWorkloadConfig", extID)
    }
    return nil
}
```

### 3. Build a WorkloadKey from resource attributes

Use the helper that parses standard Kubernetes resource attributes (e.g. from OTLP resources):

```go
import (
    "go.opentelemetry.io/collector/pdata/pcommon"
    odigosworkloadconfigextension "github.com/odigos-io/odigos/collector/extension/odigosworkloadconfigextension"
)

// For a trace/log/metric resource:
key := odigosworkloadconfigextension.WorkloadKeyFromResourceAttributes(resource.Attributes())
```

`WorkloadKeyFromResourceAttributes` reads:

- **Namespace:** `k8s.namespace.name`
- **Kind and name:** first present among `k8s.deployment.name`, `k8s.statefulset.name`, `k8s.daemonset.name`, `k8s.job.name`, `k8s.cronjob.name`, `k8s.argoproj.rollout.name` (and sets Kind to Deployment, StatefulSet, DaemonSet, Job, CronJob, Rollout respectively).

Any missing attribute leaves that field empty. If you already have namespace/kind/name from elsewhere, you can build `odigosworkloadconfigextension.WorkloadKey{Namespace: ns, Kind: kind, Name: name}` directly.

### 4. Look up workload config

```go
cfg, ok := p.workloadConfig.GetWorkloadSamplingConfig(key)
if !ok {
    // No InstrumentationConfig for this workload; use defaults or skip.
    return
}
// Use cfg.WorkloadCollectorConfig (e.g. tail sampling, collector config per container).
```

`WorkloadSamplingConfig` contains:

- **WorkloadCollectorConfig** – slice of collector config (e.g. tail sampling) per container, as defined on the InstrumentationConfig spec.

### 5. Optional: iterate over cached keys

For debugging or batch use, you can access the underlying cache:

```go
cache := p.workloadConfig.Cache()
for _, k := range cache.AllKeys() {
    cfg, _ := cache.Get(k)
    // ...
}
```

Do not modify the cache; use `GetWorkloadSamplingConfig` for reads.

## Types

| Type | Description |
|------|-------------|
| `WorkloadKey` | Identifies a workload: `Namespace`, `Kind`, `Name` (e.g. Deployment, StatefulSet). Fields may be empty. |
| `WorkloadSamplingConfig` | Sampling/collector config for a workload; contains `WorkloadCollectorConfig` (per-container config). |
| `WorkloadKeyFromResourceAttributes(attrs pcommon.Map) WorkloadKey` | Builds a `WorkloadKey` from OTel resource attributes when present. |

## Requirements

- **Kubernetes:** Extension is intended to run in-cluster so it can watch InstrumentationConfigs. Without in-cluster config, the cache stays empty and lookups return not found.
- **InstrumentationConfig CRD:** Odigos InstrumentationConfig resources (odigos.io/v1alpha1) must exist; the extension only reads them and does not create them.

## Roadmap

This extension can be generalized for more functionality. TODO:

1. VM Agent support: The workload key can be easily extended for other identifiers that are not k8s specific and the informer replaced with a VM config reader.
2. More values: The cache value struct can store more than just sampling, such as service name or other values used by the collector.
