"""MCP tools exposed by graph-mcp.

Phase 0 shipped `graph_ping` and `graph_metadata`. Phase 1d adds the
diagnostic-graph surface: search, neighbors, shortest path, community
inspection, god-nodes, community listing, wiki read.
"""

from __future__ import annotations

from collections import deque
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

import networkx as nx
from mcp.server.fastmcp import FastMCP

from .loader import GraphArtifact, read_wiki_file

# Caps. These match the agent-side per-session budget in the plan (max 30
# graph/wiki queries per /debug request).
DEFAULT_TOP_K = 20
MAX_TOP_K = 100
MAX_NEIGHBORS = 200
MAX_PATH_HOPS = 6
DEFAULT_PATH_HOPS = 4
DEFAULT_WIKI_MAX_LINES = 400
MAX_WIKI_MAX_LINES = 2000
DEFAULT_COMMUNITY_TOP_NODES = 30
MAX_COMMUNITY_TOP_NODES = 100


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
            "hub_count": len(artifact.hub_order),
            "graph_path": str(artifact.graph_path),
            "wiki_dir": str(artifact.wiki_dir),
        }

    @server.tool()
    def graph_query(query: str, kind: str | None = None, top_k: int = DEFAULT_TOP_K) -> dict:
        """Substring search across node labels.

        Returns up to top_k matches ranked by simple lexical score (exact >
        prefix > substring) with degree as the tie-breaker. `kind` optionally
        filters by node `file_type` (e.g. `code`, `concept`, `doc`).
        """
        if not query:
            return {"query": query, "matches": [], "total": 0}
        top_k = _clamp(top_k, 1, MAX_TOP_K)
        query_lower = query.lower()
        scored: list[tuple[int, int, str, dict]] = []
        for node_id, attributes in artifact.graph.nodes(data=True):
            if kind and str(attributes.get("file_type", "")).lower() != kind.lower():
                continue
            label = str(attributes.get("label", node_id))
            norm = str(attributes.get("norm_label", label))
            haystack = f"{label}\n{norm}".lower()
            if query_lower not in haystack:
                continue
            score = _label_match_score(label, norm, query_lower)
            scored.append((score, artifact.graph.degree(node_id), node_id, attributes))
        scored.sort(key=lambda item: (-item[0], -item[1], item[2]))
        matches = [
            _summarize_node(node_id, attributes, artifact)
            for _, _, node_id, attributes in scored[:top_k]
        ]
        return {"query": query, "kind": kind, "matches": matches, "total": len(scored)}

    @server.tool()
    def graph_neighbors(node_id: str, depth: int = 1) -> dict:
        """Return BFS-bounded neighbors of node_id up to `depth` hops (max 2).

        Capped at 200 nodes; `truncated` flags the cap. Edges are returned
        with their relation attribute for traceability.
        """
        if node_id not in artifact.graph:
            return {"found": False, "node_id": node_id}
        depth = _clamp(depth, 1, 2)
        visited: dict[str, int] = {node_id: 0}
        queue: deque[str] = deque([node_id])
        edges_seen: list[dict] = []
        truncated = False
        while queue:
            current = queue.popleft()
            current_depth = visited[current]
            if current_depth >= depth:
                continue
            for neighbor in artifact.graph.neighbors(current):
                if neighbor not in visited:
                    visited[neighbor] = current_depth + 1
                    if len(visited) > MAX_NEIGHBORS:
                        truncated = True
                        break
                    queue.append(neighbor)
                edge_attributes = artifact.graph.get_edge_data(current, neighbor, default={})
                edges_seen.append({
                    "source": current,
                    "target": neighbor,
                    "relation": edge_attributes.get("relation"),
                })
            if truncated:
                break
        neighbors_list = [
            {
                **_summarize_node(other_id, artifact.graph.nodes[other_id], artifact),
                "hops": hops,
            }
            for other_id, hops in visited.items()
            if other_id != node_id
        ]
        return {
            "found": True,
            "node_id": node_id,
            "depth": depth,
            "neighbors": neighbors_list,
            "edges": edges_seen[:500],
            "truncated": truncated,
        }

    @server.tool()
    def graph_path(from_id: str, to_id: str, max_hops: int = DEFAULT_PATH_HOPS) -> dict:
        """Return the shortest path between two nodes (cutoff at max_hops).

        If no path exists within max_hops, `found` is false. Paths are
        returned as the ordered node-id sequence plus the edges traversed.
        """
        max_hops = _clamp(max_hops, 1, MAX_PATH_HOPS)
        if from_id not in artifact.graph or to_id not in artifact.graph:
            return {"found": False, "from": from_id, "to": to_id, "reason": "node not found"}
        try:
            path = nx.shortest_path(artifact.graph, source=from_id, target=to_id)
        except nx.NetworkXNoPath:
            return {"found": False, "from": from_id, "to": to_id, "reason": "no path"}
        if len(path) - 1 > max_hops:
            return {
                "found": False,
                "from": from_id,
                "to": to_id,
                "reason": f"shortest path is {len(path) - 1} hops, exceeds max_hops={max_hops}",
            }
        edges = []
        for index in range(len(path) - 1):
            edge_attributes = artifact.graph.get_edge_data(path[index], path[index + 1], default={})
            edges.append({
                "source": path[index],
                "target": path[index + 1],
                "relation": edge_attributes.get("relation"),
            })
        node_summaries = [
            _summarize_node(node_id, artifact.graph.nodes[node_id], artifact) for node_id in path
        ]
        return {
            "found": True,
            "from": from_id,
            "to": to_id,
            "hops": len(path) - 1,
            "nodes": node_summaries,
            "edges": edges,
        }

    @server.tool()
    def graph_community(
        community_id_or_name: str,
        top_nodes: int = DEFAULT_COMMUNITY_TOP_NODES,
    ) -> dict:
        """Inspect a community by id or case-insensitive name.

        Returns the resolved id+name, node count, the top-N nodes by degree,
        and an excerpt of the associated wiki page when one exists.
        """
        community_id = _resolve_community(community_id_or_name, artifact)
        if community_id is None:
            return {"found": False, "input": community_id_or_name}
        top_nodes = _clamp(top_nodes, 1, MAX_COMMUNITY_TOP_NODES)
        nodes = _nodes_for_community(community_id, artifact)
        nodes_sorted = sorted(nodes, key=lambda item: artifact.graph.degree(item[0]), reverse=True)
        top = [
            _summarize_node(node_id, attributes, artifact)
            for node_id, attributes in nodes_sorted[:top_nodes]
        ]
        wiki_excerpt, wiki_filename = _read_wiki_excerpt(community_id, artifact)
        return {
            "found": True,
            "community_id": community_id,
            "community_name": artifact.community_labels.get(community_id, f"Community {community_id}"),
            "node_count": len(nodes),
            "top_nodes": top,
            "wiki_filename": wiki_filename,
            "wiki_excerpt": wiki_excerpt,
        }

    @server.tool()
    def graph_god_nodes(top_k: int = DEFAULT_TOP_K, scope: str | None = None) -> dict:
        """Return the highest-degree nodes overall or scoped to a community.

        `scope` accepts a community id or name; when set, results are
        restricted to nodes in that community.
        """
        top_k = _clamp(top_k, 1, MAX_TOP_K)
        scope_community_id: int | None = None
        if scope:
            scope_community_id = _resolve_community(scope, artifact)
            if scope_community_id is None:
                return {"scope": scope, "scope_resolved": False, "matches": []}
        if scope_community_id is None:
            candidates = artifact.graph.nodes(data=True)
        else:
            candidates = _nodes_for_community(scope_community_id, artifact)
        ranked = sorted(candidates, key=lambda item: artifact.graph.degree(item[0]), reverse=True)
        matches = [
            {
                **_summarize_node(node_id, attributes, artifact),
                "degree": artifact.graph.degree(node_id),
            }
            for node_id, attributes in ranked[:top_k]
        ]
        return {"scope": scope, "scope_community_id": scope_community_id, "matches": matches}

    @server.tool()
    def graph_list_communities(filter: str | None = None) -> dict:
        """List all communities (id, name, node_count).

        `filter` does a case-insensitive substring match on the name. Hub-order
        communities come first (the GRAPH_REPORT ranking), followed by the
        remainder sorted by name.
        """
        filter_lower = filter.lower() if filter else None
        in_hub_order: list[dict] = []
        in_hub_ids: set[int] = set(artifact.hub_order)
        for community_id in artifact.hub_order:
            entry = _community_entry(community_id, artifact)
            if filter_lower and filter_lower not in entry["name"].lower():
                continue
            in_hub_order.append(entry)
        rest = []
        for community_id in sorted(artifact.community_labels.keys()):
            if community_id in in_hub_ids:
                continue
            entry = _community_entry(community_id, artifact)
            if filter_lower and filter_lower not in entry["name"].lower():
                continue
            rest.append(entry)
        return {
            "filter": filter,
            "communities": in_hub_order + rest,
            "total": len(in_hub_order) + len(rest),
        }

    @server.tool()
    def wiki_read(community_name: str, max_lines: int = DEFAULT_WIKI_MAX_LINES) -> dict:
        """Read the wiki page associated with a community.

        Truncates to max_lines and reports `truncated`. Returns `found=False`
        when no page matches.
        """
        max_lines = _clamp(max_lines, 1, MAX_WIKI_MAX_LINES)
        community_id = _resolve_community(community_name, artifact)
        if community_id is None:
            return {"found": False, "input": community_name}
        excerpt, filename = _read_wiki_excerpt(
            community_id, artifact, max_lines=max_lines, full=True
        )
        if excerpt is None:
            return {"found": False, "input": community_name, "reason": "no wiki page on disk"}
        return {
            "found": True,
            "community_id": community_id,
            "community_name": artifact.community_labels.get(community_id, f"Community {community_id}"),
            "wiki_filename": filename,
            "content": excerpt,
            "truncated": excerpt.count("\n") >= max_lines,
        }


