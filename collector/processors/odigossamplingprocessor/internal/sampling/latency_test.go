package sampling

import (
	"testing"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigossamplingprocessor/internal/sampling/testutil"
	"github.com/stretchr/testify/assert"
)

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
	assert.Equal(t, 0.0, fallback)
}
