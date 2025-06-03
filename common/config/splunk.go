package config

import (
	"errors"
	"fmt"

	"github.com/odigos-io/odigos/common"
)

const (
	splunkRealm = "SPLUNK_REALM"
)

type Splunk struct{}

func (s *Splunk) DestType() common.DestinationType {
	return common.SplunkDestinationType
}

func (s *Splunk) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	realm, exists := dest.GetConfig()[splunkRealm]
	if !exists {
		return nil, errors.New("Splunk realm not specified, gateway will not be configured for Splunk")
	}
	var pipelineNames []string
	if isTracingEnabled(dest) {
		exporterName := "sapm/" + dest.GetID()
		currentConfig.Exporters[exporterName] = GenericMap{
			"access_token": "${SPLUNK_ACCESS_TOKEN}",
			"endpoint":     fmt.Sprintf("https://ingest.%s.signalfx.com/v2/trace", realm),
		}

		tracesPipelineName := "traces/splunk-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, tracesPipelineName)
	}

	return pipelineNames, nil
}
