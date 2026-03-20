package collectorconfig

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/common/datacollectorcfg"
)

func GetSpanMetricsConfig(spanMetricsConfig common.MetricsSourceSpanMetricsConfiguration) (config.Config, []string, []string) {
	connectors := datacollectorcfg.GetSpanMetricsConnectors(spanMetricsConfig)
	processors, processorNames := datacollectorcfg.GetSpanMetricsPipelineProcessors(spanMetricsConfig)

	// this config domain api to the outside world.
	// when set, the caller also needs to:
	// - add the returned exporters to the trace pipeline
	// - add the returned recivers to the metrics pipeline
	//
	// NOTICE: temporarily bypass the normal metrics pipeline,
	// as it might add more metric resource attributes that user of span metrics do not want,
	// use the exporter directly instead.
	additionalTraceExporters := []string{datacollectorcfg.SpanMetricsTracesInConnectorName}
	additionalMetricsReceivers := []string{datacollectorcfg.SpanMetricsConnectorName}

	configDomain := config.Config{
		Connectors: connectors,
		Processors: processors,
		Service: config.Service{
			Pipelines: map[string]config.Pipeline{
				datacollectorcfg.SpanMetricsPipelineName: {
					Receivers:  []string{datacollectorcfg.SpanMetricsTracesInConnectorName},
					Processors: processorNames,
					Exporters:  []string{datacollectorcfg.SpanMetricsConnectorName},
				},
				datacollectorcfg.SpanMetricsExportingPipelineName: {
					Receivers: []string{datacollectorcfg.SpanMetricsConnectorName},
					Exporters: []string{clusterCollectorMetricsExporterName},
				},
			},
		},
	}

	return configDomain, additionalTraceExporters, additionalMetricsReceivers
}
