"""Dev CLI for the odigos agent.

Two subcommands:

- `ping` (default, kept from Phase 0): a one-shot ReAct loop bound to the
  merged MCP tool catalog. Used as a smoke test that both MCPs are reachable
  and tool calls round-trip.
- `debug --namespace --kind --name`: runs the Phase 2 diagnostic LangGraph
  against a workload and prints the final Report as JSON. The graph stops
  after proposing a mutation (status=pending_approval); Phase 3 will resume
  it after a human approves.
"""

from __future__ import annotations

import argparse
import asyncio
import json
import os
import sys

from langchain_anthropic import ChatAnthropic
from langgraph.prebuilt import create_react_agent

from .graph import DEFAULT_MODEL, build_graph, initial_state
from .mcp_client import McpEndpoints, load_tools
from .state import WorkloadInput

PING_PROMPT = (
    "Call cluster_ping, graph_ping, and graph_metadata. Report each ping "
    "response verbatim and the graph's built_at_commit."
)

PING_SYSTEM_PROMPT = (
    "You are the odigos-agent dev smoke harness. "
    "You have tools from two MCP servers: the cluster MCP (Go) and the graph "
    "MCP (Python). Exercise the tools the user asks about and report back "
    "concisely."
)


async def run_ping(prompt: str, model: str) -> str:
    tools = await load_tools(McpEndpoints.from_env())
    llm = ChatAnthropic(model=model, max_tokens=2048)
    agent = create_react_agent(llm, tools, prompt=PING_SYSTEM_PROMPT)
    result = await agent.ainvoke({"messages": [{"role": "user", "content": prompt}]})
    final = result["messages"][-1]
    return getattr(final, "content", str(final))


async def run_debug(workload: WorkloadInput, model: str) -> dict:
    tools = await load_tools(McpEndpoints.from_env())
    graph = build_graph(tools, model=model)
    final_state = await graph.ainvoke(initial_state(workload))
    report = final_state.get("report")
    return {
        "workload": workload.model_dump(),
        "triage": (
            final_state["triage"].model_dump()
            if final_state.get("triage") is not None
            else None
        ),
        "report": report.model_dump() if report is not None else None,
        "step_log": final_state.get("step_log", []),
    }


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog="odigos-agent", description="odigos-agent dev CLI")
    parser.add_argument(
        "--model",
        default=os.environ.get("ODIGOS_AGENT_MODEL", DEFAULT_MODEL),
        help="Anthropic model id (env ODIGOS_AGENT_MODEL overrides default).",
    )
    subparsers = parser.add_subparsers(dest="command")

    ping_parser = subparsers.add_parser(
        "ping", help="Smoke-test both MCPs by calling cluster_ping + graph_ping."
    )
    ping_parser.add_argument(
        "prompt",
        nargs="?",
        default=PING_PROMPT,
        help="User prompt for the ReAct agent.",
    )

    debug_parser = subparsers.add_parser(
        "debug",
        help="Run the diagnostic LangGraph against a workload and print the Report.",
    )
    debug_parser.add_argument("--namespace", required=True, help="Workload namespace.")
    debug_parser.add_argument(
        "--kind",
        required=True,
        help="Workload kind (Deployment, StatefulSet, DaemonSet, CronJob, Job, ...).",
    )
    debug_parser.add_argument("--name", required=True, help="Workload name.")

    return parser


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = build_parser()
    args = parser.parse_args(argv)
    if args.command is None:
        args.command = "ping"
        args.prompt = PING_PROMPT
    return args


def main(argv: list[str] | None = None) -> None:
    args = parse_args(argv if argv is not None else sys.argv[1:])
    if not os.environ.get("ANTHROPIC_API_KEY"):
        print("ANTHROPIC_API_KEY not set", file=sys.stderr)
        sys.exit(2)

    if args.command == "ping":
        output = asyncio.run(run_ping(args.prompt, args.model))
        print(output)
        return

    if args.command == "debug":
        workload = WorkloadInput(
            namespace=args.namespace, kind=args.kind, name=args.name
        )
        result = asyncio.run(run_debug(workload, args.model))
        print(json.dumps(result, indent=2, default=str))
        return

    print(f"unknown command: {args.command}", file=sys.stderr)
    sys.exit(2)


if __name__ == "__main__":
    main()
