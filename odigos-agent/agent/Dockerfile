# syntax=docker/dockerfile:1.7

FROM python:3.12-slim AS build
WORKDIR /app
ENV PIP_NO_CACHE_DIR=1
COPY pyproject.toml ./
COPY src ./src
RUN pip install --no-cache-dir .

FROM python:3.12-slim
WORKDIR /app
ENV PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1
COPY --from=build /usr/local/lib/python3.12/site-packages /usr/local/lib/python3.12/site-packages
COPY --from=build /usr/local/bin /usr/local/bin
ENV MCP_CLUSTER_URL=http://127.0.0.1:9090/mcp \
    MCP_GRAPH_URL=http://127.0.0.1:9091/mcp \
    ODIGOS_AGENT_MODEL=claude-sonnet-4-5
USER 65532:65532
ENTRYPOINT ["odigos-agent"]
