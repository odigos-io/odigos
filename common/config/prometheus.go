package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/common"
)

const (
	promRWurlKey = "PROMETHEUS_REMOTEWRITE_URL"
)

type Prometheus struct{}

func (p *Prometheus) DestType() common.DestinationType {
	return common.PrometheusDestinationType
}

func (p *Prometheus) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	uniqueUri := "prometheus-" + dest.GetID()
	config := dest.GetConfig()

	url, exists := config[promRWurlKey]
	if !exists {
		return nil, errors.New("Prometheus remote writer url not specified, gateway will not be configured for prometheus")
	}

	if !isMetricsEnabled(dest) {
		return nil, errors.New("metrics not enabled for prometheus destination, gateway will not be configured for prometheus")
	}

	url = addProtocol(url)
	url = strings.TrimSuffix(url, "/api/v1/write")
	rwExporterName := "prometheusremotewrite/" + uniqueUri
	spanMetricNames := applySpanMetricsConnector(currentConfig, uniqueUri)

	currentConfig.Exporters[rwExporterName] = GenericMap{
		"endpoint": fmt.Sprintf("%s/api/v1/write", url),
		"resource_to_telemetry_conversion": GenericMap{
			"enabled": true,
		},
		"external_labels": map[string]string{
			"job": "odigos-remote-write",
		},
	}

	resourceAttributesLabels, exists := config[prometheusResourceAttributesLabelsKey]
	processors, err := promResourceAttributesProcessors(resourceAttributesLabels, exists, uniqueUri)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to parse prometheus resource attributes labels, gateway will not be configured for prometheus"))
	}
	processorNames := []string{}
	for k, v := range processors {
		currentConfig.Processors[k] = v
		processorNames = append(processorNames, k)
	}

	metricsPipelineName := "metrics/" + uniqueUri
	currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
		Receivers:  []string{spanMetricNames.SpanMetricsConnector},
		Exporters:  []string{rwExporterName},
		Processors: processorNames,
	}

	return []string{metricsPipelineName, spanMetricNames.TracesPipeline}, nil
}
