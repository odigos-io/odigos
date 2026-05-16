"""LangGraph diagnostic workflow.

Triage classifies the symptom, then the matching subgraph(s) run as small
ReAct loops bound to their domain's MCP tools. The synthesize stage merges
the per-domain Findings into a single structured Report.

Phase 2 only PROPOSES mutations - it never applies. If the source subgraph
calls `propose_create_source`, the resulting request_id + yaml + diff is
captured into `state.proposed_remediation` with `status=pending_approval`,
and execution continues to synthesis. Phase 3 adds the interrupt/resume
machinery for human approval and the subsequent `apply_create_source` call.
"""

from __future__ import annotations

import json
from typing import Any

from langchain_anthropic import ChatAnthropic
from langchain_core.messages import ToolMessage
from langchain_core.tools import BaseTool
from langgraph.graph import END, START, StateGraph
from langgraph.graph.state import CompiledStateGraph
from langgraph.prebuilt import create_react_agent

from .prompts import (
    COLLECTOR_SUBGRAPH_PROMPT,
    DESTINATION_SUBGRAPH_PROMPT,
    SOURCE_SUBGRAPH_PROMPT,
    SYNTHESIS_PROMPT,
    TRIAGE_PROMPT,
)
from .state import (
    AgentState,
    Finding,
    ProposedRemediation,
    Report,
    TriageResult,
    WorkloadInput,
    make_step,
)

DEFAULT_MODEL = "claude-sonnet-4-5"


# Tool-name partitioning. Source intentionally omits `apply_create_source` -
# Phase 2 stops at the proposal; Phase 3 wires up the actual apply behind a
# human approval. Triage uses a small subset of source tools so the cheap
# initial probe doesn't pull collector / destination clients into the
# prompt's tool catalog.

_TRIAGE_TOOL_NAMES = frozenset({
    "get_source",
    "get_instrumentation_config",
    "list_workload_pods",
})

_SOURCE_TOOL_NAMES = frozenset({
    "get_source",
    "get_instrumentation_config",
    "list_instrumentation_instances",
    "get_workload",
    "list_workload_pods",
    "get_pod_env",
    "get_odiglet_logs_for_node",
    "list_instrumentation_rules",
    "propose_create_source",
})

_COLLECTOR_TOOL_NAMES = frozenset({
    "get_collectors_group",
    "get_collector_config",
    "list_collector_pods",
    "get_collector_logs",
    "get_collector_metrics",
    "get_processors",
    "get_actions",
})

_DESTINATION_TOOL_NAMES = frozenset({
    "list_destinations",
    "get_destination",
    "inspect_destination_secret",
    "get_destination_config_in_gateway",
    "get_gateway_export_errors",
    "probe_destination_endpoint",
})

_GRAPH_TOOL_NAMES = frozenset({
    "graph_query",
    "graph_neighbors",
    "graph_path",
    "graph_community",
    "graph_god_nodes",
    "graph_list_communities",
    "wiki_read",
    "graph_metadata",
    "gh_read_file",
})


def partition_tools(tools: list[BaseTool]) -> dict[str, list[BaseTool]]:
    """Split the merged MCP tool list into per-subgraph buckets.

    Tool names that don't match any bucket are dropped from subgraph
    catalogs - the agent only sees what's relevant to its phase.
    """
    by_name: dict[str, BaseTool] = {tool.name: tool for tool in tools}

    def pick(names: frozenset[str]) -> list[BaseTool]:
        return [by_name[name] for name in names if name in by_name]

    graph_tools = pick(_GRAPH_TOOL_NAMES)
    return {
        "triage": pick(_TRIAGE_TOOL_NAMES),
        "source": pick(_SOURCE_TOOL_NAMES) + graph_tools,
        "collector": pick(_COLLECTOR_TOOL_NAMES) + graph_tools,
        "destination": pick(_DESTINATION_TOOL_NAMES) + graph_tools,
        "graph": graph_tools,
    }


def route_after_triage(state: AgentState) -> list[str]:
    """Conditional-edge function: decide which subgraphs run.

    LangGraph dispatches each name in the returned list as a parallel branch
    out of the triage node. For unambiguous classifications a single subgraph
    runs; for `ambiguous` all three run; for `unknown` we skip straight to
    synthesis (which will emit a low-confidence report with empty evidence).
    """
    triage = state.get("triage")
    if triage is None:
        return ["synthesize"]
    classification = triage.classification
    if classification in ("source", "collector", "destination"):
        return [classification]
    if classification == "ambiguous":
        return ["source", "collector", "destination"]
    return ["synthesize"]


