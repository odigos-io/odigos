"""SSE event schema for the Phase 3 streaming API.

Each event is a small dataclass with an `event` name and a JSON-serializable
`data` payload. `to_sse_bytes()` returns the wire format that the
sse-starlette `EventSourceResponse` consumes.

Factory helpers exist for each event name documented in PLAN.md so the API
layer doesn't have to remember the exact key names.
"""

from __future__ import annotations

import json
from dataclasses import asdict, dataclass, field, is_dataclass
from typing import Any

from pydantic import BaseModel


def _default_encoder(value: Any) -> Any:
    if isinstance(value, BaseModel):
        return value.model_dump(mode="json")
    if is_dataclass(value):
        return asdict(value)
    if isinstance(value, (set, frozenset)):
        return list(value)
    raise TypeError(f"unserializable type {type(value).__name__}")


def encode_data(data: dict[str, Any]) -> str:
    return json.dumps(data, default=_default_encoder, ensure_ascii=False)


@dataclass
class SSEEvent:
    """A single Server-Sent Event.

    `event` is the SSE event name. `data` is a JSON object that gets
    serialized to the `data:` line. We never split data across multiple
    lines - sse-starlette and EventSource handle the single-line case.
    """

    event: str
    data: dict[str, Any] = field(default_factory=dict)

    def to_sse_dict(self) -> dict[str, str]:
        return {"event": self.event, "data": encode_data(self.data)}

    def to_sse_bytes(self) -> bytes:
        return f"event: {self.event}\ndata: {encode_data(self.data)}\n\n".encode("utf-8")


def session(session_id: str) -> SSEEvent:
    return SSEEvent("session", {"debug_session_id": session_id})


def step(phase: str, message: str, ts: float, detail: dict | None = None) -> SSEEvent:
    payload: dict[str, Any] = {"phase": phase, "message": message, "ts": ts}
    if detail:
        payload["detail"] = detail
    return SSEEvent("step", payload)


def knowledge_query(phase: str, tool: str, query: str) -> SSEEvent:
    return SSEEvent("knowledge_query", {"phase": phase, "tool": tool, "query": query})


def codebase_read(path: str, lines: str, phase: str | None = None) -> SSEEvent:
    payload: dict[str, Any] = {"path": path, "lines": lines}
    if phase:
        payload["phase"] = phase
    return SSEEvent("codebase_read", payload)


def finding(phase: str, summary: str) -> SSEEvent:
    return SSEEvent("finding", {"phase": phase, "summary": summary})


def approval_required(
    request_id: str,
    op: str,
    yaml: str,
    diff: str,
    rollback_command: str,
) -> SSEEvent:
    return SSEEvent(
        "approval_required",
        {
            "request_id": request_id,
            "op": op,
            "yaml": yaml,
            "diff": diff,
            "rollback_command": rollback_command,
        },
    )


def approval_resolved(
    request_id: str, decision: str, result: str | None = None
) -> SSEEvent:
    payload: dict[str, Any] = {"request_id": request_id, "decision": decision}
    if result is not None:
        payload["result"] = result
    return SSEEvent("approval_resolved", payload)


def report(report_payload: dict[str, Any]) -> SSEEvent:
    return SSEEvent("report", report_payload)


def done() -> SSEEvent:
    return SSEEvent("done", {})


def error(message: str, code: str | None = None) -> SSEEvent:
    payload: dict[str, Any] = {"message": message}
    if code:
        payload["code"] = code
    return SSEEvent("error", payload)
