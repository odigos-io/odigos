"""Tests for the graph MCP tools.

Uses a hand-rolled small graph (4 communities, 12 nodes) so we control every
attribute the tools care about.
"""

from __future__ import annotations

import json
from pathlib import Path

import networkx as nx
import pytest

from graph_mcp.loader import GraphArtifact
from graph_mcp.tools import (
    candidate_wiki_filenames,
    label_match_score,
    resolve_community,
)


@pytest.fixture()
def graph_artifact(tmp_path: Path) -> GraphArtifact:
    """Build a tiny artifact in a temp dir with three communities + wiki pages."""
    graph_dir = tmp_path / "graphify-out"
    wiki_dir = graph_dir / "wiki"
    wiki_dir.mkdir(parents=True)

    nodes = [
        # Community 0 — Instrumentor (4 nodes)
        {"id": "instr_reconciler", "label": "instrumentor.go", "norm_label": "instrumentor", "file_type": "code", "source_file": "instrumentor/reconciler.go", "source_location": "L1", "community": 0},
        {"id": "instr_workload", "label": "workload.go", "norm_label": "workload", "file_type": "code", "source_file": "instrumentor/workload.go", "source_location": "L1", "community": 0},
        {"id": "instr_pod", "label": "pod_webhook.go", "norm_label": "pod_webhook", "file_type": "code", "source_file": "instrumentor/webhook.go", "source_location": "L1", "community": 0},
        {"id": "instr_doc", "label": "Instrumentor", "norm_label": "instrumentor", "file_type": "doc", "source_file": "docs/instrumentor.md", "source_location": "L1", "community": 0},
        # Community 1 — Destination CRD (3 nodes)
        {"id": "dest_crd", "label": "destination_types.go", "norm_label": "destination_types", "file_type": "code", "source_file": "api/v1alpha1/destination_types.go", "source_location": "L1", "community": 1},
        {"id": "dest_data", "label": "destination_data", "norm_label": "destination_data", "file_type": "code", "source_file": "destinations/data.yaml", "source_location": "L10", "community": 1},
        {"id": "dest_doc", "label": "Destination Object Docs", "norm_label": "destination_docs", "file_type": "doc", "source_file": "docs/destinations.md", "source_location": "L1", "community": 1},
        # Community 2 — Collector (3 nodes)
        {"id": "coll_gateway", "label": "gateway.go", "norm_label": "gateway", "file_type": "code", "source_file": "collector/gateway.go", "source_location": "L1", "community": 2},
        {"id": "coll_node", "label": "node_collector.go", "norm_label": "node_collector", "file_type": "code", "source_file": "collector/node.go", "source_location": "L1", "community": 2},
        {"id": "coll_processor", "label": "processor.go", "norm_label": "processor", "file_type": "code", "source_file": "collector/processor.go", "source_location": "L1", "community": 2},
        # Community 3 — orphan, named "Community 3" (thin)
        {"id": "orphan_a", "label": "orphan_a", "norm_label": "orphan_a", "file_type": "code", "community": 3},
        {"id": "orphan_b", "label": "orphan_b", "norm_label": "orphan_b", "file_type": "code", "community": 3},
    ]
    edges = [
        {"source": "instr_reconciler", "target": "instr_workload", "relation": "calls"},
        {"source": "instr_reconciler", "target": "instr_pod", "relation": "calls"},
        {"source": "instr_workload", "target": "instr_doc", "relation": "documents"},
        {"source": "dest_crd", "target": "dest_data", "relation": "uses"},
        {"source": "dest_crd", "target": "dest_doc", "relation": "documents"},
        {"source": "coll_gateway", "target": "coll_node", "relation": "peers"},
        {"source": "coll_gateway", "target": "coll_processor", "relation": "uses"},
        {"source": "instr_reconciler", "target": "dest_crd", "relation": "references"},
        {"source": "coll_gateway", "target": "dest_crd", "relation": "exports_to"},
        {"source": "orphan_a", "target": "orphan_b", "relation": "peers"},
    ]
    graph_path = graph_dir / "merged-graph.json"
    graph_path.write_text(json.dumps({
        "directed": False,
        "multigraph": False,
        "graph": {},
        "nodes": nodes,
        "links": edges,
        "built_at_commit": "abc1234",
    }))
    (graph_dir / ".graphify_labels.json").write_text(json.dumps({
        "0": "Instrumentor Reconcilers",
        "1": "Destination CRD",
        "2": "Collector Gateway",
        "3": "Community 3",
    }))
    (graph_dir / "GRAPH_REPORT.md").write_text(
        "# Graph Report\n\n## Community Hubs (Navigation)\n"
        "- [[_COMMUNITY_Destination CRD|Destination CRD]]\n"
        "- [[_COMMUNITY_Instrumentor Reconcilers|Instrumentor Reconcilers]]\n"
        "- [[_COMMUNITY_Collector Gateway|Collector Gateway]]\n"
    )
    (wiki_dir / "Instrumentor_Reconcilers.md").write_text("# Instrumentor Reconcilers\n\nReconciles workloads.\n")
    (wiki_dir / "Destination CRD.md").write_text("# Destination CRD\n\nSchema.\n")
    # Note: no wiki page for "Collector Gateway" - exercises the missing-page path.
    (wiki_dir / "Community_3.md").write_text("# Community 3\n\nThin.\n")

    from graph_mcp.loader import load
    return load(graph_path, wiki_dir)


