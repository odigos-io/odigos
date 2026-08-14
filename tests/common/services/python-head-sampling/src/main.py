import os
import signal
import sys
import threading
import time
import urllib.error
import urllib.request
from typing import Any

from flask import Flask, jsonify, request

app = Flask(__name__)

PORT = 8080
START_TIME_MS = int(time.time() * 1000)
STARTUP_DELAY_MS = 10000
READY_DELAY_MS = 12000
SHUTDOWN_TIMEOUT_SEC = 10

SIMULATE_STARTUP_DELAY = os.environ.get("SIMULATE_STARTUP_DELAY", "").lower() == "true"
HTTP_MATCH_ENABLE_ROUTES = os.environ.get("HTTP_MATCH_ENABLE_ROUTES", "true").lower() != "false"

_outbound_stop = threading.Event()
_outbound_thread: threading.Thread | None = None
_server = None
_force_exit_timer: threading.Timer | None = None


def _elapsed_ms() -> int:
    return int(time.time() * 1000) - START_TIME_MS


def _health_startup():
    if not SIMULATE_STARTUP_DELAY:
        return jsonify({"status": "started", "simulated": False}), 200
    elapsed = _elapsed_ms()
    if elapsed >= STARTUP_DELAY_MS:
        return jsonify({"status": "started", "elapsed_ms": elapsed}), 200
    return (
        jsonify({"status": "starting", "remaining_ms": STARTUP_DELAY_MS - elapsed}),
        503,
    )


def _health_ready():
    if not SIMULATE_STARTUP_DELAY:
        return jsonify({"status": "ready", "simulated": False}), 200
    elapsed = _elapsed_ms()
    if elapsed >= READY_DELAY_MS:
        return jsonify({"status": "ready", "elapsed_ms": elapsed}), 200
    return (
        jsonify({"status": "not_ready", "remaining_ms": READY_DELAY_MS - elapsed}),
        503,
    )


@app.get("/healthz/startup")
def health_startup():
    return _health_startup()


@app.get("/healthz/ready")
def health_ready():
    return _health_ready()


@app.get("/healthz/live")
def health_live():
    return jsonify({"status": "alive"}), 200


@app.get("/healthz")
def healthz():
    return jsonify({"status": "healthy"}), 200


def _resolve_graphql_health_check_probe_type() -> str | None:
    operation_name = request.args.get("operationName")
    if operation_name == "HealthCheckStartup":
        return "startup"
    if operation_name == "HealthCheckReady":
        return "ready"
    if operation_name == "HealthCheckLive":
        return "live"

    queries = request.args.getlist("query")
    if len(queries) >= 2:
        return "live"

    query = queries[0] if queries else ""
    if "HealthCheckStartup" in query:
        return "startup"
    if "HealthCheckReady" in query:
        return "ready"
    if "HealthCheckLive" in query:
        return "live"
    return None


def _graphql_health_check_data(probe_type: str) -> dict[str, Any]:
    if probe_type == "live":
        return {"data": {"status": "alive", "__typename": "Query"}}
    if probe_type == "startup":
        return {"data": {"status": "started", "__typename": "Query"}}
    return {"data": {"status": "ready", "__typename": "Query"}}


def _graphql_applicative_data() -> dict[str, Any]:
    operation_name = request.args.get("operationName")
    queries = request.args.getlist("query")
    query = queries[0] if queries else ""

    if operation_name == "GetUser" or "GetUser" in query:
        return {
            "data": {
                "user": {"id": "1", "name": "Alice", "__typename": "User"},
                "__typename": "Query",
            }
        }
    if operation_name == "ListItems" or "ListItems" in query:
        return {
            "data": {
                "items": [{"id": "1", "title": "Item 1", "__typename": "Item"}],
                "__typename": "Query",
            }
        }
    if operation_name == "GetItem" or "GetItem" in query:
        return {
            "data": {
                "item": {"id": "42", "title": "Item 42", "__typename": "Item"},
                "__typename": "Query",
            }
        }
    return {"data": {"ok": True, "__typename": "Query"}}


@app.get("/graphql")
def graphql_health_check():
    probe_type = _resolve_graphql_health_check_probe_type()
    if probe_type is None:
        return jsonify(_graphql_applicative_data()), 200

    if probe_type == "live":
        return jsonify(_graphql_health_check_data("live")), 200

    if not SIMULATE_STARTUP_DELAY:
        return jsonify(_graphql_health_check_data(probe_type)), 200

    elapsed = _elapsed_ms()
    if probe_type == "startup":
        if elapsed >= STARTUP_DELAY_MS:
            return jsonify(_graphql_health_check_data("startup")), 200
        return (
            jsonify(
                {
                    "errors": [
                        {
                            "message": "starting",
                            "extensions": {"remaining_ms": STARTUP_DELAY_MS - elapsed},
                        }
                    ]
                }
            ),
            503,
        )

    if elapsed >= READY_DELAY_MS:
        return jsonify(_graphql_health_check_data("ready")), 200
    return (
        jsonify(
            {
                "errors": [
                    {
                        "message": "not_ready",
                        "extensions": {"remaining_ms": READY_DELAY_MS - elapsed},
                    }
                ]
            }
        ),
        503,
    )


