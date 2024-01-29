package config

import (
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
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

func (l *Logzio) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	region := dest.Spec.Data["LOGZIO_REGION"]
	if isTracingEnabled(dest) {
		exporterName := "logzio/tracing-" + dest.Name
		currentConfig.Exporters[exporterName] = commonconf.GenericMap{
			"region":        region,
			"account_token": "${LOGZIO_TRACING_TOKEN}",
		}
		tracesPipelineName := "traces/logzio-" + dest.Name
		currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{exporterName},
		}
	}

	if isMetricsEnabled(dest) {
		listenerUrl := l.GetListenerUrl(region)
		exporterName := "prometheusremotewrite/logzio-" + dest.Name
		currentConfig.Exporters[exporterName] = commonconf.GenericMap{
			"endpoint": listenerUrl,
			"external_labels": commonconf.GenericMap{
				"p8s_logzio_name": "odigos",
			},
			"headers": commonconf.GenericMap{
				"authorization": "Bearer ${LOGZIO_METRICS_TOKEN}",
			},
		}
		metricsPipelineName := "metrics/logzio-" + dest.Name
		currentConfig.Service.Pipelines[metricsPipelineName] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		exporterName := "logzio/logs-" + dest.Name
		currentConfig.Exporters[exporterName] = commonconf.GenericMap{
			"region":        region,
			"account_token": "${LOGZIO_LOGS_TOKEN}",
		}
		currentConfig.Processors["attributes/logzio"] = commonconf.GenericMap{
			"actions": []commonconf.GenericMap{
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
		logsPipelineName := "logs/logzio-" + dest.Name
		currentConfig.Service.Pipelines[logsPipelineName] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch", "attributes/logzio"},
			Exporters:  []string{exporterName},
		}
	}
}
