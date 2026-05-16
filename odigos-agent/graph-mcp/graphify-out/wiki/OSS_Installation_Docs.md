# OSS Installation Docs

> 13 nodes · cohesion 0.15

## Key Concepts

- **ConfigProvider** (5 connections) — `pkg/ebpf/configprovider.go`
- **IdentityServer** (4 connections) — `pkg/csi/identity_server.go`
- **HealthService** (3 connections) — `pkg/csi/health_server.go`
- **checkRequiredPaths()** (3 connections) — `pkg/csi/helpers.go`
- **.Check()** (2 connections) — `pkg/csi/health_server.go`
- **.Watch()** (2 connections) — `pkg/csi/health_server.go`
- **.Probe()** (2 connections) — `pkg/csi/identity_server.go`
- **.SendConfig()** (2 connections) — `pkg/ebpf/configprovider.go`
- **.GetPluginCapabilities()** (1 connections) — `pkg/csi/identity_server.go`
- **.GetPluginInfo()** (1 connections) — `pkg/csi/identity_server.go`
- **.InitialConfig()** (1 connections) — `pkg/ebpf/configprovider.go`
- **.Shutdown()** (1 connections) — `pkg/ebpf/configprovider.go`
- **health_server.go** (1 connections) — `pkg/csi/health_server.go`

## Relationships

- [[Odiglet Agent Filesystem Setup]] (24 shared connections)
- [[Filter Apply Configurations (api)]] (1 shared connections)
- [[Odiglet File Copy]] (1 shared connections)
- [[Odiglet Pod Manager]] (1 shared connections)
- [[Source Object Docs]] (1 shared connections)

## Source Files

- `pkg/csi/health_server.go`
- `pkg/csi/helpers.go`
- `pkg/csi/identity_server.go`
- `pkg/ebpf/configprovider.go`

## Audit Trail

- EXTRACTED: 23 (82%)
- INFERRED: 5 (18%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*