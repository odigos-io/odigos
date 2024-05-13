package config

import (
	"errors"

	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
)

const (
	qwUrlKey = "QUICKWIT_URL"
)

type Quickwit struct{}

func (e *Quickwit) DestType() common.DestinationType {
	return common.QuickwitDestinationType
}

func (e *Quickwit) ModifyConfig(dest common.ExporterConfigurer, currentConfig *commonconf.Config) error {
	if url, exists := dest.GetConfig()[qwUrlKey]; exists {
		exporterName := "otlp/quickwit-" + dest.GetName()

		currentConfig.Exporters[exporterName] = commonconf.GenericMap{
			"endpoint": url,
			"tls": commonconf.GenericMap{
				"insecure": true,
			},
		}

		if isTracingEnabled(dest) {
			tracesPipelineName := "traces/quickwit-" + dest.GetName()
			currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
				Exporters: []string{exporterName},
			}
		}

		if isLoggingEnabled(dest) {
			logsPipelineName := "logs/quickwit-" + dest.GetName()
			currentConfig.Service.Pipelines[logsPipelineName] = commonconf.Pipeline{
				Exporters: []string{exporterName},
			}
		}

		return nil
	}

	return errors.New("Quickwit url not specified, gateway will not be configured for Quickwit")
}
