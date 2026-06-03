# Odigos Native Symbolize Processor

Symbolizes native (C/C++/Rust) frames in OpenTelemetry profiles in-place, then forwards the batch.

The eBPF profiler symbolizes the kernel (kallsyms), Go, and interpreted runtimes (Java, Python, …) — those locations already carry `Line`/`Function` entries and pass through untouched. Native userspace frames arrive as `mapping + address` only. This processor walks the profiles dictionary and, for each unsymbolized location, resolves the address against the on-disk binary's `.symtab`/`.dynsym`, MiniDebugInfo (`.gnu_debugdata`), and local separate debuginfo (by Build ID / `.gnu_debuglink`), appending a `Function` and `Line` so downstream sees a function name.

Symbolization is best-effort: when the binary cannot be found or the address cannot be resolved, the frame is left raw and the batch is never errored.

## Binary path resolution

For each mapping the on-disk path is derived from the owning `ResourceProfiles` resource attributes:

1. `process.executable.path` — the authoritative path for the workload's main executable (matched by basename).
2. `/proc/<process.pid>/root/<mapping-basename>` — the process root mount, which exposes the same on-disk binaries (including shared libraries such as `libc.so.6`) for containerized workloads where only `process.pid` is known.

## Configuration

```yaml
processors:
  odigosnativesymbolizeprocessor/profiles: {}
```

Optional (native symbolization is on by default; set to `false` for a pure pass-through):

```yaml
processors:
  odigosnativesymbolizeprocessor/profiles:
    native: true
```