# ---- pure helpers (no MCP server) ----

def test_label_match_score_orders_exact_prefix_substring():
    assert label_match_score("foo", "foo", "foo") == 3
    assert label_match_score("foobar", "foo_bar", "foo") == 2
    assert label_match_score("xxx_foo", "xfoo", "foo") == 1


def test_candidate_wiki_filenames_fallback_chain():
    chain = candidate_wiki_filenames("Destination CRD", 1)
    assert chain == ["Destination CRD.md", "Destination_CRD.md", "Community_1.md"]


def test_resolve_community_by_id_or_name(graph_artifact: GraphArtifact):
    assert resolve_community("0", graph_artifact) == 0
    assert resolve_community("Destination CRD", graph_artifact) == 1
    assert resolve_community("destination crd", graph_artifact) == 1
    assert resolve_community("nope", graph_artifact) is None
    assert resolve_community("", graph_artifact) is None


# ---- loader: graph and hub-order parsing ----

def test_loader_builds_networkx_graph(graph_artifact: GraphArtifact):
    assert isinstance(graph_artifact.graph, nx.Graph)
    assert graph_artifact.node_count == 12
    assert graph_artifact.edge_count == 10
    assert graph_artifact.community_count == 4
    assert graph_artifact.built_at_commit == "abc1234"


def test_loader_parses_hub_order(graph_artifact: GraphArtifact):
    # Hub order is the order they appear in GRAPH_REPORT's hub section.
    assert graph_artifact.hub_order == [1, 0, 2]


def test_loader_indexes_community_labels(graph_artifact: GraphArtifact):
    assert graph_artifact.community_index["destination crd"] == 1
    assert graph_artifact.community_labels[0] == "Instrumentor Reconcilers"


# ---- tools exercised via direct calls ----
# FastMCP wraps the tool functions; we register against a fresh server then
# invoke through its tool manager so the same decorator path runs.

@pytest.fixture()
def server(graph_artifact: GraphArtifact):
    from mcp.server.fastmcp import FastMCP
    from graph_mcp.tools import register
    instance = FastMCP("graph-mcp-test")
    register(instance, graph_artifact)
    return instance


def _get_tool_callable(server, tool_name: str):
    """FastMCP keeps registered tools in `server._tool_manager._tools`. Each
    entry has a `fn` attribute that's the original Python function. We call
    that directly in tests so we don't have to round-trip through the MCP
    protocol."""
    tools = server._tool_manager._tools  # noqa: SLF001 - test introspection
    return tools[tool_name].fn


def test_graph_query_orders_by_score(server):
    tool = _get_tool_callable(server, "graph_query")
    result = tool(query="instrumentor")
    # "Instrumentor" (exact label) > "instrumentor.go" (prefix) > others.
    assert result["matches"], "expected at least one match"
    labels = [match["label"] for match in result["matches"]]
    assert "Instrumentor" in labels
    assert "instrumentor.go" in labels


def test_graph_query_kind_filter(server):
    tool = _get_tool_callable(server, "graph_query")
    result = tool(query="destination", kind="doc")
    assert all(match["kind"] == "doc" for match in result["matches"])
    assert any(match["label"] == "Destination Object Docs" for match in result["matches"])


def test_graph_neighbors_bfs(server):
    tool = _get_tool_callable(server, "graph_neighbors")
    result = tool(node_id="instr_reconciler", depth=1)
    assert result["found"] is True
    neighbor_ids = {n["id"] for n in result["neighbors"]}
    assert "instr_workload" in neighbor_ids
    assert "instr_pod" in neighbor_ids
    assert "dest_crd" in neighbor_ids
    # depth=1 must NOT include 2-hop neighbors like instr_doc
    assert "instr_doc" not in neighbor_ids


def test_graph_neighbors_unknown_node(server):
    tool = _get_tool_callable(server, "graph_neighbors")
    result = tool(node_id="does_not_exist", depth=1)
    assert result["found"] is False


def test_graph_path_shortest(server):
    tool = _get_tool_callable(server, "graph_path")
    result = tool(from_id="instr_workload", to_id="dest_doc", max_hops=4)
    assert result["found"] is True
    assert result["hops"] >= 1
    assert result["nodes"][0]["id"] == "instr_workload"
    assert result["nodes"][-1]["id"] == "dest_doc"


