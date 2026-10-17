# CORE-1872 archive E2E evidence

Captured in Chrome on 2026-10-13. Local File Archive has no vendor UI or cloud
account. These screenshots show a local readback inspector over actual persisted
Odigos demo telemetry, not a vendor APM dashboard or a catalog mock.

Ordinary demo telemetry was copied from the existing Odigos gateway to an isolated
OSS collector built from this branch. The destination helper generated its three
file exporters, with 1 MiB rotation and three backups. After stopping the writer,
a stable snapshot was replayed with contrib collector 0.159.0's `otlp_json_file`
receiver. Comparison includes every record plus its resource, scope, and schema URLs:

- 1,024 spans, equal before and after replay.
- 7,919 metric records (each may contain multiple data points), equal before and after replay.
- 2,211 log records, equal before and after replay.

`readback.json` records source file sizes, SHA-256 hashes, counts and services.
Run the reusable verifier with:

```sh
python3 tests/localfilearchive/readback.py /path/to/snapshot /path/to/replayed
```

The replay config is documented in the destination setup page. In this capture,
`max_log_size` was 4 MiB, `fingerprint_size` was 64 KiB, and filename enrichment
was disabled. The initial default 1 KiB fingerprint skipped a rotated trace file;
the larger fingerprint and enrichment setting produced exact equality for all signals.

Separately, the real common config and Kubernetes deployment helpers generated a
two-replica archive deployment in kind. A test Filesystem RWX PVC backed by a
kind-node directory was mounted read-write. Writers ran as UID/GID/fsGroup 65534.
Each pod wrote a separate directory. After a rolling replacement, the previous
pods' trace/metric/log files remained and the new pods created their own archives.
This validates the destination mount/config helpers, rather than a complete new
instrumentor image rollout. The VM path was exercised with the native OSS collector;
a full VM-agent installation was not part of this local fixture.

`disk-full.json` records a Linux `/dev/full` probe: the collector returned HTTP 503
with `no space left on device` rather than reporting successful persistence.
The file exporter has no destination retry queue or OTLP partial-success protocol;
filesystem permissions replace network TLS/authentication for local archival.

Bundled-exporter regression tests cover all three signals in JSON/protobuf, with
and without framed zstd compression, with rotation and append mode. They check
multi-resource and scope preservation after exporter shutdown/restart, rotation
size and oversized-batch rejection. Kubernetes tests cover PVC access validation,
idempotent/shared mounts, missing claims, and overlapping paths.
