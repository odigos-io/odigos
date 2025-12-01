package actions

import (
	"fmt"

	"github.com/odigos-io/odigos/common"
)

type filterProcessorConfig struct {
	ErrorMode string        `json:"error_mode"`
	Traces    tracesConfig  `json:"traces"`
	Metrics   metricsConfig `json:"metrics"`
	Logs      logsConfig    `json:"logs"`
}

type tracesConfig struct {
	Span      []string `json:"span"`
	SpanEvent []string `json:"spanevent"`
}

type metricsConfig struct {
	Metric    []string `json:"metric"`
	DataPoint []string `json:"datapoint"`
}

type logsConfig struct {
	LogRecord []string `json:"log_record"`
}

func filtersConfig(attributes map[string]string, resourceAttributes map[string]string, signals []common.ObservabilitySignal) (any, error) {
	config := filterProcessorConfig{
		ErrorMode: "ignore",
	}

	// Build filter lists once
	var filters []string
	for key, value := range attributes {
		filters = append(filters, fmt.Sprintf("IsMatch(attributes[\"%s\"], \"%s\")", key, value))
	}
	for key, value := range resourceAttributes {
		filters = append(filters, fmt.Sprintf("IsMatch(resource.attributes[\"%s\"], \"%s\")", key, value))
	}

	// Apply to each enabled signal
	for _, signal := range signals {
		switch signal {
		case common.TracesObservabilitySignal:
			config.Traces = tracesConfig{
				Span:      filters,
				SpanEvent: filters,
			}
		case common.MetricsObservabilitySignal:
			config.Metrics = metricsConfig{
				Metric:    filters,
				DataPoint: filters,
			}
		case common.LogsObservabilitySignal:
			config.Logs = logsConfig{
				LogRecord: filters,
			}
		}
	}

	return config, nil
}
