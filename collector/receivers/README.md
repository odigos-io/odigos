# Collector receivers

Odigos ships two receiver styles in `builder-config.yaml`:

| Receiver | Module | `replaces:` in builder-config | Signals |
|----------|--------|------------------------------|---------|
| **`odigosebpf`** | `github.com/odigos-io/odigos/collector/receivers/odigosebpfreceiver` | **Yes** → `../receivers/odigosebpfreceiver` | Traces, metrics (in-node eBPF instrumentation path) |
| **`profiling`** | `go.opentelemetry.io/ebpf-profiler` with `import: .../collector` | **No** (public module; receiver is the upstream `collector` package) | Profiles (CPU / OpenTelemetry profiles signal) |

**Adding or bumping the profiling receiver:** edit `gomod` / tag on the `go.opentelemetry.io/ebpf-profiler` line in `../builder-config.yaml`, then run `make genodigoscol` from the `collector/` directory (see `../README.md`). There is no local fork under `receivers/` for that component—reuse is **upstream** + OCB `import`.
