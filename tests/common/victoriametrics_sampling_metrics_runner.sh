#!/bin/bash
#
# Record and assert VictoriaMetrics sampling counter deltas (Prometheus API).
#
# Usage:
#   victoriametrics_sampling_metrics_runner.sh record-baseline <test.yaml> [baseline.json]
#   victoriametrics_sampling_metrics_runner.sh assert-delta <test.yaml> [baseline.json]
#
# SamplingTraceMetricsTest YAML:
#   victoriaMetrics: { namespace, service, port }
#   metrics: { match, keep, drop }  # Prometheus metric names
#   rules:
#     - ruleId: ...           # optional; resolved from InstrumentationConfig when omitted
#     - ruleName: ...
#     - deployment: ...
#       expected: { match: N, keep: N, drop: N }
#
# Env:
#   TESTS_NAMESPACE  — namespace of InstrumentationConfigs (apps). Required when ruleId is omitted.
#   ODIGOS_NAMESPACE — namespace for temporary curl probe pods (defaults to victoriaMetrics.namespace).
#
set -euo pipefail

resolve_rule_id() {
  local file=$1
  local rule_index=$2
  local rule_id rule_name deployment namespace ic_name

  rule_id=$(yq e ".rules[${rule_index}].ruleId // \"\"" "$file")
  if [[ -n "$rule_id" && "$rule_id" != "null" ]]; then
    printf '%s' "$rule_id"
    return
  fi

  rule_name=$(yq e ".rules[${rule_index}].ruleName" "$file")
  deployment=$(yq e ".rules[${rule_index}].deployment" "$file")
  namespace="${TESTS_NAMESPACE:-}"
  if [[ -z "$namespace" ]]; then
    echo "ruleId missing for rules[${rule_index}] and TESTS_NAMESPACE is not set" >&2
    exit 1
  fi

  ic_name="deployment-${deployment}"
  rule_id=$(
    kubectl get instrumentationconfig "$ic_name" -n "$namespace" -o json | jq -r --arg name "$rule_name" '
      .spec.workloadCollectorConfig[0].samplingCollectorConfig as $sc |
      (($sc.highlyRelevantOperations // []) + ($sc.costReductionRules // []))
      | map(select(.name == $name))
      | .[0].id // empty
    '
  )
  if [[ -z "$rule_id" ]]; then
    echo "Could not resolve rule id for ${deployment}/${rule_name} in namespace ${namespace}" >&2
    exit 1
  fi
  printf '%s' "$rule_id"
}

yq_read() {
  local file=$1
  local field=$2
  local expr=$3
  yq e ".${field}${expr}" "$file"
}

vm_namespace() {
  local file=$1
  yq_read "$file" "victoriaMetrics" '.namespace // "odigos-test"'
}

vm_base_url() {
  local file=$1
  local ns svc port
  ns=$(vm_namespace "$file")
  svc=$(yq_read "$file" "victoriaMetrics" '.service // "odigos-victoriametrics"')
  port=$(yq_read "$file" "victoriaMetrics" '.port // "8428"')
  printf 'http://%s.%s.svc.cluster.local:%s' "$svc" "$ns" "$port"
}

probe_pod_namespace() {
  local file=$1
  if [[ -n "${ODIGOS_NAMESPACE:-}" ]]; then
    printf '%s' "$ODIGOS_NAMESPACE"
    return
  fi
  vm_namespace "$file"
}

query_vm_scalar() {
  local file=$1
  local base_url=$2
  local promql=$3
  local probe_ns response json value
  probe_ns=$(probe_pod_namespace "$file")
  response=$(
    kubectl run "vm-promql-$RANDOM" \
      --rm -i --restart=Never \
      -n "$probe_ns" \
      --image=curlimages/curl:8.4.0 \
      --command -- \
      curl -sS -G "${base_url}/api/v1/query" \
      --data-urlencode "query=${promql}" 2>&1
  )
  json=$(printf '%s' "$response" | sed 's/pod ".*$//')
  value=$(echo "$json" | jq -r '
    if .status != "success" then
      error("VM query failed: " + (.error // .errorType // "unknown"))
    elif (.data.result | length) == 0 then
      "0"
    else
      .data.result[0].value[1]
    end
  ')
  if [[ -z "$value" || "$value" == "null" ]]; then
    value=0
  fi
  printf '%s' "$value"
}

sum_metric_for_rule_id() {
  local file=$1
  local base_url=$2
  local metric=$3
  local rule_id=$4
  local promql
  promql=$(printf 'sum(%s{odigos.sampling.rule.id="%s"})' "$metric" "$rule_id")
  query_vm_scalar "$file" "$base_url" "$promql"
}

metric_delta() {
  local current=$1
  local baseline=$2
  awk -v cur="$current" -v base="$baseline" 'BEGIN { printf "%.0f", cur - base }'
}

record_baseline() {
  local file=$1
  local baseline_file=$2
  local base_url rule_count i rule_id
  local metric_match metric_keep metric_drop
  local match_v keep_v drop_v

  base_url=$(vm_base_url "$file")
  metric_match=$(yq_read "$file" "metrics" '.match')
  metric_keep=$(yq_read "$file" "metrics" '.keep')
  metric_drop=$(yq_read "$file" "metrics" '.drop')
  rule_count=$(yq e '.rules | length' "$file")

  echo "Recording sampling trace metrics baseline -> ${baseline_file}"
  echo "VictoriaMetrics: ${base_url}"
  echo "Counters: match=${metric_match} keep=${metric_keep} drop=${metric_drop}"

  rm -f "$baseline_file.tmp"
  for ((i = 0; i < rule_count; i++)); do
    rule_id=$(resolve_rule_id "$file" "$i")
    match_v=$(sum_metric_for_rule_id "$file" "$base_url" "$metric_match" "$rule_id")
    keep_v=$(sum_metric_for_rule_id "$file" "$base_url" "$metric_keep" "$rule_id")
    drop_v=$(sum_metric_for_rule_id "$file" "$base_url" "$metric_drop" "$rule_id")
    jq -n \
      --arg id "$rule_id" \
      --arg match "$match_v" \
      --arg keep "$keep_v" \
      --arg drop "$drop_v" \
      '{($id): {match: ($match|tonumber), keep: ($keep|tonumber), drop: ($drop|tonumber)}}' >>"$baseline_file.tmp"
    echo "  baseline ${rule_id}: match=${match_v} keep=${keep_v} drop=${drop_v}"
  done

  jq -s 'add' "$baseline_file.tmp" | jq '.' >"$baseline_file"
  rm -f "$baseline_file.tmp"
}

assert_counter_delta() {
  local counter_name=$1
  local baseline_val=$2
  local current_val=$3
  local delta=$4
  local expected=$5

  echo "  ${counter_name}: baseline=${baseline_val} current=${current_val} delta=${delta} expected=${expected}"
  if [[ "$delta" -ne "$expected" ]]; then
    echo "FAILED: ${counter_name} expected delta ${expected}, got ${delta}" >&2
    return 1
  fi
  return 0
}

assert_delta() {
  local file=$1
  local baseline_file=$2
  local base_url rule_count i rule_id rule_name deployment description
  local metric_match metric_keep metric_drop
  local expected_match expected_keep expected_drop
  local baseline_match baseline_keep baseline_drop
  local current_match current_keep current_drop
  local delta_match delta_keep delta_drop sum_keep_drop

  base_url=$(vm_base_url "$file")
  metric_match=$(yq_read "$file" "metrics" '.match')
  metric_keep=$(yq_read "$file" "metrics" '.keep')
  metric_drop=$(yq_read "$file" "metrics" '.drop')
  rule_count=$(yq e '.rules | length' "$file")

  if [[ ! -f "$baseline_file" ]]; then
    echo "Baseline file not found: $baseline_file (run record-baseline before trigger)" >&2
    exit 1
  fi

  echo "Asserting sampling trace metric deltas using baseline ${baseline_file}"
  echo "VictoriaMetrics: ${base_url}"

  for ((i = 0; i < rule_count; i++)); do
    rule_id=$(resolve_rule_id "$file" "$i")
    rule_name=$(yq e ".rules[${i}].ruleName" "$file")
    deployment=$(yq e ".rules[${i}].deployment" "$file")
    description=$(yq e ".rules[${i}].description // \"\"" "$file")
    expected_match=$(yq e ".rules[${i}].expected.match" "$file")
    expected_keep=$(yq e ".rules[${i}].expected.keep" "$file")
    expected_drop=$(yq e ".rules[${i}].expected.drop" "$file")
    baseline_match=$(jq -r --arg id "$rule_id" '.[$id].match // 0' "$baseline_file")
    baseline_keep=$(jq -r --arg id "$rule_id" '.[$id].keep // 0' "$baseline_file")
    baseline_drop=$(jq -r --arg id "$rule_id" '.[$id].drop // 0' "$baseline_file")

    echo "Rule [${deployment}/${rule_name}] id=${rule_id}"
    if [[ -n "$description" && "$description" != "null" ]]; then
      echo "  ${description}"
    fi

    current_match=$(sum_metric_for_rule_id "$file" "$base_url" "$metric_match" "$rule_id")
    current_keep=$(sum_metric_for_rule_id "$file" "$base_url" "$metric_keep" "$rule_id")
    current_drop=$(sum_metric_for_rule_id "$file" "$base_url" "$metric_drop" "$rule_id")
    delta_match=$(metric_delta "$current_match" "$baseline_match")
    delta_keep=$(metric_delta "$current_keep" "$baseline_keep")
    delta_drop=$(metric_delta "$current_drop" "$baseline_drop")
    sum_keep_drop=$((delta_keep + delta_drop))

    if ! assert_counter_delta "match" "$baseline_match" "$current_match" "$delta_match" "$expected_match"; then
      exit 1
    fi

    echo "  keep+drop: keep_delta=${delta_keep} + drop_delta=${delta_drop} = ${sum_keep_drop} (match_delta=${delta_match})"
    if [[ "$sum_keep_drop" -ne "$delta_match" ]]; then
      echo "FAILED: keep+drop delta (${sum_keep_drop}) != match delta (${delta_match})" >&2
      exit 1
    fi

    if [[ -n "$expected_keep" && "$expected_keep" != "null" ]]; then
      assert_counter_delta "keep" "$baseline_keep" "$current_keep" "$delta_keep" "$expected_keep" ||
        exit 1
    else
      echo "  keep: value not asserted (probabilistic / cost-reduction rule)"
    fi
    if [[ -n "$expected_drop" && "$expected_drop" != "null" ]]; then
      assert_counter_delta "drop" "$baseline_drop" "$current_drop" "$delta_drop" "$expected_drop" ||
        exit 1
    else
      echo "  drop: value not asserted (probabilistic / cost-reduction rule)"
    fi
    echo "PASSED"
  done
}

if ! command -v yq &>/dev/null; then
  echo "yq command not found. Please install yq." >&2
  exit 1
fi

if ! command -v jq &>/dev/null; then
  echo "jq command not found. Please install jq." >&2
  exit 1
fi

if [[ $# -lt 2 ]]; then
  cat >&2 <<'EOF'
Usage:
  victoriametrics_sampling_metrics_runner.sh record-baseline <test.yaml> [baseline.json]
  victoriametrics_sampling_metrics_runner.sh assert-delta <test.yaml> [baseline.json]
EOF
  exit 1
fi

COMMAND=$1
TEST_FILE=$2
BASELINE_FILE=${3:-./.trace-sampling-metrics-baseline.json}

kind=$(yq e '.kind // ""' "$TEST_FILE")
if [[ "$kind" != "SamplingTraceMetricsTest" ]]; then
  echo "Unsupported kind '$kind' in $TEST_FILE (expected SamplingTraceMetricsTest)" >&2
  exit 1
fi

case "$COMMAND" in
record-baseline)
  record_baseline "$TEST_FILE" "$BASELINE_FILE"
  ;;
assert-delta)
  assert_delta "$TEST_FILE" "$BASELINE_FILE"
  ;;
*)
  echo "Unknown command: $COMMAND" >&2
  exit 1
  ;;
esac
