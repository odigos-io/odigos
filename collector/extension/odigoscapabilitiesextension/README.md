# odigoscapabilitiesextension

Raises the collector process's **effective** Linux capabilities to match its
**permitted** set on startup, before any pipeline component (e.g. the eBPF
receiver) starts.

## Why it exists

Odigos ships a single collector image/binary used in two roles:

- **Node collector** — runs the eBPF receiver and needs kernel privileges
  (`CAP_BPF`, `CAP_PERFMON`, `CAP_SYS_ADMIN`, `CAP_IPC_LOCK`) to load programs and
  read perf/ring buffers.
- **Cluster (gateway) collector** — only forwards/processes telemetry and needs
  **no** capabilities.

When run as non-root, the node collector must obtain these privileges without
turning the gateway into a privileged process.

## Effective vs permitted capabilities

The kernel tracks several per-thread capability sets. Two matter here
(see [capabilities(7)](https://man7.org/linux/man-pages/man7/capabilities.7.html)):

- **Permitted** — "a limiting superset for the effective capabilities that the
  thread may assume." A capability must be in the permitted set before it can be
  made effective.
- **Effective** — "the capabilities used by the kernel to perform permission
  checks for the thread." A privileged syscall only succeeds if the required
  capability is currently in the **effective** set.

So a process can hold a capability in *permitted* but still be denied, because it
is not yet *effective*. This extension copies permitted → effective at startup.

## File capabilities (`=p`) vs the pod's bounding set

Two independent mechanisms decide which capabilities the collector ends up with:

- **File capabilities on the binary.** The image runs
  `setcap ...=p` (permitted only, no effective bit). At `execve()` the permitted
  file capabilities become the process's permitted set. We intentionally omit the
  effective (`e`) bit: with it set, the kernel treats the binary as
  "capability-dumb" and fails `execve()` with `EPERM` if any file capability is
  masked out by the bounding set — which would crash the gateway (no caps) and
  break node collectors whenever a user drops a capability.
- **The bounding set from Helm values.** The Kubernetes `securityContext`
  capabilities (from `values.yaml`) form the process's bounding set, which "is a
  limiting superset for the capabilities that a thread can add to its permitted
  set." At exec the permitted set is effectively `file-permitted ∩ bounding`, so a
  capability the user removes from Helm simply never appears in permitted.

Because the binary is stamped with the effective bit **off**, it is up to the
process to raise its own effective capabilities — that is what this extension
does. It reads the permitted set, sets effective equal to it (across all OS
threads), and is a no-op when they already match (e.g. running as root, or the
gateway where permitted is empty).
