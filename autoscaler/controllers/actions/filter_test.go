package actions

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFiltersConfig_TracesOnly(t *testing.T) {
	attributes := map[string]string{
		"http.method": "GET",
		"http.status": "200",
	}
	signals := []common.ObservabilitySignal{common.TracesObservabilitySignal}

	result, err := attributeBasedFiltersConfig(attributes, signals)
	require.NoError(t, err)

	config, ok := result.(filterProcessorConfig)
	require.True(t, ok)

	assert.Equal(t, "ignore", config.ErrorMode)

	// Check traces config - literal values use equality check (2 filters per attribute: attributes + resource.attributes)
	assert.Len(t, config.Traces.Span, 4)
	assert.Contains(t, config.Traces.Span, `attributes["http.method"] == "GET"`)
	assert.Contains(t, config.Traces.Span, `resource.attributes["http.method"] == "GET"`)
	assert.Contains(t, config.Traces.Span, `attributes["http.status"] == "200"`)
	assert.Contains(t, config.Traces.Span, `resource.attributes["http.status"] == "200"`)

	// Metrics and logs should be zero-value
	assert.Empty(t, config.Metrics.Metric)
	assert.Empty(t, config.Metrics.DataPoint)
	assert.Empty(t, config.Logs.LogRecord)
}

func TestFiltersConfig_MetricsOnly(t *testing.T) {
	attributes := map[string]string{
		"metric.type": "counter",
	}
	signals := []common.ObservabilitySignal{common.MetricsObservabilitySignal}

	result, err := attributeBasedFiltersConfig(attributes, signals)
	require.NoError(t, err)

	config, ok := result.(filterProcessorConfig)
	require.True(t, ok)

	assert.Equal(t, "ignore", config.ErrorMode)

	// Check metrics config - literal values use equality check (2 filters per attribute: attributes + resource.attributes)
	assert.Len(t, config.Metrics.Metric, 2)
	assert.Contains(t, config.Metrics.Metric, `attributes["metric.type"] == "counter"`)
	assert.Contains(t, config.Metrics.Metric, `resource.attributes["metric.type"] == "counter"`)
	assert.Len(t, config.Metrics.DataPoint, 2)
	assert.Contains(t, config.Metrics.DataPoint, `attributes["metric.type"] == "counter"`)
	assert.Contains(t, config.Metrics.DataPoint, `resource.attributes["metric.type"] == "counter"`)

	// Traces and logs should be zero-value
	assert.Empty(t, config.Traces.Span)
	assert.Empty(t, config.Traces.SpanEvent)
	assert.Empty(t, config.Logs.LogRecord)
}

func TestFiltersConfig_LogsOnly(t *testing.T) {
	attributes := map[string]string{
		"log.level": "error",
	}
	signals := []common.ObservabilitySignal{common.LogsObservabilitySignal}

	result, err := attributeBasedFiltersConfig(attributes, signals)
	require.NoError(t, err)

	config, ok := result.(filterProcessorConfig)
	require.True(t, ok)

	assert.Equal(t, "ignore", config.ErrorMode)

	// Check logs config - literal values use equality check (2 filters per attribute: attributes + resource.attributes)
	assert.Len(t, config.Logs.LogRecord, 2)
	assert.Contains(t, config.Logs.LogRecord, `attributes["log.level"] == "error"`)
	assert.Contains(t, config.Logs.LogRecord, `resource.attributes["log.level"] == "error"`)

	// Traces and metrics should be zero-value
	assert.Empty(t, config.Traces.Span)
	assert.Empty(t, config.Traces.SpanEvent)
	assert.Empty(t, config.Metrics.Metric)
	assert.Empty(t, config.Metrics.DataPoint)
}

func TestFiltersConfig_AllSignals(t *testing.T) {
	attributes := map[string]string{
		"env": "production",
	}
	signals := []common.ObservabilitySignal{
		common.TracesObservabilitySignal,
		common.MetricsObservabilitySignal,
		common.LogsObservabilitySignal,
	}

	result, err := attributeBasedFiltersConfig(attributes, signals)
	require.NoError(t, err)

	config, ok := result.(filterProcessorConfig)
	require.True(t, ok)

	assert.Equal(t, "ignore", config.ErrorMode)

	// Literal values use equality check (2 filters per attribute: attributes + resource.attributes)
	expectedFilters := []string{
		`attributes["env"] == "production"`,
		`resource.attributes["env"] == "production"`,
	}

	// All signals should have the same filters
	assert.Len(t, config.Traces.Span, 2)
	assert.ElementsMatch(t, expectedFilters, config.Traces.Span)

	assert.Len(t, config.Metrics.Metric, 2)
	assert.ElementsMatch(t, expectedFilters, config.Metrics.Metric)
	assert.Len(t, config.Metrics.DataPoint, 2)
	assert.ElementsMatch(t, expectedFilters, config.Metrics.DataPoint)

	assert.Len(t, config.Logs.LogRecord, 2)
	assert.ElementsMatch(t, expectedFilters, config.Logs.LogRecord)
}

func TestFiltersConfig_EmptyAttributes(t *testing.T) {
	attributes := map[string]string{}
	signals := []common.ObservabilitySignal{common.TracesObservabilitySignal}

	result, err := attributeBasedFiltersConfig(attributes, signals)
	require.NoError(t, err)

	config, ok := result.(filterProcessorConfig)
	require.True(t, ok)

	assert.Equal(t, "ignore", config.ErrorMode)

	// Should have empty filter lists
	assert.Empty(t, config.Traces.Span)
	assert.Empty(t, config.Traces.SpanEvent)
}

