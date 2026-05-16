"""HTTP MCP client.

Connects to both the cluster MCP (Go) and the graph MCP (Python) and returns
a merged LangChain tool list. In the in-cluster pod, both servers listen on
`127.0.0.1`. In local docker compose, they're reachable by service name.
"""

from __future__ import annotations

import os
from dataclasses import dataclass

from langchain_mcp_adapters.client import MultiServerMCPClient
from langchain_core.tools import BaseTool


@dataclass(frozen=True)
class McpEndpoints:
    cluster_url: str
    graph_url: str

    @classmethod
    def from_env(cls) -> "McpEndpoints":
        return cls(
            cluster_url=os.environ.get("MCP_CLUSTER_URL", "http://127.0.0.1:9090/mcp"),
            graph_url=os.environ.get("MCP_GRAPH_URL", "http://127.0.0.1:9091/mcp"),
        )


def build_client(endpoints: McpEndpoints | None = None) -> MultiServerMCPClient:
    endpoints = endpoints or McpEndpoints.from_env()
    return MultiServerMCPClient(
        {
            "cluster": {"transport": "streamable_http", "url": endpoints.cluster_url},
            "graph": {"transport": "streamable_http", "url": endpoints.graph_url},
        }
    )


async def load_tools(endpoints: McpEndpoints | None = None) -> list[BaseTool]:
    """Return the merged tool catalog from both MCPs."""
    client = build_client(endpoints)
    return await client.get_tools()
