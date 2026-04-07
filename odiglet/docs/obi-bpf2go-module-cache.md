# OBI, bpf2go, `GOMODCACHE`, and `GODEBUG=goindex=0`

## What you need

When building **odiglet**, we run `go generate` (via **bpf2go**) inside a copy of `go.opentelemetry.io/obi` under [**the module cache**](https://go.dev/ref/mod#module-cache) (`$GOMODCACHE`, often `/go/pkg/mod`). That step adds generated **`.go`** files (and **`.o`** objects) next to the package sources.

The Go command can use a **module file index** to avoid re-reading every source file on each build. **New `.go` files** added *after* the module was first indexed are not always reflected in that index, so `go build` can compile a package **without** the generated Go sources and fail with `undefined: Bpf…` (or similar) even though the files exist on disk.

**Fix (used in `odiglet/Makefile`):** set [`GODEBUG=goindex=0`](https://go.dev/doc/godebug) to disable the module index so the loader falls back to scanning the package directory (see the `modindex` / `ImportDir` discussion in the implementation: [`load/pkg.go`](https://github.com/golang/go/blob/master/src/cmd/go/internal/load/pkg.go), [`modindex/read.go`](https://github.com/golang/go/blob/master/src/cmd/go/internal/modindex/read.go)). Any existing `GODEBUG` values are preserved; `goindex=0` is prepended.

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

## Why the old “obi-generate image + `cp` into `GOMODCACHE`” Dockerfile *can* work (no `replace` / `vendor`)

An earlier pattern ([example: `damemi/odigos` at `b8ef2d1…`](https://github.com/damemi/odigos/blob/b8ef2d1be4b07b54b8e4404096f78f3e4d58f65b/odiglet/Dockerfile)) did **not** use `replace` or `vendor` for OBI. It:

1. Ran **`go mod download`**, then copied the resolved OBI tree out to `/tmp/obi-module` for a separate **`obi-generate` stage** where `go generate ./...` ran on that copy.
2. Brought the generated tree back in **`COPY --from=obi-generate`**, then in a **`RUN`**: `go mod download`, `OBI_DIR=$(go list -m -f '{{.Dir}}' go.opentelemetry.io/obi)`, **`cp -r /tmp/obi-generated/* "$OBI_DIR/"`**. That **writes the generated `.go` / `.o` directly into the real module path under** [`$GOMODCACHE`](https://go.dev/ref/mod#module-cache).
3. Ran **`make build-odiglet` in a later `RUN`**, with the copy already on disk *before* that step starts.

That is still “mutate the module in the cache,” so it is *not* inherently different from “generate in place” in terms of **pathnames**. The important difference is usually **order and when the module index is first built**:

- The [module index](https://github.com/golang/go/blob/master/src/cmd/go/internal/modindex/scan.go) is populated when `cmd/go` first needs it; it reflects whatever files exist when **`indexModule`** walks the module tree. If the **first** full walk (and the mmap stored under [`$GOCACHE`](https://go.dev/doc/gocache) for that module key) happens **after** the `cp` has laid down all bpf2go outputs, the indexed package file lists can include the generated **`.go`** files.
- A **single** `RUN` that runs several `go` invocations (e.g. `go list` / other work) **before** `go generate` in the *same* cache directory can pre-build an index from the **pre-generate** tree, then new files are added; the next `go build` can still use that **stale** snapshot — the failure mode this doc started from.

`replace` / `vendor` are *other* ways to avoid the cached `path@version` layout, but the historical Dockerfile above shows you do **not** need them: careful **multi-stage** ordering so the “first time we need a full index for this modroot” lines up with a disk state that **already** contains the generated sources can work. `GODEBUG=goindex=0` is still the **robust** fix when those orderings or persistent [BuildKit cache](https://docs.docker.com/build/cache/optimize/) mounts on `/go/pkg` and `$GOCACHE` make the ordering hard to guarantee.
