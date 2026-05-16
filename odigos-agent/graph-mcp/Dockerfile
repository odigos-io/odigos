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
COPY graphify-out /graph
ENV GRAPH_FILE=/graph/merged-graph.json \
    WIKI_DIR=/graph/wiki \
    MCP_GRAPH_HOST=0.0.0.0 \
    MCP_GRAPH_PORT=9091
EXPOSE 9091
USER 65532:65532
ENTRYPOINT ["python", "-m", "graph_mcp.server"]