def _coerce_finding(value: Any, phase: str) -> Finding:
    """Coerce a structured_response into a Finding, forcing the phase field.

    The LLM's structured output may set `phase` to a literal that doesn't
    match this subgraph (e.g. it picks 'collector' from the prompt context).
    Overwrite it so the caller can trust the field.
    """
    if isinstance(value, Finding):
        return value.model_copy(update={"phase": phase})
    if isinstance(value, dict):
        value = {**value, "phase": phase}
        return Finding.model_validate(value)
    return Finding(
        phase=phase,
        summary="(subgraph produced no structured finding)",
    )


def _tool_message_text(message: ToolMessage) -> str | None:
    """Return the text payload of a ToolMessage regardless of content shape.

    langchain-mcp-adapters may return a plain string, a list of content
    blocks, or a structured dict depending on transport. Normalize to a
    JSON-parseable string when possible.
    """
    content = message.content
    if isinstance(content, str):
        return content
    if isinstance(content, list):
        for block in content:
            if isinstance(block, str):
                return block
            if not isinstance(block, dict):
                continue
            if block.get("type") == "text":
                text = block.get("text")
                if isinstance(text, str):
                    return text
                continue
            if block.get("type") == "json" and isinstance(block.get("json"), dict):
                return json.dumps(block["json"])
            # Bare dict block with no recognized type wrapper - try it as-is.
            if "type" not in block:
                return json.dumps(block)
        return None
    if isinstance(content, dict):
        return json.dumps(content)
    return None


def _extract_proposed_remediation(messages: list[Any]) -> ProposedRemediation | None:
    """Walk back through ReAct messages for a successful propose_create_source.

    Phase 2's only mutation surface is create_source. We only honor the most
    recent successful proposal - if the LLM re-proposed (which the prompt
    forbids), the last one wins. Failed proposals (`status == "error"`) are
    skipped so a transient k8s error doesn't masquerade as a pending
    approval in the final report.
    """
    for message in reversed(messages):
        if not isinstance(message, ToolMessage):
            continue
        if getattr(message, "name", None) != "propose_create_source":
            continue
        if getattr(message, "status", "success") == "error":
            continue
        text = _tool_message_text(message)
        if not text:
            continue
        try:
            data = json.loads(text)
        except json.JSONDecodeError:
            continue
        if not isinstance(data, dict) or "request_id" not in data:
            continue
        return ProposedRemediation(
            op="create_source",
            request_id=str(data["request_id"]),
            yaml=str(data.get("yaml", "")),
            diff=str(data.get("diff", "")),
            rollback_command=str(data.get("rollback_command", "")),
            status="pending_approval",
        )
    return None


def _override_remediation_from_state(
    report: Report, state: AgentState
) -> Report:
    """Force `report.proposed_remediation` to match `state.proposed_remediation`.

    The synthesizer is an `llm.with_structured_output(Report)` call and the
    LLM can ignore the prompt instruction to leave the field null. Trusting
    the model here would let a hallucinated `request_id` reach the UI and
    Phase 3 approval modal, where it would 404 on apply. The MCP's approval
    cache is the single source of truth for which mutations actually exist
    - this helper enforces that boundary unconditionally.
    """
    return report.model_copy(
        update={"proposed_remediation": state.get("proposed_remediation")}
    )


def _format_findings_for_synthesis(state: AgentState) -> str:
    """Render the per-domain findings as a compact prompt section."""
    lines: list[str] = []
    triage = state.get("triage")
    if triage is not None:
        lines.append(f"Triage classification: {triage.classification}")
        lines.append(f"Triage reasoning: {triage.reasoning}")
        if triage.symptoms_observed:
            lines.append("Triage symptoms:")
            for symptom in triage.symptoms_observed:
                lines.append(f"  - {symptom}")
    for phase in ("source", "collector", "destination"):
        finding = state.get(f"{phase}_findings")
        if finding is None:
            continue
        lines.append(f"\n[{phase}] {finding.summary}")
        if finding.evidence:
            lines.append("evidence:")
            for item in finding.evidence:
                lines.append(f"  - {item}")
        if finding.suggested_actions:
            lines.append("suggested actions:")
            for item in finding.suggested_actions:
                lines.append(f"  - {item}")
    return "\n".join(lines) if lines else "(no findings produced)"


