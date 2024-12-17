package config

import (
	"errors"

	"github.com/odigos-io/odigos/common"
)

const (
	JaegerUrlKey = "JAEGER_URL"
	JaegerCa     = "JAEGER_CA"
)

var (
	ErrorJaegerMissingURL        = errors.New("Jaeger is missing a required field (\"JAEGER_URL\"), Jaeger will not be configured")
	ErrorJaegerTracingDisabled   = errors.New("Jaeger is missing a required field (\"TRACES\"), Jaeger will not be configured")
	ErrorJaegerMetricsNotAllowed = errors.New("Jaeger has a forbidden field (\"METRICS\"), Jaeger will not be configured")
	ErrorJaegerLogsNotAllowed    = errors.New("Jaeger has a forbidden field (\"LOGS\"), Jaeger will not be configured")
)

type Jaeger struct{}

// compile time checks
var _ Configer = (*Jaeger)(nil)

func (j *Jaeger) DestType() common.DestinationType {
	// DestinationType defined in common/dests.go
	return common.JaegerDestinationType
}

func (j *Jaeger) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	config := dest.GetConfig()
	uniqueUri := "jaeger-" + dest.GetID()

	url, exists := config[JaegerUrlKey]
	if !exists {
		return ErrorJaegerMissingURL
	}

	endpoint, err := parseUnencryptedOtlpGrpcUrl(url)
	if err != nil {
		return err
	}

	// Create config for exporter

	exporterName := "otlp/" + uniqueUri
	exporterConfig := GenericMap{
		"endpoint": endpoint,
	}

	ca, exists := dest.GetConfig()[JaegerCa]
	if exists && ca != "" {
		exporterConfig["tls"] = GenericMap{
			"ca_pem": ca,
		}
	} else {
		exporterConfig["tls"] = GenericMap{
			"insecure": true,
		}
	}

	currentConfig.Exporters[exporterName] = exporterConfig

	pipelineName := "traces/" + uniqueUri
	currentConfig.Service.Pipelines[pipelineName] = Pipeline{
		Exporters: []string{exporterName},
	}

	// Apply configs to service

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/" + uniqueUri
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	} else {
		return ErrorJaegerTracingDisabled
	}

	if isMetricsEnabled(dest) {
		return ErrorJaegerMetricsNotAllowed
	}

	if isLoggingEnabled(dest) {
		return ErrorJaegerLogsNotAllowed
	}

	return nil
}
