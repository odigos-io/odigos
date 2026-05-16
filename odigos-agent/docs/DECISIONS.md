# Architecture decisions

Short ADRs. One per major directional choice. Each has Context / Decision /
Consequences. Keep entries short - PLAN.md has the long-form reasoning.

---

## ADR-001 - Language split: Go for cluster MCP, Python for agent and graph MCP

**Context.** Two MCP servers and one agent. Cluster MCP must use odigos's typed
k8s clients (already Go in this repo). Agent must integrate with LangGraph
(Python). Graph MCP needs `networkx` for graph traversals (Python).

**Decision.** Go for `mcp/`, Python for `agent/` and `graph-mcp/`.

**Consequences.** Three images instead of one, but each uses the
ecosystem-native tooling. MCP HTTP over `127.0.0.1` between containers in the
same pod insulates the agent from the cluster-MCP's go.mod blast radius.

---

## ADR-002 - Sidecar pattern with HTTP MCP on localhost, not stdio

**Context.** MCP supports stdio or HTTP transports. Sub-processing the Go MCP
from the Python agent (stdio) would couple their lifecycles and complicate
language-specific tooling.

**Decision.** Three sidecar containers in one pod. MCPs listen on
`127.0.0.1:9090` and `127.0.0.1:9091`. Agent connects over HTTP MCP.

**Consequences.** Clean container boundaries (separate images, independent
restarts, independent Dockerfiles). No service exposure - MCPs are pod-local.
Local dev uses docker compose with a shared network bridging the three.

---

## ADR-003 - Frontend integration via Go backend SSE proxy, not direct webapp → agent

**Context.** Webapp could call the agent directly via a Service. Would require
CORS, agent-side auth surface visible to browsers, and bypasses the existing
auth/audit path through the Go backend.

**Decision.** Webapp → existing Go backend (`/api/ai/debug`) → agent. Go
backend opens an SSE stream upstream and pipes bytes back to the browser.

**Consequences.** Single auth boundary (the Go backend). Agent token never
leaves the cluster. No CORS. Reuses the audit log path the Go backend already
has. Phase 5 work.

---

## ADR-004 - Codebase access via baked-in Graphify artifact + minimal gh_read_file

**Context.** Agent needs to reason about odigos source. Live grep / live RAG is
expensive and brittle (graph relationships are invisible to grep, RAG over a
constantly-changing repo is wrong). Graphify already produced a pre-built
knowledge graph with 497 communities and 508 wiki pages.

**Decision.** Bundle the Graphify artifact (`graphify-out/`) into the
`mcp-graph` image at build time. Expose graph/wiki tools through MCP. Provide
a single `gh_read_file(path, lineRange)` tool for citation expansion only,
locked to the artifact's pinned commit (`37cf1aee` for v1).

**Consequences.** Token-efficient queries, airgapped reads at runtime, no
exploration via raw file reads. Artifact is immutable per agent release. CI
rebuild per odigos release is a v2 concern (Phase 8+).

---

## ADR-005 - graphify-out lives under graph-mcp/, committed in-repo

**Context.** Phase 0 needs the artifact inside the `mcp-graph` Docker build
context. Originally `graphify-out/` was at the upstream repo root.

**Decision.** Move to `odigos-agent/graph-mcp/graphify-out/`. Commit as-is for
v1 (19 MB, immutable per release).

**Consequences.** Docker build context stays local. Repo grows by ~19 MB once.
Revisit (Git LFS, OCI artifact, release-bucket pull) in v2 when CI rebuild
lands.

---

## ADR-006 - Cluster MCP imports odigos modules via `replace` directives

**Context.** The cluster MCP needs the odigos typed clientset (`api/generated/odigos/clientset/versioned`), CRD type defs (`api/odigos/v1alpha1`), and shared helpers (`k8sutils/pkg/workload`, `api/k8sconsts`, `common/consts`). Publishing those modules to a registry just for the agent is over-engineering for an in-repo POC.

**Decision.** `odigos-agent/mcp/go.mod` uses local `replace` directives pointing at `../../api`, `../../common`, `../../k8sutils`, `../../odigosauth`. Matches the pattern already used elsewhere in odigos for sibling modules.

**Consequences.** Cluster MCP is locked to whatever odigos commit it sits on top of, which is exactly what we want for v1 (agent is built alongside odigos, not separately released). Out-of-tree builds would need to vendor or fork.

---

## ADR-007 - Approval cache is in-process and ephemeral

**Context.** The mutation approval flow (`propose_X` -> user approve -> `apply_X`) needs to remember the dry-run state across two tool calls. Options: external store (Redis/Postgres), file/PVC, or in-process memory.

**Decision.** `sync.Mutex`-guarded `map[string]*PendingMutation` in the MCP process, 5-minute TTL, garbage-collected on every `Put`/`Take`. UUID v4 request IDs.

**Consequences.** v1 ships a single MCP replica - no cross-process state to worry about. Pod restart drops in-flight approvals, which matches the 5-minute TTL semantics anyway. Scaling out (HPA on the agent pod) would require swapping this for a backing store; deferred until we hit it.

---

## ADR-008 - Mutation audit is `log.Printf` placeholder, not OTLP, in v1

**Context.** PLAN.md says every mutation emits an OTLP audit event. The OTLP audit pipeline doesn't exist yet - building it is its own (cross-cutting) workstream.

**Decision.** Every `propose_*` / `apply_*` call logs a single structured line via `log.Printf` (`audit: op=... ns=... result=...`). Stdout flows into the pod's logs, which odigos itself can scrape later.

**Consequences.** Real auditability lands when OTLP audit is wired up (likely Phase 7 or 8 alongside batch-plan approval). Until then, ops team has logs, not traces. Tracking debt explicitly here so it isn't forgotten.

---

## ADR-009 - Collector metrics scraped via direct in-cluster HTTP, not pods/exec

**Context.** PLAN.md calls for `pods/exec` to `wget -qO- localhost:8888/metrics` inside each collector pod for `get_collector_metrics`. That requires the agent's RBAC to include `pods/exec` `create` plus SPDY wiring (`k8s.io/client-go/tools/remotecommand`).

**Decision.** Scrape collector `/metrics` directly via an HTTP GET to `http://<podIP>:8888/metrics` from the MCP container, with a 5-second timeout. The MCP pod sits in the same cluster network as the collectors, so PodIP is reachable.

**Consequences.** No `pods/exec` RBAC verb needed. No SPDY code. Smaller blast radius. If the collector binds the metrics endpoint to localhost-only (it doesn't, by default in odigos's rendered config) the scrape would fail and we'd revisit. Phase 6 kind validation will confirm reachability end-to-end. Same direction applies to `probe_destination_endpoint` in Phase 1c (direct dial, not exec into a debug pod).
