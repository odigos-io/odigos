# Recommendations

Catalog of Odigos recommendations — suggested configuration changes that improve
usability, reduce noise, or enable optional capabilities.

Each recommendation is defined as a YAML manifest under [`manifests/`](./manifests)
and loaded at runtime via `go:embed`.

## Usage

```go
import (
    "github.com/odigos-io/odigos/common"
    "github.com/odigos-io/odigos/recommendations"
)

if err := recommendations.Load(); err != nil {
    return err
}

for _, rec := range recommendations.Get() {
    fmt.Println(rec.Type, rec.Title)
}

rec, ok := recommendations.GetByType(common.RecommendationTypeInferDBAttributes)
if ok {
    fmt.Println(rec.Summary)
}
```

- `Load()` parses all embedded manifests once (call during startup).
- `Get()` returns every loaded recommendation.
- `GetByType(type)` looks up a single recommendation by its `spec.type`.
- `GetByK8sObjectName(name)` looks up a single recommendation by its `spec.k8sObjectName`.

## Types

| Type | Description |
|------|-------------|
| `InferDBAttributes` | Derive missing DB semantic attributes from query text |
| `AutoGoOffsetUpdater` | Keep Go eBPF offsets up to date automatically (enterprise) |
| `EnableOwnMetrics` | Include a lean metrics DB for Odigos self-observability in the UI |
| `SampleHealthProbes` | Sample out noisy Kubernetes health probe traces |
| `UrlTemplatization` | Replace dynamic URL path segments with low-cardinality templates |
