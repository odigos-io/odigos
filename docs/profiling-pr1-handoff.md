# PR1 handoff — resume on Linux VM

**Paused:** 2026-03-23. Continue profiling work from here after moving to a VM for faster builds.

## Goal (PR1)

Collector: **ebpf-profiler receiver + `odigosotelcol` build** — image/CI include `go.opentelemetry.io/ebpf-profiler`; builds succeed on **Linux** (CI path).

## Repo state

- **Branch:** `profiling/pr-1-collector-ebpf-receiver` (already checked out locally).
- **Already in tree:** `collector/builder-config.yaml` lists `go.opentelemetry.io/ebpf-profiler` with `import: go.opentelemetry.io/ebpf-profiler/collector`; `collector/odigosotelcol/components.go` registers the receiver factory.

## What was tried (macOS)

- Ran: `make -C collector build-odigoscol`
- **Failed:** `go.opentelemetry.io/ebpf-profiler/rlimit` — `undefined: unix.Prlimit` (darwin). This is expected; ebpf-profiler targets Linux. **Do not treat macOS failure as a blocker** if Linux build passes.
- **Docker** `docker build -f collector/Dockerfile …` from repo root was **not completed** here (command did not finish in session). **Re-run on the VM.**

## What to run on the VM (in order)

1. Clone/fetch this repo, checkout `profiling/pr-1-collector-ebpf-receiver`.
2. **Native Linux build (preferred quick check):**
   ```bash
   cd collector && make build-odigoscol
   ```
3. **Docker build (matches CI image build):** from repository root:
   ```bash
   docker build -f collector/Dockerfile -t odigos-collector:pr1 .
   ```
4. **Align with CI “verify collector OCB”** (`.github/workflows/verify-collector-ocb.yml`):
   ```bash
   cd collector && make genodigoscol generate
   ```
   If this changes generated files, commit them as part of PR1.

## Next steps after green build

- Single commit for PR1 (per stacked-PR plan), push to **upstream** branch `profiling/pr-1-collector-ebpf-receiver`, open PR against `main`.
- Then branch **PR2** from this branch’s tip.

## Reference

- Stacked PR plan (if present): `docs/profiling-pr-split.md` or team Notion.

---

*Delete or update this file once PR1 is merged or no longer needed.*
