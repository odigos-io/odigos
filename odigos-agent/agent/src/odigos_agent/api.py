"""FastAPI HTTP surface for the odigos diagnostic agent.

Endpoints:

- POST /debug              -> SSE stream of agent reasoning + final report.
- POST /approve/{sid}/{rid} -> resolves a pending human-in-loop approval.
- GET  /healthz            -> liveness; always 200.
- GET  /readyz             -> readiness; 503 until MCP tools load.

The /debug handler drives `graph.astream(...)` and maps the heterogeneous
chunks (updates, messages, custom) to typed SSE events. When the
`apply_remediation` node hits `interrupt(...)`, this handler emits
`approval_required`, awaits the registry, then resumes the graph with the
decision via `Command(resume=...)`.
"""

from __future__ import annotations

import asyncio
import json
import logging
import os
import uuid
from contextlib import asynccontextmanager
from dataclasses import dataclass
from typing import Any, AsyncIterator

from fastapi import Depends, FastAPI, HTTPException, Request, Response, status
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer
from langchain_core.messages import AIMessage, AIMessageChunk, ToolMessage
from langgraph.checkpoint.memory import MemorySaver
from langgraph.types import Command
from pydantic import BaseModel, Field
from sse_starlette.sse import EventSourceResponse

from . import events as ev
from .approvals import ApprovalRegistry, Decision
from .graph import (
    DEFAULT_MODEL,
    _GRAPH_TOOL_NAMES,
    build_graph,
    initial_state,
)
from .mcp_client import McpEndpoints, load_tools
from .state import WorkloadInput

logger = logging.getLogger(__name__)

APPROVAL_TIMEOUT_SECONDS = float(os.environ.get("APPROVAL_TIMEOUT_SECONDS", "300"))


class DebugRequest(BaseModel):
    namespace: str = Field(min_length=1)
    kind: str = Field(min_length=1)
    name: str = Field(min_length=1)


class ApproveRequest(BaseModel):
    decision: Decision


@dataclass
class _AppState:
    graph: Any
    approvals: ApprovalRegistry
    ready: bool
    ready_error: str | None


_bearer = HTTPBearer(auto_error=False)


def _expected_token() -> str | None:
    token = os.environ.get("ODIGOS_AGENT_TOKEN")
    return token or None


async def _require_bearer(
    creds: HTTPAuthorizationCredentials | None = Depends(_bearer),
) -> None:
    expected = _expected_token()
    if expected is None:
        # No token configured -> allow (dev convenience). Production sets
        # the env var so this branch never fires.
        return
    if creds is None or creds.scheme.lower() != "bearer" or creds.credentials != expected:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="invalid or missing bearer token",
        )


@asynccontextmanager
async def lifespan(app: FastAPI):
    approvals = ApprovalRegistry()
    try:
        tools = await load_tools(McpEndpoints.from_env())
        model = os.environ.get("ODIGOS_AGENT_MODEL", DEFAULT_MODEL)
        graph = build_graph(tools, model=model, checkpointer=MemorySaver())
        app.state.agent = _AppState(
            graph=graph, approvals=approvals, ready=True, ready_error=None
        )
        logger.info("agent ready: loaded %d MCP tools", len(tools))
    except Exception as exc:  # noqa: BLE001
        logger.exception("startup failed: %s", exc)
        app.state.agent = _AppState(
            graph=None,
            approvals=approvals,
            ready=False,
            ready_error=f"{type(exc).__name__}: {exc}",
        )
    try:
        yield
    finally:
        pass


app = FastAPI(title="odigos-agent", lifespan=lifespan)


@app.get("/healthz")
async def healthz() -> dict[str, str]:
    return {"status": "ok"}


@app.get("/readyz")
async def readyz(response: Response) -> dict[str, Any]:
    agent: _AppState = app.state.agent
    if not agent.ready:
        response.status_code = status.HTTP_503_SERVICE_UNAVAILABLE
        return {"status": "not_ready", "error": agent.ready_error}
    return {"status": "ready"}


@app.post("/debug", dependencies=[Depends(_require_bearer)])
async def debug(req: DebugRequest, request: Request) -> EventSourceResponse:
    agent: _AppState = app.state.agent
    if not agent.ready or agent.graph is None:
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail=agent.ready_error or "agent not ready",
        )
    workload = WorkloadInput(namespace=req.namespace, kind=req.kind, name=req.name)
    session_id = uuid.uuid4().hex
    stream = _run_debug_session(
        graph=agent.graph,
        approvals=agent.approvals,
        workload=workload,
        session_id=session_id,
        client_disconnect=request.is_disconnected,
    )
    return EventSourceResponse(_to_sse_dicts(stream))


