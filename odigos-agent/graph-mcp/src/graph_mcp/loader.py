"""Load the bundled Graphify artifact at startup.

Phase 0 reads only the metadata we need for `graph_metadata` (commit, node /
edge / community counts plus a wiki page count). Full graph traversal lands
in Phase 1d.
"""

from __future__ import annotations

import json
from dataclasses import dataclass
from pathlib import Path


@dataclass(frozen=True)
class GraphArtifact:
    graph_path: Path
    wiki_dir: Path
    built_at_commit: str
    node_count: int
    edge_count: int
    community_count: int
    wiki_page_count: int


def load(graph_path: Path, wiki_dir: Path) -> GraphArtifact:
    if not graph_path.is_file():
        raise FileNotFoundError(f"graph file missing: {graph_path}")
    if not wiki_dir.is_dir():
        raise FileNotFoundError(f"wiki dir missing: {wiki_dir}")

    with graph_path.open("r", encoding="utf-8") as handle:
        data = json.load(handle)

    nodes = data.get("nodes", [])
    edges = data.get("links", data.get("edges", []))
    communities = {node.get("community") for node in nodes if "community" in node}
    communities.discard(None)

    wiki_pages = sum(1 for _ in wiki_dir.rglob("*.md"))

    return GraphArtifact(
        graph_path=graph_path,
        wiki_dir=wiki_dir,
        built_at_commit=_resolve_commit(graph_path, data),
        node_count=len(nodes),
        edge_count=len(edges),
        community_count=len(communities),
        wiki_page_count=wiki_pages,
    )


def _resolve_commit(graph_path: Path, data: dict) -> str:
    """Locate the Graphify build commit.

    `merged-graph.json` doesn't carry `built_at_commit` (only `graph.json`
    does), so fall back to the sibling `graph.json` when the served file
    omits it.
    """
    commit = data.get("built_at_commit")
    if commit:
        return str(commit)
    sibling = graph_path.with_name("graph.json")
    if sibling.is_file() and sibling != graph_path:
        with sibling.open("r", encoding="utf-8") as handle:
            sibling_data = json.load(handle)
        if sibling_data.get("built_at_commit"):
            return str(sibling_data["built_at_commit"])
    return "unknown"
