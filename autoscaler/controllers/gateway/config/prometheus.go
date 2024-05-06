package config

import (
	"errors"
	"fmt"
	"strings"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
)

const (
	promRWurlKey = "PROMETHEUS_REMOTEWRITE_URL"
)

type Prometheus struct{}

func (p *Prometheus) DestType() common.DestinationType {
	return common.PrometheusDestinationType
}

func (p *Prometheus) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) error {
	url, exists := dest.Spec.Data[promRWurlKey]
	if !exists {
		return errors.New("Prometheus remote writer url not specified, gateway will not be configured for prometheus")
	}

	if !isMetricsEnabled(dest) {
		return errors.New("Metrics not enabled for prometheus destination, gateway will not be configured for prometheus")
	}

	url = addProtocol(url)
	url = strings.TrimSuffix(url, "/api/v1/write")
	rwExporterName := "prometheusremotewrite/prometheus-" + dest.Name
	spanMetricsProcessorName := "spanmetrics/prometheus-" + dest.Name
	currentConfig.Exporters[rwExporterName] = commonconf.GenericMap{
		"endpoint": fmt.Sprintf("%s/api/v1/write", url),
	}

	metricsPipelineName := "metrics/prometheus-" + dest.Name
	currentConfig.Service.Pipelines[metricsPipelineName] = commonconf.Pipeline{
		Exporters: []string{rwExporterName},
	}

	// Send SpanMetrics to prometheus
	tracesPipelineName := "traces/spanmetrics-" + dest.Name
	currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
		Processors: []string{spanMetricsProcessorName},
		Exporters:  []string{"logging"},
	}
	currentConfig.Exporters["logging"] = struct{}{} // Dummy exporter, needed only because pipeline must have an exporter
	currentConfig.Processors[spanMetricsProcessorName] = commonconf.GenericMap{
		"metrics_exporter":          rwExporterName,
		"latency_histogram_buckets": []string{"100us", "1ms", "2ms", "6ms", "10ms", "100ms", "250ms"},
		"dimensions": []commonconf.GenericMap{
			{
				"name":    "http.method",
				"default": "GET",
			},
			{
				"name": "http.status_code",
			},
		},
		"dimensions_cache_size":   1000,
		"aggregation_temporality": "AGGREGATION_TEMPORALITY_CUMULATIVE",
	}

	return nil
}
