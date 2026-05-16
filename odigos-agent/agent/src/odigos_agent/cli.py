"""Phase 0 dev CLI.

Runs a one-shot LangGraph ReAct agent bound to the merged MCP tool catalog
(cluster MCP + graph MCP). Smoke test: prompt the agent to call `ping` on
both MCPs and report the bundled graph commit via `graph_metadata`.
"""

from __future__ import annotations

import argparse
import asyncio
import os
import sys

from langchain_anthropic import ChatAnthropic
from langgraph.prebuilt import create_react_agent

from .mcp_client import McpEndpoints, load_tools

DEFAULT_PROMPT = (
    "Call cluster_ping, graph_ping, and graph_metadata. Report each ping "
    "response verbatim and the graph's built_at_commit."
)

SYSTEM_PROMPT = (
    "You are the odigos-agent Phase 0 smoke harness. "
    "You have tools from two MCP servers: the cluster MCP (Go) and the graph "
    "MCP (Python). For now, just exercise the tools the user asks about and "
    "report back concisely."
)


async def run(prompt: str, model: str) -> str:
    tools = await load_tools(McpEndpoints.from_env())
    llm = ChatAnthropic(model=model, max_tokens=2048)
    agent = create_react_agent(llm, tools, prompt=SYSTEM_PROMPT)
    result = await agent.ainvoke({"messages": [{"role": "user", "content": prompt}]})
    final = result["messages"][-1]
    return getattr(final, "content", str(final))


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="odigos-agent dev CLI")
    parser.add_argument(
        "prompt",
        nargs="?",
        default=DEFAULT_PROMPT,
        help="User prompt for the ReAct agent.",
    )
    parser.add_argument(
        "--model",
        default=os.environ.get("ODIGOS_AGENT_MODEL", "claude-sonnet-4-5"),
        help="Anthropic model id (env ODIGOS_AGENT_MODEL overrides default).",
    )
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> None:
    args = parse_args(argv if argv is not None else sys.argv[1:])
    if not os.environ.get("ANTHROPIC_API_KEY"):
        print("ANTHROPIC_API_KEY not set", file=sys.stderr)
        sys.exit(2)
    output = asyncio.run(run(args.prompt, args.model))
    print(output)


if __name__ == "__main__":
    main()