def build_graph(
    tools: list[BaseTool],
    model: str = DEFAULT_MODEL,
    max_tokens: int = 4096,
) -> CompiledStateGraph:
    """Compile the diagnostic StateGraph against the merged MCP tool catalog."""
    llm = ChatAnthropic(model=model, max_tokens=max_tokens)
    partitioned = partition_tools(tools)

    triage_agent = create_react_agent(
        llm,
        partitioned["triage"],
        prompt=TRIAGE_PROMPT,
        response_format=TriageResult,
    )
    source_agent = create_react_agent(
        llm,
        partitioned["source"],
        prompt=SOURCE_SUBGRAPH_PROMPT,
        response_format=Finding,
    )
    collector_agent = create_react_agent(
        llm,
        partitioned["collector"],
        prompt=COLLECTOR_SUBGRAPH_PROMPT,
        response_format=Finding,
    )
    destination_agent = create_react_agent(
        llm,
        partitioned["destination"],
        prompt=DESTINATION_SUBGRAPH_PROMPT,
        response_format=Finding,
    )
    synthesizer = llm.with_structured_output(Report)

    async def triage_node(state: AgentState) -> dict:
        workload = state["input_workload"]
        user_message = (
            f"Triage {workload.kind} {workload.namespace}/{workload.name}. "
            "Run the minimum probes needed to classify."
        )
        result = await triage_agent.ainvoke(
            {"messages": [{"role": "user", "content": user_message}]}
        )
        triage = result.get("structured_response")
        if not isinstance(triage, TriageResult):
            triage = TriageResult(
                classification="unknown",
                reasoning="triage produced no structured response",
            )
        return {
            "triage": triage,
            "step_log": [
                make_step(
                    "triage",
                    f"classified as {triage.classification}",
                    {"reasoning": triage.reasoning},
                )
            ],
        }

    async def _run_subgraph(
        agent: CompiledStateGraph,
        state: AgentState,
        phase: str,
        prompt_hint: str,
    ) -> dict:
        workload = state["input_workload"]
        user_message = (
            f"{prompt_hint} Workload: {workload.kind} "
            f"{workload.namespace}/{workload.name}."
        )
        result = await agent.ainvoke(
            {"messages": [{"role": "user", "content": user_message}]}
        )
        finding = _coerce_finding(result.get("structured_response"), phase=phase)
        update: dict = {
            f"{phase}_findings": finding,
            "step_log": [
                make_step(phase, "subgraph completed", {"summary": finding.summary})
            ],
        }
        if phase == "source":
            proposed = _extract_proposed_remediation(result.get("messages", []))
            if proposed is not None:
                update["proposed_remediation"] = proposed
                update["step_log"].append(
                    make_step(
                        "source",
                        "remediation proposed",
                        {"op": proposed.op, "request_id": proposed.request_id},
                    )
                )
        return update

    async def source_node(state: AgentState) -> dict:
        return await _run_subgraph(
            source_agent,
            state,
            "source",
            "Diagnose source / instrumentation issues.",
        )

    async def collector_node(state: AgentState) -> dict:
        return await _run_subgraph(
            collector_agent,
            state,
            "collector",
            "Diagnose collector / gateway pipeline issues.",
        )

    async def destination_node(state: AgentState) -> dict:
        return await _run_subgraph(
            destination_agent,
            state,
            "destination",
            "Diagnose destination configuration issues.",
        )

    async def synthesize_node(state: AgentState) -> dict:
        findings_text = _format_findings_for_synthesis(state)
        user_message = (
            "Combine the following per-domain findings into a single Report.\n\n"
            f"{findings_text}"
        )
        report = await synthesizer.ainvoke(
            [
                {"role": "system", "content": SYNTHESIS_PROMPT},
                {"role": "user", "content": user_message},
            ]
        )
        if not isinstance(report, Report):
            report = Report(root_cause="unknown", confidence=0.0)
        report = _override_remediation_from_state(report, state)
        return {
            "report": report,
            "step_log": [
                make_step(
                    "synthesize",
                    f"root_cause={report.root_cause}",
                    {"confidence": report.confidence},
                )
            ],
        }

    builder = StateGraph(AgentState)
    builder.add_node("triage", triage_node)
    builder.add_node("source", source_node)
    builder.add_node("collector", collector_node)
    builder.add_node("destination", destination_node)
    builder.add_node("synthesize", synthesize_node)

    builder.add_edge(START, "triage")
    builder.add_conditional_edges(
        "triage",
        route_after_triage,
        ["source", "collector", "destination", "synthesize"],
    )
    builder.add_edge("source", "synthesize")
    builder.add_edge("collector", "synthesize")
    builder.add_edge("destination", "synthesize")
    builder.add_edge("synthesize", END)

    return builder.compile()


def initial_state(workload: WorkloadInput) -> AgentState:
    """Build the starting state for a /debug run."""
    return {
        "input_workload": workload,
        "step_log": [
            make_step(
                "input",
                "received debug request",
                {
                    "namespace": workload.namespace,
                    "kind": workload.kind,
                    "name": workload.name,
                },
            )
        ],
    }
