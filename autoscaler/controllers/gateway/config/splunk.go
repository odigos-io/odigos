package config

import (
	"errors"
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

func (s *Splunk) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) error {
	realm, exists := dest.Spec.Data[splunkRealm]
	if !exists {
		return errors.New("Splunk realm not specified, gateway will not be configured for Splunk")
	}

	if isTracingEnabled(dest) {
		exporterName := "sapm/" + dest.Name
		currentConfig.Exporters[exporterName] = commonconf.GenericMap{
			"access_token": "${SPLUNK_ACCESS_TOKEN}",
			"endpoint":     fmt.Sprintf("https://ingest.%s.signalfx.com/v2/trace", realm),
		}

		tracesPipelineName := "traces/splunk-" + dest.Name
		currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}
