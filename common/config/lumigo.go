package config

import (
	"errors"
	"strings"

	"github.com/odigos-io/odigos/common"
)

const (
	LumigoEndpoint = "LUMIGO_ENDPOINT"
	LumigoToken    = "LUMIGO_TOKEN"
)

var (
	ErrorLumigoEndpointMissing = errors.New("Lumigo is missing a required field (\"LUMIGO_ENDPOINT\"), Lumigo will not be configured")
	ErrorLumigoTokenMissing    = errors.New("Lumigo is missing a required field (\"LUMIGO_TOKEN\"), Lumigo will not be configured")
)

type Lumigo struct{}

func (j *Lumigo) DestType() common.DestinationType {
	return common.LumigoDestinationType
}

func (j *Lumigo) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	config := dest.GetConfig()
	uniqueUri := "lumigo-" + dest.GetID()

	url, exists := config[LumigoEndpoint]
	if !exists {
		return ErrorJaegerMissingURL
	}
	if !strings.HasPrefix(url, "https://") {
		return errors.New("Lumigo Endpoint (\"LUMIGO_ENDPOINT\") malformed, HTTPS prefix is required, Lumigo will not be configured")
	}
	if strings.HasSuffix(url, "/") {
		return errors.New("Lumigo Endpoint (\"LUMIGO_ENDPOINT\") malformed, forward-slash \"/\" suffix is forbidden, Lumigo will not be configured")
	}

	exporterName := "otlphttp/" + uniqueUri
	exporterConfig := GenericMap{
		"endpoint": url,
		"headers": GenericMap{
			"Authorization": "LumigoToken ${LUMIGO_TOKEN}",
		},
	}

	currentConfig.Exporters[exporterName] = exporterConfig

	if isTracingEnabled(dest) {
		pipeName := "traces/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isMetricsEnabled(dest) {
		pipeName := "metrics/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		pipeName := "logs/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}
