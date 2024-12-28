package config

import (
	"errors"

	"github.com/odigos-io/odigos/common"
)

const (
	TracetestEndpoint = "TRACETEST_ENDPOINT"
)

var (
	ErrorTracetestEndpointMissing = errors.New("Tracetest is missing a required field (\"TRACETEST_ENDPOINT\"), Tracetest will not be configured")
)

type Tracetest struct{}

func (j *Tracetest) DestType() common.DestinationType {
	return common.TracetestDestinationType
}

func (j *Tracetest) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	config := dest.GetConfig()
	uniqueUri := "tracetest-" + dest.GetID()

	url, urlExists := config[TracetestEndpoint]
	if !urlExists {
		return ErrorTracetestEndpointMissing
	}

	endpoint, err := parseOtlpGrpcUrl(url, false)
	if err != nil {
		return err
	}

	exporterName := "otlp/" + uniqueUri
	exporterConfig := GenericMap{
		"endpoint": endpoint,
		"tls": GenericMap{
			"insecure": true,
		},
	}

	currentConfig.Exporters[exporterName] = exporterConfig

	if isTracingEnabled(dest) {
		pipeName := "traces/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}
