package config

import (
	"fmt"
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
	"strings"
)

const (
	promRWurlKey = "PROMETHEUS_REMOTEWRITE_URL"
)

type Prometheus struct{}

func (p *Prometheus) DestType() common.DestinationType {
	return common.PrometheusDestinationType
}

func (p *Prometheus) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	if url, exists := dest.Spec.Data[promRWurlKey]; exists && isMetricsEnabled(dest) {
		url := addProtocol(url)
		url = strings.TrimSuffix(url, "/api/v1/write")
		rwExporterName := "prometheusremotewrite/prometheus"
		spanMetricsProcessorName := "spanmetrics"
		currentConfig.Exporters[rwExporterName] = commonconf.GenericMap{
			"endpoint": fmt.Sprintf("%s/api/v1/write", url),
		}

		currentConfig.Service.Pipelines["metrics/prometheus"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{rwExporterName},
		}

		// Send SpanMetrics to prometheus
		currentConfig.Service.Pipelines["traces/spanmetrics"] = commonconf.Pipeline{
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
}
