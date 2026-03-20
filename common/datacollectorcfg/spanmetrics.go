package datacollectorcfg

import (
	"maps"
	"slices"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const (
	SpanMetricsConnectorName         = "spanmetrics"
	SpanMetricsTracesInConnectorName = "forward/trace/spanmetrics"
	SpanMetricsPipelineName          = "traces/spanmetrics"
	SpanMetricsExportingPipelineName = "metrics/spanmetrics-exporting"

	spanMetricsResourceRemoveDimensionsProcessorName = "resource/spanmetrics/remove-dimensions"
	spanMetricsCopyScopeSpanMetricsProcessorName     = "transform/copy-scope-span-metrics"

	SpanInstrumentationScopeNameAttributeName = "span.instrumentation.scope.name"
)

var SpanMetricsDefaultDimensions = []string{
	"http.method",
	string(semconv.HTTPRequestMethodKey),
	"http.status_code",
	string(semconv.HTTPResponseStatusCodeKey),
	string(semconv.HTTPRouteKey),
	SpanInstrumentationScopeNameAttributeName,
}

var SpanMetricsAlwaysExcludedResourceAttributes = []string{
	string(semconv.TelemetrySDKNameKey),
	string(semconv.TelemetrySDKVersionKey),
}

var SpanMetricsProcessResourceAttributes = []string{
	string(semconv.ProcessCommandKey),
	string(semconv.ProcessCommandArgsKey),
	string(semconv.ProcessExecutableNameKey),
	string(semconv.ProcessExecutablePathKey),
	string(semconv.ProcessPIDKey),
	string(semconv.ProcessVpidKey),
	string(semconv.ProcessParentPIDKey),
}

func GetSpanMetricsConnectorConfig(spanMetricsConfig common.MetricsSourceSpanMetricsConfiguration) config.GenericMap {
	histogramConfig := config.GenericMap{}

	if spanMetricsConfig.HistogramDisabled {
		histogramConfig["disable"] = true
	} else if spanMetricsConfig.ExplicitHistogramBuckets != nil {
		histogramConfig["explicit"] = config.GenericMap{
			"buckets": spanMetricsConfig.ExplicitHistogramBuckets,
		}
	}

	dimensionsAttributeNames := append([]string{}, SpanMetricsDefaultDimensions...)
	if len(spanMetricsConfig.AdditionalDimensions) > 0 {
		dimensionsAttributeNames = append(dimensionsAttributeNames, spanMetricsConfig.AdditionalDimensions...)
	}

	dimensions := make([]config.GenericMap, len(dimensionsAttributeNames))
	for i, name := range dimensionsAttributeNames {
		dimensions[i] = config.GenericMap{"name": name}
	}

	cfg := config.GenericMap{
		"histogram":  histogramConfig,
		"dimensions": dimensions,
		"exemplars": config.GenericMap{
			"enabled": false,
		},
		"aggregation_temporality": "AGGREGATION_TEMPORALITY_CUMULATIVE",
		"metrics_flush_interval":  spanMetricsConfig.Interval,
		"metrics_expiration":      spanMetricsConfig.MetricsExpiration,
		"events": config.GenericMap{
			"enabled": true,
			"dimensions": []config.GenericMap{
				{"name": "exception.type"},
				{"name": "exception.message"},
			},
		},
	}

	if len(spanMetricsConfig.ResourceMetricsKeyAttributes) > 0 {
		cfg["resource_metrics_key_attributes"] = spanMetricsConfig.ResourceMetricsKeyAttributes
	}

	return cfg
}

func GetSpanMetricsConnectors(spanMetricsConfig common.MetricsSourceSpanMetricsConfiguration) config.GenericMap {
	return config.GenericMap{
		SpanMetricsConnectorName:         GetSpanMetricsConnectorConfig(spanMetricsConfig),
		SpanMetricsTracesInConnectorName: &config.GenericMap{},
	}
}

func GetSpanMetricsPipelineProcessors(spanMetricsConfig common.MetricsSourceSpanMetricsConfiguration) (config.GenericMap, []string) {
	processors := config.GenericMap{}
	processorNames := []string{}

	resourceAttrToExclude := append([]string{}, SpanMetricsAlwaysExcludedResourceAttributes...)

	if spanMetricsConfig.IncludedProcessInDimensions == nil || !*spanMetricsConfig.IncludedProcessInDimensions {
		resourceAttrToExclude = append(resourceAttrToExclude, SpanMetricsProcessResourceAttributes...)
	}

	if len(spanMetricsConfig.ExcludedResourceAttributes) > 0 {
		resourceAttrToExclude = append(resourceAttrToExclude, spanMetricsConfig.ExcludedResourceAttributes...)
	}

	slices.Sort(resourceAttrToExclude)
	resourceAttrToExclude = slices.Compact(resourceAttrToExclude)

	processors[spanMetricsCopyScopeSpanMetricsProcessorName] = config.GenericMap{
		"trace_statements": []config.GenericMap{
			{
				"context": "span",
				"statements": []string{
					"set(span.attributes[\"" + SpanInstrumentationScopeNameAttributeName + "\"], instrumentation_scope.name)",
				},
			},
		},
	}
	processorNames = append(processorNames, spanMetricsCopyScopeSpanMetricsProcessorName)

	attributes := make([]config.GenericMap, 0, len(resourceAttrToExclude))
	for _, attributeName := range resourceAttrToExclude {
		attributes = append(attributes, config.GenericMap{
			"key":    attributeName,
			"action": "delete",
		})
	}

	processors[spanMetricsResourceRemoveDimensionsProcessorName] = config.GenericMap{
		"attributes": attributes,
	}
	processorNames = append(processorNames, spanMetricsResourceRemoveDimensionsProcessorName)

	return processors, processorNames
}

// ApplySpanMetrics wires span metrics connectors, processors, and pipelines
// into the collector config. It adds the forward connector to all existing
// traces/* pipelines and creates the spanmetrics processing + exporting pipelines.
func ApplySpanMetrics(cfg *config.Config, spanMetricsConfig common.MetricsSourceSpanMetricsConfiguration, metricsExporters []string) {
	spanMetricsConnectors := GetSpanMetricsConnectors(spanMetricsConfig)
	processors, processorNames := GetSpanMetricsPipelineProcessors(spanMetricsConfig)

	if cfg.Connectors == nil {
		cfg.Connectors = config.GenericMap{}
	}
	maps.Copy(cfg.Connectors, spanMetricsConnectors)

	if cfg.Processors == nil {
		cfg.Processors = config.GenericMap{}
	}
	maps.Copy(cfg.Processors, processors)

	for pipelineName, pipeline := range cfg.Service.Pipelines {
		if !strings.HasPrefix(pipelineName, "traces/") {
			continue
		}
		if !slices.Contains(pipeline.Exporters, SpanMetricsTracesInConnectorName) {
			pipeline.Exporters = append(pipeline.Exporters, SpanMetricsTracesInConnectorName)
			cfg.Service.Pipelines[pipelineName] = pipeline
		}
	}

	cfg.Service.Pipelines[SpanMetricsPipelineName] = config.Pipeline{
		Receivers:  []string{SpanMetricsTracesInConnectorName},
		Processors: processorNames,
		Exporters:  []string{SpanMetricsConnectorName},
	}

	cfg.Service.Pipelines[SpanMetricsExportingPipelineName] = config.Pipeline{
		Receivers: []string{SpanMetricsConnectorName},
		Exporters: metricsExporters,
	}
}
