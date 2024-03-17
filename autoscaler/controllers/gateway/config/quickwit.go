package config

import (
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
)

const (
	qwUrlKey = "QUICKWIT_URL"
)

type Quickwit struct{}

func (e *Quickwit) DestType() common.DestinationType {
	return common.QuickwitDestinationType
}

func (e *Quickwit) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	if url, exists := dest.Spec.Data[qwUrlKey]; exists {
		exporterName := "otlp/quickwit-" + dest.Name

		currentConfig.Exporters[exporterName] = commonconf.GenericMap{
			"endpoint": url,
			"tls": commonconf.GenericMap{
				"insecure": true,
			},
		}

		if isTracingEnabled(dest) {
			tracesPipelineName := "traces/quickwit-" + dest.Name
			currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
				Receivers:  []string{"otlp"},
				Processors: []string{"batch"},
				Exporters:  []string{exporterName},
			}
		}

		if isLoggingEnabled(dest) {
			currentConfig.Service.Pipelines["logs/quickwit"] = commonconf.Pipeline{
				Receivers:  []string{"otlp"},
				Processors: []string{"batch"},
				Exporters:  []string{exporterName},
			}
		}
	}
}