@app.post(
    "/approve/{session_id}/{request_id}",
    status_code=status.HTTP_204_NO_CONTENT,
    dependencies=[Depends(_require_bearer)],
)
async def approve(session_id: str, request_id: str, body: ApproveRequest) -> Response:
    agent: _AppState = app.state.agent
    ok = await agent.approvals.resolve(session_id, request_id, body.decision)
    if not ok:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="no pending approval for that session+request_id",
        )
    return Response(status_code=status.HTTP_204_NO_CONTENT)


async def _to_sse_dicts(stream: AsyncIterator[ev.SSEEvent]) -> AsyncIterator[dict]:
    async for event in stream:
        yield event.to_sse_dict()


async def _run_debug_session(
    *,
    graph: Any,
    approvals: ApprovalRegistry,
    workload: WorkloadInput,
    session_id: str,
    client_disconnect,
) -> AsyncIterator[ev.SSEEvent]:
    """Drive the LangGraph workflow and yield SSE events.

    Handles a single approval interrupt round-trip. v1 emits at most one
    `create_source` mutation per session so a single resume covers it.
    """
    yield ev.session(session_id)
    config = {
        "configurable": {
            "thread_id": session_id,
            "enable_approval_interrupt": True,
        }
    }
    pending_interrupt: dict[str, Any] | None = None
    mapper = _StreamMapper()
    try:
        async for event in _iter_graph_stream(
            graph, initial_state(workload), config, mapper
        ):
            if isinstance(event, _InterruptSignal):
                pending_interrupt = event.payload
                break
            yield event

        if pending_interrupt is not None:
            request_id = str(pending_interrupt.get("request_id", ""))
            if not request_id:
                yield ev.error("interrupt missing request_id")
                return
            await approvals.register(session_id, request_id)
            yield ev.approval_required(
                request_id=request_id,
                op=str(pending_interrupt.get("op", "")),
                yaml=str(pending_interrupt.get("yaml", "")),
                diff=str(pending_interrupt.get("diff", "")),
                rollback_command=str(pending_interrupt.get("rollback_command", "")),
            )
            decision = await approvals.wait(
                session_id, request_id, timeout=APPROVAL_TIMEOUT_SECONDS
            )
            yield ev.approval_resolved(request_id=request_id, decision=decision)

            resume_payload = {"decision": decision}
            async for event in _iter_graph_stream(
                graph, Command(resume=resume_payload), config, mapper
            ):
                if isinstance(event, _InterruptSignal):
                    # v1 only supports a single mutation per session.
                    yield ev.error(
                        "unexpected second interrupt in session", code="too_many_interrupts"
                    )
                    return
                yield event
    except asyncio.CancelledError:
        raise
    except Exception as exc:  # noqa: BLE001
        logger.exception("debug session %s failed", session_id)
        yield ev.error(f"{type(exc).__name__}: {exc}", code="agent_error")
    finally:
        await approvals.cleanup(session_id)
        yield ev.done()


@dataclass
class _InterruptSignal:
    payload: dict[str, Any]


