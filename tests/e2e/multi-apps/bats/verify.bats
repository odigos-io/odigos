#!/usr/bin/env bats

load utilities

GO_SCOPE="go.opentelemetry.io/auto/net/http"
JAVA_SCOPE="io.opentelemetry.tomcat-7.0"
JAVA_CLIENT_SCOPE="io.opentelemetry.http-url-connection"
JS_SCOPE="@opentelemetry/instrumentation-http"

@test "all :: includes service.name in resource attributes" {
  result=$(resource_attributes_received | jq "select(.key == \"service.name\").value.stringValue" | sort | uniq)
  result_separated=$(echo $result | sed 's/\n/,/g')
  assert_equal "$result_separated" '"coupon" "frontend" "inventory" "membership" "pricing"'
}

@test "all :: includes odigos.version in resource attributes" {
  result=$(resource_attributes_received | jq -r "select(.key == \"odigos.version\").value.stringValue")

  # Count occurrences of "e2e-test"
  e2e_test_count=$(echo "$result" | grep -Fx "e2e-test" | wc -l | xargs)

  # Ensure all values match "e2e-test" by comparing counts
  total_count=$(echo "$result" | wc -l | xargs)

  assert_equal "$e2e_test_count" "$total_count"

  # Ensure there are at least 5 elements in the array (currently 5 services)
  assert_ge "$total_count" 5
}

@test "go :: emits a span name '{http.method}' (per semconv)" {
  result=$(server_span_names_for ${GO_SCOPE})
  assert_equal "$result" '"GET"'
}

@test "java :: emits a span name '{http.method} {http.route}''" {
  result=$(server_span_names_for ${JAVA_SCOPE} | sort)
  result_separated=$(echo $result | sed 's/\n/,/g')
  assert_equal "$result_separated" '"GET /price" "POST /buy"'
}

@test "js :: emits a span name '{http.method}' (per semconv)" {
  result=$(server_span_names_for ${JS_SCOPE})
  assert_equal "$result" '"POST"'
}

@test "go :: includes http.request.method attribute" {
  result=$(server_span_attributes_for ${GO_SCOPE} | jq "select(.key == \"http.request.method\").value.stringValue")
  assert_equal "$result" '"GET"'
}

@test "java :: includes http.request.method attribute" {
  result=$(server_span_attributes_for ${JAVA_SCOPE} | jq "select(.key == \"http.request.method\").value.stringValue" | sort)
  result_separated=$(echo $result | sed 's/\n/,/g')
  assert_equal "$result_separated" '"GET" "POST"'
}

@test "js :: includes http.method attribute" {
  result=$(server_span_attributes_for ${JS_SCOPE} | jq "select(.key == \"http.method\").value.stringValue")
  assert_equal "$result" '"POST"'
}

@test "go :: includes url.path attribute" {
  result=$(server_span_attributes_for ${GO_SCOPE} | jq "select(.key == \"url.path\").value.stringValue")
  assert_equal "$result" '"/isMember"'
}

@test "java :: includes url.path attributes" {
  result=$(server_span_attributes_for ${JAVA_SCOPE} | jq "select(.key == \"url.path\").value.stringValue" | sort)
  result_separated=$(echo $result | sed 's/\n/,/g')
  assert_equal "$result_separated" '"/buy" "/price"'
}

@test "js :: includes http.target attribute" {
  result=$(server_span_attributes_for ${JS_SCOPE} | jq "select(.key == \"http.target\").value.stringValue")
  assert_equal "$result" '"/apply-coupon"'
}

@test "go :: includes http.response.status_code attribute" {
  result=$(server_span_attributes_for ${GO_SCOPE} | jq "select(.key == \"http.response.status_code\").value.intValue")
  assert_equal "$result" '"200"'
}

@test "java :: includes http.response.status_code attribute" {
  result=$(server_span_attributes_for ${JAVA_SCOPE} | jq "select(.key == \"http.response.status_code\").value.intValue" | sort)
  result_separated=$(echo $result | sed 's/\n/,/g')
  assert_equal "$result_separated" '"200" "200"'
}