@app.get("/sampling/percentage/no-rule")
def sampling_no_rule():
    return jsonify(
        {
            "endpoint": "/sampling/percentage/no-rule",
            "description": "no sampling rule matches this endpoint",
        }
    )


@app.get("/sampling/percentage/sampled-0")
def sampling_sampled_0():
    return jsonify(
        {
            "endpoint": "/sampling/percentage/sampled-0",
            "description": "sampling rule with 0% rate",
        }
    )


@app.get("/sampling/percentage/sampled-50")
def sampling_sampled_50():
    return jsonify(
        {
            "endpoint": "/sampling/percentage/sampled-50",
            "description": "sampling rule with 50% rate",
        }
    )


@app.get("/sampling/percentage/sampled-100")
def sampling_sampled_100():
    return jsonify(
        {
            "endpoint": "/sampling/percentage/sampled-100",
            "description": "sampling rule with 100% rate",
        }
    )


@app.get("/sampling/percentage/sampled-fallback")
def sampling_sampled_fallback():
    return jsonify(
        {
            "endpoint": "/sampling/percentage/sampled-fallback",
            "description": "sampling rule with no percentage, falls back to 0%",
        }
    )


@app.get("/sampling/route/prefix")
def sampling_route_prefix():
    return jsonify(
        {
            "endpoint": "/sampling/route/prefix",
            "description": "route prefix sampling rule with 50% rate",
        }
    )


@app.get("/sampling/route/prefix/part-one")
def sampling_route_prefix_part_one():
    return jsonify(
        {
            "endpoint": "/sampling/route/prefix/part-one",
            "description": "matches the /sampling/route/prefix sampling rule",
        }
    )


@app.get("/sampling/route/prefix/part-one/part-two")
def sampling_route_prefix_part_two():
    return jsonify(
        {
            "endpoint": "/sampling/route/prefix/part-one/part-two",
            "description": "matches the /sampling/route/prefix sampling rule with more route parts",
        }
    )


@app.get("/sampling/route/exact/<item_id>")
def sampling_route_exact(item_id: str):
    return jsonify(
        {
            "endpoint": "/sampling/route/exact/:itemId",
            "item_id": item_id,
            "description": "exact sampling rule for a templated route",
        }
    )


@app.get("/sampling/route/exact/<item_id>/details/<detail_id>")
def sampling_route_exact_details(item_id: str, detail_id: str):
    return jsonify(
        {
            "endpoint": "/sampling/route/exact/:itemId/details/:detailId",
            "item_id": item_id,
            "detail_id": detail_id,
            "description": "exact sampling rule for a templated route with multiple parameters",
        }
    )


def _register_http_match_routes() -> None:
    @app.get("/http-match/control/no-rule")
    def http_match_control():
        return jsonify(
            {
                "category": "control",
                "endpoint": "/http-match/control/no-rule",
                "description": "no sampling rule targets this path; traces follow default pipeline behavior",
            }
        )

    @app.get("/http-match/exact/target")
    def http_match_exact_target():
        return jsonify(
            {
                "category": "exact_route",
                "endpoint": "/http-match/exact/target",
                "description": "exact HTTP route match (full literal path)",
            }
        )

    @app.post("/http-match/exact/post-target")
    def http_match_exact_post_target():
        return jsonify(
            {
                "category": "exact_route_post",
                "endpoint": "/http-match/exact/post-target",
                "description": "POST-only exact path for client outbound tests",
            }
        )

    @app.get("/http-match/prefix/segment")
    def http_match_prefix_segment():
        return jsonify(
            {
                "category": "prefix_route",
                "endpoint": "/http-match/prefix/segment",
                "description": "prefix HTTP route match (first segment after /http-match/prefix)",
            }
        )

    @app.get("/http-match/prefix/segment/nested")
    def http_match_prefix_segment_nested():
        return jsonify(
            {
                "category": "prefix_route",
                "endpoint": "/http-match/prefix/segment/nested",
                "description": "prefix HTTP route match (deeper path under the same static prefix)",
            }
        )

    @app.get("/http-match/texact/<resource_id>")
    def http_match_texact(resource_id: str):
        return jsonify(
            {
                "category": "templatized_exact",
                "endpoint": "/http-match/texact/:resourceId",
                "resource_id": resource_id,
                "description": "templatized exact route (one dynamic segment)",
            }
        )

    @app.get("/http-match/tprefix/<tenant_id>/items")
    def http_match_tprefix_items(tenant_id: str):
        return jsonify(
            {
                "category": "templatized_prefix",
                "endpoint": "/http-match/tprefix/:tenantId/items",
                "tenant_id": tenant_id,
                "description": "templatized prefix (tenant/items)",
            }
        )

    @app.get("/http-match/tprefix/<tenant_id>/items/<item_id>")
    def http_match_tprefix_items_item(tenant_id: str, item_id: str):
        return jsonify(
            {
                "category": "templatized_prefix",
                "endpoint": "/http-match/tprefix/:tenantId/items/:itemId",
                "tenant_id": tenant_id,
                "item_id": item_id,
                "description": "templatized prefix with extra path under items",
            }
        )


