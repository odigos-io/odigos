package odigossamplingprocessor

import (
	"testing"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigossamplingprocessor/internal/sampling"
	"github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigossamplingprocessor/internal/sampling/testutil"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// TestRuleEngine_LatencyRuleSatisfied verifies that the rule engine samples
// a trace when an endpoint latency rule is satisfied.
func TestRuleEngine_LatencyRuleSatisfied(t *testing.T) {
	cfg := &Config{
		EndpointRules: []Rule{
			{
				Name: "LatencyRule",
				Type: "http_latency",
				RuleDetails: &sampling.HttpRouteLatencyRule{
					HttpRoute:             "/api",
					ServiceName:           "auth-service",
					Threshold:             100,
					FallbackSamplingRatio: 0,
				},
			},
		},
	}

	engine := NewRuleEngine(cfg, nil)

	trace := testutil.NewTrace().
		AddResource("auth-service").
		AddSpan("GET /api", testutil.WithAttribute("http.route", "/api"), testutil.WithLatency(150*time.Millisecond)).
		Done().Build()

	assert.True(t, engine.ShouldSample(trace))
}

// TestRuleEngine_ErrorRuleFallbackOnly ensures fallback is used if no error spans are present
func TestRuleEngine_ErrorRuleFallbackOnly(t *testing.T) {
	cfg := &Config{
		GlobalRules: []Rule{
			{
				Name: "ErrorRule",
				Type: "error",
				RuleDetails: &sampling.ErrorRule{
					FallbackSamplingRatio: 100,
				},
			},
		},
	}

	engine := NewRuleEngine(cfg, nil)

	trace := testutil.NewTrace().
		AddResource("some-service").
		AddSpan("do something").
		Done().Build()

	assert.True(t, engine.ShouldSample(trace))
}

// TestRuleEngine_EndpointOverridesGlobal confirms endpoint-level rules override global fallback
func TestRuleEngine_EndpointOverridesGlobal(t *testing.T) {
	cfg := &Config{
		EndpointRules: []Rule{
			{
				Name: "Latency",
				Type: "http_latency",
				RuleDetails: &sampling.HttpRouteLatencyRule{
					HttpRoute:             "/x",
					ServiceName:           "svc",
					Threshold:             50,
					FallbackSamplingRatio: 0,
				},
			},
		},
		GlobalRules: []Rule{
			{
				Name: "Errors",
				Type: "error",
				RuleDetails: &sampling.ErrorRule{
					FallbackSamplingRatio: 100,
				},
			},
		},
	}

	engine := NewRuleEngine(cfg, nil)

	trace := testutil.NewTrace().
		AddResource("svc").
		AddSpan("GET /x", testutil.WithAttribute("http.route", "/x"), testutil.WithLatency(60*time.Millisecond)).
		Done().Build()

	assert.True(t, engine.ShouldSample(trace)) // Endpoint rule satisfied
}

// TestRuleEngine_NoMatchingRules ensures nothing is sampled if no rules match and fallback is 0
func TestRuleEngine_NoMatchingRules(t *testing.T) {
	cfg := &Config{
		GlobalRules: []Rule{
			{
				Name: "ErrorOnly",
				Type: "error",
				RuleDetails: &sampling.ErrorRule{
					FallbackSamplingRatio: 0,
				},
			},
		},
	}

	engine := NewRuleEngine(cfg, nil)

	trace := testutil.NewTrace().
		AddResource("service").
		AddSpan("ok op").
		Done().Build()

	assert.False(t, engine.ShouldSample(trace))
}

// TestRuleEngine_MixedSatisfiedAndFallback confirms fallback is used if matched but not satisfied
func TestRuleEngine_MixedSatisfiedAndFallback(t *testing.T) {
	cfg := &Config{
		GlobalRules: []Rule{
			{
				Name: "ErrorRule",
				Type: "error",
				RuleDetails: &sampling.ErrorRule{
					FallbackSamplingRatio: 100,
				},
			},
		},
	}

	engine := NewRuleEngine(cfg, nil)

	trace := testutil.NewTrace().
		AddResource("no-errors").
		AddSpan("ok span").
		Done().Build()

	assert.True(t, engine.ShouldSample(trace)) // fallback 100%
}

// TestRuleEngine_MultipleEndpointRules_OneSatisfied confirms sampling if one rule is satisfied
func TestRuleEngine_MultipleEndpointRules_OneSatisfied(t *testing.T) {
	cfg := &Config{
		EndpointRules: []Rule{
			{
				Name: "Rule1",
				Type: "http_latency",
				RuleDetails: &sampling.HttpRouteLatencyRule{
					HttpRoute:             "/miss",
					ServiceName:           "svc-a",
					Threshold:             200,
					FallbackSamplingRatio: 0,
				},
			},
			{
				Name: "Rule2",
				Type: "http_latency",
				RuleDetails: &sampling.HttpRouteLatencyRule{
					HttpRoute:             "/api",
					ServiceName:           "svc-a",
					Threshold:             10,
					FallbackSamplingRatio: 0,
				},
			},
		},
	}

	engine := NewRuleEngine(cfg, nil)

	trace := testutil.NewTrace().
		AddResource("svc-a").
		AddSpan("GET /api", testutil.WithAttribute("http.route", "/api"), testutil.WithLatency(50*time.Millisecond)).
		Done().Build()

	assert.True(t, engine.ShouldSample(trace)) // Rule2 satisfied
}

// TestRuleEngine_PreferHigherLevelSatisfied ensures global satisfied wins over service fallback
func TestRuleEngine_PreferHigherLevelSatisfied(t *testing.T) {
	cfg := &Config{
		ServiceRules: []Rule{
			{
				Name: "ServiceFallback",
				Type: "error",
				RuleDetails: &sampling.ErrorRule{
					FallbackSamplingRatio: 100,
				},
			},
		},
		GlobalRules: []Rule{
			{
				Name: "GlobalError",
				Type: "error",
				RuleDetails: &sampling.ErrorRule{
					FallbackSamplingRatio: 0,
				},
			},
		},
	}

	engine := NewRuleEngine(cfg, nil)

	// Trace with an actual error → global rule satisfied → takes precedence
	trace := testutil.NewTrace().
		AddResource("any-service").
		AddSpan("bad op", testutil.WithStatus(ptrace.StatusCodeError)).
		Done().Build()

	assert.True(t, engine.ShouldSample(trace))
}

// TestRuleEngine_EmptyRules ensures engine handles config with no rules
func TestRuleEngine_EmptyRules(t *testing.T) {
	cfg := &Config{}

	engine := NewRuleEngine(cfg, nil)

	trace := testutil.NewTrace().
		AddResource("empty").
		AddSpan("noop").
		Done().Build()

	assert.True(t, engine.ShouldSample(trace))
}
