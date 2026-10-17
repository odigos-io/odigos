package config

import (
	"errors"

	"github.com/odigos-io/odigos/common"
)

const (
	posthogRegion       = "POSTHOG_REGION"
	posthogProjectToken = "POSTHOG_PROJECT_TOKEN"
	posthogCompression  = "POSTHOG_COMPRESSION"
)

type PostHog struct{}

func (p *PostHog) DestType() common.DestinationType {
	return common.PostHogDestinationType
}

func (p *PostHog) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	region := dest.GetConfig()[posthogRegion]
	if region == "" {
		region = "us"
	}
	if region != "us" && region != "eu" {
		return nil, errors.New("POSTHOG_REGION must be us or eu")
	}
	compression := dest.GetConfig()[posthogCompression]
	if compression == "" {
		compression = "none"
	}
	if compression != "none" && compression != "gzip" {
		return nil, errors.New("POSTHOG_COMPRESSION must be none or gzip")
	}

	endpoint := "https://" + region + ".i.posthog.com/i/v1/"
	exporterName := "otlp_http/posthog-" + dest.GetID()
	currentConfig.Exporters[exporterName] = GenericMap{
		"logs_endpoint":    endpoint + "logs",
		"metrics_endpoint": endpoint + "metrics",
		"compression":      compression,
		"headers": GenericMap{
			"Authorization": "Bearer " + SecretEnvPlaceholder(posthogProjectToken, dest),
		},
	}

	var pipelineNames []string
	if isMetricsEnabled(dest) {
		name := "metrics/posthog-" + dest.GetID()
		currentConfig.Service.Pipelines[name] = Pipeline{Exporters: []string{exporterName}}
		pipelineNames = append(pipelineNames, name)
	}
	if isLoggingEnabled(dest) {
		name := "logs/posthog-" + dest.GetID()
		currentConfig.Service.Pipelines[name] = Pipeline{Exporters: []string{exporterName}}
		pipelineNames = append(pipelineNames, name)
	}
	return pipelineNames, nil
}
