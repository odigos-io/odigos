package config

import (
	"github.com/odigos-io/odigos/common"
)

const (
	OBSERVE_CUSTOMER_ID = "OBSERVE_CUSTOMER_ID"
	OBSERVE_ENDPOINT    = "OBSERVE_ENDPOINT"
)

type Observe struct{}

func (j *Observe) DestType() common.DestinationType {
	return common.ObserveDestinationType
}

func (j *Observe) ModifyConfig(dest ExporterConfigurer, cfg *Config) ([]string, error) {
	config := dest.GetConfig()
	uniqueUri := "observe-" + dest.GetID()
	var pipelineNames []string

	// OBSERVE_ENDPOINT lets users provide the full collection endpoint directly (e.g. for regional
	// tenants such as "https://<customer_id>.collect.eu1.observeinc.com/v2/otel"). When it is not set,
	// the endpoint is derived from the customer ID for the default (US) region.
	endpoint := config[OBSERVE_ENDPOINT]
	if endpoint == "" {
		customerId, exists := config[OBSERVE_CUSTOMER_ID]
		if !exists {
			return nil, errorMissingKey(OBSERVE_CUSTOMER_ID)
		}
		endpoint = "https://" + customerId + ".collect.observeinc.com/v2/otel"
	}

	// Observe routes OTLP data to a Datastream using the "x-observe-target-package" header.
	// Without it the collect endpoint rejects requests with HTTP 415 "no datastream can process
	// the request". The value selects the Observe app that models the data and differs per signal,
	// so each signal needs its own exporter. See
	// https://docs.observeinc.com/docs/configure-your-own-otel-collector
	newExporter := func(signal, targetPackage string) string {
		exporterName := "otlp_http/" + signal + "-" + uniqueUri
		cfg.Exporters[exporterName] = GenericMap{
			"endpoint": endpoint,
			"headers": GenericMap{
				"authorization":            "Bearer ${OBSERVE_TOKEN}",
				"x-observe-target-package": targetPackage,
			},
		}
		return exporterName
	}

	if isTracingEnabled(dest) {
		exporterName := newExporter("traces", "Tracing")
		pipeName := "traces/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	if isMetricsEnabled(dest) {
		exporterName := newExporter("metrics", "Metrics")
		pipeName := "metrics/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	if isLoggingEnabled(dest) {
		exporterName := newExporter("logs", "Host Explorer")
		pipeName := "logs/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	return pipelineNames, nil
}
