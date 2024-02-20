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
	signozUrlKey = "SIGNOZ_URL"
)

type Signoz struct{}

func (s *Signoz) DestType() common.DestinationType {
	return common.SignozDestinationType
}

func (s *Signoz) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	url, exists := dest.Spec.Data[signozUrlKey]
	if !exists {
		log.Log.V(0).Info("Signoz url not specified, gateway will not be configured for Signoz")
		return
	}

	if strings.HasPrefix(url, "https://") {
		log.Log.V(0).Info("Signoz does not currently supports tls export, gateway will not be configured for Signoz")
		return
	}

	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimSuffix(url, ":4317")
	signozExporterName := "otlp/signoz-" + dest.Name
	currentConfig.Exporters[signozExporterName] = commonconf.GenericMap{
		"endpoint": fmt.Sprintf("%s:4317", url),
		"tls": commonconf.GenericMap{
			"insecure": true,
		},
	}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/signoz-" + dest.Name
		currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
			Exporters: []string{signozExporterName},
		}
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/signoz-" + dest.Name
		currentConfig.Service.Pipelines[metricsPipelineName] = commonconf.Pipeline{
			Exporters: []string{signozExporterName},
		}
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/signoz-" + dest.Name
		currentConfig.Service.Pipelines[logsPipelineName] = commonconf.Pipeline{
			Exporters: []string{signozExporterName},
		}
	}
}
