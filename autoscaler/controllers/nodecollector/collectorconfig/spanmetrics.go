package collectorconfig

import (
	"slices"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

var (
	spanMetricsConnectorName                         = "spanmetrics"
	spanMetricsTracesInConnectorName                 = "forward/trace/spanmetrics"
	spanMetricsPipelineName                          = "traces/spanmetrics"
	spanMetricsResourceRemoveDimensionsProcessorName = "resource/spanmetrics/remove-dimensions"
)

func getSpanMetricsConnectorConfig(spanMetricsConfig common.MetricsSourceSpanMetricsConfiguration) config.GenericMap {
	histogramConfig := config.GenericMap{}

	if spanMetricsConfig.HistogramDisabled {
		histogramConfig["disabled"] = true
	} else {
		if spanMetricsConfig.ExplicitHistogramBuckets != nil {
			histogramConfig["explicit"] = config.GenericMap{
				"buckets": spanMetricsConfig.ExplicitHistogramBuckets,
			}
		}
	}

	dimensionsAttributeNames := []string{
		"http.method",
		"http.request.method",
		"http.status_code",
		"http.response.status_code",
		"http.route",
	}
	if spanMetricsConfig.AdditionalDimensions != nil {
		dimensionsAttributeNames = append(dimensionsAttributeNames, spanMetricsConfig.AdditionalDimensions...)
	}

	dimensions := make([]config.GenericMap, len(dimensionsAttributeNames))
	for i, dimensionAttributeName := range dimensionsAttributeNames {
		dimensions[i] = config.GenericMap{
			"name": dimensionAttributeName,
		}
	}

	return config.GenericMap{
		"histogram": histogramConfig,
		// Taking into account changes in the semantic conventions, to support a range of instrumentation libraries
		"dimensions": dimensions,
		"exemplars": config.GenericMap{
			"enabled": true,
		},
		"dimensions_cache_size":           1000,
		"aggregation_temporality":         "AGGREGATION_TEMPORALITY_CUMULATIVE",
		"metrics_flush_interval":          spanMetricsConfig.Interval,
		"metrics_expiration":              spanMetricsConfig.MetricsExpiration,
		"resource_metrics_key_attributes": []string{"service.name", "telemetry.sdk.language", "telemetry.sdk.name"},
		"events": config.GenericMap{
			"enabled": true,
			"dimensions": []config.GenericMap{
				{
					"name": "exception.type",
				},
				{
					"name": "exception.message",
				},
			},
		},
	}
}

func getSpanMetricsConnectors(spanMetricsConfig common.MetricsSourceSpanMetricsConfiguration) config.GenericMap {
	return config.GenericMap{
		spanMetricsConnectorName:         getSpanMetricsConnectorConfig(spanMetricsConfig),
		spanMetricsTracesInConnectorName: &config.GenericMap{},
	}
}

func getSpanMetricsPipelineProcessors(spanMetricsConfig common.MetricsSourceSpanMetricsConfiguration) (config.GenericMap, []string) {

	processors := config.GenericMap{}
	processorNames := []string{}
	resourceAttrToExclude := []string{
		// always delete these two attributes, as they are just noise in span metrics
		// TODO: consider making it an opt-in configuration option one day
		string(semconv.TelemetrySDKNameKey),
		string(semconv.TelemetrySDKVersionKey),
	}

	if spanMetricsConfig.IncludedProcessInDimensions == nil || !*spanMetricsConfig.IncludedProcessInDimensions {
		// if include process is not specifically set,
		// we want by default to remove all "process.*" attributes from the resource,
		// so the span metrics dimensions will be aggregated without it.
		resourceAttrToExclude = append(resourceAttrToExclude, []string{
			string(semconv.ProcessCommandKey),
			string(semconv.ProcessCommandArgsKey),
			string(semconv.ProcessExecutableNameKey),
			string(semconv.ProcessExecutablePathKey),
			string(semconv.ProcessPIDKey),
			string(semconv.ProcessVpidKey),
			string(semconv.ProcessParentPIDKey),
		}...)
	}

	if spanMetricsConfig.ExcludedResourceAttributes != nil {
		resourceAttrToExclude = append(resourceAttrToExclude, spanMetricsConfig.ExcludedResourceAttributes...)
	}

	// remove duplicates. doing sort and compact, which might not be most efficient
	// but the list is expected to be small anyway.
	slices.Sort(resourceAttrToExclude)
	resourceAttrToExclude = slices.Compact(resourceAttrToExclude)

	attributes := []config.GenericMap{}
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

func GetSpanMetricsConfig(spanMetricsConfig common.MetricsSourceSpanMetricsConfiguration) (config.Config, []string, []string) {
	connectors := getSpanMetricsConnectors(spanMetricsConfig)
	processors, processorNames := getSpanMetricsPipelineProcessors(spanMetricsConfig)

	// this config domain api to the outside world.
	// when set, the caller also needs to:
	// - add the returned exporters to the trace pipeline
	// - add the returned recivers to the metrics pipeline
	additionalTraceExporters := []string{spanMetricsTracesInConnectorName}
	additionalMetricsRecivers := []string{spanMetricsConnectorName}
	configDomain := config.Config{
		Connectors: connectors,
		Processors: processors,
		Service: config.Service{
			Pipelines: map[string]config.Pipeline{
				spanMetricsPipelineName: {
					Receivers:  []string{spanMetricsTracesInConnectorName},
					Processors: processorNames,
					Exporters:  []string{spanMetricsConnectorName},
				},
			},
		},
	}

	return configDomain, additionalTraceExporters, additionalMetricsRecivers
}
