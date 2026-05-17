"""uvicorn entrypoint for the Phase 3 API.

Binds 0.0.0.0:8765 by default so the in-cluster Service can reach the pod.
Override the port with PORT and the log level with LOG_LEVEL.
"""

from __future__ import annotations

import os

import uvicorn


def main() -> None:
    port = int(os.environ.get("PORT", "8765"))
    log_level = os.environ.get("LOG_LEVEL", "info").lower()
    uvicorn.run(
        "odigos_agent.api:app",
        host="0.0.0.0",
        port=port,
        log_level=log_level,
    )


if __name__ == "__main__":
    main()