func TestFiltersConfig_NoSignals(t *testing.T) {
	attributes := map[string]string{
		"key": "value",
	}
	signals := []common.ObservabilitySignal{}

	result, err := attributeBasedFiltersConfig(attributes, signals)
	require.NoError(t, err)

	config, ok := result.(filterProcessorConfig)
	require.True(t, ok)

	assert.Equal(t, "ignore", config.ErrorMode)

	// All should be zero-value since no signals provided
	assert.Empty(t, config.Traces.Span)
	assert.Empty(t, config.Traces.SpanEvent)
	assert.Empty(t, config.Metrics.Metric)
	assert.Empty(t, config.Metrics.DataPoint)
	assert.Empty(t, config.Logs.LogRecord)
}

func TestFiltersConfig_MultipleAttributes(t *testing.T) {
	attributes := map[string]string{
		"attr1": "value1",
		"attr2": "value2",
		"attr3": "value3",
	}
	signals := []common.ObservabilitySignal{common.TracesObservabilitySignal}

	result, err := attributeBasedFiltersConfig(attributes, signals)
	require.NoError(t, err)

	config, ok := result.(filterProcessorConfig)
	require.True(t, ok)

	// Should have 6 total filters (2 per attribute: attributes + resource.attributes)
	assert.Len(t, config.Traces.Span, 6)

	// Check attributes - literal values use equality check
	assert.Contains(t, config.Traces.Span, `attributes["attr1"] == "value1"`)
	assert.Contains(t, config.Traces.Span, `resource.attributes["attr1"] == "value1"`)
	assert.Contains(t, config.Traces.Span, `attributes["attr2"] == "value2"`)
	assert.Contains(t, config.Traces.Span, `resource.attributes["attr2"] == "value2"`)
	assert.Contains(t, config.Traces.Span, `attributes["attr3"] == "value3"`)
	assert.Contains(t, config.Traces.Span, `resource.attributes["attr3"] == "value3"`)
}

func TestFiltersConfig_TracesAndMetrics(t *testing.T) {
	attributes := map[string]string{
		"env": "production",
	}
	signals := []common.ObservabilitySignal{
		common.TracesObservabilitySignal,
		common.MetricsObservabilitySignal,
	}

	result, err := attributeBasedFiltersConfig(attributes, signals)
	require.NoError(t, err)

	config, ok := result.(filterProcessorConfig)
	require.True(t, ok)

	// Literal values use equality check
	expectedFilter := `attributes["env"] == "production"`
	expectedResourceFilter := `resource.attributes["env"] == "production"`

	// Traces and metrics should both have the filter
	assert.Contains(t, config.Traces.Span, expectedFilter)
	assert.Contains(t, config.Traces.Span, expectedResourceFilter)
	assert.Contains(t, config.Metrics.Metric, expectedFilter)
	assert.Contains(t, config.Metrics.Metric, expectedResourceFilter)
	assert.Contains(t, config.Metrics.DataPoint, expectedFilter)
	assert.Contains(t, config.Metrics.DataPoint, expectedResourceFilter)

	// Logs should be empty
	assert.Empty(t, config.Logs.LogRecord)
}

func TestFiltersConfig_RegexPattern(t *testing.T) {
	attributes := map[string]string{
		"http.url": ".*health.*",
	}
	signals := []common.ObservabilitySignal{common.TracesObservabilitySignal}

	result, err := attributeBasedFiltersConfig(attributes, signals)
	require.NoError(t, err)

	config, ok := result.(filterProcessorConfig)
	require.True(t, ok)

	// Regex patterns should use IsMatch (2 filters: attributes + resource.attributes)
	assert.Len(t, config.Traces.Span, 2)
	assert.Contains(t, config.Traces.Span, `IsMatch(attributes["http.url"], ".*health.*")`)
	assert.Contains(t, config.Traces.Span, `IsMatch(resource.attributes["http.url"], ".*health.*")`)
}

func TestFiltersConfig_MixedLiteralAndRegex(t *testing.T) {
	attributes := map[string]string{
		"http.method": "GET",       // literal
		"http.url":    "^/api/.*$", // regex
	}
	signals := []common.ObservabilitySignal{common.TracesObservabilitySignal}

	result, err := attributeBasedFiltersConfig(attributes, signals)
	require.NoError(t, err)

	config, ok := result.(filterProcessorConfig)
	require.True(t, ok)

	// Should have 4 filters (2 per attribute: attributes + resource.attributes)
	assert.Len(t, config.Traces.Span, 4)

	// Literal value uses equality check
	assert.Contains(t, config.Traces.Span, `attributes["http.method"] == "GET"`)
	assert.Contains(t, config.Traces.Span, `resource.attributes["http.method"] == "GET"`)

	// Regex pattern uses IsMatch
	assert.Contains(t, config.Traces.Span, `IsMatch(attributes["http.url"], "^/api/.*$")`)
	assert.Contains(t, config.Traces.Span, `IsMatch(resource.attributes["http.url"], "^/api/.*$")`)
}

func TestIsRegexPattern(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"simple", false},
		{"hello-world", false},
		{"GET", false},
		{"200", false},
		{"production", false},
		{".*", true},
		{"^start", true},
		{"end$", true},
		{"a+b", true},
		{"a?b", true},
		{"a*b", true},
		{"[abc]", true},
		{"(a|b)", true},
		{`a\d`, true},
		{"a{2,3}", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, isRegexPattern(tt.input))
		})
	}
}
