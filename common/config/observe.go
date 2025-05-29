package config

import (
	"github.com/odigos-io/odigos/common"
)

const (
	OBSERVE_CUSTOMER_ID = "OBSERVE_CUSTOMER_ID"
)

type Observe struct{}

func (j *Observe) DestType() common.DestinationType {
	return common.ObserveDestinationType
}

func (j *Observe) ModifyConfig(dest ExporterConfigurer, cfg *Config) ([]string, error) {
	config := dest.GetConfig()
	uniqueUri := "observe-" + dest.GetID()
	var pipelineNames []string

	customerId, exists := config[OBSERVE_CUSTOMER_ID]
	if !exists {
		return nil, errorMissingKey(OBSERVE_CUSTOMER_ID)
	}

	exporterName := "otlphttp/" + uniqueUri
	cfg.Exporters[exporterName] = GenericMap{
		"endpoint": "https://" + customerId + ".collect.observeinc.com/v2/otel",
		"headers": GenericMap{
			"Authorization": "Bearer ${OBSERVE_TOKEN}",
		},
	}

	if IsTracingEnabled(dest) {
		pipeName := "traces/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	if IsMetricsEnabled(dest) {
		pipeName := "metrics/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	if IsLoggingEnabled(dest) {
		pipeName := "logs/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	return pipelineNames, nil
}
