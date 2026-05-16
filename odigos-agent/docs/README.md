# odigos-agent

In-cluster "Fix with AI" debugging agent for odigos.

Triggered from the webapp on a source with missing spans. Diagnoses one of three
root causes (destination misconfigured, source not instrumented, collector
misconfigured) by inspecting cluster state via a Go MCP server, queries a
pre-built codebase knowledge graph via a Python MCP server, and streams its
reasoning back to the UI through the existing frontend Go backend.

## Layout

```
odigos-agent/
  docs/                  # README, PLAN, PROGRESS, DECISIONS
  mcp/                   # Go MCP server (cluster state) - listens 127.0.0.1:9090
  graph-mcp/             # Python MCP server (codebase knowledge graph) - listens 127.0.0.1:9091
    graphify-out/        # Bundled Graphify artifact (immutable per release)
  agent/                 # Python LangGraph + FastAPI agent
  deploy/                # Helm subchart + raw kustomize manifests (Phase 4)
  docker-compose.yml     # Local 3-container dev setup
```

## Status

Phase 0 - scaffold + smoke test. See [PLAN.md](PLAN.md).

## Local dev (Phase 0 smoke)

```bash
export ANTHROPIC_API_KEY=...
cd odigos-agent
docker compose up -d
docker compose --profile cli run --rm agent \
  "Call cluster_ping, graph_ping, and graph_metadata and report each."
```

Expected: ping responses from both MCPs and the bundled commit hash `37cf1aee`.
