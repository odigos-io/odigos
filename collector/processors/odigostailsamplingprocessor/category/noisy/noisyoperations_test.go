package noisy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
)

func TestEvaluateSkipsDisabledNoisyOperations(t *testing.T) {
	span := ptrace.NewSpan()
	span.SetKind(ptrace.SpanKindServer)
	span.Attributes().PutStr(string(semconv.HTTPRequestMethodKey), "GET")

	matched, rule := Evaluate(span, []commonapisampling.NoisyOperation{
		{
			Id:       "disabled-noisy-rule",
			Disabled: true,
			Operation: &commonapisampling.HeadSamplingOperationMatcher{
				HttpServer: &commonapisampling.HeadSamplingHttpServerOperationMatcher{
					Method: "GET",
				},
			},
		},
	})

	assert.False(t, matched)
	assert.Nil(t, rule)
}

func TestEvaluateChoosesLeastPercentageEnabledNoisyOperation(t *testing.T) {
	span := ptrace.NewSpan()
	span.SetKind(ptrace.SpanKindServer)
	span.Attributes().PutStr(string(semconv.HTTPRequestMethodKey), "GET")

	disabledPercentage := 0.0
	enabledPercentage := 25.0

	matched, rule := Evaluate(span, []commonapisampling.NoisyOperation{
		{
			Id:               "disabled-most-restrictive-rule",
			Disabled:         true,
			PercentageAtMost: &disabledPercentage,
			Operation: &commonapisampling.HeadSamplingOperationMatcher{
				HttpServer: &commonapisampling.HeadSamplingHttpServerOperationMatcher{
					Method: "GET",
				},
			},
		},
		{
			Id:               "enabled-rule",
			PercentageAtMost: &enabledPercentage,
			Operation: &commonapisampling.HeadSamplingOperationMatcher{
				HttpServer: &commonapisampling.HeadSamplingHttpServerOperationMatcher{
					Method: "GET",
				},
			},
		},
	})

	assert.True(t, matched)
	assert.NotNil(t, rule)
	assert.Equal(t, "enabled-rule", rule.Id)
}
