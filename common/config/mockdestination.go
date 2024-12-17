package config

import (
	"fmt"
	"strconv"

	"github.com/odigos-io/odigos/common"
)

const (
	mockResponseDurationMs = "MOCK_RESPONSE_DURATION_MS"
	rejectFraction         = "MOCK_REJECT_FRACTION"
)

type Mock struct{}

func (s *Mock) DestType() common.DestinationType {
	return common.MockDestinationType
}

func (s *Mock) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	exporterName := "mockdestination/" + dest.GetID()

	responseDuration := dest.GetConfig()[mockResponseDurationMs]
	// this destination is for development, assuming the input is always valid
	rejectFractionStr := dest.GetConfig()[rejectFraction]
	reject, _ := strconv.ParseFloat(rejectFractionStr, 64)

	currentConfig.Exporters[exporterName] = GenericMap{
		"response_duration": fmt.Sprintf("%sms", responseDuration),
		// convert string to float
		"reject_fraction": reject,
	}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/mockdestination-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/mockdestination-" + dest.GetID()
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/mockdestination-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}
