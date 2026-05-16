"""Agent state and structured-output schemas.

`AgentState` is a TypedDict because LangGraph reduces over it. Inner records
use pydantic to play well with `with_structured_output` and to give the agent
predictable JSON in tool messages.

`step_log` uses `operator.add` so parallel subgraphs (the ambiguous-triage
fan-out) can each append without clobbering one another.
"""

from __future__ import annotations

import operator
import time
from typing import Annotated, Literal, TypedDict

from pydantic import BaseModel, Field


Classification = Literal[
    "source",
    "collector",
    "destination",
    "ambiguous",
    "unknown",
]


RootCause = Literal[
    "source_not_instrumented",
    "collector_misconfigured",
    "destination_misconfigured",
    "unknown",
]


RemediationStatus = Literal[
    "pending_approval",
    "approved_applied",
    "denied",
    "timed_out",
    "failed",
]


class WorkloadInput(BaseModel):
    namespace: str
    kind: str
    name: str


class TriageResult(BaseModel):
    """Output of the triage node.

    The classification drives the routing decision after triage. `reasoning`
    is a short human-readable summary that lands in the step log and the
    final report. Triage runs before any deep diagnosis - keep the prompt
    cheap and the classification a single dispatch choice.
    """

    classification: Classification
    reasoning: str
    symptoms_observed: list[str] = Field(default_factory=list)


class Finding(BaseModel):
    """A single subgraph's output."""

    phase: Literal["source", "collector", "destination"]
    summary: str
    evidence: list[str] = Field(default_factory=list)
    suggested_actions: list[str] = Field(default_factory=list)


class ProposedRemediation(BaseModel):
    """A pending or resolved mutation proposal.

    Phase 2 only produces `status="pending_approval"`. Phase 3 will resume
    the graph after a user approve/deny decision and update `status` +
    `result` accordingly.
    """

    op: Literal["create_source"]
    request_id: str
    yaml: str
    diff: str
    rollback_command: str
    status: RemediationStatus = "pending_approval"
    result: str | None = None


class Report(BaseModel):
    """Final structured output emitted by the synthesize node."""

    root_cause: RootCause
    confidence: float = Field(ge=0.0, le=1.0)
    evidence: list[str] = Field(default_factory=list)
    suggested_actions: list[str] = Field(default_factory=list)
    proposed_remediation: ProposedRemediation | None = None


class StepLogEntry(TypedDict, total=False):
    phase: str
    action: str
    ts: float
    detail: dict


def make_step(phase: str, action: str, detail: dict | None = None) -> StepLogEntry:
    """Build a step-log entry with a UTC epoch timestamp.

    The step log is the audit trail Phase 3 will stream as SSE `step` events.
    Keep entries small and structured so the UI can render them as a feed.
    """
    entry: StepLogEntry = {"phase": phase, "action": action, "ts": time.time()}
    if detail is not None:
        entry["detail"] = detail
    return entry


class AgentState(TypedDict, total=False):
    """LangGraph state.

    `step_log` carries an `operator.add` reducer so parallel subgraph nodes
    (when triage classifies the symptom as "ambiguous") can each contribute
    entries without overwriting one another.
    """

    input_workload: WorkloadInput
    triage: TriageResult | None
    source_findings: Finding | None
    collector_findings: Finding | None
    destination_findings: Finding | None
    proposed_remediation: ProposedRemediation | None
    report: Report | None
    step_log: Annotated[list[StepLogEntry], operator.add]
