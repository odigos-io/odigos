package odigossamplingprocessor

import (
	"testing"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigossamplingprocessor/internal/sampling"
	"github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigossamplingprocessor/internal/sampling/testutil"

	"github.com/stretchr/testify/assert"
)

func TestRuleEngine_ShouldSample(t *testing.T) {
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
		AddResource("auth-service").
		AddSpan("GET /api", testutil.WithAttribute("http.route", "/api"), testutil.WithLatency(150*time.Millisecond)).
		Done().Build()

	assert.True(t, engine.ShouldSample(trace))
}
