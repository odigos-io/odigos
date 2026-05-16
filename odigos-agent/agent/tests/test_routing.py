"""Routing-function tests.

These exercise pure helper functions in `odigos_agent.graph` without
spinning up an LLM or any MCP transport. The dispatch behaviour is the
load-bearing piece of Phase 2 - if triage classifies wrong, the wrong
subgraph runs and the report is meaningless.
"""

from __future__ import annotations

from langchain_core.tools import BaseTool, tool

from odigos_agent.graph import partition_tools, route_after_triage
from odigos_agent.state import (
    AgentState,
    TriageResult,
    WorkloadInput,
)


def _make_stub(name: str) -> BaseTool:
    @tool(name)
    def stub() -> str:
        """Stub tool for tests."""
        return "ok"

    return stub


def test_route_unambiguous_source():
    state: AgentState = {
        "triage": TriageResult(classification="source", reasoning="no Source CR"),
    }
    assert route_after_triage(state) == ["source"]


def test_route_unambiguous_collector():
    state: AgentState = {
        "triage": TriageResult(classification="collector", reasoning="exporter dropped"),
    }
    assert route_after_triage(state) == ["collector"]


def test_route_unambiguous_destination():
    state: AgentState = {
        "triage": TriageResult(classification="destination", reasoning="exporter 401"),
    }
    assert route_after_triage(state) == ["destination"]


def test_route_ambiguous_fans_out_to_all_three():
    state: AgentState = {
        "triage": TriageResult(classification="ambiguous", reasoning="signals overlap"),
    }
    assert sorted(route_after_triage(state)) == ["collector", "destination", "source"]


def test_route_unknown_skips_to_synthesize():
    state: AgentState = {
        "triage": TriageResult(classification="unknown", reasoning="no signal"),
    }
    assert route_after_triage(state) == ["synthesize"]


def test_route_missing_triage_skips_to_synthesize():
    state: AgentState = {
        "input_workload": WorkloadInput(namespace="default", kind="Deployment", name="payments"),
    }
    assert route_after_triage(state) == ["synthesize"]


def test_partition_tools_buckets_by_name():
    tools = [
        _make_stub("get_source"),
        _make_stub("get_instrumentation_config"),
        _make_stub("list_workload_pods"),
        _make_stub("propose_create_source"),
        _make_stub("get_collector_config"),
        _make_stub("get_collector_metrics"),
        _make_stub("list_destinations"),
        _make_stub("probe_destination_endpoint"),
        _make_stub("graph_query"),
        _make_stub("wiki_read"),
        _make_stub("gh_read_file"),
        _make_stub("unrelated_tool"),
    ]
    partitioned = partition_tools(tools)

    source_names = {tool.name for tool in partitioned["source"]}
    collector_names = {tool.name for tool in partitioned["collector"]}
    destination_names = {tool.name for tool in partitioned["destination"]}
    triage_names = {tool.name for tool in partitioned["triage"]}
    graph_names = {tool.name for tool in partitioned["graph"]}

    assert "get_source" in source_names
    assert "propose_create_source" in source_names
    assert "graph_query" in source_names, "graph tools available to source subgraph"

    assert "get_collector_config" in collector_names
    assert "get_collector_metrics" in collector_names
    assert "graph_query" in collector_names
    assert "get_source" not in collector_names

    assert "list_destinations" in destination_names
    assert "probe_destination_endpoint" in destination_names
    assert "graph_query" in destination_names
    assert "get_collector_config" not in destination_names

    assert triage_names == {"get_source", "get_instrumentation_config", "list_workload_pods"}
    assert "graph_query" in graph_names
    assert "wiki_read" in graph_names
    assert "unrelated_tool" not in graph_names


def test_partition_excludes_apply_create_source():
    """apply_create_source is intentionally never exposed to the source subgraph
    in Phase 2 - the apply step lives in the Phase 3 API layer after human
    approval. If the tool ever leaks into the catalog, the LLM might call it
    directly and bypass the approval gate."""
    tools = [
        _make_stub("propose_create_source"),
        _make_stub("apply_create_source"),
        _make_stub("get_source"),
    ]
    partitioned = partition_tools(tools)
    source_names = {tool.name for tool in partitioned["source"]}
    assert "propose_create_source" in source_names
    assert "apply_create_source" not in source_names
