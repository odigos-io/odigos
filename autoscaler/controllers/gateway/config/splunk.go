package config

import (
	"fmt"
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
)

const (
	splunkRealm = "SPLUNK_REALM"
)

type Splunk struct{}

func (s *Splunk) DestType() common.DestinationType {
	return common.SplunkDestinationType
}

func (s *Splunk) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	if isTracingEnabled(dest) {
		if realm, exists := dest.Spec.Data[splunkRealm]; exists {
			currentConfig.Exporters["sapm"] = commonconf.GenericMap{
				"access_token": "${SPLUNK_ACCESS_TOKEN}",
				"endpoint":     fmt.Sprintf("https://ingest.%s.signalfx.com/v2/trace", realm),
			}

			currentConfig.Service.Pipelines["traces/splunk"] = commonconf.Pipeline{
				Receivers:  []string{"otlp"},
				Processors: []string{"batch"},
				Exporters:  []string{"sapm"},
			}
		}
	}
}
