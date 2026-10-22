#!/usr/bin/env bash
set -euo pipefail

chart_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

for image_tag in before after; do
  rendered="$(helm template rollout-test "$chart_dir" \
    --namespace odigos-central \
    --show-only templates/connectors/connector-runtime-deployment.yaml \
    --set cloudConnectors.enabled=true \
    --set-string onPremToken=rollout-test \
    --set-string "cloudConnectors.managedCollectors.collectorImage.reference=example.test/collector:$image_tag")"

  # One replica is not enough: RollingUpdate may temporarily run old and new controllers together.
  printf '%s\n' "$rendered" | awk '
    /^  replicas: 1$/ { single_replica = 1 }
    /^  strategy:$/ { in_strategy = 1; next }
    in_strategy && /^    type: Recreate$/ { recreate = 1 }
    in_strategy && /rollingUpdate:/ && !/rollingUpdate: null$/ { unsafe = 1 }
    /^  [^ ]/ { in_strategy = 0 }
    END {
      if (!single_replica || !recreate || unsafe) {
        print "connector-runtime must have one replica and a Recreate rollout" > "/dev/stderr"
        exit 1
      }
    }
  '
done

printf '%s\n' 'connector-runtime rollout checks passed'
