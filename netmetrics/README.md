# odigos/netmetrics

Shared, environment-agnostic enrichment of OBI network flows with process/service
identity. Imported by **both** the VM agent and odiglet (k8s) so the resolution logic
is written once — mirroring how CPU profiling shares `odigos/common/unixfd` and the
collector processors.

## Why this module exists

OBI's network pipeline (netolly) emits flows as a bare 5-tuple — `src/dst ip:port` with
byte/RTT counts, **no process identity**. To turn `IP:port → IP:port` into
`service → service`, each flow's *local* endpoint must be resolved to the owning process,
then to a service. The lookup logic is identical on VM and k8s; only the **identity
source** differs. This module holds the shared logic and takes the identity source as an
injected dependency.

## Layering

```
EndpointResolver   local socket (ip:port) -> owning PID        (github.com/prometheus/procfs;
                   via /proc/net/{tcp,udp} inode join.          already an odigos dependency)
                   Pure, env-agnostic, no eBPF.

ServiceResolver    joins a flow to identity, composing
                   EndpointResolver with two INJECTED lookups:
                     PIDToService   (VM: profileattrs table; k8s: informer)
                     PeerToService  (VM: CIDR/DNS registry;  k8s: informer)
```

Only `PIDToService` / `PeerToService` are per-environment. `Resolve` (which endpoint is
local, peer side, stable server_port, comm fallback) is shared.

When the OBI fork stamps `process.pid` on the flow (socktrack_map), `EndpointResolver`
becomes optional — callers pass the flow's PID straight to the same join.

## Usage

```go
endpoints, _ := netmetrics.NewEndpointResolver()
go func() { for { endpoints.Refresh(); time.Sleep(2*time.Second) } }()

resolver := netmetrics.NewServiceResolver(
    endpoints,
    func(pid int) (netmetrics.Service, bool) { /* VM: profileattrs / k8s: informer */ },
    func(ip string) (netmetrics.Service, bool) { /* peer registry */ },
)

fi, ok := resolver.Resolve(srcIP, srcPort, dstIP, dstPort)
// fi.Local.Name -> service.name, fi.Peer.Name -> peer.service.name, fi.ServerPort -> server.port
```

`cmd/enricher` is a reference consumer (stand-in for the per-agent glue + collector
processor): scrapes OBI's raw flows, resolves via the shared `ServiceResolver`, and
re-exposes OTel-semconv-named `network_flow_bytes_total{service_name,peer_service_name,…}`.