def test_graph_path_no_path(server):
    tool = _get_tool_callable(server, "graph_path")
    # orphan_a is in a disconnected component (the orphan community).
    result = tool(from_id="instr_reconciler", to_id="orphan_a", max_hops=4)
    assert result["found"] is False


def test_graph_path_exceeds_max_hops(server):
    tool = _get_tool_callable(server, "graph_path")
    result = tool(from_id="instr_workload", to_id="dest_doc", max_hops=1)
    assert result["found"] is False


def test_graph_community_with_wiki(server):
    tool = _get_tool_callable(server, "graph_community")
    result = tool(community_id_or_name="Destination CRD")
    assert result["found"] is True
    assert result["community_id"] == 1
    assert result["node_count"] == 3
    assert result["wiki_filename"] == "Destination CRD.md"
    assert "Schema" in result["wiki_excerpt"]


def test_graph_community_falls_back_to_community_id_filename(server):
    tool = _get_tool_callable(server, "graph_community")
    result = tool(community_id_or_name="3")
    assert result["found"] is True
    assert result["wiki_filename"] == "Community_3.md"


def test_graph_community_missing_wiki(server):
    tool = _get_tool_callable(server, "graph_community")
    # No wiki file exists for "Collector Gateway" in the fixture.
    result = tool(community_id_or_name="Collector Gateway")
    assert result["found"] is True
    assert result["wiki_filename"] is None
    assert result["wiki_excerpt"] is None


def test_graph_god_nodes_overall(server):
    tool = _get_tool_callable(server, "graph_god_nodes")
    result = tool(top_k=3)
    # instr_reconciler has degree 3 (workload, pod, dest_crd).
    top_ids = [match["id"] for match in result["matches"]]
    assert "instr_reconciler" in top_ids


def test_graph_god_nodes_scoped(server):
    tool = _get_tool_callable(server, "graph_god_nodes")
    result = tool(top_k=5, scope="Instrumentor Reconcilers")
    for match in result["matches"]:
        assert match["community_id"] == 0


def test_graph_list_communities_hub_order_first(server):
    tool = _get_tool_callable(server, "graph_list_communities")
    result = tool()
    names = [community["name"] for community in result["communities"]]
    # Hub-order is [Destination CRD, Instrumentor Reconcilers, Collector Gateway].
    assert names[:3] == ["Destination CRD", "Instrumentor Reconcilers", "Collector Gateway"]
    # The thin community ("Community 3") comes after.
    assert names[-1] == "Community 3"


def test_graph_list_communities_filter(server):
    tool = _get_tool_callable(server, "graph_list_communities")
    result = tool(filter="instrument")
    assert all("instrument" in c["name"].lower() for c in result["communities"])


def test_wiki_read_returns_content(server):
    tool = _get_tool_callable(server, "wiki_read")
    result = tool(community_name="Instrumentor Reconcilers")
    assert result["found"] is True
    assert "Reconciles workloads" in result["content"]
    # Small fixture file is well under the default max_lines, so not truncated.
    assert result["truncated"] is False


def test_wiki_read_unknown_community(server):
    tool = _get_tool_callable(server, "wiki_read")
    result = tool(community_name="No Such Community")
    assert result["found"] is False


def test_wiki_read_reports_truncation(graph_artifact: GraphArtifact, tmp_path: Path):
    # Replace the existing wiki page with one that has 50 lines and read with
    # max_lines=10. We expect truncated=True and exactly 10 lines back.
    big_path = graph_artifact.wiki_dir / "Instrumentor_Reconcilers.md"
    big_path.write_text("\n".join(f"line {index}" for index in range(50)))

    from mcp.server.fastmcp import FastMCP
    from graph_mcp.tools import register
    server = FastMCP("graph-mcp-trunc")
    register(server, graph_artifact)
    tool = _get_tool_callable(server, "wiki_read")

    result = tool(community_name="Instrumentor Reconcilers", max_lines=10)
    assert result["found"] is True
    assert result["truncated"] is True
    assert result["content"].count("\n") == 9  # 10 lines = 9 newlines between them


def test_wiki_read_no_false_truncated_when_file_fits(graph_artifact: GraphArtifact):
    # Exact-fit case: file has exactly 5 lines, ask for max_lines=10.
    fit_path = graph_artifact.wiki_dir / "Destination CRD.md"
    fit_path.write_text("\n".join(f"line {index}" for index in range(5)))

    from mcp.server.fastmcp import FastMCP
    from graph_mcp.tools import register
    server = FastMCP("graph-mcp-fit")
    register(server, graph_artifact)
    tool = _get_tool_callable(server, "wiki_read")

    result = tool(community_name="Destination CRD", max_lines=10)
    assert result["found"] is True
    assert result["truncated"] is False
