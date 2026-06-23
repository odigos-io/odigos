// Package symbolize is the on-host native (C/C++/Rust) address->name symbolizer
// used by odigossymbolizeprocessor. It resolves the native frames the eBPF
// profiler leaves unnamed (emitted as mapping basename + file offset).
//
// It is internal to the processor on purpose: the only consumer is the
// processor, and the operation (address->name) is distinct from the
// name->offset symbol work in the instrumentation agents. Keeping it here
// avoids a cross-repo dependency.
//
// # Scope
//
// Native frames only. Interpreted runtimes (Python/JVM/Ruby/PHP/Node/.NET/
// Perl/BEAM) and Go are symbolized in-agent by the profiler and arrive already
// named; the processor passes those through untouched. DWARF source lines and
// inline expansion are out of scope (a future symblib-backed engine); this is
// function-name (Tier-1) resolution from the live binary's symbol table.
//
// # How it resolves
//
// For a frame, the symbolizer reads /proc/<pid>/maps to turn the mapping
// basename into an openable host path (container-aware: it prefers
// /proc/<pid>/root/<path>, honoring HOST_PROC), parses the binary's ELF symbols
// (.symtab, falling back to .dynsym), translates the runtime address to a file
// virtual address via the PT_LOAD segments, looks up the enclosing function, and
// demangles it. When the profile carries a GNU build-id it is verified against
// the on-disk file (strict): a mismatch leaves the frame raw rather than risk a
// wrong name from a redeployed binary.
//
// # Performance model
//
// Symbolization runs in the collector pipeline, so it must never block on slow
// work. The expensive step — parsing a binary's ELF symbol table (e.g. a ~50 MB
// libclntsh) — is done on a BACKGROUND worker, not in Resolve. On a cache miss
// Resolve enqueues the parse and returns "unresolved" for that frame; the next
// batch (once the parse has completed) resolves it. PreWarm parses a process's
// mapped binaries ahead of time so even the first batch hits a warm cache.
//
// Cost on the hot path is therefore O(unique locations) with a cached binary:
// an O(log n) symbol lookup plus a /proc/<pid>/maps read (small, cached). The
// hot path takes a read lock (RWMutex) so cache hits run concurrently across
// pipelines. No network, no large parse inline.
//
// # Scale & safety
//
// Built to run across large fleets at low, bounded overhead:
//
//   - Memory-bounded caches: parsed symbols are evicted by total BYTES (one
//     Oracle binary can hold tens of MB), not just entry count — the real guard
//     against node-collector OOM. The per-pid maps cache is LRU- and TTL-bounded.
//   - Dead-process sweeper: a background loop drops cached maps for processes
//     that have exited (stat /proc/<pid>), bounding memory and preventing a
//     reused pid from resolving against stale maps within the TTL window.
//   - Negative cache + back-off: a failed or oversized binary is not re-parsed
//     every batch under process churn; entries expire so transient failures
//     recover.
//   - Parse limits: pathological/corrupt binaries (excessive size or symbol
//     count) are skipped so one bad file cannot pin a worker or exhaust memory.
//   - Graceful degradation: any unresolved frame stays module+offset; the
//     symbolizer never errors the pipeline.
//
// All cache sizes, the maps TTL, and the parse-worker count are configurable;
// the defaults work out of the box.
package symbolize
