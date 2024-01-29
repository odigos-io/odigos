package config

import (
	"fmt"
	"strings"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	promRWurlKey = "PROMETHEUS_REMOTEWRITE_URL"
)

type Prometheus struct{}

func (p *Prometheus) DestType() common.DestinationType {
	return common.PrometheusDestinationType
}

func (p *Prometheus) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	url, exists := dest.Spec.Data[promRWurlKey]
	if !exists {
		log.Log.V(0).Info("Prometheus remote writer url not specified, gateway will not be configured for prometheus")
		return
	}

	if !isMetricsEnabled(dest) {
		log.Log.V(0).Info("Metrics not enabled for prometheus destination, gateway will not be configured for prometheus")
		return
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
		Receivers:  []string{"otlp"},
		Processors: []string{"batch"},
		Exporters:  []string{rwExporterName},
	}

	// Send SpanMetrics to prometheus
	tracesPipelineName := "traces/spanmetrics-" + dest.Name
	currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
		Receivers:  []string{"otlp"},
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
}
