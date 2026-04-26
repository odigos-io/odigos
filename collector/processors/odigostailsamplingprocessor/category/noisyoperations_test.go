package category

import (
	"testing"

	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func spanWithHTTPServerRoute(route string) ptrace.Span {
	traces := ptrace.NewTraces()
	span := traces.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.SetKind(ptrace.SpanKindServer)
	span.Attributes().PutStr(string(semconv.HTTPRequestMethodKey), "GET")
	span.Attributes().PutStr(string(semconv.HTTPRouteKey), route)
	return span
}

func noisyOperation(id string, disabled bool, route string, percentage *float64) commonapisampling.NoisyOperation {
	return commonapisampling.NoisyOperation{
		Id:       id,
		Disabled: disabled,
		Operation: &commonapisampling.HeadSamplingOperationMatcher{
			HttpServer: &commonapisampling.HeadSamplingHttpServerOperationMatcher{
				Method: "GET",
				Route:  route,
			},
		},
		PercentageAtMost: percentage,
	}
}

func TestEvaluateNoisyOperationsSkipsDisabledRules(t *testing.T) {
	span := spanWithHTTPServerRoute("/health")

	matched, rule := EvaluateNoisyOperations(span, []commonapisampling.NoisyOperation{
		noisyOperation("disabled-health", true, "/health", nil),
	})

	require.False(t, matched)
	require.Nil(t, rule)
}

func TestEvaluateNoisyOperationsChoosesLeastPercentageEnabledRule(t *testing.T) {
	span := spanWithHTTPServerRoute("/health")
	enabledPercentage := 10.0

	matched, rule := EvaluateNoisyOperations(span, []commonapisampling.NoisyOperation{
		noisyOperation("disabled-health", true, "/health", nil),
		noisyOperation("enabled-health", false, "/health", &enabledPercentage),
	})

	require.True(t, matched)
	require.NotNil(t, rule)
	require.Equal(t, "enabled-health", rule.Id)
}
