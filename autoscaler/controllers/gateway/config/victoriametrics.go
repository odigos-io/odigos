package config

import (
	"fmt"
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
)

const (
	victoriaMetricsUrl     = "VICTORIA_METRICS_URL"
	victoriaMetricsPort    = "VICTORIA_METRICS_PORT"
	victoriaMetricsPromApi = "VICTORIA_METRICS_PROM_API"
)

type VictoriaMetrics struct{}

func (v VictoriaMetrics) DestType() common.DestinationType {
	return common.VictoriaMetricsDestinationType
}

func (v VictoriaMetrics) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	if isMetricsEnabled(dest) {
		url, exists := dest.Spec.Data[victoriaMetricsUrl]
		if !exists {
			log.Log.V(0).Info("Victoria metrics URL not specified, gateway will not be configured for victoria metrics")
			return
		}
		promImportApi, exists := dest.Spec.Data[victoriaMetricsPromApi]
		if !exists {
			log.Log.V(0).Info("Victoria metrics prom API not specified, defaulting to /api/v1/import/prometheus")
			promImportApi = "/api/v1/import/prometheus"
		}
		promImportApi = strings.TrimPrefix(promImportApi, "/")
		url = strings.TrimSuffix(url, promImportApi)
		port, exists := dest.Spec.Data[victoriaMetricsPort]
		if !exists {
			log.Log.V(0).Info("Victoria metrics port not specified, defaulting to 8428")
			port = ":8428"
		}
		url = strings.TrimSuffix(url, port)
		url = addProtocol(url)
		currentConfig.Exporters["victoriametrics"] = commonconf.GenericMap{
			"endpoint": fmt.Sprintf("%s:%s/%s", url, port, promImportApi),
		}

		currentConfig.Service.Pipelines["metrics/victoriametrics"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"victoriametrics"},
		}
	}
}
