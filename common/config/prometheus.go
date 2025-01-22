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
	url, exists := dest.GetConfig()[promRWurlKey]
	if !exists {
		return nil, errors.New("Prometheus remote writer url not specified, gateway will not be configured for prometheus")
	}

	if !isMetricsEnabled(dest) {
		return nil, errors.New("metrics not enabled for prometheus destination, gateway will not be configured for prometheus")
	}

	url = addProtocol(url)
	url = strings.TrimSuffix(url, "/api/v1/write")
	rwExporterName := "prometheusremotewrite/prometheus-" + dest.GetID()
	spanMetricsConnectorName := "spanmetrics/prometheus-" + dest.GetID()
	currentConfig.Exporters[rwExporterName] = GenericMap{
		"endpoint": fmt.Sprintf("%s/api/v1/write", url),
	}

	metricsPipelineName := "metrics/prometheus-" + dest.GetID()
	currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
		Receivers: []string{spanMetricsConnectorName},
		Exporters: []string{rwExporterName},
	}

	// Send SpanMetrics to prometheus
	// configure a connector which will convert spans to metrics, this should ideally be configurable,
	// and available for all metrics destinations
	// TODO: this should be an action ("SpanMetrics connector")?
	tracesPipelineName := "traces/spanmetrics-" + dest.GetID()
	currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
		Exporters: []string{spanMetricsConnectorName},
	}
	currentConfig.Connectors[spanMetricsConnectorName] = GenericMap{
		"histogram": GenericMap{
			"explicit": GenericMap{
				"buckets": []string{"100us", "1ms", "2ms", "6ms", "10ms", "100ms", "250ms"},
			},
		},
		// Taking into account changes in the semantic conventions, to support a range of instrumentation libraries
		"dimensions": []GenericMap{
			{
				"name": "http.method",
			},
			{
				"name": "http.request.method",
			},
			{
				"name": "http.status_code",
			},
			{
				"name": "http.response.status_code",
			},
			{
				"name": "http.route",
			},
		},
		"exemplars": GenericMap{
			"enabled": true,
		},
		"exclude_dimensions":              []string{"status.code"},
		"dimensions_cache_size":           1000,
		"aggregation_temporality":         "AGGREGATION_TEMPORALITY_CUMULATIVE",
		"metrics_flush_interval":          "15s",
		"metrics_expiration":              "5m",
		"resource_metrics_key_attributes": []string{"service.name", "telemetry.sdk.language", "telemetry.sdk.name"},
		"events": GenericMap{
			"enabled": true,
			"dimensions": []GenericMap{
				{
					"name": "exception.type",
				},
				{
					"name": "exception.message",
				},
			},
		},
	}
	pipeLineNames := []string{metricsPipelineName, tracesPipelineName}
	return pipeLineNames, nil
}
