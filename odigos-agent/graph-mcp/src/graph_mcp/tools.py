"""MCP tools exposed by graph-mcp.

Phase 0: `graph_ping` and `graph_metadata`. Phase 1d adds `graph_query`,
`graph_neighbors`, `graph_path`, `graph_community`, `graph_god_nodes`,
`graph_list_communities`, `wiki_read`.
"""

from __future__ import annotations

from datetime import datetime, timezone

from mcp.server.fastmcp import FastMCP

from .loader import GraphArtifact


def register(server: FastMCP, artifact: GraphArtifact) -> None:
    @server.tool()
    def graph_ping() -> str:
        """Graph MCP health check. Returns "pong" plus the server name and a UTC timestamp."""
        return f"pong from graph-mcp at {datetime.now(timezone.utc).isoformat()}"

    @server.tool()
    def graph_metadata() -> dict:
        """Return metadata about the bundled Graphify artifact.

        Includes the pinned odigos commit used to build the graph, plus node,
        edge, community, and wiki-page counts.
        """
        return {
            "built_at_commit": artifact.built_at_commit,
            "node_count": artifact.node_count,
            "edge_count": artifact.edge_count,
            "community_count": artifact.community_count,
            "wiki_page_count": artifact.wiki_page_count,
            "graph_path": str(artifact.graph_path),
            "wiki_dir": str(artifact.wiki_dir),
        }
