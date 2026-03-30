# Odigos patches on the vendored service graph connector

This package is copied from [opentelemetry-collector-contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/connector/servicegraphconnector) (module path unchanged). These notes list **Odigos-only behavior** so you can re-apply it after upgrading the vendored tree.

---

## 1. `upsertPeerAttributes` — no `break` (fill the whole `Peer` map)

| Item | Detail |
|------|--------|
| **Where** | `connector.go` — `upsertPeerAttributes` |
| **Upstream** | After the first `VirtualNodePeerAttributes` entry found on the span, the loop **`break`s**, so `e.Peer` holds at most **one** key. |
| **Odigos** | **`break` removed** — every attribute name in the config list that exists on the span is written into `peers` (same loop, no early exit). |
| **Why** | `getPeerHost` still uses **priority order** only for the primary `server` label. Odigos also needs **all** present peer attributes in `e.Peer` for §2–§3 (aggregation key + extra `server_*` metric labels). |

---

## 2. `buildMetricKeyFromEdge` (internal series key)

| Item | Detail |
|------|--------|
| **Where** | `connector.go` — `aggregateMetricsForEdge` calls `buildMetricKeyFromEdge(e)` instead of `buildMetricKey(..., e.Dimensions)`. |
| **Why** | The `server` label uses the **first** matching `virtual_node_peer_attributes` value (`getPeerHost`). Other peer fields can differ while `server` stays the same; the key must include them or series merge incorrectly. |
| **What it does** | Starts from `buildMetricKey`. For `virtual_node` + non-empty `e.Peer`, appends sorted `peer\|<key>\|<value>`. Skips keys already in `e.Dimensions` as `client_*` / `server_*`. |

---

## 3. `addVirtualNodePeerDimensions` + `sortedMapKeys` (exported metric labels)

| Item | Detail |
|------|--------|
| **Where** | `connector.go` — `buildDimensions` + helpers |
| **Why** | Odigos UI expects `server_*` labels for virtual-node peers (e.g. `server_db.system`). Upstream only copies `e.Dimensions` onto the datapoint. |
| **What it does** | Adds `server_<peerKey>` for each `e.Peer` entry unless already in `edgeDims`. Uses **sorted** peer keys (same order as §2). |
