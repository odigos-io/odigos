package config

import (
	"errors"
	"fmt"

	"github.com/odigos-io/odigos/common"
)

var (
	ErrorOdigosTracingDisabled   = errors.New("Odigos is missing a required field (\"TRACES\"), Odigos will not be configured")
	ErrorOdigosMetricsNotAllowed = errors.New("Odigos has a forbidden field (\"METRICS\"), Odigos will not be configured")
	ErrorOdigosLogsNotAllowed    = errors.New("Odigos has a forbidden field (\"LOGS\"), Odigos will not be configured")
)

type Odigos struct{}

// compile time checks
var _ Configer = (*Odigos)(nil)

func (j *Odigos) DestType() common.DestinationType {
	return common.OdigosDestinationType
}

func (j *Odigos) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	uniqueUri := "odigos-" + dest.GetID()

	domain := fmt.Sprintf("%s.%s", "ingester", "odigos-system")
	endpoint, err := parseOtlpGrpcUrl(domain, false)
	if err != nil {
		return nil, err
	}

	exporterName := "otlp/" + uniqueUri

	currentConfig.Exporters[exporterName] = GenericMap{
		"endpoint": endpoint,
		"tls": GenericMap{
			"insecure": true,
		},
	}

	pipelineNames := []string{}
	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/" + uniqueUri
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, tracesPipelineName)
	} else {
		return nil, ErrorJaegerTracingDisabled
	}

	if isMetricsEnabled(dest) {
		return nil, ErrorJaegerMetricsNotAllowed
	}

	if isLoggingEnabled(dest) {
		return nil, ErrorJaegerLogsNotAllowed
	}

	return pipelineNames, nil
}
