# OBI, bpf2go, `GOMODCACHE`, and the module file index

## What you need

When building **odiglet**, **bpf2go** (via **`go generate`**) adds generated **`.go`** files (and **`.o`** objects) next to `go.opentelemetry.io/obi` package sources. Those files must be on disk **before** `go build` treats the package as complete.

The Go command can use a **module file index** to avoid re-reading every source file on each invocation. **New `.go` files** added *after* the module was first indexed are not always reflected in that index, so `go build` can compile a package **without** the generated Go sources and fail with `undefined: Bpf…` (or similar) even though the files exist on disk.

**`go build -a`** does *not* fix that by itself: it only invalidates the **build** (compile) cache, not the module file index; see *Why this affects…* below.

## Copy method (current Docker / CI build)

The **`odiglet/Dockerfile`** does **not** run **`go generate`** for OBI directly inside the read-only-ish module cache first. It uses a **copy → generate → overlay** flow:

1. **`builder`**: **`go mod download`**, then **`cp -a`** the resolved OBI tree to **`/tmp/obi-module`** (full module layout so **`go list`** / **`go generate`** work).
2. **`obi-generate`**: image [**`obi-generator`**](https://github.com/open-telemetry/opentelemetry-ebpf-instrumentation/pkgs/container/obi-generator) runs **`go generate ./...`** on that copy under **`/src`** (clang/llvm + bpf2go provided by the image).
3. **`odiglet-build`**: **`COPY --from=obi-generate /src /tmp/obi-generated`**, then **`go mod download`**, then **`chmod`** + **`cp`** the generated tree **into** the real OBI path under [`$GOMODCACHE`](https://go.dev/ref/mod#module-cache) (paths from **`go list -m -f '{{.Dir}}'`** — use **inline backticks** in **`RUN`**, not **`$$VAR`**, [see `odiglet/Dockerfile`](https://github.com/odigos-io/odigos/blob/main/odiglet/Dockerfile)).
4. **`SKIP_OBI_GENERATE=1 make build-odiglet`**: the Makefile skips OBI’s **`go generate`** because the cache already contains bpf2go outputs.

That ordering keeps “mutate the tree in **`$GOMODCACHE`**” to a **single overlay** right before compile, instead of generating in place. **Why order matters for the module index:** the [module index](https://github.com/golang/go/blob/master/src/cmd/go/internal/modindex/scan.go) reflects whatever **`.go`** files exist when **`indexModule`** first walks that module root; the mmap lives under [**`$GOCACHE`**](https://go.dev/doc/gocache). If the first walk happens **after** the overlay has written all bpf2go **`.go`** outputs, the indexed package lists can include them. The fragile pattern is a **single `RUN`** that runs **`go`** (e.g. **`go list`**, **`go mod download`**) **before** **`go generate`** against the **same** module directory, then adds generated files in that same layer—**`go build`** can still use a **pre-generate** snapshot of the index. Splitting **generate** onto **`/tmp`** and overlaying in a dedicated step avoids that. You do **not** need **`replace`** or **`vendor`** for OBI if you keep this separation (they are other ways to leave the plain **`path@version`** mod-cache layout).

Persistent [BuildKit cache](https://docs.docker.com/build/cache/optimize/) mounts on **`/go/pkg`** and **`$GOCACHE`** can still surface **stale** index data across layers in theory; the odiglet **Makefile does not set `GODEBUG`** so CI/Docker can validate the copy flow on its own. If you hit **`undefined: Bpf…`** or similar after mutating the module cache, try the optional **`goindex=0`** workaround in the next section.

**Related ideas**

- [Module cache layout and immutability](https://go.dev/ref/mod#module-cache)
- [Organizing a Go module](https://go.dev/doc/modules/layout) (where generated `.go` must live to be part of a package)
- [bpf2go](https://github.com/cilium/ebpf/tree/master/cmd/bpf2go) (command that emits the `*_bpfel.go` sources and the embedded `.o`)

## Optional: `GODEBUG=goindex=0` (manual workaround)

You can **turn off** the module file index for a shell session: export [`GODEBUG=goindex=0`](https://go.dev/doc/godebug) (and merge with any other flags you need, comma-separated) so **`go build`** / **`go list`** rescans module source trees instead of trusting a possibly stale mmap’d index (see [`load/pkg.go`](https://github.com/golang/go/blob/master/src/cmd/go/internal/load/pkg.go), [`modindex/read.go`](https://github.com/golang/go/blob/master/src/cmd/go/internal/modindex/read.go)).

That is a **correctness-first** switch, not a structural fix. The **copy method** above is meant to avoid needing it for normal Docker/CI builds.

**Downsides of `goindex=0`**

- **Slower `go` work**: every load that would use the index walks the module tree from disk again instead of using the cached index under **`$GOCACHE`**.
- **More I/O**: large modules or many dependencies multiply directory reads and stat traffic.
- **Cold / CI cost**: repeated rescans hurt worst on clean caches and big graphs; you pay the “full scan” price on paths that the index was meant to make cheap.
- **Hides ordering bugs**: builds can succeed even when generate-vs-index ordering is wrong, which makes it harder to notice Dockerfile or cache issues until someone runs **without** that flag.

## Why this affects generated **`.go`** and not the **`.o`**

When generating other eBPF libraries like `go.opentelemetry.io/auto`, we simply `cd` into the modcache and run `go generate` there. Why does that work for them but not OBI?

The answer is because those libraries commit generated bpf2go files (`*_bpfel.go`). The only thing that is generated in the cache at build time are the `*.o` binary eBPF files.

The `go` command’s **module file index** (`GODEBUG` **`goindex`**) is about **which paths count as this package’s Go source** when resolving imports and building the [`build.Package`](https://pkg.go.dev/go/build#Package) (e.g. [`GoFiles`](https://pkg.go.dev/go/build#Package) from [`ImportDir`](https://pkg.go.dev/go/build#Context.ImportDir) / the loader in [`load/pkg.go`](https://github.com/golang/go/blob/master/src/cmd/go/internal/load/pkg.go)). It does **not** implement “read this blob for `//go:embed`.” Non-`.go` artifacts under the same directory are **not** listed in that set; a bpf2go **`.o`** is an ordinary file on disk in `$GOMODCACHE`, not a “package member” the index must enumerate for the compiler to open later.

**`//go:embed`** is handled when the **compiler** compiles a **`.go` file** that is already part of the build and contains an embed directive. The compiler then reads the embedded path (the `.o`, etc.) with normal **filesystem I/O** under the package’s source directory (still inside the mod cache for a dependency). So yes: the blob lives in the mod cache, and the toolchain **will** read it from there **once** that `*_bpfel.go` (or equivalent) is **selected for compilation**. The index is not a separate “fetch the `.o`” step—it’s “which `.go` files are in this package.” If a stale index **omits** a generated `*_bpfel.go`, that file is never compiled, so the compiler never runs the `//go:embed` for the `.o`; you see **`undefined: Bpf…`** from other packages that **were** built without those generated types, not a missing-embed error about the object file.

So the failure mode here is a stale list of **`.go`** files, not the index “failing to find” the **`.o`**: the `.o` is not a separate line item in the indexed Go file list; the bug is the generated **`.go`** that references it not being treated as part of the package.
