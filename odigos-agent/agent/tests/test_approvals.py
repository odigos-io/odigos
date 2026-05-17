"""Tests for the in-memory ApprovalRegistry."""

from __future__ import annotations

import asyncio

import pytest

from odigos_agent.approvals import ApprovalRegistry


@pytest.mark.asyncio
async def test_resolve_unblocks_waiter():
    registry = ApprovalRegistry()
    await registry.register("s1", "r1")

    async def resolver():
        await asyncio.sleep(0.05)
        ok = await registry.resolve("s1", "r1", "approve")
        assert ok is True

    decision_task = asyncio.create_task(registry.wait("s1", "r1", timeout=2.0))
    await asyncio.gather(resolver())
    decision = await decision_task
    assert decision == "approve"


@pytest.mark.asyncio
async def test_wait_times_out_when_no_decision():
    registry = ApprovalRegistry()
    await registry.register("s1", "r1")
    decision = await registry.wait("s1", "r1", timeout=0.05)
    assert decision == "timed_out"


@pytest.mark.asyncio
async def test_resolve_after_timeout_returns_false():
    registry = ApprovalRegistry()
    await registry.register("s1", "r1")
    decision = await registry.wait("s1", "r1", timeout=0.05)
    assert decision == "timed_out"
    ok = await registry.resolve("s1", "r1", "approve")
    assert ok is False


@pytest.mark.asyncio
async def test_resolve_unknown_returns_false():
    registry = ApprovalRegistry()
    ok = await registry.resolve("nope", "nope", "approve")
    assert ok is False


@pytest.mark.asyncio
async def test_double_resolve_returns_false_second_time():
    registry = ApprovalRegistry()
    await registry.register("s1", "r1")
    assert await registry.resolve("s1", "r1", "approve") is True
    assert await registry.resolve("s1", "r1", "deny") is False


@pytest.mark.asyncio
async def test_wait_without_register_raises():
    registry = ApprovalRegistry()
    with pytest.raises(KeyError):
        await registry.wait("missing", "r", timeout=0.01)


@pytest.mark.asyncio
async def test_cleanup_removes_session_entries():
    registry = ApprovalRegistry()
    await registry.register("s1", "r1")
    await registry.register("s1", "r2")
    await registry.register("s2", "r1")
    await registry.cleanup("s1")
    assert await registry.pending_count() == 1
    # Resolving s2/r1 must still work (untouched by cleanup of s1)
    assert await registry.resolve("s2", "r1", "deny") is True


@pytest.mark.asyncio
async def test_deny_decision_propagates():
    registry = ApprovalRegistry()
    await registry.register("s1", "r1")
    asyncio.create_task(_delayed_resolve(registry, "s1", "r1", "deny"))
    decision = await registry.wait("s1", "r1", timeout=2.0)
    assert decision == "deny"


async def _delayed_resolve(registry: ApprovalRegistry, sid: str, rid: str, decision):
    await asyncio.sleep(0.02)
    await registry.resolve(sid, rid, decision)
