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
