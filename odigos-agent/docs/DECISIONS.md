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

---

## ADR-010 - ReAct loop per subgraph, not one big agent

**Context.** Phase 2 needs three diagnostic flows (source, collector, destination). One option is a single ReAct agent with all tools and a long system prompt that switches mode based on triage. Another is one small ReAct agent per domain, dispatched from a LangGraph router node.

**Decision.** Per-subgraph ReAct agents. Each is created with `create_react_agent` bound to a filtered tool catalog and a focused system prompt. The LangGraph router node (`route_after_triage`) returns a list of node names; LangGraph fans out in parallel when the list has more than one entry (the `ambiguous` case).

**Consequences.** Smaller tool catalogs per call -> cheaper, less prompt-bloat, less off-domain tool drift. Each subgraph's prompt can be tuned independently in Phase 6. Cost: one extra LLM call for triage. Parallel fan-out for ambiguous cases works out of the box because `step_log` uses an `operator.add` reducer in `AgentState`.

---

## ADR-011 - Defer `apply_create_source` to Phase 3, never expose it in Phase 2

**Context.** PLAN.md Phase 2 says the source subgraph "yields control back to the API layer to await user decision." The `apply_create_source` MCP tool exists (Phase 1a) but applying without human approval would defeat the gated-mutation design.

**Decision.** Phase 2 deliberately excludes `apply_create_source` from the source subgraph's tool catalog. The subgraph may call `propose_create_source` (caching a dry-run + request_id in the MCP), then stops. The node code extracts the proposal from the conversation, writes it to `state.proposed_remediation` with `status="pending_approval"`, and execution proceeds to synthesis. Phase 3 will add an interrupt-based approval node and wire `apply_create_source` behind it.

**Consequences.** Phase 2 produces a complete Report including a pending mutation, but never mutates the cluster. Tests pin this contract (`test_partition_excludes_apply_create_source`). Phase 3 has a clean seam: insert an `await_approval` node between source and synthesize, gated on `state.proposed_remediation is not None`.

---

## ADR-012 - Tool partitioning by explicit name set, not pattern

**Context.** The merged MCP tool catalog mixes cluster + graph tools. Each subgraph should only see its own domain's tools (plus the graph/wiki tools as cross-cutting reference). Options: prefix matching (`get_collector_*`), regex, or an explicit set of names per bucket.

**Decision.** Explicit `frozenset` per bucket in `graph.py`. Tools not in any bucket are dropped from subgraph catalogs.

**Consequences.** Adding a new MCP tool requires a one-line update to the right bucket. Worth it: prefix matching would accidentally include `apply_create_source` in the source bucket (the very thing ADR-011 forbids). Naming MCP tools is part of the agent's contract; making the partition explicit forces the conversation when a new tool is added.

---

## ADR-013 - Graph/wiki tools available to every subgraph, not just a dedicated node

**Context.** The codebase knowledge graph (Phase 1d) is most useful when a domain subgraph is mid-diagnosis and needs to look up how some controller works. We considered a dedicated "research" node that owns graph access and a tool-router that hands relevant code excerpts to the domain subgraphs.

**Decision.** Each domain subgraph's tool catalog includes the full graph/wiki tool set (`graph_query`, `graph_neighbors`, `graph_path`, `graph_community`, `graph_god_nodes`, `graph_list_communities`, `wiki_read`, `graph_metadata`, `gh_read_file`).

**Consequences.** Subgraphs can reach for codebase context in the same loop where they're reading cluster state - no extra round trip through a router. Cost: every subgraph's prompt sees the graph tool descriptions. The per-session budget (max 30 graph/wiki queries + 10 citation reads) lives at the MCP / agent loop level, not per-subgraph, so the catalog overlap doesn't multiply the budget.
