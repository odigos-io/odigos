"""Load the bundled Graphify artifact at startup.

Phase 1d loads the full NetworkX graph plus the community-label map and the
GRAPH_REPORT's hub-ordering. Wiki content is read lazily on demand via an
LRU-cached helper so the steady-state memory footprint stays small.
"""

from __future__ import annotations

import functools
import json
import logging
import re
from dataclasses import dataclass, field
from pathlib import Path

import networkx as nx

LOGGER = logging.getLogger(__name__)

# Matches the wikilinks in GRAPH_REPORT's "## Community Hubs" section:
#   - [[_COMMUNITY_<id_or_name>|<display>]]
# The first capture is the raw target (may be a name with spaces stripped or a
# numbered "Community 200" placeholder), the second is the human display name.
_HUB_LINK_PATTERN = re.compile(r"\[\[_COMMUNITY_([^|\]]+)\|([^\]]+)\]\]")


@dataclass(frozen=True)
class GraphArtifact:
    """Loaded view of one graphify-out/ directory.

    `graph` is the live NetworkX graph; `community_labels` maps integer
    community id -> human-readable name; `community_index` is the
    case-insensitive reverse lookup (lowercase name -> id); `hub_order` is the
    list of community ids in the order they appear in GRAPH_REPORT's hub
    section (top of the list = highest-signal community).
    """

    graph_path: Path
    wiki_dir: Path
    built_at_commit: str
    node_count: int
    edge_count: int
    community_count: int
    wiki_page_count: int
    graph: nx.Graph = field(default_factory=nx.Graph)
    community_labels: dict[int, str] = field(default_factory=dict)
    community_index: dict[str, int] = field(default_factory=dict)
    hub_order: list[int] = field(default_factory=list)
    community_node_counts: dict[int, int] = field(default_factory=dict)


def load(graph_path: Path, wiki_dir: Path) -> GraphArtifact:
    if not graph_path.is_file():
        raise FileNotFoundError(f"graph file missing: {graph_path}")
    if not wiki_dir.is_dir():
        raise FileNotFoundError(f"wiki dir missing: {wiki_dir}")

    with graph_path.open("r", encoding="utf-8") as handle:
        data = json.load(handle)

    nodes = data.get("nodes", [])
    edges = data.get("links", data.get("edges", []))

    graph = _build_networkx_graph(data)
    community_labels = _load_community_labels(graph_path.parent)
    community_index = _build_community_index(community_labels)
    community_node_counts = _count_nodes_per_community(nodes)
    hub_order = _parse_hub_order(graph_path.parent / "GRAPH_REPORT.md", community_index)
    wiki_page_count = sum(1 for _ in wiki_dir.rglob("*.md"))

    LOGGER.info(
        "graph loaded: nodes=%d edges=%d communities=%d hubs=%d wiki=%d",
        len(nodes),
        len(edges),
        len(community_node_counts),
        len(hub_order),
        wiki_page_count,
    )

    return GraphArtifact(
        graph_path=graph_path,
        wiki_dir=wiki_dir,
        built_at_commit=_resolve_commit(graph_path, data),
        node_count=len(nodes),
        edge_count=len(edges),
        community_count=len(community_node_counts),
        wiki_page_count=wiki_page_count,
        graph=graph,
        community_labels=community_labels,
        community_index=community_index,
        hub_order=hub_order,
        community_node_counts=community_node_counts,
    )


@functools.lru_cache(maxsize=128)
def read_wiki_file(path: Path) -> str:
    """LRU-cached wiki page reader. Decoded as UTF-8."""
    return path.read_text(encoding="utf-8")


def _build_networkx_graph(data: dict) -> nx.Graph:
    """Convert the on-disk NodeLink JSON into a NetworkX graph.

    merged-graph.json is undirected (`directed: false`). We use the canonical
    `node_link_graph` reader so node/edge attributes survive.
    """
    directed = bool(data.get("directed", False))
    constructor: type[nx.Graph] = nx.DiGraph if directed else nx.Graph
    return nx.node_link_graph(
        data,
        directed=directed,
        multigraph=bool(data.get("multigraph", False)),
        edges="links",
    )


def _load_community_labels(graphify_dir: Path) -> dict[int, str]:
    labels_path = graphify_dir / ".graphify_labels.json"
    if not labels_path.is_file():
        return {}
    with labels_path.open("r", encoding="utf-8") as handle:
        raw = json.load(handle)
    out: dict[int, str] = {}
    for key, value in raw.items():
        try:
            out[int(key)] = str(value)
        except (TypeError, ValueError):
            continue
    return out


def _build_community_index(labels: dict[int, str]) -> dict[str, int]:
    index: dict[str, int] = {}
    for community_id, name in labels.items():
        index[name.lower()] = community_id
    return index


def _count_nodes_per_community(nodes: list[dict]) -> dict[int, int]:
    counts: dict[int, int] = {}
    for node in nodes:
        community = node.get("community")
        if community is None:
            continue
        try:
            community_id = int(community)
        except (TypeError, ValueError):
            continue
        counts[community_id] = counts.get(community_id, 0) + 1
    return counts


def _parse_hub_order(report_path: Path, community_index: dict[str, int]) -> list[int]:
    if not report_path.is_file():
        return []
    text = report_path.read_text(encoding="utf-8")
    # Only consider the "## Community Hubs" section to avoid matching links
    # elsewhere in the report.
    hubs_section_marker = "## Community Hubs"
    hubs_start = text.find(hubs_section_marker)
    if hubs_start == -1:
        return []
    hubs_text = text[hubs_start:]
    next_section = hubs_text.find("\n## ", len(hubs_section_marker))
    if next_section != -1:
        hubs_text = hubs_text[:next_section]

    order: list[int] = []
    seen: set[int] = set()
    for match in _HUB_LINK_PATTERN.finditer(hubs_text):
        display = match.group(2).strip()
        community_id = community_index.get(display.lower())
        if community_id is None or community_id in seen:
            continue
        order.append(community_id)
        seen.add(community_id)
    return order


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
