package config

import (
	"errors"
	"fmt"

	"github.com/odigos-io/odigos/common"
)

var (
	ErrorBrontoNoSignals     = errors.New("Bronto destination does not have any signals to export")
	ErrorBrontoRegionInvalid = errors.New("Bronto region must be either \"eu\" or \"us\", Bronto will not be configured")
)

const (
	brontoRegion     = "BRONTO_REGION"
	brontoCollection = "BRONTO_COLLECTION"
)

type Bronto struct{}

func (b *Bronto) DestType() common.DestinationType {
	return common.BrontoDestinationType
}

func (b *Bronto) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	if !isTracingEnabled(dest) && !isMetricsEnabled(dest) && !isLoggingEnabled(dest) {
		return nil, ErrorBrontoNoSignals
	}

	config := dest.GetConfig()
	region, exists := config[brontoRegion]
	if !exists || region == "" {
		region = "eu"
	}
	if region != "eu" && region != "us" {
		return nil, ErrorBrontoRegionInvalid
	}

	headers := GenericMap{
		"x-bronto-api-key": SecretEnvPlaceholder("BRONTO_API_KEY", dest),
	}
	// Bronto applies the collection header to logs only, other signals ignore it.
	if collection, ok := config[brontoCollection]; ok && collection != "" {
		headers["x-bronto-collection"] = collection
	}

	// Bronto accepts metrics over OTLP/HTTP only, so a single HTTP exporter serves all signals.
	exporterName := "otlp_http/bronto-" + dest.GetID()
	currentConfig.Exporters[exporterName] = GenericMap{
		"endpoint": fmt.Sprintf("https://ingestion.%s.bronto.io", region),
		"headers":  headers,
	}

	var pipelineNames []string

	if isTracingEnabled(dest) {
		tracePipelineName := "traces/bronto-" + dest.GetID()
		currentConfig.Service.Pipelines[tracePipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, tracePipelineName)
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/bronto-" + dest.GetID()
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, metricsPipelineName)
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/bronto-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, logsPipelineName)
	}

	return pipelineNames, nil
}
