package config

import (
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
)

type Sentry struct{}

func (s *Sentry) DestType() common.DestinationType {
	return common.SentryDestinationType
}

func (s *Sentry) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	if isTracingEnabled(dest) {
		currentConfig.Exporters["sentry"] = commonconf.GenericMap{
			"dsn": "${DSN}",
		}

		currentConfig.Service.Pipelines["traces/sentry"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"sentry"},
		}
	}
}
