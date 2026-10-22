package config

import (
	"strings"

	"github.com/odigos-io/odigos/common"
)

type Logzio struct{}

// logzioRegionCodes maps the listener hostnames offered by the LOGZIO_REGION
// dropdown in destinations/data/logzio.yaml to the two letter region codes that
// GetListenerUrl and the logzio exporter's "region" field both expect.
var logzioRegionCodes = map[string]string{
	"listener.logz.io":    "us",
	"listener-au.logz.io": "au",
	"listener-ca.logz.io": "ca",
	"listener-eu.logz.io": "eu",
	"listener-nl.logz.io": "nl",
	"listener-uk.logz.io": "uk",
	"listener-wa.logz.io": "wa",
}

// normalizeRegion accepts either a listener hostname, which is what the dropdown
// stores, or a two letter region code, which is what a hand written Destination
// may carry, and returns the two letter code.
func (l *Logzio) normalizeRegion(region string) string {
	region = strings.ToLower(strings.TrimSpace(region))
	if code, ok := logzioRegionCodes[region]; ok {
		return code
	}
	return region
}

// GetListenerUrl Generates logzio listener url based on aws region
func (l *Logzio) GetListenerUrl(region string) string {
	var url string
	switch region {
	case "us":
		url = "https://listener.logz.io:8053"
	case "ca":
		url = "https://listener-ca.logz.io:8053"
	case "eu":
		url = "https://listener-eu.logz.io:8053"
	case "uk":
		url = "https://listener-uk.logz.io:8053"
	case "nl":
		url = "https://listener-nl.logz.io:8053"
	case "au":
		url = "https://listener-au.logz.io:8053"
	case "wa":
		url = "https://listener-wa.logz.io:8053"
	default:
		url = "https://listener.logz.io:8053"
	}
	return url
}

func (l *Logzio) DestType() common.DestinationType {
	return common.LogzioDestinationType
}

func (l *Logzio) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	var pipelineNames []string
	region := l.normalizeRegion(dest.GetConfig()["LOGZIO_REGION"])
	if isTracingEnabled(dest) {
		exporterName := "logzio/tracing-" + dest.GetID()
		currentConfig.Exporters[exporterName] = GenericMap{
			"region":        region,
			"account_token": SecretEnvPlaceholder("LOGZIO_TRACING_TOKEN", dest),
		}
		tracesPipelineName := "traces/logzio-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, tracesPipelineName)
	}

	if isMetricsEnabled(dest) {
		listenerUrl := l.GetListenerUrl(region)
		exporterName := "prometheusremotewrite/logzio-" + dest.GetID()
		currentConfig.Exporters[exporterName] = GenericMap{
			"endpoint": listenerUrl,
			"external_labels": GenericMap{
				"p8s_logzio_name": "odigos",
			},
			"headers": GenericMap{
				"authorization": "Bearer " + SecretEnvPlaceholder("LOGZIO_METRICS_TOKEN", dest),
			},
		}
		metricsPipelineName := "metrics/logzio-" + dest.GetID()
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, metricsPipelineName)
	}

	if isLoggingEnabled(dest) {
		exporterName := "logzio/logs-" + dest.GetID()
		currentConfig.Exporters[exporterName] = GenericMap{
			"region":        region,
			"account_token": SecretEnvPlaceholder("LOGZIO_LOGS_TOKEN", dest),
		}
		currentConfig.Processors["attributes/logzio"] = GenericMap{
			"actions": []GenericMap{
				{
					"key":    "log.file.path",
					"action": "delete",
				},
				{
					"key":    "log.iostream",
					"action": "delete",
				},
				{
					"key":    "type",
					"action": "insert",
					"value":  "odigos",
				},
			},
		}
		logsPipelineName := "logs/logzio-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Processors: []string{"attributes/logzio"},
			Exporters:  []string{exporterName},
		}
		pipelineNames = append(pipelineNames, logsPipelineName)
	}

	return pipelineNames, nil
}
