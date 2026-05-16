"""Tests for state models, step-log construction, and finding coercion."""

from __future__ import annotations

import operator
import typing

from odigos_agent.graph import (
    _coerce_finding,
    _format_findings_for_synthesis,
    _override_remediation_from_state,
    initial_state,
)
from odigos_agent.state import (
    AgentState,
    Finding,
    ProposedRemediation,
    Report,
    TriageResult,
    WorkloadInput,
    make_step,
)


def test_step_log_reducer_is_additive():
    """Parallel subgraphs must be able to append step_log entries without
    overwriting one another. The Annotated[..., operator.add] reducer in
    AgentState is what makes the ambiguous fan-out safe."""
    hints = typing.get_type_hints(AgentState, include_extras=True)
    metadata = getattr(hints["step_log"], "__metadata__", ())
    assert operator.add in metadata


def test_make_step_records_timestamp_and_detail():
    entry = make_step("source", "checked InstrumentationConfig", {"name": "deployment-foo"})
    assert entry["phase"] == "source"
    assert entry["action"] == "checked InstrumentationConfig"
    assert entry["detail"] == {"name": "deployment-foo"}
    assert entry["ts"] > 0


def test_coerce_finding_overwrites_phase():
    """The LLM may set phase="collector" while running in the source subgraph
    if the prompt context drags it that way. The phase field is owned by the
    routing layer, not the model."""
    raw = Finding(phase="collector", summary="hmm")
    forced = _coerce_finding(raw, phase="source")
    assert forced.phase == "source"
    assert forced.summary == "hmm"


def test_coerce_finding_from_dict():
    raw = {"phase": "destination", "summary": "ok", "evidence": ["e1"]}
    forced = _coerce_finding(raw, phase="source")
    assert forced.phase == "source"
    assert forced.evidence == ["e1"]


def test_coerce_finding_handles_none():
    forced = _coerce_finding(None, phase="collector")
    assert forced.phase == "collector"
    assert "no structured finding" in forced.summary


def test_initial_state_seeds_step_log():
    workload = WorkloadInput(namespace="default", kind="Deployment", name="payments")
    state = initial_state(workload)
    assert state["input_workload"] == workload
    assert len(state["step_log"]) == 1
    entry = state["step_log"][0]
    assert entry["phase"] == "input"
    assert entry["detail"]["namespace"] == "default"


def test_format_findings_includes_all_phases():
    state: AgentState = {
        "input_workload": WorkloadInput(namespace="ns", kind="Deployment", name="x"),
        "triage": TriageResult(
            classification="ambiguous",
            reasoning="multiple signals",
            symptoms_observed=["no spans at gateway", "exporter 401s"],
        ),
        "source_findings": Finding(phase="source", summary="OK from source side"),
        "collector_findings": Finding(
            phase="collector",
            summary="pipeline dropping at batch processor",
            evidence=["otelcol_processor_dropped_spans=412"],
            suggested_actions=["kubectl rollout restart deploy/odigos-gateway"],
        ),
        "destination_findings": Finding(
            phase="destination",
            summary="endpoint reachable, token rejected",
            evidence=["401 Unauthorized in gateway logs"],
        ),
        "step_log": [],
    }
    formatted = _format_findings_for_synthesis(state)
    assert "Triage classification: ambiguous" in formatted
    assert "[source]" in formatted
    assert "[collector]" in formatted
    assert "[destination]" in formatted
    assert "otelcol_processor_dropped_spans=412" in formatted
    assert "kubectl rollout restart" in formatted


def test_format_findings_skips_absent_phases():
    state: AgentState = {
        "input_workload": WorkloadInput(namespace="ns", kind="Deployment", name="x"),
        "triage": TriageResult(classification="source", reasoning="no Source"),
        "source_findings": Finding(phase="source", summary="missing Source CR"),
        "step_log": [],
    }
    formatted = _format_findings_for_synthesis(state)
    assert "[source]" in formatted
    assert "[collector]" not in formatted
    assert "[destination]" not in formatted


def test_synthesize_drops_hallucinated_remediation_when_state_has_none():
    """If the LLM ignores the synthesis prompt and fabricates a
    proposed_remediation, the override must strip it. The MCP approval cache
    is the single source of truth - a fake request_id reaching the UI would
    404 at apply time."""
    hallucinated = ProposedRemediation(
        op="create_source",
        request_id="totally-made-up",
        yaml="apiVersion: odigos.io/v1alpha1\n",
        diff="+ apiVersion: odigos.io/v1alpha1\n",
        rollback_command="kubectl delete source ...",
    )
    raw_report = Report(
        root_cause="source_not_instrumented",
        confidence=0.7,
        proposed_remediation=hallucinated,
    )
    state: AgentState = {
        "input_workload": WorkloadInput(namespace="ns", kind="Deployment", name="x"),
        "step_log": [],
    }
    final = _override_remediation_from_state(raw_report, state)
    assert final.proposed_remediation is None


def test_synthesize_preserves_real_remediation_from_state():
    real = ProposedRemediation(
        op="create_source",
        request_id="req-real",
        yaml="apiVersion: odigos.io/v1alpha1\n",
        diff="+ apiVersion: odigos.io/v1alpha1\n",
        rollback_command="kubectl delete source ...",
    )
    raw_report = Report(root_cause="source_not_instrumented", confidence=0.9)
    state: AgentState = {
        "input_workload": WorkloadInput(namespace="ns", kind="Deployment", name="x"),
        "proposed_remediation": real,
        "step_log": [],
    }
    final = _override_remediation_from_state(raw_report, state)
    assert final.proposed_remediation is not None
    assert final.proposed_remediation.request_id == "req-real"


def test_synthesize_overrides_hallucination_with_state_remediation():
    """When state has a real proposal and the LLM also hallucinated a
    different one, the state value wins."""
    real = ProposedRemediation(
        op="create_source",
        request_id="req-real",
        yaml="",
        diff="",
        rollback_command="",
    )
    fake = ProposedRemediation(
        op="create_source",
        request_id="req-fake",
        yaml="",
        diff="",
        rollback_command="",
    )
    raw_report = Report(
        root_cause="source_not_instrumented",
        confidence=0.5,
        proposed_remediation=fake,
    )
    state: AgentState = {
        "input_workload": WorkloadInput(namespace="ns", kind="Deployment", name="x"),
        "proposed_remediation": real,
        "step_log": [],
    }
    final = _override_remediation_from_state(raw_report, state)
    assert final.proposed_remediation is not None
    assert final.proposed_remediation.request_id == "req-real"


def test_report_with_proposed_remediation_roundtrip():
    proposed = ProposedRemediation(
        op="create_source",
        request_id="req-1",
        yaml="apiVersion: odigos.io/v1alpha1\n",
        diff="+ apiVersion: odigos.io/v1alpha1\n",
        rollback_command="kubectl delete source -n default ...",
    )
    report = Report(
        root_cause="source_not_instrumented",
        confidence=0.9,
        evidence=["no InstrumentationConfig"],
        suggested_actions=["create Source CR"],
        proposed_remediation=proposed,
    )
    dumped = report.model_dump()
    assert dumped["proposed_remediation"]["status"] == "pending_approval"
    assert dumped["root_cause"] == "source_not_instrumented"
