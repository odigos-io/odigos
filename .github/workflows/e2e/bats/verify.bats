#!/usr/bin/env bats

load utilities

GO_SCOPE="go.opentelemetry.io/auto/net/http"
JAVA_SCOPE="io.opentelemetry.tomcat-7.0"
JS_SCOPE="@opentelemetry/instrumentation-http"

@test "all :: includes service.name in resource attributes" {
  result=$(resource_attributes_received | jq "select(.key == \"service.name\").value.stringValue") | sort
  result_separated=$(echo $result | sed 's/\n/,/g')
  assert_equal "$result_separated" '"frontend" "pricing" "membership" "inventory" "coupon"'
}

@test "go :: emits a span name '{http.method}' (per semconv)" {
  result=$(server_span_names_for ${GO_SCOPE})
  assert_equal "$result" '"GET"'
}

@test "java :: emits a span name '{http.method} {http.route}''" {
  result=$(server_span_names_for ${JAVA_SCOPE}) | sort
  result_separated=$(echo $result | sed 's/\n/,/g')
  assert_equal "$result_separated" '"GET /price" "POST /buy"'
}

@test "js :: emits a span name '{http.method}' (per semconv)" {
  result=$(server_span_names_for ${JS_SCOPE})
  assert_equal "$result" '"POST"'
}

@test "go :: includes http.method attribute" {
  result=$(server_span_attributes_for ${GO_SCOPE} | jq "select(.key == \"http.method\").value.stringValue")
  assert_equal "$result" '"GET"'
}

@test "java :: includes http.method attribute" {
  result=$(server_span_attributes_for ${JAVA_SCOPE} | jq "select(.key == \"http.method\").value.stringValue") | sort
  result_separated=$(echo $result | sed 's/\n/,/g')
  assert_equal "$result_separated" '"GET" "POST"'
}

@test "js :: includes http.method attribute" {
  result=$(server_span_attributes_for ${GO_SCOPE} | jq "select(.key == \"http.method\").value.stringValue")
  assert_equal "$result" '"GET"'
}

@test "go :: includes http.target  attribute" {
  result=$(server_span_attributes_for ${GO_SCOPE} | jq "select(.key == \"http.target\").value.stringValue")
  assert_equal "$result" '"/isMember"'
}

@test "java :: includes http.target attribute" {
  result=$(server_span_attributes_for ${JAVA_SCOPE} | jq "select(.key == \"http.target\").value.stringValue") | sort
  result_separated=$(echo $result | sed 's/\n/,/g')
  assert_equal "$result_separated" '"/price?id=123" "/buy?id=123"'
}

@test "js :: includes http.target attribute" {
  result=$(server_span_attributes_for ${JS_SCOPE} | jq "select(.key == \"http.target\").value.stringValue")
  assert_equal "$result" '"/apply-coupon"'
}

@test "server :: includes hhttp.response.status_code attribute" {
  result=$(server_span_attributes_for ${SCOPE} | jq "select(.key == \"http.response.status_code\").value.intValue")
  assert_equal "$result" '"200"'
}

@test "client :: includes http.response.status_code attribute" {
  result=$(client_span_attributes_for ${SCOPE} | jq "select(.key == \"http.response.status_code\").value.intValue")
  assert_equal "$result" '"200"'
}

@test "server :: trace ID present and valid in all spans" {
  trace_id=$(server_spans_from_scope_named ${SCOPE} | jq ".traceId")
  assert_regex "$trace_id" ${MATCH_A_TRACE_ID}
}

@test "server :: span ID present and valid in all spans" {
  span_id=$(server_spans_from_scope_named ${SCOPE} | jq ".spanId")
  assert_regex "$span_id" ${MATCH_A_SPAN_ID}
}

@test "server :: parent span ID present and valid in all spans" {
  parent_span_id=$(server_spans_from_scope_named ${SCOPE} | jq ".parentSpanId")
  assert_regex "$parent_span_id" ${MATCH_A_SPAN_ID}
}

@test "client, server :: spans have same trace ID" {
  client_trace_id=$(server_spans_from_scope_named ${SCOPE} | jq ".traceId")
  server_trace_id=$(client_spans_from_scope_named ${SCOPE} | jq ".traceId")
  assert_equal "$server_trace_id" "$client_trace_id"
}

@test "client, server :: server span has client span as parent" {
  server_parent_span_id=$(server_spans_from_scope_named ${SCOPE} | jq ".parentSpanId")
  client_span_id=$(client_spans_from_scope_named ${SCOPE} | jq ".spanId")
  assert_equal "$client_span_id" "$server_parent_span_id"
}

@test "server :: expected (redacted) trace output" {
  redact_json
  assert_equal "$(git --no-pager diff ${BATS_TEST_DIRNAME}/traces.json)" ""
}