# ---- helpers ----


def _clamp(value: int, low: int, high: int) -> int:
    if value < low:
        return low
    if value > high:
        return high
    return value


def _label_match_score(label: str, norm: str, query_lower: str) -> int:
    label_lower = label.lower()
    if label_lower == query_lower or norm.lower() == query_lower:
        return 3
    if label_lower.startswith(query_lower) or norm.lower().startswith(query_lower):
        return 2
    return 1


def _summarize_node(node_id: str, attributes: dict, artifact: GraphArtifact) -> dict:
    community_raw = attributes.get("community")
    community_id: int | None
    try:
        community_id = int(community_raw) if community_raw is not None else None
    except (TypeError, ValueError):
        community_id = None
    community_name = (
        artifact.community_labels.get(community_id, f"Community {community_id}")
        if community_id is not None
        else None
    )
    return {
        "id": node_id,
        "label": attributes.get("label"),
        "file": attributes.get("source_file"),
        "line": attributes.get("source_location"),
        "kind": attributes.get("file_type"),
        "community_id": community_id,
        "community_name": community_name,
    }


def _resolve_community(input_value: str, artifact: GraphArtifact) -> int | None:
    if input_value is None:
        return None
    stripped = str(input_value).strip()
    if not stripped:
        return None
    try:
        candidate = int(stripped)
        if candidate in artifact.community_labels or candidate in artifact.community_node_counts:
            return candidate
    except ValueError:
        pass
    return artifact.community_index.get(stripped.lower())


