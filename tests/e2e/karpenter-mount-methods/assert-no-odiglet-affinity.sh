#!/usr/bin/env bash
# With karpenter.enabled, CSI/hostPath pods must not get odiglet-installed node affinity.
set -euo pipefail

FAILED=0
for app in frontend inventory coupon currency membership pricing geolocation; do
  POD=$(kubectl get pods -n default -l "app=${app}" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
  if [[ -z "${POD}" ]]; then
    echo "❌ no pod found for app=${app}"
    FAILED=1
    continue
  fi

  AFFINITY_JSON=$(kubectl get pod "${POD}" -n default -o jsonpath='{.spec.affinity.nodeAffinity}' || true)
  if [[ -z "${AFFINITY_JSON}" || "${AFFINITY_JSON}" == "null" ]]; then
    echo "✅ ${app}/${POD}: no nodeAffinity"
    continue
  fi

  if echo "${AFFINITY_JSON}" | jq -e '
    .. | objects | select(has("key")) | select(.key | test("odiglet.*installed"))
  ' >/dev/null 2>&1; then
    echo "❌ ${app}/${POD} still has odiglet-installed node affinity:"
    echo "${AFFINITY_JSON}" | jq .
    FAILED=1
  else
    echo "✅ ${app}/${POD}: nodeAffinity has no odiglet-installed selector"
  fi
done

exit "${FAILED}"
