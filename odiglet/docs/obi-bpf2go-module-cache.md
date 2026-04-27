# OBI, bpf2go, `GOMODCACHE`, and the module file index

## What you need

For **OBI**, Odigos uses the upstream **source-generated** release (see **`setup-obi`**) instead of running **`go generate`** for that module. For other eBPF deps, **bpf2go** (via upstream **`make generate`**) can still add **`.go`** / **`.o`** files; those must be on disk before **`go build`**.

The Go command can use a **module file index** to avoid re-reading every source file on each invocation. **New `.go` files** added *after* the module was first indexed are not always reflected in that index, so `go build` can compile a package **without** the generated Go sources and fail with `undefined: Bpf…` (or similar) even though the files exist on disk.

**`go build -a`** does *not* fix that by itself: it only invalidates the **build** (compile) cache, not the module file index; see *Why this affects…* below.

## Docker / CI build (same **`Makefile`** as local)

Odigos follows [open-telemetry/opentelemetry-ebpf-instrumentation#1378](https://github.com/open-telemetry/opentelemetry-ebpf-instrumentation/pull/1378): each OBI release publishes **`obi-<tag>-source-generated.tar.gz`**. There is **no** separate OBI stage in **`odiglet/Dockerfile`**: the **`builder`** stage runs **`make build-odiglet`** (**`setup-obi`**, then **`generate`** for non-OBI libs, then **`go build`**). The **CSI** image builds **`./cmd/csi-driver`** only and does not run **`setup-obi`**. The image build uses the Makefile default **`OBI_VERSION`**; bump that (and **`go.mod`** **`require go.opentelemetry.io/obi`**) when upgrading OBI.

Locally: **`make build-odiglet`** / **`make setup-obi`**, optional **`OBI_VERSION=v…`**.

**Related ideas**

- [Module cache layout and immutability](https://go.dev/ref/mod#module-cache)
- [Organizing a Go module](https://go.dev/doc/modules/layout) (where generated `.go` must live to be part of a package)
- [bpf2go](https://github.com/cilium/ebpf/tree/master/cmd/bpf2go) (command that emits the `*_bpfel.go` sources and the embedded `.o`)

## Why this affects generated **`.go`** and not the **`.o`**

When generating other eBPF libraries like `go.opentelemetry.io/auto`, we simply `cd` into the modcache and run `go generate` there. Why does that work for them but not OBI?

The answer is because those libraries commit generated bpf2go files (`*_bpfel.go`). The only thing that is generated in the cache at build time are the `*.o` binary eBPF files.

The `go` command’s **module file index** (`GODEBUG` **`goindex`**) is about **which paths count as this package’s Go source** when resolving imports and building the [`build.Package`](https://pkg.go.dev/go/build#Package) (e.g. [`GoFiles`](https://pkg.go.dev/go/build#Package) from [`ImportDir`](https://pkg.go.dev/go/build#Context.ImportDir) / the loader in [`load/pkg.go`](https://github.com/golang/go/blob/master/src/cmd/go/internal/load/pkg.go)). It does **not** implement “read this blob for `//go:embed`.” Non-`.go` artifacts under the same directory are **not** listed in that set; a bpf2go **`.o`** is an ordinary file on disk in `$GOMODCACHE`, not a “package member” the index must enumerate for the compiler to open later.

**`//go:embed`** is handled when the **compiler** compiles a **`.go` file** that is already part of the build and contains an embed directive. The compiler then reads the embedded path (the `.o`, etc.) with normal **filesystem I/O** under the package’s source directory (still inside the mod cache for a dependency). So yes: the blob lives in the mod cache, and the toolchain **will** read it from there **once** that `*_bpfel.go` (or equivalent) is **selected for compilation**. The index is not a separate “fetch the `.o`” step—it’s “which `.go` files are in this package.” If a stale index **omits** a generated `*_bpfel.go`, that file is never compiled, so the compiler never runs the `//go:embed` for the `.o`; you see **`undefined: Bpf…`** from other packages that **were** built without those generated types, not a missing-embed error about the object file.

So the failure mode here is a stale list of **`.go`** files, not the index “failing to find” the **`.o`**: the `.o` is not a separate line item in the indexed Go file list; the bug is the generated **`.go`** that references it not being treated as part of the package.
