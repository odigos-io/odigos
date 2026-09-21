package odigosconditionalattributes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor/processortest"
)

func TestCalculateUniqueNewAttributes(t *testing.T) {
	// every attribute any rule can produce has to be collected, because that is the set the global
	// default is applied to for telemetry that matches nothing.
	cfg := &Config{
		Rules: []ConditionalRule{
			{
				FieldToCheck: OTTLScopeNameKey,
				NewAttributeValueConfigurations: map[string][]NewAttributeValueConfiguration{
					"scope.http": {
						{NewAttributeName: categoryAttr, Value: "HTTP"},
						{NewAttributeName: subCategoryAttr, FromField: "http.request.method"},
					},
					"scope.database": {
						{NewAttributeName: categoryAttr, Value: "Database"},
						{NewAttributeName: "odigos.transaction.type", Value: "non-web"},
					},
				},
			},
			{
				FieldToCheck: "http.method",
				NewAttributeValueConfigurations: map[string][]NewAttributeValueConfiguration{
					"POST": {
						{NewAttributeName: "odigos.operation_type", Value: "mutation"},
						// a configuration without a target attribute cannot produce anything.
						{NewAttributeName: "", Value: "ignored"},
					},
				},
			},
		},
	}

	assert.Equal(t, map[string]struct{}{
		categoryAttr:              {},
		subCategoryAttr:           {},
		"odigos.transaction.type": {},
		"odigos.operation_type":   {},
	}, calculateUniqueNewAttributes(cfg))
}

func TestCalculateUniqueNewAttributes_WithoutRules(t *testing.T) {
	assert.Empty(t, calculateUniqueNewAttributes(&Config{}))
}

func TestFactory_TracesProcessorEnrichesSpansThroughThePipeline(t *testing.T) {
	cfg := &Config{
		GlobalDefault: globalDefault,
		Rules: []ConditionalRule{
			{
				FieldToCheck: OTTLScopeNameKey,
				NewAttributeValueConfigurations: map[string][]NewAttributeValueConfiguration{
					"scope.http": {{NewAttributeName: categoryAttr, Value: "HTTP"}},
				},
			},
		},
	}

	sink := new(consumertest.TracesSink)
	proc, err := NewFactory().CreateTraces(t.Context(), processortest.NewNopSettings(typ), cfg, sink)
	require.NoError(t, err)

	// the processor writes into the spans it is handed, so it must declare that it mutates data.
	assert.True(t, proc.Capabilities().MutatesData)

	td := newTestTraces(nil, scopeSpanInput{
		scopeName: "scope.http",
		spanAttrs: []map[string]string{{}},
	})
	require.NoError(t, proc.ConsumeTraces(t.Context(), td))

	require.Len(t, sink.AllTraces(), 1)
	assert.Equal(t, map[string]any{categoryAttr: "HTTP"}, spanAttributesAt(t, sink.AllTraces()[0], 0, 0, 0))
}

func TestFactory_MetricsProcessorEnrichesDataPointsThroughThePipeline(t *testing.T) {
	sink := new(consumertest.MetricsSink)
	proc, err := NewFactory().CreateMetrics(t.Context(), processortest.NewNopSettings(typ), metricsTestConfig(), sink)
	require.NoError(t, err)

	assert.True(t, proc.Capabilities().MutatesData)

	md := newTestMetrics(pmetric.MetricTypeGauge, nil, map[string]string{"http.method": "POST"})
	require.NoError(t, proc.ConsumeMetrics(t.Context(), md))

	require.Len(t, sink.AllMetrics(), 1)
	assert.Equal(t, []map[string]any{{
		"http.method":   "POST",
		categoryAttr:    globalDefault,
		subCategoryAttr: "mutation",
	}}, dataPointAttributes(t, sink.AllMetrics()[0]))
}
