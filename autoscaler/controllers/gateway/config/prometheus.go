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
		currentConfig.Exporters[rwExporterName] = commonconf.GenericMap{
			"endpoint": fmt.Sprintf("%s/api/v1/write", url),
		}

		currentConfig.Service.Pipelines["metrics/prometheus"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{rwExporterName},
		}
	}
}
