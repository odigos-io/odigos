package config

import (
	"errors"

	"github.com/odigos-io/odigos/common"
)

type Sentry struct{}

func (s *Sentry) DestType() common.DestinationType {
	return common.SentryDestinationType
}

func (s *Sentry) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	if !isTracingEnabled(dest) {
		return errors.New("Sentry is not enabled for any supported signals, skipping")
	}

	if isTracingEnabled(dest) {
		exporterName := "sentry/" + dest.GetName()
		currentConfig.Exporters[exporterName] = GenericMap{
			"dsn": "${DSN}",
		}

		tracesPipelineName := "traces/sentry-" + dest.GetName()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}