if HTTP_MATCH_ENABLE_ROUTES:
    _register_http_match_routes()


def _http_get(url: str) -> None:
    try:
        with urllib.request.urlopen(url, timeout=30) as resp:
            resp.read()
    except urllib.error.URLError as err:
        print(f"http-match periodic outbound failed {url} {err.reason}", file=sys.stderr)


def _http_post(url: str) -> None:
    try:
        req = urllib.request.Request(url, method="POST")
        with urllib.request.urlopen(req, timeout=30) as resp:
            resp.read()
    except urllib.error.URLError as err:
        print(f"http-match periodic outbound POST failed {url} {err.reason}", file=sys.stderr)


def _fire_http_match_outbound() -> None:
    default_peer = os.environ.get("HTTP_MATCH_PEER_BASE_URL", f"http://127.0.0.1:{PORT}")
    exact_peer = os.environ.get("HTTP_MATCH_EXACT_PEER_BASE_URL", default_peer)
    default_base = default_peer.rstrip("/")
    exact_base = exact_peer.rstrip("/")

    exact_get_path = "/http-match/exact/target"
    post_exact_path = "/http-match/exact/post-target"
    prefix_and_templated_paths = [
        "/http-match/prefix/segment",
        "/http-match/prefix/segment/nested",
        "/http-match/texact/out-peer-res",
        "/http-match/tprefix/out-peer-tenant/items",
        "/http-match/tprefix/out-peer-tenant/items/out-peer-item",
    ]

    _http_get(default_base + exact_get_path)
    _http_get(exact_base + exact_get_path)
    _http_post(default_base + post_exact_path)
    for rel_path in prefix_and_templated_paths:
        _http_get(default_base + rel_path)


def _outbound_loop(interval_sec: float) -> None:
    while not _outbound_stop.wait(interval_sec):
        _fire_http_match_outbound()


def _start_http_match_periodic_outbound() -> None:
    global _outbound_thread
    if os.environ.get("HTTP_MATCH_PERIODIC_OUTBOUND", "").lower() != "true":
        return

    default_peer = os.environ.get("HTTP_MATCH_PEER_BASE_URL", f"http://127.0.0.1:{PORT}")
    exact_peer = os.environ.get("HTTP_MATCH_EXACT_PEER_BASE_URL", default_peer)
    try:
        interval_ms = int(os.environ.get("HTTP_MATCH_OUTBOUND_INTERVAL_MS", "10000"))
    except ValueError:
        interval_ms = 10000
    if interval_ms < 1000:
        interval_ms = 10000

    prefix_paths = 5
    requests_per_tick = 2 + 1 + prefix_paths
    print(
        f"http-match periodic outbound every {interval_ms}ms → "
        f"{requests_per_tick} requests/tick (GET exact ×2 + POST exact + prefix/templated GETs; "
        f"bases {default_peer} / {exact_peer})"
    )

    _outbound_thread = threading.Thread(
        target=_outbound_loop,
        args=(interval_ms / 1000.0,),
        name="http-match-outbound",
        daemon=True,
    )
    _outbound_thread.start()


def _shutdown(signum: int, frame: Any) -> None:
    global _force_exit_timer
    print("SIGTERM received, shutting down gracefully...")
    _outbound_stop.set()
    if _outbound_thread is not None:
        _outbound_thread.join(timeout=2)

    def _force_exit() -> None:
        print("Could not close connections in time, forcefully shutting down", file=sys.stderr)
        os._exit(1)

    _force_exit_timer = threading.Timer(SHUTDOWN_TIMEOUT_SEC, _force_exit)
    _force_exit_timer.daemon = True
    _force_exit_timer.start()

    if _server is not None:
        _server.shutdown()


if __name__ == "__main__":
    signal.signal(signal.SIGTERM, _shutdown)

    print(f"head-sampling server running at http://127.0.0.1:{PORT}/")
    if SIMULATE_STARTUP_DELAY:
        print(f"Startup probe will pass after {STARTUP_DELAY_MS // 1000}s")
        print(f"Readiness probe will pass after {READY_DELAY_MS // 1000}s")
    else:
        print("Startup delay simulation is disabled")

    from werkzeug.serving import make_server

    _server = make_server("0.0.0.0", PORT, app)
    _start_http_match_periodic_outbound()
    _server.serve_forever()
    if _force_exit_timer is not None:
        _force_exit_timer.cancel()
    print("HTTP server closed")
    sys.exit(0)
