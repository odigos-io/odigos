# Enterprise Instrumentation Docs

> 12 nodes · cohesion 0.21

## Key Concepts

- **NodeServer** (10 connections) — `pkg/csi/node_server.go`
- **extractPodUIDFromPath()** (3 connections) — `pkg/csi/helpers.go`
- **isPathMounted()** (3 connections) — `pkg/csi/node_server.go`
- **.NodePublishVolume()** (3 connections) — `pkg/csi/node_server.go`
- **.NodeUnpublishVolume()** (3 connections) — `pkg/csi/node_server.go`
- **helpers.go** (2 connections) — `pkg/csi/helpers.go`
- **.NodeExpandVolume()** (1 connections) — `pkg/csi/node_server.go`
- **.NodeGetCapabilities()** (1 connections) — `pkg/csi/node_server.go`
- **.NodeGetInfo()** (1 connections) — `pkg/csi/node_server.go`
- **.NodeGetVolumeStats()** (1 connections) — `pkg/csi/node_server.go`
- **.NodeStageVolume()** (1 connections) — `pkg/csi/node_server.go`
- **.NodeUnstageVolume()** (1 connections) — `pkg/csi/node_server.go`

## Relationships

- [[Filter Apply Configurations (api)]] (28 shared connections)
- [[Odiglet File Copy]] (1 shared connections)
- [[Odiglet Agent Filesystem Setup]] (1 shared connections)

## Source Files

- `pkg/csi/helpers.go`
- `pkg/csi/node_server.go`

## Audit Trail

- EXTRACTED: 26 (87%)
- INFERRED: 4 (13%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*