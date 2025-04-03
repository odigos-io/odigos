package sampling

import (
	"testing"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigossamplingprocessor/internal/sampling/testutil"
	"github.com/stretchr/testify/assert"
)

// TestHttpRouteLatencyRule_Evaluate checks that a trace is sampled when the service and route match
// and latency exceeds the threshold.
func TestHttpRouteLatencyRule_Evaluate(t *testing.T) {
	rule := &HttpRouteLatencyRule{
		HttpRoute:             "/api",
		ServiceName:           "auth-service",
		Threshold:             100,
		FallbackSamplingRatio: 50,
	}

	trace := testutil.NewTrace().
		AddResource("auth-service").
		AddSpan("GET /api", testutil.WithAttribute("http.route", "/api"), testutil.WithLatency(150*time.Millisecond)).
		Done().Build()

	matched, satisfied, fallback := rule.Evaluate(trace)
	assert.True(t, matched)
	assert.True(t, satisfied)
	assert.Equal(t, 100.0, fallback)
}

// TestHttpRouteLatencyRule_Evaluate_LatencyBelowThreshold ensures that when latency is below the threshold,
// the trace is only sampled via fallback.
func TestHttpRouteLatencyRule_Evaluate_LatencyBelowThreshold(t *testing.T) {
	rule := &HttpRouteLatencyRule{
		HttpRoute:             "/api",
		ServiceName:           "auth-service",
		Threshold:             200,
		FallbackSamplingRatio: 25,
	}

	trace := testutil.NewTrace().
		AddResource("auth-service").
		AddSpan("GET /api", testutil.WithAttribute("http.route", "/api"), testutil.WithLatency(100*time.Millisecond)).
		Done().Build()

	matched, satisfied, fallback := rule.Evaluate(trace)
	assert.True(t, matched)
	assert.False(t, satisfied)
	assert.Equal(t, 25.0, fallback)
}

// TestHttpRouteLatencyRule_Evaluate_ServiceMismatch ensures the rule is skipped if the service doesn't match.
func TestHttpRouteLatencyRule_Evaluate_ServiceMismatch(t *testing.T) {
	rule := &HttpRouteLatencyRule{
		HttpRoute:             "/api",
		ServiceName:           "auth-service",
		Threshold:             100,
		FallbackSamplingRatio: 10,
	}

	trace := testutil.NewTrace().
		AddResource("not-auth-service").
		AddSpan("GET /api", testutil.WithAttribute("http.route", "/api"), testutil.WithLatency(300*time.Millisecond)).
		Done().Build()

	matched, satisfied, fallback := rule.Evaluate(trace)
	assert.False(t, matched)
	assert.False(t, satisfied)
	assert.Equal(t, 0.0, fallback)
}

// TestHttpRouteLatencyRule_Evaluate_EndpointMismatch ensures the rule is skipped if the endpoint doesn't match.
func TestHttpRouteLatencyRule_Evaluate_EndpointMismatch(t *testing.T) {
	rule := &HttpRouteLatencyRule{
		HttpRoute:             "/api",
		ServiceName:           "auth-service",
		Threshold:             100,
		FallbackSamplingRatio: 15,
	}

	trace := testutil.NewTrace().
		AddResource("auth-service").
		AddSpan("GET /wrong", testutil.WithAttribute("http.route", "/wrong"), testutil.WithLatency(300*time.Millisecond)).
		Done().Build()

	matched, satisfied, fallback := rule.Evaluate(trace)
	assert.False(t, matched)
	assert.False(t, satisfied)
	assert.Equal(t, 0.0, fallback)
}

// TestHttpRouteLatencyRule_Evaluate_OnlyOneMatchingServiceAndEndpoint confirms the rule triggers correctly
// even if other resources in the trace don't match.
func TestHttpRouteLatencyRule_Evaluate_OnlyOneMatchingServiceAndEndpoint(t *testing.T) {
	rule := &HttpRouteLatencyRule{
		HttpRoute:             "/api",
		ServiceName:           "gateway",
		Threshold:             50,
		FallbackSamplingRatio: 10,
	}

	trace := testutil.NewTrace().
		AddResource("gateway").
		AddSpan("GET /api", testutil.WithAttribute("http.route", "/api"), testutil.WithLatency(75*time.Millisecond)).
		Done().
		AddResource("other-service").
		AddSpan("irrelevant").
		Done().
		Build()

	matched, satisfied, fallback := rule.Evaluate(trace)
	assert.True(t, matched)
	assert.True(t, satisfied)
	assert.Equal(t, 100.0, fallback)
}

// TestHttpRouteLatencyRule_Evaluate_NoMatchingServiceOrEndpoint ensures the rule does not apply
// if neither service nor endpoint match.
func TestHttpRouteLatencyRule_Evaluate_NoMatchingServiceOrEndpoint(t *testing.T) {
	rule := &HttpRouteLatencyRule{
		HttpRoute:             "/admin",
		ServiceName:           "admin-service",
		Threshold:             200,
		FallbackSamplingRatio: 40,
	}

	trace := testutil.NewTrace().
		AddResource("metrics-service").
		AddSpan("scrape", testutil.WithAttribute("http.route", "/metrics"), testutil.WithLatency(500*time.Millisecond)).
		Done().Build()

	matched, satisfied, fallback := rule.Evaluate(trace)
	assert.False(t, matched)
	assert.False(t, satisfied)
	assert.Equal(t, 0.0, fallback)
}

// TestHttpRouteLatencyRule_Evaluate_LatencyEqualsThreshold ensures equality is treated as satisfied.
func TestHttpRouteLatencyRule_Evaluate_LatencyEqualsThreshold(t *testing.T) {
	rule := &HttpRouteLatencyRule{
		HttpRoute:             "/api",
		ServiceName:           "user-service",
		Threshold:             100,
		FallbackSamplingRatio: 20,
	}

	trace := testutil.NewTrace().
		AddResource("user-service").
		AddSpan("GET /api", testutil.WithAttribute("http.route", "/api"), testutil.WithLatency(100*time.Millisecond)).
		Done().Build()

	matched, satisfied, fallback := rule.Evaluate(trace)
	assert.True(t, matched)
	assert.True(t, satisfied)
	assert.Equal(t, 100.0, fallback)
}
