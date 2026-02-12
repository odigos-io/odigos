#!/bin/bash

# This test verifies that collector metrics continue to be reported after a
# rollout restart of the UI pod.
#
# Background:
#   The UI pod runs metric-watchers that consume OTLP metrics from collectors.
#   These watchers pin ResourceVersion to '1' (rather than using a dynamic
#   list-derived version) so that after a pod restart the watchers replay
#   from the beginning and existing sources continue to produce metrics.
#
#   If the watchers were using dynamic ResourceVersions, sources that were
#   instrumented *before* the restart would no longer produce metrics in the
#   UI because the watcher would only see events newer than its stale version.
#
# Flow:
#   1. Assert collector metrics are already being reported (pre-restart).
#   2. Rollout-restart the odigos-ui deployment.
#   3. Wait for the new UI pod to become ready.
#   4. Assert collector metrics are still being reported (post-restart).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

print_help() {
    echo "Usage: $0 [namespace]"
    echo ""
    echo "Arguments:"
    echo "  namespace       (optional) The Kubernetes namespace of the UI service (default: 'odigos-system')"
    exit 1
}

if [[ $# -gt 1 ]]; then
    echo "❌ Error: Invalid number of arguments."
    print_help
fi

NAMESPACE="${1:-odigos-system}"

echo "==========================================="
echo "  Collector Metrics – UI Restart Survival  "
echo "==========================================="
echo "ℹ️ Using namespace: $NAMESPACE"
echo ""

# ------------------------------------------------------------------
# Step 1 – Verify metrics are reported BEFORE the restart
# ------------------------------------------------------------------
echo "📊 Step 1: Asserting collector metrics are reported before UI restart..."

"$SCRIPT_DIR/assert_collector_metrics_from_ui.sh" "$NAMESPACE"
echo "✅ Step 1 passed: Metrics are being reported."
echo ""

# ------------------------------------------------------------------
# Step 2 – Rollout-restart the UI deployment
# ------------------------------------------------------------------
UI_DEPLOYMENT="odigos-ui"

echo "🔄 Step 2: Performing rollout restart of deployment/$UI_DEPLOYMENT in namespace $NAMESPACE..."
kubectl rollout restart "deployment/$UI_DEPLOYMENT" -n "$NAMESPACE"
echo "⏳ Waiting for rollout to complete..."
kubectl rollout status "deployment/$UI_DEPLOYMENT" -n "$NAMESPACE" --timeout=120s
echo "✅ Step 2 passed: UI deployment rolled out successfully."
echo ""

# ------------------------------------------------------------------
# Step 3 – Wait for the new UI pod to be ready
# ------------------------------------------------------------------
echo "⏳ Step 3: Waiting for the new UI pod to become ready..."
kubectl wait --for=condition=ready pod -l app=odigos-ui -n "$NAMESPACE" --timeout=120s
echo "✅ Step 3 passed: UI pod is ready."
echo ""

# ------------------------------------------------------------------
# Step 4 – Wait for metrics to accumulate, then verify post-restart
# ------------------------------------------------------------------
echo "📊 Step 4: Asserting collector metrics are reported after UI restart..."
echo "⏳ Giving the UI some time to start collecting metrics..."

# The UI needs a bit of time after startup for the OTLP receiver to start
# and for collectors to push metrics. Retry with backoff.
MAX_RETRIES=30
RETRY_DELAY=10
for attempt in $(seq 1 $MAX_RETRIES); do
    if "$SCRIPT_DIR/assert_collector_metrics_from_ui.sh" "$NAMESPACE"; then
        echo ""
        echo "✅ Step 4 passed: Metrics are still being reported after UI restart."
        echo ""
        echo "==========================================="
        echo "  ✅ All checks passed!                    "
        echo "==========================================="
        exit 0
    fi

    if [[ $attempt -eq $MAX_RETRIES ]]; then
        echo ""
        echo "❌ Step 4 failed: Metrics were NOT reported after UI restart (exhausted $MAX_RETRIES retries)."
        echo "   This likely means the metric watchers lost track of pre-existing sources after the restart."
        exit 1
    fi

    echo "   Attempt $attempt/$MAX_RETRIES failed, retrying in ${RETRY_DELAY}s..."
    sleep "$RETRY_DELAY"
done
