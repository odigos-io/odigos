"""In-memory async approval registry.

The agent pauses on a LangGraph `interrupt(...)` when it has a remediation
to apply. The API handler awaits `ApprovalRegistry.wait(...)`. A separate
HTTP request to `POST /approve/{session}/{request_id}` calls
`ApprovalRegistry.resolve(...)`, which sets the `asyncio.Event` and unblocks
the waiter.

Scoped per process. v1 runs a single agent replica so this is sufficient.
"""

from __future__ import annotations

import asyncio
from dataclasses import dataclass, field
from typing import Literal


Decision = Literal["approve", "deny", "timed_out"]


@dataclass
class _PendingApproval:
    event: asyncio.Event = field(default_factory=asyncio.Event)
    decision: Decision | None = None


class ApprovalRegistry:
    """Tracks pending human-in-loop approvals keyed by (session_id, request_id)."""

    def __init__(self) -> None:
        self._pending: dict[tuple[str, str], _PendingApproval] = {}
        self._lock = asyncio.Lock()

    async def register(self, session_id: str, request_id: str) -> None:
        async with self._lock:
            self._pending[(session_id, request_id)] = _PendingApproval()

    async def wait(
        self, session_id: str, request_id: str, timeout: float
    ) -> Decision:
        """Block until resolved or timeout.

        Returns the resolved decision, or "timed_out" if no decision arrived
        in `timeout` seconds. On timeout the entry is left in place so a
        late-arriving POST still sees the slot (and returns 404 because
        decision is set); cleanup happens via `cleanup(session_id)`.
        """
        async with self._lock:
            pending = self._pending.get((session_id, request_id))
            if pending is None:
                raise KeyError(f"no pending approval for {session_id}/{request_id}")

        try:
            await asyncio.wait_for(pending.event.wait(), timeout=timeout)
        except asyncio.TimeoutError:
            async with self._lock:
                if pending.decision is None:
                    pending.decision = "timed_out"
                    pending.event.set()
        return pending.decision or "timed_out"

    async def resolve(
        self, session_id: str, request_id: str, decision: Decision
    ) -> bool:
        """Set the decision. Returns False if no pending request or already resolved."""
        async with self._lock:
            pending = self._pending.get((session_id, request_id))
            if pending is None or pending.decision is not None:
                return False
            pending.decision = decision
            pending.event.set()
            return True

    async def cleanup(self, session_id: str) -> None:
        async with self._lock:
            for key in [k for k in self._pending if k[0] == session_id]:
                del self._pending[key]

    async def pending_count(self) -> int:
        async with self._lock:
            return len(self._pending)
