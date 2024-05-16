package config

import (
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
)

const (
	causelyUrl = "CAUSELY_URL"
)

type Causely struct{}

func (e *Causely) DestType() common.DestinationType {
	return common.CauselyDestinationType
}

func (e *Causely) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	if url, exists := dest.Spec.Data[causelyUrl]; exists {
		exporterName := "otlp/causely-" + dest.Name

		currentConfig.Exporters[exporterName] = commonconf.GenericMap{
			"endpoint": url,
			"tls": commonconf.GenericMap{
				"insecure": true,
			},
		}

		if isTracingEnabled(dest) {
			tracesPipelineName := "traces/causely-" + dest.Name
			currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
				Exporters: []string{exporterName},
			}
		}

		if isMetricsEnabled(dest) {
			logsPipelineName := "metrics/causely-" + dest.Name
			currentConfig.Service.Pipelines[logsPipelineName] = commonconf.Pipeline{
				Exporters: []string{exporterName},
			}
		}
	}
}