def _nodes_for_community(community_id: int, artifact: GraphArtifact) -> list[tuple[str, dict]]:
    nodes: list[tuple[str, dict]] = []
    for node_id, attributes in artifact.graph.nodes(data=True):
        community_raw = attributes.get("community")
        try:
            attribute_id = int(community_raw) if community_raw is not None else None
        except (TypeError, ValueError):
            attribute_id = None
        if attribute_id == community_id:
            nodes.append((node_id, attributes))
    return nodes


def _community_entry(community_id: int, artifact: GraphArtifact) -> dict:
    return {
        "id": community_id,
        "name": artifact.community_labels.get(community_id, f"Community {community_id}"),
        "node_count": artifact.community_node_counts.get(community_id, 0),
    }


def _read_wiki_excerpt(
    community_id: int,
    artifact: GraphArtifact,
    max_lines: int = 80,
    full: bool = False,
) -> tuple[str | None, str | None]:
    """Locate and read the community's wiki page.

    Resolution order:
      1. `<community_name>.md` (with spaces preserved as-is from the labels file)
      2. `<community_name with spaces replaced by underscores>.md`
      3. `Community_<id>.md`
    Returns (content_or_None, filename_or_None). When found=True content is
    truncated to max_lines lines unless `full` is True (then the truncation
    is still applied but the caller asked for a longer view).
    """
    name = artifact.community_labels.get(community_id, f"Community {community_id}")
    candidates = [
        f"{name}.md",
        f"{name.replace(' ', '_')}.md",
        f"Community_{community_id}.md",
    ]
    for candidate in candidates:
        candidate_path = (artifact.wiki_dir / candidate)
        if candidate_path.is_file():
            text = read_wiki_file(candidate_path)
            lines = text.split("\n")
            limit = max_lines if full else min(max_lines, len(lines))
            return ("\n".join(lines[:limit]), candidate)
    return (None, None)


def _candidate_wiki_filenames(name: str, community_id: int) -> list[str]:
    """Public-ish helper exposed for tests so the fallback chain is locked."""
    return [
        f"{name}.md",
        f"{name.replace(' ', '_')}.md",
        f"Community_{community_id}.md",
    ]


# Exposed for tests: a wrapper around the module-level helper above.
def candidate_wiki_filenames(name: str, community_id: int) -> list[str]:
    return _candidate_wiki_filenames(name, community_id)


def label_match_score(label: str, norm: str, query_lower: str) -> int:
    return _label_match_score(label, norm, query_lower)


def resolve_community(input_value: str, artifact: GraphArtifact) -> int | None:
    return _resolve_community(input_value, artifact)


# Avoid linter complaints about Path being unused on systems that strip
# type-only imports.
_ = Path
_ = Any
