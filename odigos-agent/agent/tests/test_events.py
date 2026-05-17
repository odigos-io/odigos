"""Tests for SSE event encoding."""

from __future__ import annotations

import json

from odigos_agent import events as ev
from odigos_agent.state import ProposedRemediation


def test_session_event_wire_format():
    event = ev.session("abc123")
    text = event.to_sse_bytes().decode()
    assert text.startswith("event: session\n")
    assert "data: " in text
    assert text.endswith("\n\n")
    data = _parse_data(text)
    assert data == {"debug_session_id": "abc123"}


def test_step_event_includes_phase_ts_message():
    event = ev.step(phase="source", message="reading config", ts=1234.5)
    data = _parse_data(event.to_sse_bytes().decode())
    assert data["phase"] == "source"
    assert data["message"] == "reading config"
    assert data["ts"] == 1234.5
    assert "detail" not in data


def test_step_event_optional_detail():
    event = ev.step(phase="x", message="m", ts=0.0, detail={"k": "v"})
    data = _parse_data(event.to_sse_bytes().decode())
    assert data["detail"] == {"k": "v"}


def test_approval_required_event_carries_full_payload():
    event = ev.approval_required(
        request_id="r1",
        op="create_source",
        yaml="apiVersion: x\n",
        diff="+ apiVersion: x\n",
        rollback_command="kubectl delete ...",
    )
    data = _parse_data(event.to_sse_bytes().decode())
    assert data["request_id"] == "r1"
    assert data["op"] == "create_source"
    assert data["yaml"].startswith("apiVersion:")
    assert data["diff"].startswith("+ ")
    assert data["rollback_command"].startswith("kubectl")


def test_approval_resolved_event_optional_result():
    e1 = ev.approval_resolved("r1", "deny")
    assert "result" not in _parse_data(e1.to_sse_bytes().decode())
    e2 = ev.approval_resolved("r1", "approve", result="created")
    assert _parse_data(e2.to_sse_bytes().decode())["result"] == "created"


def test_done_event_wire_format():
    text = ev.done().to_sse_bytes().decode()
    assert text.startswith("event: done\n")


def test_report_event_handles_pydantic_payload():
    """Report payloads may contain pydantic models nested inside dicts.
    The encoder must serialize them without choking."""
    proposed = ProposedRemediation(
        op="create_source",
        request_id="r1",
        yaml="y",
        diff="d",
        rollback_command="rc",
    )
    payload = {"root_cause": "source_not_instrumented", "proposed_remediation": proposed}
    event = ev.report(payload)
    data = _parse_data(event.to_sse_bytes().decode())
    assert data["proposed_remediation"]["request_id"] == "r1"


def test_to_sse_dict_keys():
    event = ev.session("s")
    d = event.to_sse_dict()
    assert d["event"] == "session"
    assert json.loads(d["data"]) == {"debug_session_id": "s"}


def _parse_data(wire: str) -> dict:
    for line in wire.split("\n"):
        if line.startswith("data: "):
            return json.loads(line[len("data: ") :])
    raise AssertionError(f"no data line in {wire!r}")