@test "js :: includes http.status_code attribute" {
  result=$(server_span_attributes_for ${JS_SCOPE} | jq "select(.key == \"http.status_code\").value.intValue")
  assert_equal "$result" '"200"'
}

@test "client :: includes http.response.status_code attribute" {
  result=$(client_span_attributes_for ${JAVA_CLIENT_SCOPE} | jq "select(.key == \"http.response.status_code\").value.intValue" | sort)
  result_separated=$(echo $result | sed 's/\n/,/g')
  assert_equal "$result_separated" '"200" "200" "200"'
}

@test "server :: trace ID present and valid in all spans" {
  trace_id=$(server_spans_from_scope_named ${GO_SCOPE} | jq ".traceId")
  assert_regex "$trace_id" ${MATCH_A_TRACE_ID}
  trace_ids=$(server_spans_from_scope_named ${JAVA_SCOPE} | jq ".traceId")
  while read -r line; do
    assert_regex "$line" ${MATCH_A_TRACE_ID}
  done <<< "$trace_ids"
  trace_ids=$(server_spans_from_scope_named ${JS_SCOPE} | jq ".traceId")
  while read -r line; do
    assert_regex "$line" ${MATCH_A_TRACE_ID}
  done <<< "$trace_ids"
}

@test "server :: span ID present and valid in all spans" {
  span_id=$(server_spans_from_scope_named ${GO_SCOPE} | jq ".spanId")
  assert_regex "$span_id" ${MATCH_A_SPAN_ID}
  span_ids=$(server_spans_from_scope_named ${JAVA_SCOPE} | jq ".spanId")
  while read -r line; do
    assert_regex "$line" ${MATCH_A_SPAN_ID}
  done <<< "$span_ids"
  span_ids=$(server_spans_from_scope_named ${JS_SCOPE} | jq ".spanId")
  while read -r line; do
    assert_regex "$line" ${MATCH_A_SPAN_ID}
  done <<< "$span_ids"
}

@test "server :: parent span ID present and valid in all spans" {
  parent_span_id=$(server_spans_from_scope_named ${GO_SCOPE} | jq ".parentSpanId")
  assert_regex "$parent_span_id" ${MATCH_A_SPAN_ID}
  parent_span_ids=$(server_spans_from_scope_named ${JAVA_SCOPE} | jq ".parentSpanId" | sort)
  while read -r line; do
    assert_regex "$line" ${MATCH_A_SPAN_ID}
  done <<< "$parent_span_ids"
  parent_span_ids=$(server_spans_from_scope_named ${JS_SCOPE} | jq ".parentSpanId" | sort)
  while read -r line; do
    assert_regex "$line" ${MATCH_A_SPAN_ID}
  done <<< "$parent_span_ids"
}

@test "client, server :: spans have same trace ID" {
  client_trace_id=$(client_spans_from_scope_named ${JAVA_CLIENT_SCOPE} | jq ".traceId" | uniq)
  assert_not_empty "$client_trace_id"
  server_trace_id=$(server_spans_from_scope_named ${JAVA_SCOPE} | jq ".traceId" | uniq)
  assert_not_empty "$server_trace_id"
  assert_equal "$server_trace_id" "$client_trace_id"
}

@test "client, server :: server span has client span as parent" {
  server_parent_span_ids=$(server_spans_from_scope_named ${JAVA_SCOPE} | jq ".parentSpanId" | sort)
  client_span_ids=$(client_spans_from_scope_named ${JAVA_CLIENT_SCOPE} | jq ".spanId" | sort)
  # Verify client_span_ids is contained in server_parent_span_ids
    while read -r line; do
        if [[ "$client_span_ids" != *"$line"* ]]; then
            echo "client span ID $line not found in server parent span IDs"
            exit 1
        fi
    done <<< "$server_parent_span_ids"

  # Verify Go server span has JS client span as parent
  go_parent_span_id=$(server_spans_from_scope_named ${GO_SCOPE} | jq ".parentSpanId")
  assert_not_empty "$go_parent_span_id"
  js_client_span_id=$(client_spans_from_scope_named ${JS_SCOPE} | jq ".spanId")
  assert_not_empty "$js_client_span_id"
  assert_equal "$go_parent_span_id" "$js_client_span_id"
}