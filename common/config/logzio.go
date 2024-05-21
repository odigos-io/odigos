package config

import (
	"github.com/odigos-io/odigos/common"
)

type Logzio struct{}

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

func (l *Logzio) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	region := dest.GetConfig()["LOGZIO_REGION"]
	if isTracingEnabled(dest) {
		exporterName := "logzio/tracing-" + dest.GetName()
		currentConfig.Exporters[exporterName] = GenericMap{
			"region":        region,
			"account_token": "${LOGZIO_TRACING_TOKEN}",
		}
		tracesPipelineName := "traces/logzio-" + dest.GetName()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isMetricsEnabled(dest) {
		listenerUrl := l.GetListenerUrl(region)
		exporterName := "prometheusremotewrite/logzio-" + dest.GetName()
		currentConfig.Exporters[exporterName] = GenericMap{
			"endpoint": listenerUrl,
			"external_labels": GenericMap{
				"p8s_logzio_name": "odigos",
			},
			"headers": GenericMap{
				"authorization": "Bearer ${LOGZIO_METRICS_TOKEN}",
			},
		}
		metricsPipelineName := "metrics/logzio-" + dest.GetName()
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		exporterName := "logzio/logs-" + dest.GetName()
		currentConfig.Exporters[exporterName] = GenericMap{
			"region":        region,
			"account_token": "${LOGZIO_LOGS_TOKEN}",
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
		logsPipelineName := "logs/logzio-" + dest.GetName()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Processors: []string{"attributes/logzio"},
			Exporters:  []string{exporterName},
		}
	}

	return nil
}
