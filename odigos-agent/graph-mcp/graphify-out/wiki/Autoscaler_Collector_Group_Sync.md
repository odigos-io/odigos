# Autoscaler Collector Group Sync

> 15 nodes · cohesion 0.18

## Key Concepts

- **main()** (7 connections) — `cmd/main.go`
- **.Run()** (7 connections) — `odiglet.go`
- **main.go** (5 connections) — `cmd/main.go`
- **NewCSIDriver()** (4 connections) — `cmd/csi-driver/main.go`
- **OdigletInitPhase()** (4 connections) — `odiglet.go`
- **ebpfInstrumentationFactories()** (3 connections) — `cmd/main.go`
- **CSIDriver** (3 connections) — `cmd/csi-driver/main.go`
- **nodeRegistrar** (3 connections) — `cmd/csi-driver/main.go`
- **ApplyOpenShiftSELinuxSettings()** (3 connections) — `pkg/instrumentation/fs/agents.go`
- **odiglet.go** (3 connections) — `odiglet.go`
- **.registerWithKubelet()** (2 connections) — `cmd/csi-driver/main.go`
- **NewIdentityServer()** (2 connections) — `pkg/csi/identity_server.go`
- **NewNodeServer()** (2 connections) — `pkg/csi/node_server.go`
- **.GetInfo()** (1 connections) — `cmd/csi-driver/main.go`
- **.NotifyRegistrationStatus()** (1 connections) — `cmd/csi-driver/main.go`

## Relationships

- [[Odiglet File Copy]] (38 shared connections)
- [[Odiglet Pod Manager]] (4 shared connections)
- [[Source Object Docs]] (2 shared connections)
- [[Instrumentor Manager]] (2 shared connections)
- [[Community 236]] (1 shared connections)
- [[Odiglet Agent Filesystem Setup]] (1 shared connections)
- [[Filter Apply Configurations (api)]] (1 shared connections)
- [[Community 225]] (1 shared connections)

## Source Files

- `cmd/csi-driver/main.go`
- `cmd/main.go`
- `odiglet.go`
- `pkg/csi/identity_server.go`
- `pkg/csi/node_server.go`
- `pkg/instrumentation/fs/agents.go`

## Audit Trail

- EXTRACTED: 37 (74%)
- INFERRED: 13 (26%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*