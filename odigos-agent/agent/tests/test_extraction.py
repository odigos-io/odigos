"""Tests for the ProposedRemediation extractor.

The Phase 2 source subgraph captures a pending mutation by walking back
through the ReAct conversation for the most recent successful
`propose_create_source` ToolMessage. These tests cover the three content
shapes langchain-mcp-adapters can produce (string, content-block list,
dict) plus the failure / no-call paths.
"""

from __future__ import annotations

import json

from langchain_core.messages import AIMessage, HumanMessage, ToolMessage

from odigos_agent.graph import _extract_proposed_remediation


def _make_propose_tool_message(payload: dict, status: str = "success") -> ToolMessage:
    message = ToolMessage(
        content=json.dumps(payload),
        tool_call_id="call-1",
        name="propose_create_source",
    )
    message.status = status
    return message


def test_extracts_from_string_content():
    payload = {
        "request_id": "req-abc-123",
        "yaml": "apiVersion: odigos.io/v1alpha1\nkind: Source\n",
        "diff": "+ apiVersion: odigos.io/v1alpha1\n",
        "rollback_command": "kubectl delete source -n default -l x=y",
    }
    messages = [
        HumanMessage(content="hi"),
        AIMessage(content="thinking"),
        _make_propose_tool_message(payload),
    ]
    proposed = _extract_proposed_remediation(messages)
    assert proposed is not None
    assert proposed.op == "create_source"
    assert proposed.request_id == "req-abc-123"
    assert proposed.yaml.startswith("apiVersion")
    assert proposed.status == "pending_approval"


def test_extracts_from_content_block_list():
    payload = {
        "request_id": "req-block",
        "yaml": "y",
        "diff": "d",
        "rollback_command": "kubectl delete",
    }
    message = ToolMessage(
        content=[{"type": "text", "text": json.dumps(payload)}],
        tool_call_id="call-2",
        name="propose_create_source",
    )
    message.status = "success"
    proposed = _extract_proposed_remediation([message])
    assert proposed is not None
    assert proposed.request_id == "req-block"


def test_extracts_from_json_typed_content_block():
    """langchain-mcp-adapters may surface structured tool output as
    {"type": "json", "json": <dict>} blocks. The extractor must handle
    this transport shape, not just text blocks."""
    payload = {
        "request_id": "req-json-block",
        "yaml": "y",
        "diff": "d",
        "rollback_command": "kubectl delete",
    }
    message = ToolMessage(
        content=[{"type": "json", "json": payload}],
        tool_call_id="call-j",
        name="propose_create_source",
    )
    message.status = "success"
    proposed = _extract_proposed_remediation([message])
    assert proposed is not None
    assert proposed.request_id == "req-json-block"


def test_extracts_from_bare_dict_content_block():
    """Some transport versions surface raw dict blocks without a "type"
    wrapper. Treat them as the payload itself."""
    payload = {
        "request_id": "req-bare-dict",
        "yaml": "y",
        "diff": "d",
        "rollback_command": "kubectl delete",
    }
    message = ToolMessage(
        content=[payload],
        tool_call_id="call-b",
        name="propose_create_source",
    )
    message.status = "success"
    proposed = _extract_proposed_remediation([message])
    assert proposed is not None
    assert proposed.request_id == "req-bare-dict"


def test_returns_none_when_no_propose_call():
    messages = [
        HumanMessage(content="hi"),
        AIMessage(content="all good, fully instrumented"),
    ]
    assert _extract_proposed_remediation(messages) is None


def test_skips_failed_propose():
    failure = ToolMessage(
        content="Source already exists for foo/bar - nothing to create",
        tool_call_id="call-3",
        name="propose_create_source",
    )
    failure.status = "error"
    assert _extract_proposed_remediation([failure]) is None


def test_picks_most_recent_proposal():
    first = _make_propose_tool_message(
        {"request_id": "old", "yaml": "", "diff": "", "rollback_command": ""}
    )
    second = _make_propose_tool_message(
        {"request_id": "new", "yaml": "", "diff": "", "rollback_command": ""}
    )
    proposed = _extract_proposed_remediation([first, AIMessage(content="..."), second])
    assert proposed is not None
    assert proposed.request_id == "new"


def test_ignores_other_tool_calls():
    other = ToolMessage(
        content=json.dumps({"request_id": "unrelated"}),
        tool_call_id="call-x",
        name="get_source",
    )
    other.status = "success"
    assert _extract_proposed_remediation([other]) is None


def test_ignores_malformed_json():
    bad = ToolMessage(
        content="not json at all",
        tool_call_id="call-bad",
        name="propose_create_source",
    )
    bad.status = "success"
    assert _extract_proposed_remediation([bad]) is None
