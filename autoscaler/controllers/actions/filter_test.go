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
	resourceAttributes := map[string]string{
		"service.name": "my-service",
	}
	signals := []common.ObservabilitySignal{common.TracesObservabilitySignal}

	result, err := filtersConfig(attributes, resourceAttributes, signals)
	require.NoError(t, err)

	config, ok := result.(filterProcessorConfig)
	require.True(t, ok)

	assert.Equal(t, "ignore", config.ErrorMode)
	
	// Check traces config
	assert.Len(t, config.Traces.Span, 3)
	assert.Contains(t, config.Traces.Span, `IsMatch(attributes["http.method"], "GET")`)
	assert.Contains(t, config.Traces.Span, `IsMatch(attributes["http.status"], "200")`)
	assert.Contains(t, config.Traces.Span, `IsMatch(resource.attributes["service.name"], "my-service")`)
	assert.Len(t, config.Traces.SpanEvent, 3)
	
	// Metrics and logs should be zero-value
	assert.Empty(t, config.Metrics.Metric)
	assert.Empty(t, config.Metrics.DataPoint)
	assert.Empty(t, config.Logs.LogRecord)
}

func TestFiltersConfig_MetricsOnly(t *testing.T) {
	attributes := map[string]string{
		"metric.type": "counter",
	}
	resourceAttributes := map[string]string{
		"service.name": "metrics-service",
	}
	signals := []common.ObservabilitySignal{common.MetricsObservabilitySignal}

	result, err := filtersConfig(attributes, resourceAttributes, signals)
	require.NoError(t, err)

	config, ok := result.(filterProcessorConfig)
	require.True(t, ok)

	assert.Equal(t, "ignore", config.ErrorMode)
	
	// Check metrics config
	assert.Len(t, config.Metrics.Metric, 2)
	assert.Contains(t, config.Metrics.Metric, `IsMatch(attributes["metric.type"], "counter")`)
	assert.Contains(t, config.Metrics.Metric, `IsMatch(resource.attributes["service.name"], "metrics-service")`)
	assert.Len(t, config.Metrics.DataPoint, 2)
	assert.Contains(t, config.Metrics.DataPoint, `IsMatch(attributes["metric.type"], "counter")`)
	
	// Traces and logs should be zero-value
	assert.Empty(t, config.Traces.Span)
	assert.Empty(t, config.Traces.SpanEvent)
	assert.Empty(t, config.Logs.LogRecord)
}

func TestFiltersConfig_LogsOnly(t *testing.T) {
	attributes := map[string]string{
		"log.level": "error",
	}
	resourceAttributes := map[string]string{
		"service.name": "log-service",
	}
	signals := []common.ObservabilitySignal{common.LogsObservabilitySignal}

	result, err := filtersConfig(attributes, resourceAttributes, signals)
	require.NoError(t, err)

	config, ok := result.(filterProcessorConfig)
	require.True(t, ok)

	assert.Equal(t, "ignore", config.ErrorMode)
	
	// Check logs config
	assert.Len(t, config.Logs.LogRecord, 2)
	assert.Contains(t, config.Logs.LogRecord, `IsMatch(attributes["log.level"], "error")`)
	assert.Contains(t, config.Logs.LogRecord, `IsMatch(resource.attributes["service.name"], "log-service")`)
	
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
	resourceAttributes := map[string]string{
		"service.namespace": "default",
	}
	signals := []common.ObservabilitySignal{
		common.TracesObservabilitySignal,
		common.MetricsObservabilitySignal,
		common.LogsObservabilitySignal,
	}

	result, err := filtersConfig(attributes, resourceAttributes, signals)
	require.NoError(t, err)

	config, ok := result.(filterProcessorConfig)
	require.True(t, ok)

	assert.Equal(t, "ignore", config.ErrorMode)
	
	expectedFilters := []string{
		`IsMatch(attributes["env"], "production")`,
		`IsMatch(resource.attributes["service.namespace"], "default")`,
	}
	
	// All signals should have the same filters
	assert.Len(t, config.Traces.Span, 2)
	assert.ElementsMatch(t, expectedFilters, config.Traces.Span)
	assert.Len(t, config.Traces.SpanEvent, 2)
	assert.ElementsMatch(t, expectedFilters, config.Traces.SpanEvent)
	
	assert.Len(t, config.Metrics.Metric, 2)
	assert.ElementsMatch(t, expectedFilters, config.Metrics.Metric)
	assert.Len(t, config.Metrics.DataPoint, 2)
	assert.ElementsMatch(t, expectedFilters, config.Metrics.DataPoint)
	
	assert.Len(t, config.Logs.LogRecord, 2)
	assert.ElementsMatch(t, expectedFilters, config.Logs.LogRecord)
}

func TestFiltersConfig_EmptyAttributes(t *testing.T) {
	attributes := map[string]string{}
	resourceAttributes := map[string]string{}
	signals := []common.ObservabilitySignal{common.TracesObservabilitySignal}

	result, err := filtersConfig(attributes, resourceAttributes, signals)
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
	resourceAttributes := map[string]string{
		"resource.key": "resource.value",
	}
	signals := []common.ObservabilitySignal{}

	result, err := filtersConfig(attributes, resourceAttributes, signals)
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

func TestFiltersConfig_MultipleAttributesAndResourceAttributes(t *testing.T) {
	attributes := map[string]string{
		"attr1": "value1",
		"attr2": "value2",
		"attr3": "value3",
	}
	resourceAttributes := map[string]string{
		"res1": "resvalue1",
		"res2": "resvalue2",
	}
	signals := []common.ObservabilitySignal{common.TracesObservabilitySignal}

	result, err := filtersConfig(attributes, resourceAttributes, signals)
	require.NoError(t, err)

	config, ok := result.(filterProcessorConfig)
	require.True(t, ok)

	// Should have 5 total filters (3 attributes + 2 resource attributes)
	assert.Len(t, config.Traces.Span, 5)
	
	// Check attributes
	assert.Contains(t, config.Traces.Span, `IsMatch(attributes["attr1"], "value1")`)
	assert.Contains(t, config.Traces.Span, `IsMatch(attributes["attr2"], "value2")`)
	assert.Contains(t, config.Traces.Span, `IsMatch(attributes["attr3"], "value3")`)
	
	// Check resource attributes
	assert.Contains(t, config.Traces.Span, `IsMatch(resource.attributes["res1"], "resvalue1")`)
	assert.Contains(t, config.Traces.Span, `IsMatch(resource.attributes["res2"], "resvalue2")`)
}

func TestFiltersConfig_TracesAndMetrics(t *testing.T) {
	attributes := map[string]string{
		"common.attr": "common.value",
	}
	resourceAttributes := map[string]string{}
	signals := []common.ObservabilitySignal{
		common.TracesObservabilitySignal,
		common.MetricsObservabilitySignal,
	}

	result, err := filtersConfig(attributes, resourceAttributes, signals)
	require.NoError(t, err)

	config, ok := result.(filterProcessorConfig)
	require.True(t, ok)

	expectedFilter := `IsMatch(attributes["common.attr"], "common.value")`
	
	// Traces and metrics should both have the filter
	assert.Contains(t, config.Traces.Span, expectedFilter)
	assert.Contains(t, config.Traces.SpanEvent, expectedFilter)
	assert.Contains(t, config.Metrics.Metric, expectedFilter)
	assert.Contains(t, config.Metrics.DataPoint, expectedFilter)
	
	// Logs should be empty
	assert.Empty(t, config.Logs.LogRecord)
}

