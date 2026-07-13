# odigossymbolizeprocessor

Resolves **native** (C/C++/Rust) profiling frames to function names, in the
collector, on the host where the process runs.

## The problem

The eBPF profiler unwinds native stacks but does **not** name them — a native
frame arrives as `module + offset`, e.g. `libCXOPSX00.so+0x47a38`. Interpreted
runtimes (Python, JVM, Ruby, PHP, Node, .NET, Perl, BEAM) and Go are symbolized
in-agent by the profiler and arrive already named; native frames are left for the
backend. Without this processor, flamegraphs show raw addresses for native code.

## What it does

For each native frame, on the node where the process lives:

```
  process.pid + mapping basename (+ build-id)            from the OTLP profile
        │
        ▼
  /proc/<pid>/maps  ──►  open the on-disk ELF (container-aware: /proc/<pid>/root)
        │
        ▼
  parse .symtab (fallback .dynsym) ─► find the function containing the address
        │                              (build-id verified against the mapped file)
        ▼
  demangle ─► fill the OTLP Location's Lines with the function name
```

Frames the profiler already named are passed through untouched, so it is correct
for **any** language with zero configuration.

## Configuration

Zero-config — `odigossymbolizeprocessor: {}` works out of the box. Optional knobs:
`pid_attribute`, `max_symbol_cache`, `max_symbol_bytes`, `max_maps_cache`,
`maps_ttl_seconds`, `parse_workers`.

It runs in the **k8s node collector** and the **VM agent** collector — one code
path — and is enabled by default when profiling is on (opt out with
`profiling.symbolization.native: false`).

## Built to run at scale

- **Never blocks the pipeline:** the expensive step (parsing a ~50 MB symbol
  table) runs on background workers; a cache miss resolves on a later batch.
  `PreWarm` warms a process's binaries on first sight.
- **Bounded memory:** the symbol cache evicts by total **bytes** (not just count),
  plus a dead-process sweeper, a negative cache with back-off, and ELF parse
  limits for pathological binaries.
- **Safe:** strict build-id verification, and any unresolved frame simply stays
  `module+offset` — it never errors the pipeline.

## How it compares to debuginfod

[debuginfod](https://sourceware.org/elfutils/Debuginfod.html) is a build-id → file
**server**: you host it, and for stripped binaries users must **upload** debug
info to it. It's the right tool for stripped binaries, exited processes, or
central fleet symbolization.

This processor takes the other, complementary path for the common case: the
symbols are **already on the host** for unstripped binaries (the typical
proprietary C++/Oracle deployment), so it reads them directly from `/proc` + ELF.

| | this processor (on-host) | debuginfod |
|---|---|---|
| Setup | none | host a server (+ upload for stripped) |
| Unstripped binary on host | ✅ resolves, zero touch | works, but adds infra for no gain |
| Stripped binary / exited process | ❌ not covered | ✅ |
| Network / air-gapped | local, none needed | server required |

They are **layers, not alternatives**: this is the zero-setup default (Tier 1);
a debuginfod-backed fallback for stripped/remote cases is a planned follow-up
(Tier 2). Both key off the same build-id the profiler already provides.
