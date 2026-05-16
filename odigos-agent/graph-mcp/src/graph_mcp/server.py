"""Entry point for the graph MCP server."""

from __future__ import annotations

import argparse
import logging
import os
from pathlib import Path

from mcp.server.fastmcp import FastMCP

from .loader import load
from .tools import register


def build_server(graph_path: Path, wiki_dir: Path, host: str, port: int) -> FastMCP:
    artifact = load(graph_path, wiki_dir)
    logging.info(
        "loaded graph artifact: commit=%s nodes=%d edges=%d communities=%d wiki_pages=%d",
        artifact.built_at_commit,
        artifact.node_count,
        artifact.edge_count,
        artifact.community_count,
        artifact.wiki_page_count,
    )

    server = FastMCP("odigos-agent-graph-mcp", host=host, port=port)
    register(server, artifact)
    return server


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="odigos-agent graph MCP server")
    parser.add_argument(
        "--graph",
        default=os.getenv("GRAPH_FILE", "/graph/merged-graph.json"),
        type=Path,
    )
    parser.add_argument(
        "--wiki",
        default=os.getenv("WIKI_DIR", "/graph/wiki"),
        type=Path,
    )
    parser.add_argument(
        "--host",
        default=os.getenv("MCP_GRAPH_HOST", "0.0.0.0"),
    )
    parser.add_argument(
        "--port",
        default=int(os.getenv("MCP_GRAPH_PORT", "9091")),
        type=int,
    )
    return parser.parse_args()


def main() -> None:
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
    args = parse_args()
    server = build_server(args.graph, args.wiki, args.host, args.port)
    logging.info("graph-mcp listening on %s:%d (endpoint /mcp)", args.host, args.port)
    server.run(transport="streamable-http")


if __name__ == "__main__":
    main()
