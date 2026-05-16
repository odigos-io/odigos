# Progress

One-line dated entries pointing to the chat that did the work.

- 2026-05-16 - Phase 0 scaffold (three containers, docker compose smoke test, graphify-out relocated under graph-mcp).
- 2026-05-16 - Phase 1a source / instrumentation MCP tools (8 reads + propose/apply_create_source approval pair). Shared `tools.BuildClients`, `ApprovalCache`. ADR-006/007/008.
- 2026-05-16 - Phase 1b collector MCP tools (7 tools incl. direct-HTTP `/metrics` scrape, regex log grep, parsed pipelines). ADR-009.
- 2026-05-16 - Phase 1c destination MCP tools (6 tools incl. masked secret inspection, gateway exporter cross-reference, direct TCP/TLS endpoint probe).