class _StreamMapper:
    """Converts heterogeneous astream chunks to SSE events.

    Stateful: tracks the last-known node phase so step events get a stable
    `phase` field, and tracks AIMessage tool_calls so we can emit
    `knowledge_query` / `codebase_read` events with their arguments when the
    matching ToolMessage arrives.
    """

    def __init__(self) -> None:
        self._current_phase: str = "input"
        self._tool_calls: dict[str, dict[str, Any]] = {}

    def updates_to_events(self, chunk: dict[str, Any]) -> list[ev.SSEEvent]:
        events: list[ev.SSEEvent] = []
        for node_name, update in chunk.items():
            if node_name == "__interrupt__":
                continue
            if isinstance(update, dict):
                self._current_phase = node_name
                events.extend(self._update_to_events(node_name, update))
        return events

    def _update_to_events(
        self, node_name: str, update: dict[str, Any]
    ) -> list[ev.SSEEvent]:
        events: list[ev.SSEEvent] = []
        for entry in update.get("step_log") or []:
            phase = str(entry.get("phase") or node_name)
            action = str(entry.get("action") or "")
            ts = float(entry.get("ts") or 0.0)
            detail = entry.get("detail") if isinstance(entry.get("detail"), dict) else None
            events.append(ev.step(phase=phase, message=action, ts=ts, detail=detail))
        for phase in ("source", "collector", "destination"):
            finding = update.get(f"{phase}_findings")
            if finding is None:
                continue
            summary = getattr(finding, "summary", None) or (
                finding.get("summary") if isinstance(finding, dict) else ""
            )
            events.append(ev.finding(phase=phase, summary=str(summary)))
        report = update.get("report")
        if report is not None:
            events.append(ev.report(_to_jsonable(report)))
        return events

    def message_to_events(self, message: Any) -> list[ev.SSEEvent]:
        events: list[ev.SSEEvent] = []
        if isinstance(message, (AIMessage, AIMessageChunk)):
            for call in getattr(message, "tool_calls", None) or []:
                tool_id = call.get("id") if isinstance(call, dict) else None
                if not tool_id:
                    continue
                name = str(call.get("name") or "") if isinstance(call, dict) else ""
                args = call.get("args") if isinstance(call, dict) else None
                self._tool_calls[tool_id] = {"name": name, "args": args or {}}
                if name in _GRAPH_TOOL_NAMES and name != "gh_read_file":
                    events.append(
                        ev.knowledge_query(
                            phase=self._current_phase,
                            tool=name,
                            query=_render_query(args),
                        )
                    )
        elif isinstance(message, ToolMessage):
            tool_id = getattr(message, "tool_call_id", None)
            recorded = self._tool_calls.pop(tool_id, None) if tool_id else None
            tool_name = getattr(message, "name", None) or (
                recorded["name"] if recorded else ""
            )
            if tool_name == "gh_read_file":
                args = (recorded or {}).get("args") or {}
                events.append(
                    ev.codebase_read(
                        path=str(args.get("path") or args.get("file_path") or ""),
                        lines=_render_lines(args),
                        phase=self._current_phase,
                    )
                )
        return events


async def _iter_graph_stream(
    graph: Any,
    input_value: Any,
    config: dict[str, Any],
    mapper: _StreamMapper,
) -> AsyncIterator[ev.SSEEvent | _InterruptSignal]:
    """Iterate astream and yield SSE events or an _InterruptSignal sentinel."""
    async for mode, chunk in graph.astream(
        input_value,
        config=config,
        stream_mode=["updates", "messages", "custom"],
    ):
        if mode == "updates":
            if isinstance(chunk, dict) and "__interrupt__" in chunk:
                payload = _extract_interrupt_payload(chunk["__interrupt__"])
                if payload is not None:
                    yield _InterruptSignal(payload=payload)
                    return
            if isinstance(chunk, dict):
                for event in mapper.updates_to_events(chunk):
                    yield event
        elif mode == "messages":
            message = chunk[0] if isinstance(chunk, tuple) and chunk else None
            if message is not None:
                for event in mapper.message_to_events(message):
                    yield event
        # mode == "custom": ignored in v1


def _extract_interrupt_payload(value: Any) -> dict[str, Any] | None:
    """Pull the dict passed to `interrupt(...)` out of an Interrupt object/tuple."""
    if value is None:
        return None
    if isinstance(value, dict):
        return value
    if isinstance(value, (list, tuple)):
        for item in value:
            payload = _extract_interrupt_payload(item)
            if payload is not None:
                return payload
        return None
    inner = getattr(value, "value", None)
    if isinstance(inner, dict):
        return inner
    return None


def _to_jsonable(value: Any) -> dict[str, Any]:
    if isinstance(value, BaseModel):
        return value.model_dump(mode="json")
    if isinstance(value, dict):
        return value
    try:
        return json.loads(json.dumps(value, default=str))
    except (TypeError, ValueError):
        return {"value": str(value)}


def _render_query(args: Any) -> str:
    if not args:
        return ""
    if isinstance(args, dict):
        for key in ("query", "q", "name", "path", "file_path"):
            value = args.get(key)
            if value:
                return str(value)
        try:
            return json.dumps(args, default=str)
        except (TypeError, ValueError):
            return str(args)
    return str(args)


def _render_lines(args: Any) -> str:
    if not isinstance(args, dict):
        return ""
    start = args.get("start_line") or args.get("start")
    end = args.get("end_line") or args.get("end")
    if start and end:
        return f"{start}-{end}"
    if start:
        return str(start)
    return ""
